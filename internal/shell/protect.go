package shell

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// Lớp bảo vệ tiến trình điều khiển: chặn CỨNG trong trình thực thi shell (như
// mọi BlockFunc, chạy trước cả YOLO và quyền đã cấp cho session) các lệnh dừng
// chính tiến trình agent-cli đang giữ phiên, hay tắt/khởi động lại cả máy.
//
// Sự cố 15/10: agent coi client/server agent-cli là "mồ côi" rồi chạy
// `/bin/kill -9 <client> <server>`, giết đúng tiến trình đang chạy việc. Cùng
// UID / cùng tên / bị coi là mồ côi không đủ để được giết; mục tiêu là chính
// cây tiến trình agent-cli thì chặn, không hỏi.

// processLaunchers là các lệnh bọc; lệnh thật nằm sau cờ của chúng. Trình
// thực thi shell đã tách sẵn argv từng lệnh, nên chỉ cần bóc vài lớp phổ biến.
var processLaunchers = map[string]bool{
	"env": true, "nohup": true, "timeout": true, "nice": true,
	"stdbuf": true, "setsid": true, "ionice": true, "command": true,
	"sudo": true, "doas": true, "xargs": true,
	"exec": true, "builtin": true, "time": true, "taskset": true, "busybox": true,
}

// launcherValueFlags là các cờ của lớp bọc ăn thêm một GIÁ TRỊ ở arg kế (vd
// `sudo -u root`, `xargs -I{} ...`): phải bỏ cả giá trị để không nhầm nó là
// lệnh thật. Không gồm cờ nhận giá trị số (đã bị bỏ như digit) hay cờ không
// giá trị (vd `sudo -n`).
var launcherValueFlags = map[string]bool{
	"-u": true, "--user": true, "-g": true, "--group": true,
	"-C": true, "-p": true, "-U": true, "-R": true, "-h": true,
	"-I": true, "--replace": true, "-a": true, "--arg-file": true,
	"-E": true, "--eof": true, "-d": true, "--delimiter": true,
	"-L": true, "--max-lines": true, "-P": true, "--max-procs": true,
}

// shutdownCommands tắt/khởi động lại cả máy: box chạy dịch vụ thật nên chặn
// cứng, không bao giờ tự duyệt.
var shutdownCommands = map[string]bool{
	"shutdown": true, "reboot": true, "halt": true, "poweroff": true, "telinit": true,
}

// killByPID là các lệnh dừng tiến trình theo PID.
var killByPID = map[string]bool{"kill": true}

// killByName là các lệnh dừng tiến trình theo tên/pattern.
var killByName = map[string]bool{"pkill": true, "killall": true}

// shellInterpreters chạy được script qua -c; `kill` trong script đó là builtin
// thật của shell con nên argv của nó không bao giờ tới BlockFunc.
var shellInterpreters = map[string]bool{"sh": true, "bash": true, "zsh": true, "dash": true, "ksh": true, "rbash": true}

// killWords là các từ lệnh mà guard chặn; dùng để quyết khi script có tên lệnh
// động (`$k X`) mà không thể biết nó là gì.
var killWords = regexp.MustCompile(`\b(kill|pkill|killall|shutdown|reboot|halt|poweroff|telinit)\b`)

// dynamicWord đánh dấu một từ có expansion ($x, $(...), ${...}) nên không biết
// giá trị lúc phân tích.
const dynamicWord = "\x00"

// Guard KHÔNG phủ (cố ý): script nạp từ tệp (`bash x.sh`, shebang), từ stdin
// (`echo 'kill N' | sh`, here-doc), `env -S`, `find -exec`, `script -c`, và
// ngôn ngữ khác (python/node `os.kill`). Chỉ phủ lệnh gọi trực tiếp và script
// nằm ngay trên dòng lệnh (sh/bash/... -c, eval).

// maxShellDepth là số lớp `sh -c` lồng nhau được soi; sâu hơn thì chặn luôn.
const maxShellDepth = 3

// baseName chuẩn hoá argv[0] về tên lệnh (bỏ đường dẫn, bỏ đuôi Windows).
func baseName(arg string) string {
	name := strings.ToLower(filepath.Base(arg))
	return strings.TrimSuffix(name, ".exe")
}

// peelLaunchers bỏ các lớp bọc (env FOO=1, timeout 5, nice -n 9...) để tới
// lệnh thật.
func peelLaunchers(args []string) []string {
	for len(args) > 0 && processLaunchers[baseName(args[0])] {
		args = args[1:]
		for len(args) > 0 {
			a := args[0]
			// Bỏ cờ, gán biến (A=b) và tham số số (timeout 5) của lớp bọc.
			if strings.HasPrefix(a, "-") {
				args = args[1:]
				if launcherValueFlags[a] && len(args) > 0 {
					args = args[1:] // bỏ luôn giá trị đi kèm
				}
				continue
			}
			if strings.Contains(a, "=") || isAllDigits(a) || isDuration(a) {
				args = args[1:]
				continue
			}
			break
		}
	}
	return args
}

// isDuration nhận đối số thời gian của timeout/sleep: 5, 5s, 1.5m, 2h, 1d.
var durationArg = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?[smhd]?$`)

func isDuration(s string) bool { return durationArg.MatchString(s) }

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// killPIDs trả các PID (dương) mà một lệnh `kill` nhắm tới. Bỏ cờ tín hiệu
// (-9, -TERM, -s KILL); PID âm (nhóm tiến trình) trả trị tuyệt đối vì giết
// nhóm cũng giết tiến trình trong đó.
func killPIDs(args []string) []int {
	var pids []int
	sawSignal := false // đã gặp đối số tín hiệu (-9, -s KILL) chưa
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-s" || a == "--signal" || a == "-n":
			i++ // tên/số tín hiệu ở arg kế
			sawSignal = true
		case a == "--":
			// phần còn lại là PID
		case strings.HasPrefix(a, "-"):
			// "-N" đầu tiên (chưa có tín hiệu, chưa có PID) là TÍN HIỆU như
			// `kill -9`; "-N" sau đó là PID ÂM (nhóm tiến trình) như
			// `kill -9 -946242`.
			if !sawSignal && len(pids) == 0 {
				sawSignal = true
				continue
			}
			if n, err := strconv.Atoi(strings.TrimPrefix(a, "-")); err == nil && n > 0 {
				pids = append(pids, n)
			}
		case strings.HasPrefix(a, "%"):
			// job spec của shell, không phải PID hệ thống
		default:
			if n, err := strconv.Atoi(a); err == nil && n > 0 {
				pids = append(pids, n)
			}
		}
	}
	return pids
}

// killNames trả các pattern/tên mà pkill/killall nhắm tới (bỏ cờ).
func killNames(args []string) []string {
	var names []string
	for i := 1; i < len(args); i++ {
		a := args[i]
		if a == "-s" || a == "--signal" || a == "-u" || a == "--user" || a == "-U" {
			i++
			continue
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		names = append(names, a)
	}
	return names
}

// shellScript trả script đứng sau cờ -c của một shell (kể cả cờ gộp -lc, -ec;
// bỏ "--" đứng ngay sau) và các đối số vị trí còn lại ($0 $1...).
func shellScript(args []string) (script string, positional []string, ok bool) {
	for i := 1; i < len(args)-1; i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.Contains(a, "c") {
			j := i + 1
			if args[j] == "--" && j+1 < len(args) {
				j++
			}
			return args[j], args[j+1:], true
		}
	}
	return "", nil, false
}

// scriptCommands phân tích script như bash và trả argv của MỌI lệnh đơn trong nó
// (kể cả trong vòng lặp, if, hàm, $(...), backtick). Từ chỉ gồm chữ và nháy
// được giải thành chuỗi (`ki""ll` -> kill); từ có expansion là dynamicWord.
func scriptCommands(script string) (cmds [][]string, ok bool) {
	file, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(strings.NewReader(script), "")
	if err != nil {
		return nil, false
	}
	syntax.Walk(file, func(n syntax.Node) bool {
		call, isCall := n.(*syntax.CallExpr)
		if !isCall || len(call.Args) == 0 {
			return true
		}
		argv := make([]string, len(call.Args))
		for i, w := range call.Args {
			argv[i] = staticWord(w)
		}
		cmds = append(cmds, argv)
		return true
	})
	return cmds, true
}

func staticWord(w *syntax.Word) string {
	var b strings.Builder
	for _, part := range w.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			b.WriteString(strings.ReplaceAll(p.Value, `\`, ""))
		case *syntax.SglQuoted:
			b.WriteString(p.Value)
		case *syntax.DblQuoted:
			for _, q := range p.Parts {
				lit, isLit := q.(*syntax.Lit)
				if !isLit {
					return dynamicWord
				}
				b.WriteString(lit.Value)
			}
		default:
			return dynamicWord
		}
	}
	return b.String()
}

// killListsSignals: `kill -l` chỉ liệt kê tên tín hiệu, không giết ai.
func killListsSignals(args []string) bool {
	for _, a := range args[1:] {
		if a == "-l" || a == "-L" || a == "--list" || a == "--table" {
			return true
		}
	}
	return false
}

// ProcessGuard quyết định một PID hay tên tiến trình có được agent-cli bảo vệ
// (không được tự giết). Tách ra để test thay bằng đồ giả.
type ProcessGuard struct {
	// ProtectedPID cho biết pid thuộc cây tiến trình agent-cli (chính nó,
	// tổ tiên, hoặc tiến trình khác chạy cùng binary).
	ProtectedPID func(pid int) bool
	// ProtectedName cho biết tên/pattern của pkill/killall khớp tiến trình
	// agent-cli.
	ProtectedName func(name string) bool
}

// Blocked báo lệnh args có bị chặn không, và lý do để ghi log.
func (g ProcessGuard) Blocked(args []string) (reason string, blocked bool) {
	return g.blocked(args, 0)
}

func (g ProcessGuard) blocked(args []string, depth int) (reason string, blocked bool) {
	args = peelLaunchers(args)
	if len(args) == 0 {
		return "", false
	}
	cmd := baseName(args[0])
	switch {
	case shutdownCommands[cmd]:
		return "shutting the machine down or rebooting is never allowed", true
	case shellInterpreters[cmd] || cmd == "eval":
		var script string
		var positional []string
		if cmd == "eval" {
			script = strings.Join(args[1:], " ") // eval nối đối số rồi chạy như script
		} else {
			var ok bool
			if script, positional, ok = shellScript(args); !ok {
				return "", false
			}
		}
		return g.blockedScript(script, positional, depth)
	case killByPID[cmd]:
		pids := killPIDs(args)
		// Không có PID tường minh thì PID đến từ nơi khác (xargs, stdin): không
		// kiểm được, nên chặn. `kill -l` chỉ liệt kê tín hiệu.
		if len(pids) == 0 && !killListsSignals(args) {
			return "kill without an explicit PID (fed by xargs or stdin) could hit agent-cli's own process", true
		}
		for _, pid := range pids {
			// `kill -9 -1` / pid 1: nhắm mọi tiến trình hoặc init.
			if pid == 1 {
				return "kill -1 (every process) or pid 1 (init) is never allowed", true
			}
			if g.ProtectedPID != nil && g.ProtectedPID(pid) {
				return "refusing to kill agent-cli's own process (pid " + strconv.Itoa(pid) + "): it is running this session", true
			}
		}
	case killByName[cmd]:
		for _, name := range killNames(args) {
			if g.ProtectedName != nil && g.ProtectedName(name) {
				return "refusing to " + cmd + " a process matching agent-cli (" + name + "): it would kill this session's own server or client", true
			}
		}
	}
	return "", false
}

// blockedScript soi từng lệnh đơn trong script của `sh -c` / `eval`.
func (g ProcessGuard) blockedScript(script string, positional []string, depth int) (string, bool) {
	if depth >= maxShellDepth {
		return "shell commands nested too deep to inspect for process-killing", true
	}
	cmds, ok := scriptCommands(script)
	if !ok {
		// Không phân tích được: chỉ chặn khi script có nhắc tới lệnh bị cấm.
		if killWords.MatchString(script) {
			return "cannot parse a shell script that mentions kill or shutdown", true
		}
		return "", false
	}
	dynamicHead := false
	for _, argv := range cmds {
		if argv[0] == dynamicWord {
			dynamicHead = true
			continue
		}
		if reason, blocked := g.blocked(argv, depth+1); blocked {
			return reason, true
		}
	}
	if dynamicHead {
		// Tên lệnh chỉ biết lúc chạy (`$k X`, `"$@"`): chặn khi script nhắc tới
		// kill/shutdown, và soi các đối số vị trí (`bash -c '"$@"' _ /bin/kill N`).
		if killWords.MatchString(script) {
			return "shell script runs a command whose name is only known at run time, next to kill", true
		}
		for i := range positional {
			if reason, blocked := g.blocked(positional[i:], depth+1); blocked {
				return reason, true
			}
		}
	}
	return "", false
}

// selfExe là đường dẫn binary agent-cli đã giải symlink; selfTree là các PID
// trong cây tiến trình của chính nó (nó + tổ tiên tới init). Tính một lần.
var (
	selfExe  = resolveSelfExe()
	selfTree = selfProcessTree()
)

func resolveSelfExe() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if r, err := filepath.EvalSymlinks(exe); err == nil {
		return r
	}
	return exe
}

// selfProcessTree trả PID của chính tiến trình và các tổ tiên (client/server/
// launcher/shell cha đều cần được bảo vệ). Leo theo /proc; thiếu /proc thì chỉ
// có pid và ppid.
func selfProcessTree() map[int]bool {
	tree := map[int]bool{os.Getpid(): true, os.Getppid(): true}
	pid := os.Getppid()
	for range 32 {
		ppid := parentPID(pid)
		if ppid <= 1 || tree[ppid] {
			break
		}
		tree[ppid] = true
		pid = ppid
	}
	return tree
}

// parentPID đọc PPid từ /proc/<pid>/status (Linux); 0 nếu không đọc được.
func parentPID(pid int) int {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/status")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if rest, ok := strings.CutPrefix(line, "PPid:"); ok {
			if n, err := strconv.Atoi(strings.TrimSpace(rest)); err == nil {
				return n
			}
		}
	}
	return 0
}

// pidRunsSelfExe cho biết tiến trình pid đang chạy cùng binary với agent-cli
// (client, server, launcher đều là cùng một binary crush).
func pidRunsSelfExe(pid int) bool {
	if selfExe == "" {
		return false
	}
	target, err := filepath.EvalSymlinks("/proc/" + strconv.Itoa(pid) + "/exe")
	return err == nil && target == selfExe
}

// patternHitsProtected báo mẫu của pkill/killall có đe doạ một tiến trình được
// bảo vệ không. pkill/killall khớp theo CHUỖI CON / REGEX trên tên tiến trình
// (hoặc cả dòng lệnh với -f), nên so tên chính xác là chưa đủ: chặn khi mẫu
// chứa, bị chứa, hoặc (coi như regex) khớp một tên được bảo vệ. Nghiêng về
// chặn (fail-closed) vì đây là lệnh giết tiến trình.
func patternHitsProtected(pattern string, names []string) bool {
	p := strings.ToLower(strings.TrimSpace(pattern))
	if p == "" {
		return false
	}
	re, reErr := regexp.Compile(p)
	for _, n := range names {
		n = strings.ToLower(n)
		if n == "" {
			continue
		}
		if strings.Contains(n, p) || strings.Contains(p, n) {
			return true
		}
		if reErr == nil && re.MatchString(n) {
			return true
		}
	}
	return false
}

// defaultProcessGuard là bộ bảo vệ dùng lúc chạy thật.
var defaultProcessGuard = ProcessGuard{
	ProtectedPID: func(pid int) bool {
		return selfTree[pid] || pidRunsSelfExe(pid)
	},
	ProtectedName: func(pattern string) bool {
		names := []string{"crush", "agent-cli"}
		if selfExe != "" {
			names = append(names, baseName(selfExe))
		}
		return patternHitsProtected(pattern, names)
	},
}

// ProtectControlProcess là BlockFunc chặn lệnh giết tiến trình agent-cli hay
// tắt máy. onBlock (có thể nil) nhận lý do để ghi audit log.
func ProtectControlProcess(onBlock func(reason string, args []string)) BlockFunc {
	return func(args []string) bool {
		reason, blocked := defaultProcessGuard.Blocked(args)
		if blocked && onBlock != nil {
			onBlock(reason, args)
		}
		return blocked
	}
}
