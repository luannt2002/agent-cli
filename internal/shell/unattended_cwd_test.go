package shell

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// workspaceBesideCheckout dựng parent/{wt,main} như routine thật: wt là workspace
// (worktree của routine, có repo git riêng), main là checkout của người dùng có
// precious.txt đã commit và sub/keep.
func workspaceBesideCheckout(t *testing.T) (ws, main string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	parent := t.TempDir()
	ws, main = filepath.Join(parent, "wt"), filepath.Join(parent, "main")
	for _, d := range []string{filepath.Join(ws, "sub"), filepath.Join(main, "sub")} {
		require.NoError(t, os.MkdirAll(d, 0o755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(main, "sub", "keep"), nil, 0o644))
	gitInitWithFile(t, main)
	gitInitWithFile(t, ws)
	return ws, main
}

// checkoutState chụp những gì lệnh thoát workspace đã phá được ở lần tái lập 22/10:
// nhánh hiện tại, tệp đã commit, thư mục con.
func checkoutState(t *testing.T, main string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = main
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "%s", out)
	state := strings.TrimSpace(string(out))
	for _, f := range []string{"precious.txt", filepath.Join("sub", "keep")} {
		if _, err := os.Stat(filepath.Join(main, f)); err == nil {
			state += " " + f
		}
	}
	return state
}

func runInWorkspace(t *testing.T, ws, command string, enabled bool) (error, []string) {
	t.Helper()
	var mu sync.Mutex
	var reasons []string
	ctx := WithWorkspaceGuard(context.Background(), func() bool { return enabled }, ws, func(reason string, _ []string) {
		mu.Lock()
		defer mu.Unlock()
		reasons = append(reasons, reason)
	})
	err := Run(ctx, RunOptions{Command: command, Cwd: ws, Env: os.Environ()})
	mu.Lock()
	defer mu.Unlock()
	return err, reasons
}

// P1 audit 22/10: dưới guard, `cd ../main && git checkout -b`, `git rm -f`, `rm -rf
// main/sub` đều chạy và đổi checkout thật, vì BlockFunc chỉ thấy argv. Không người
// trực thì lệnh có thư mục làm việc thật (sau cd, pushd, subshell, symlink, job nền)
// ngoài workspace bị từ chối kèm lý do; cùng lệnh khi có người trực thì phá được
// (đối chứng: test không mù).
func TestWorkspaceGuardRefusesCommandsThatLeaveTheWorkspace(t *testing.T) {
	t.Parallel()
	for name, command := range map[string]string{
		"cd .. then git checkout -b": `cd ../main && git checkout -q -b agent-branch`,
		"cd .. then git rm":          `cd ../main && git rm -q -f precious.txt`,
		"cd .. then rm -rf":          `cd .. && rm -rf main/sub`,
		"cd absolute":                `cd "$(dirname "$PWD")/main" && git rm -q -f precious.txt`,
		"subshell":                   `(cd .. && rm -rf main/sub)`,
		"pushd":                      `pushd ../main >/dev/null && git rm -q -f precious.txt`,
		"symlinked dir":              `ln -s ../main r && cd r && git rm -q -f precious.txt`,
		"git -C symlink":             `ln -s ../main r && git -C r rm -q -f precious.txt`,
		"git --work-tree symlink":    `ln -s ../main r && git --git-dir=r/.git --work-tree r rm -q -f precious.txt`,
		"background job &":           `cd ../main && git rm -q -f precious.txt & wait $!`,
		"shebang script after cd":    `cd .. && ./main/wipe.sh`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			requireGuardStopsEscape(t, command)
		})
	}
}

// requireGuardStopsEscape: có người trực thì command phá được checkout (đối chứng: test
// không mù); không người trực thì bị từ chối, lỗi nói lý do và checkout còn nguyên.
func requireGuardStopsEscape(t *testing.T, command string) {
	t.Helper()
	for _, enabled := range []bool{false, true} {
		ws, main := workspaceBesideCheckout(t)
		require.NoError(t, os.WriteFile(filepath.Join(main, "wipe.sh"), []byte("#!/bin/sh\nrm -rf \"$(dirname \"$0\")/sub\"\n"), 0o755))
		before := checkoutState(t, main)
		err, reasons := runInWorkspace(t, ws, command, enabled)
		if !enabled {
			require.NotEqual(t, before, checkoutState(t, main), "control: attended, the command really changes the checkout (err=%v)", err)
			continue
		}
		require.Error(t, err)
		require.Equal(t, before, checkoutState(t, main), "unattended: the user's checkout is untouched")
		require.NotEmpty(t, reasons)
		require.Contains(t, reasons[0], "outside the workspace")
		require.Contains(t, err.Error(), "outside the workspace", "the error says why")
	}
}

// Lệnh chỉ đọc cũng bị từ chối khi đã ra ngoài (`cd /tmp`, `git -C <symlink> log` của lần
// tái lập 22/10): guard xét thư mục, không xét lệnh.
func TestWorkspaceGuardRefusesReadsOutsideToo(t *testing.T) {
	t.Parallel()
	commands := []string{`ln -s ../main r && git -C r log -1`, `cd .. && ls`}
	if runtime.GOOS != "windows" {
		commands = append(commands, `cd /tmp && ls`)
	}
	for _, command := range commands {
		ws, _ := workspaceBesideCheckout(t)
		err, reasons := runInWorkspace(t, ws, command, true)
		require.Errorf(t, err, "%q", command)
		require.NotEmptyf(t, reasons, "%q", command)
	}
}

// Vế ngược: đi lại BÊN TRONG workspace (thư mục con, đường dẫn tuyệt đối trong
// workspace, symlink trỏ vào trong, cd rồi quay về) vẫn chạy.
func TestWorkspaceGuardAllowsMovingInsideTheWorkspace(t *testing.T) {
	t.Parallel()
	for _, command := range []string{
		`cd sub && git status --short`,
		`cd "$PWD/sub" && ls`,
		`cd sub && cd .. && git status --short`,
		`git -C sub status --short`,
		`ln -s sub inner && cd inner && ls`,
		`pushd sub >/dev/null && ls && popd >/dev/null && ls`,
		`(cd sub && ls) && ls`,
		`cd sub && ls & wait`,
	} {
		ws, _ := workspaceBesideCheckout(t)
		err, reasons := runInWorkspace(t, ws, command, true)
		require.NoErrorf(t, err, "%q", command)
		require.Emptyf(t, reasons, "%q", command)
	}
}

// Đường dẫn workspace đi qua symlink (macOS: /var -> /private/var) không làm mọi lệnh bị
// chặn: gốc và thư mục hiện tại đều được giải symlink trước khi so.
func TestWorkspaceGuardWorkspaceReachedThroughASymlink(t *testing.T) {
	t.Parallel()
	ws, _ := workspaceBesideCheckout(t)
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(ws, link))
	err, reasons := runInWorkspace(t, link, `cd sub && ls && cd .. && git status --short`, true)
	require.NoError(t, err)
	require.Empty(t, reasons)
	err, reasons = runInWorkspace(t, link, `cd .. && ls`, true)
	require.Error(t, err)
	require.NotEmpty(t, reasons)
}

// Soát 22/10 #5: workspace đi qua symlink mà script `sh -c` dùng dạng đã giải (hay ngược
// lại) vẫn là ở trong; đi lên quá gốc thì vẫn chặn.
func TestWorkspaceGuardFollowsCdAcrossRootSpellings(t *testing.T) {
	t.Parallel()
	ws, _ := workspaceBesideCheckout(t)
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(ws, link))
	resolved, err := filepath.EvalSymlinks(ws)
	require.NoError(t, err)
	for _, c := range []struct{ root, command string }{
		{link, `sh -c 'cd "` + resolved + `/sub" && ls && cd .. && ls'`},
		{ws, `sh -c 'cd "` + link + `/sub" && ls && cd .. && ls'`},
	} {
		err, reasons := runInWorkspace(t, c.root, c.command, true)
		require.NoErrorf(t, err, "%q", c.command)
		require.Emptyf(t, reasons, "%q", c.command)
	}
	err, reasons := runInWorkspace(t, link, `sh -c 'cd "`+resolved+`/sub" && cd ../.. && ls'`, true)
	require.Error(t, err)
	require.NotEmpty(t, reasons)
}

// Thư mục làm việc không giải được (đã bị xoá sau khi cd vào) coi như ở ngoài.
func TestWorkspaceGuardTreatsUnresolvableDirAsOutside(t *testing.T) {
	t.Parallel()
	ws, _ := workspaceBesideCheckout(t)
	err, reasons := runInWorkspace(t, ws, `mkdir gone && cd gone && rmdir ../gone && ls`, true)
	require.Error(t, err)
	require.NotEmpty(t, reasons)
}

// Job nền của tool bash (BackgroundShellManager, cả run_in_background lẫn lệnh tự
// chuyển nền) chạy dưới ctx mà tool truyền vào: guard theo cùng job đó.
func TestWorkspaceGuardCoversBackgroundJobs(t *testing.T) {
	t.Parallel()
	ws, main := workspaceBesideCheckout(t)
	before := checkoutState(t, main)
	ctx := WithWorkspaceGuard(context.Background(), func() bool { return true }, ws, nil)
	bg, err := newBackgroundShellManager().StartFor(ctx, JobOwner{Root: ws}, ws, nil, `cd ../main && git rm -q -f precious.txt`, "escape")
	require.NoError(t, err)
	bg.Wait()
	_, _, done, execErr := bg.GetOutput()
	require.True(t, done)
	require.ErrorContains(t, execErr, "outside the workspace")
	require.Equal(t, before, checkoutState(t, main))
}

// `cd` trong script của `sh -c` chạy ở shell CON: trình thông dịch của tool không thấy thư
// mục đổi. Guard theo dõi thư mục theo từ vựng: bắt đầu từ thư mục hiện tại, áp từng
// cd/pushd/popd theo thứ tự (vế sau && chỉ chạy khi cd thành công; sau `;`, `||`, subshell,
// vòng lặp thì lấy ước lượng nông nhất), chặn khi một bước ra ngoài hoặc đích không biết
// trước. Mẫu hợp lệ `cd web && npm test && cd ..` phải qua.
func TestWorkspaceGuardFollowsCdInsideShellScripts(t *testing.T) {
	t.Parallel()
	for _, command := range []string{
		`sh -c 'cd sub && ls && cd ..'`,
		`sh -c "cd sub && cd .. && git status --short"`,
		`bash -c 'pushd sub >/dev/null && ls && popd >/dev/null'`,
		`sh -c '(cd sub && ls); ls'`,
		`sh -c 'mkdir -p build/out && cd build/out && cd ../.. && ls'`,
		`sh -c "cd '$PWD/sub' && ls"`,
		`sh -c 'cd sub; ls'`,
		`bash -c 'cd sub && bash -c "cd .. && ls"'`,
		`sh -c 'for f in a b; do echo $f; done && cd sub && cd ..'`,
		`cd sub && sh -c 'cd .. && ls'`,
		`sh -c 'cd sub || exit 1; ls; cd ..'`,
		// soát 22/10 #4: thân if/for đi tuần tự, cd vào thư mục có sẵn rồi quay về
		`sh -c 'if [ -d sub ]; then cd sub; ls; cd ..; fi'`,
		`sh -c 'for i in 1 2; do cd sub; ls; cd ..; done'`,
		`sh -c 'f() { cd sub && ls && cd ..; }; f'`,
		// soát lại 22/10 (C): eval của đầu ra lệnh khác (conda/ssh-agent/opam) như ở 5337a9f7
		`bash -lc 'eval "$(printf true)" && ls'`,
		`bash -c 'eval "$(echo true)" && ls'`,
		`sh -c 'eval $(echo true) && ls'`,
	} {
		ws, _ := workspaceBesideCheckout(t)
		err, reasons := runInWorkspace(t, ws, command, true)
		require.NoErrorf(t, err, "%q", command)
		require.Emptyf(t, reasons, "%q", command)
	}
}

// Vế chặn của TestWorkspaceGuardFollowsCdInsideShellScripts: mỗi ca có đối chứng.
func TestWorkspaceGuardStopsCdOutOfShellScripts(t *testing.T) {
	t.Parallel()
	for name, command := range map[string]string{
		"cd .. first":            `sh -c 'cd .. && rm -rf main/sub'`,
		"down then up past root": `sh -c 'cd sub && cd ../.. && rm -rf main/sub'`,
		"subshell cd not kept":   `sh -c '(cd sub) && cd .. && rm -rf main/sub'`,
		"failed cd then ;":       `sh -c 'cd nonexistent; cd .. && rm -rf main/sub'`,
		"failed cd then ||":      `sh -c 'cd nonexistent || true; cd .. && rm -rf main/sub'`,
		"popd back to root":      `bash -c 'pushd sub >/dev/null && popd >/dev/null && cd .. && rm -rf main/sub'`,
		"nested bash -c":         `bash -c 'cd sub && bash -c "cd ../.. && rm -rf main/sub"'`,
		"eval":                   `sh -c 'eval "cd .." && rm -rf main/sub'`,
		"loop":                   `sh -c 'for i in 1; do cd ..; done; rm -rf main/sub'`,
		"dynamic target":         `sh -c 'cd "$(dirname "$PWD")/main" && git rm -q -f precious.txt'`,
		"existing symlink":       `ln -s ../main r && sh -c 'cd r && git rm -q -f precious.txt'`,
		// soát 22/10 #3: `\"` trong nháy kép, bash bỏ dấu `\` nên cd tới ../main thật
		"escaped quotes": `sh -c 'bash -c "cd \"../main\" && rm -rf sub"'`,
		// cd lên trong thân if vẫn bị soi
		"cd up inside if": `sh -c 'if true; then cd ..; fi; rm -rf main/sub'`,
		// `!` đảo kết quả: vế sau && chạy khi cd HỎNG (vẫn ở gốc)
		"negated failed cd": `bash -c '! cd nonexistent && cd .. && rm -rf main/sub'`,
		// soát lại 22/10 (D): cd vào TỆP hỏng nên shell vẫn ở gốc, `cd ..` ra ngoài thật
		"cd into a file": `touch README.md && sh -c 'cd README.md; cd ..; rm -rf main/sub'`,
		// vòng 3: eval động mà chữ tĩnh có cd thì chặn (idiom conda/ssh-agent/opam vẫn qua)
		"dynamic eval of cd": `D=.. sh -c 'eval "cd $D" && rm -rf main/sub'`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			requireGuardStopsEscape(t, command)
		})
	}
}

// Soát lại 22/10 (D): thư mục không vào được (không có quyền x) thì cd hỏng, shell vẫn ở
// gốc. Root bỏ qua quyền nên bỏ qua khi chạy bằng root.
func TestWorkspaceGuardTreatsUnsearchableDirAsFailedCd(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs POSIX permissions and a non-root user")
	}
	requireGuardStopsEscape(t, `mkdir locked && chmod 000 locked && sh -c 'cd locked; cd ..; rm -rf main/sub'`)
}

// Đích cd chỉ biết lúc chạy thì chặn.
func TestWorkspaceGuardRefusesUnknownCdTargetsInShellScripts(t *testing.T) {
	t.Parallel()
	commands := []string{
		`sh -c 'cd "$D" && ls'`,
		`sh -c 'cd ~ && ls'`,
		`sh -c 'cd && ls'`,
		`sh -c 'cd - && ls'`,
		`sh -c 'cd {..,sub} && ls'`,
	}
	if runtime.GOOS != "windows" {
		commands = append(commands, `sh -c 'cd /tmp && ls'`)
	}
	for _, command := range commands {
		ws, _ := workspaceBesideCheckout(t)
		err, reasons := runInWorkspace(t, ws, command, true)
		require.Errorf(t, err, "%q", command)
		require.NotEmptyf(t, reasons, "%q", command)
	}
}
