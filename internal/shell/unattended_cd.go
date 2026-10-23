package shell

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/charmbracelet/crush/internal/filepathext"
	"mvdan.cc/sh/v3/syntax"
)

// cdTracker theo dõi thư mục của một script `sh -c` (chạy ở shell con, trình thông dịch
// của tool không thấy nó đổi thư mục) theo TỪ VỰNG như `cd` mặc định (-L) của shell: bắt
// đầu từ thư mục hiện tại của lệnh, áp từng cd/pushd/popd theo thứ tự; chặn khi một bước
// ra ngoài gốc hay đích không biết trước (biến, $(…), `-`, `~`, glob, không đích = $HOME).
//
// Mỗi lệnh cho hai thư mục: ok (lệnh thành công, vế sau && chạy ở đó) và fail (lệnh hỏng,
// vế sau || chạy ở đó; cd hỏng thì vẫn ở chỗ cũ). Lệnh sau `;` chạy ở tổ tiên chung của
// hai thư mục: ước lượng NÔNG nhất, nên `..` tính từ đó không lọt ra ngoài. "" là không
// tới được (sau `exit`). Thân if/vòng lặp/hàm chạy 0..n lần ở chỗ không biết: soi từ gốc,
// có đổi thư mục thì sau đó coi như ở gốc.
//
// shortcut: symlink tạo ngay trong script, tệp nạp bằng `source`/`.`, `eval` động, CDPATH không theo
// được; chặn hẳn phải dùng sandbox của OS.
type cdTracker struct {
	root   string // gốc theo từ vựng, cùng hệ với thư mục đang theo dõi
	real   string // gốc đã giải symlink
	reason string
	moved  bool // đã gặp cd/pushd/popd
}

// cdWords: script có nhắc lệnh đổi thư mục (dùng khi không phân tích được).
var cdWords = regexp.MustCompile(`\b(cd|pushd|popd)\b`)

// followCd soi script của `sh -c` chạy ở dir.
func followCd(g workspaceGuard, dir, script string) (string, bool) {
	dir = filepath.Clean(dir)
	t := &cdTracker{real: g.root}
	switch {
	case within(g.logical, dir):
		t.root = g.logical
	case within(g.root, dir):
		t.root = g.root
	default:
		t.root = dir // không biết dir ở đâu trong gốc: coi nó là gốc, không cho đi lên
	}
	t.script(script, dir, 0)
	return t.reason, t.reason != ""
}

func (t *cdTracker) block(reason string) {
	if t.reason == "" {
		t.reason = reason + "; not allowed while unattended"
	}
}

func (t *cdTracker) script(script, dir string, depth int) (ok, fail string) {
	if depth >= maxShellDepth {
		t.block("shell scripts nested too deep to follow whether their cd goes outside the workspace")
		return dir, dir
	}
	file, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(strings.NewReader(script), "")
	if err != nil {
		if cdWords.MatchString(script) {
			t.block("cannot parse a shell script that changes directory, so cannot tell whether it goes outside the workspace")
		}
		return dir, dir
	}
	return t.stmts(file.Stmts, dir, depth)
}

func (t *cdTracker) stmts(list []*syntax.Stmt, dir string, depth int) (ok, fail string) {
	ok, fail = dir, dir
	for _, s := range list {
		start := commonDir(ok, fail)
		if start == "" {
			break // sau exit
		}
		ok, fail = t.stmt(s, start, depth)
	}
	return ok, fail
}

func (t *cdTracker) stmt(s *syntax.Stmt, dir string, depth int) (ok, fail string) {
	for _, r := range s.Redirs {
		t.nested(r, dir, depth)
	}
	if s.Cmd == nil {
		return dir, dir
	}
	ok, fail = t.cmd(s.Cmd, dir, depth)
	switch {
	case s.Background || s.Coprocess:
		return dir, dir // chạy ở shell con
	case s.Negated:
		return fail, ok
	}
	return ok, fail
}

func (t *cdTracker) cmd(c syntax.Command, dir string, depth int) (ok, fail string) {
	switch c := c.(type) {
	case *syntax.CallExpr:
		return t.call(c, dir, depth)
	case *syntax.Block:
		return t.stmts(c.Stmts, dir, depth)
	case *syntax.Subshell:
		t.stmts(c.Stmts, dir, depth)
		return dir, dir
	case *syntax.TimeClause:
		if c.Stmt != nil {
			return t.stmt(c.Stmt, dir, depth)
		}
		return dir, dir
	case *syntax.BinaryCmd:
		switch c.Op {
		case syntax.AndStmt:
			okL, failL := t.stmt(c.X, dir, depth)
			if okL == "" {
				return "", failL
			}
			okR, failR := t.stmt(c.Y, okL, depth)
			return okR, commonDir(failL, failR)
		case syntax.OrStmt:
			okL, failL := t.stmt(c.X, dir, depth)
			if failL == "" {
				return okL, ""
			}
			okR, failR := t.stmt(c.Y, failL, depth)
			return commonDir(okL, okR), failR
		}
		// Ống: mỗi vế một shell con.
		t.stmt(c.X, dir, depth)
		t.stmt(c.Y, dir, depth)
		return dir, dir
	}
	// if/while/for/case/hàm/...: thân chạy 0..n lần, có khi ở chỗ khác: soi từng thân tuần
	// tự từ gốc (nông nhất, nên lặp lại cũng không lọt); thân có đổi thư mục thì sau khối
	// coi như ở gốc.
	before := t.moved
	t.moved = false
	t.bodies(c, depth)
	moved := t.moved
	t.moved = before || moved
	if !moved {
		return dir, dir
	}
	return t.root, t.root
}

// bodies soi thân của khối điều khiển, mỗi thân tuần tự từ gốc.
func (t *cdTracker) bodies(c syntax.Command, depth int) {
	switch c := c.(type) {
	case *syntax.IfClause:
		start := t.root
		for ; c != nil; c = c.Else {
			ok, fail := t.stmts(c.Cond, start, depth)
			t.stmts(c.Then, ok, depth)
			start = fail // elif/else chạy khi điều kiện hỏng
		}
	case *syntax.WhileClause:
		ok, _ := t.stmts(c.Cond, t.root, depth)
		t.stmts(c.Do, ok, depth)
	case *syntax.ForClause:
		t.nested(c.Loop, t.root, depth)
		t.stmts(c.Do, t.root, depth)
	case *syntax.CaseClause:
		t.nested(c.Word, t.root, depth)
		for _, item := range c.Items {
			for _, p := range item.Patterns {
				t.nested(p, t.root, depth)
			}
			t.stmts(item.Stmts, t.root, depth)
		}
	case *syntax.FuncDecl:
		if c.Body != nil {
			t.stmt(c.Body, t.root, depth)
		}
	default:
		t.nested(c, t.root, depth)
	}
}

// nested soi lệnh lồng trong n ($(…) và <(…) trong từ và chuyển hướng: shell con chạy
// tuần tự từ dir; lệnh khác lồng trong n cũng từ dir).
func (t *cdTracker) nested(n syntax.Node, dir string, depth int) {
	syntax.Walk(n, func(m syntax.Node) bool {
		switch m := m.(type) {
		case *syntax.CmdSubst:
			t.stmts(m.Stmts, dir, depth)
			return false
		case *syntax.ProcSubst:
			t.stmts(m.Stmts, dir, depth)
			return false
		case *syntax.Stmt:
			t.stmt(m, dir, depth)
			return false
		}
		return true
	})
}

func (t *cdTracker) call(c *syntax.CallExpr, dir string, depth int) (ok, fail string) {
	for _, a := range c.Assigns {
		t.nested(a, dir, depth)
	}
	argv := make([]string, len(c.Args))
	for i, w := range c.Args {
		t.nested(w, dir, depth)
		argv[i] = staticWord(w)
	}
	argv = peelLaunchers(argv)
	if len(argv) == 0 {
		return dir, dir
	}
	switch name := baseName(argv[0]); {
	case name == "cd" || name == "pushd" || name == "popd":
		t.moved = true
		return t.cd(name, argv[1:], dir)
	case name == "exit":
		return "", ""
	case name == "eval":
		if slices.Contains(argv[1:], dynamicWord) {
			// shortcut: eval của đầu ra lệnh khác (`eval "$(conda shell.bash hook)"`) không
			// theo được, như `source`; không chặn để idiom thường gặp vẫn chạy. Chữ tĩnh của
			// chính eval có cd (`eval "cd $D"`) thì chặn.
			if literalMentionsCd(c.Args) {
				t.block("eval of a cd whose target is only known at run time may go outside the workspace")
			}
			return dir, dir
		}
		return t.script(strings.Join(argv[1:], " "), dir, depth+1) // cùng shell: thư mục đi tiếp
	case shellInterpreters[name]:
		if script, _, isScript := shellScript(argv); isScript {
			t.script(script, dir, depth+1) // shell con: không đổi thư mục của shell này
		}
	}
	return dir, dir
}

// cd áp một cd/pushd/popd lên dir.
func (t *cdTracker) cd(name string, args []string, dir string) (ok, fail string) {
	target, given := "", false
	for i, a := range args {
		if a == "--" {
			if given = i+1 < len(args); given {
				target = args[i+1]
			}
			break
		}
		// -L/-P/-e/-@/-n là cờ; "-", "+N", "-N" là đích.
		if a == "-" || !strings.HasPrefix(a, "-") || isAllDigits(a[1:]) {
			target, given = a, true
			break
		}
	}
	rotate := len(target) > 1 && (target[0] == '+' || target[0] == '-')
	if name == "popd" || (name == "pushd" && (!given || rotate)) {
		// Về một thư mục đã qua (đã soát là ở trong) hay xoay ngăn xếp: ở trong, không biết chỗ nào.
		return t.root, dir
	}
	if !given || target == "-" || rotate || strings.HasPrefix(target, "~") || strings.ContainsAny(target, dynamicWord+"*?[{") {
		shown := strings.ReplaceAll(target, dynamicWord, "$…")
		if !given {
			shown = "(no directory: $HOME)"
		}
		t.block(name + " " + shown + " in a shell script goes somewhere only known at run time, possibly outside the workspace")
		return dir, dir
	}
	next := filepath.Clean(filepathext.SmartJoin(dir, target))
	if r, err := filepath.EvalSymlinks(next); err == nil {
		// Đã có: chỗ thật quyết (symlink trỏ ra ngoài bị chặn, gốc viết dạng nào cũng nhận).
		if !within(t.real, r) {
			t.block(name + " " + target + " in a shell script goes to " + r + ", outside the workspace " + t.real + ": the command left the workspace")
			return dir, dir
		}
		// Thư mục vào được thì coi cd là thành công; tệp hay thư mục không có quyền x thì
		// cd hỏng, shell ở lại dir.
		// shortcut: quyền x xét theo bit chế độ, không theo chủ sở hữu; script tự xoá thư
		// mục trước khi cd (`rm -rf x; cd x; cd ..`) thì lọt, đó không phải lối vô tình.
		if fi, err := os.Stat(next); err == nil && fi.IsDir() && (runtime.GOOS == "windows" || fi.Mode().Perm()&0o111 != 0) {
			return next, next
		}
		return next, dir
	}
	// Chưa có (script tự tạo, hay cd sẽ hỏng): so theo từ vựng với cả hai dạng của gốc.
	if !within(t.root, next) && !within(t.real, next) {
		t.block(name + " " + target + " in a shell script goes to " + next + ", outside the workspace " + t.real + ": the command left the workspace")
		return dir, dir
	}
	return next, dir
}

// literalMentionsCd: chữ tĩnh của các từ (ngoài $(…), vốn chạy ở shell con) nhắc cd/pushd/popd.
func literalMentionsCd(words []*syntax.Word) bool {
	found := false
	for _, w := range words {
		syntax.Walk(w, func(n syntax.Node) bool {
			switch n := n.(type) {
			case *syntax.CmdSubst, *syntax.ProcSubst:
				return false
			case *syntax.Lit:
				found = found || cdWords.MatchString(n.Value)
			case *syntax.SglQuoted:
				found = found || cdWords.MatchString(n.Value)
			}
			return !found
		})
	}
	return found
}

// commonDir trả tổ tiên chung gần nhất của a và b (theo từ vựng); "" (không tới được) bị bỏ qua.
func commonDir(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	}
	for !within(a, b) {
		parent := filepath.Dir(a)
		if parent == a {
			break
		}
		a = parent
	}
	return a
}
