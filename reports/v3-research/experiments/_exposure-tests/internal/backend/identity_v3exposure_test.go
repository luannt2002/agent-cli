package backend

// Test phơi nhiễm V3 T1 (reports/v3-research/experiments/00_EXPOSURE_TESTS.md):
// bảng chân trị các resolver danh tính workspace. Bất biến mong muốn, mỗi bất
// biến một subtest:
//   - spelling_invariant: cùng một thư mục vật lý, viết cách nào (symlink, "/"
//     cuối, tương đối) thì mỗi resolver cũng chỉ tới cùng thư mục vật lý (khác
//     chuỗi mà cùng thư mục KHÔNG tính là lệch);
//   - data_dir_inside_project: data dir nằm trong biên vật lý của dự án;
//   - branch_equals_physical: nhánh đọc qua đường đã cho bằng nhánh vật lý.
// Hai workspace lồng nhau (monorepo) không xét ở đây: tác hại đo ở T1M
// (scripts/eval/exposure.sh). ĐỎ = phơi nhiễm được xác nhận; không chỉnh test
// cho ra màu mong muốn.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/gitutil"
	"github.com/charmbracelet/crush/internal/projects"
	"github.com/stretchr/testify/require"
)

func v3git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
}

// physical là đường vật lý (Abs + EvalSymlinks). Đường chưa tồn tại thì giải
// thư mục cha rồi ghép tên, để data dir chưa tạo vẫn so được.
func physical(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	if r, err := filepath.EvalSymlinks(abs); err == nil {
		return r
	}
	if parent := filepath.Dir(abs); parent != abs {
		return filepath.Join(physical(parent), filepath.Base(abs))
	}
	return abs
}

// projectTop là biên vật lý của dự án chứa dir: git toplevel của đường vật
// lý, ngoài git thì chính nó. Tính bằng git trực tiếp, không qua resolver
// đang được đo.
func projectTop(dir string) string {
	out, err := exec.Command("git", "-C", physical(dir), "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return physical(dir)
	}
	return physical(strings.TrimSpace(string(out)))
}

func within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// launcherRoot chép đúng biểu thức chọn gốc của launcher
// (scripts/install-agent-cli.sh:115). Chỉ để tham khảo, không vào phép assert.
func launcherRoot(dir string) string {
	out, err := exec.Command("bash", "-c", `cd -- "$1" 2>/dev/null && { git rev-parse --show-toplevel 2>/dev/null || pwd -P; }`, "_", dir).Output()
	if err != nil {
		return "ERR"
	}
	return strings.TrimSpace(string(out))
}

func TestV3Exposure_T1_IdentityResolversAgree(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("CRUSH_GLOBAL_CONFIG", filepath.Join(home, "cfg"))
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(home, "data"))
	t.Setenv("CRUSH_DISABLE_PROVIDER_AUTO_UPDATE", "1")
	t.Setenv("CRUSH_DISABLE_METRICS", "1")

	// Fixture:
	//   T/.crush            data dir có sẵn ở thư mục cha của dự án (vd ~/code)
	//   T/real/repo         repo git, nhánh main, đã mở một lần (có repo/.crush)
	//   T/real/repo/sub     thư mục con (monorepo), chưa có .crush riêng
	//   T/real/wt           worktree liên kết, nhánh feature
	//   T/link-root  ->  T/real/repo
	//   T/link-sub   ->  T/real/repo/sub
	//   T/plain             ngoài git
	//   U/link-sub   ->  T/real/repo/sub  (U không có .crush nào phía trên)
	T := physical(t.TempDir())
	repo := filepath.Join(T, "real", "repo")
	sub := filepath.Join(repo, "sub")
	wt := filepath.Join(T, "real", "wt")
	plain := filepath.Join(T, "plain")
	for _, d := range []string{filepath.Join(T, ".crush"), sub, filepath.Join(repo, ".crush"), plain} {
		require.NoError(t, os.MkdirAll(d, 0o700))
	}
	require.NoError(t, os.WriteFile(filepath.Join(repo, "README.md"), []byte("x\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "a.go"), []byte("package a\n"), 0o600))
	v3git(t, repo, "init", "-q", "-b", "main")
	v3git(t, repo, "add", "README.md", "sub/a.go")
	v3git(t, repo, "commit", "-q", "-m", "init")
	v3git(t, repo, "worktree", "add", "-q", "-b", "feature", wt)
	require.NoError(t, os.Symlink(repo, filepath.Join(T, "link-root")))
	require.NoError(t, os.Symlink(sub, filepath.Join(T, "link-sub")))
	// Cùng symlink vào sub nhưng đặt ở thư mục không có .crush nào phía trên.
	U := physical(t.TempDir())
	require.NoError(t, os.Symlink(sub, filepath.Join(U, "link-sub")))

	type ids struct{ k, n, data, branch string }
	resolve := func(in string) ids {
		k, err := resolveWorkspaceKey(in)
		require.NoError(t, err)
		n, err := projects.NormalizeWorkspace(in)
		require.NoError(t, err)
		store, err := config.Load(in, "", false)
		require.NoError(t, err, in)
		return ids{k, n, store.Config().Options.DataDirectory, gitutil.CurrentBranch(in)}
	}

	// ref là cách viết tuyệt đối, không symlink, của cùng thư mục vật lý.
	type row struct{ name, input, ref, cwd string }
	rows := []row{
		{"repo-root", repo, repo, ""},
		{"symlink-to-root", filepath.Join(T, "link-root"), repo, ""},
		{"symlink-into-sub", filepath.Join(T, "link-sub"), sub, ""},
		{"symlink-into-sub-no-parent-crush", filepath.Join(U, "link-sub"), sub, ""},
		{"monorepo-sub", sub, sub, ""},
		{"linked-worktree", wt, wt, ""},
		{"trailing-slash", repo + "/", repo, ""},
		{"relative", "sub", sub, repo},
		{"outside-git", plain, plain, ""},
	}
	short := func(p string) string { return strings.Replace(strings.Replace(p, T, "T", 1), U, "U", 1) }
	table := []string{
		"| ca | đầu vào | K resolveWorkspaceKey | N projects.NormalizeWorkspace | data dir | nhánh | biên vật lý | L launcher (tham khảo) | kết quả |",
		"|---|---|---|---|---|---|---|---|---|",
	}
	for _, r := range rows {
		want := resolve(r.ref)
		if r.cwd != "" {
			t.Chdir(r.cwd)
		}
		got := resolve(r.input)
		top := projectTop(r.ref)
		bPhys := gitutil.CurrentBranch(physical(r.ref))
		var bad []string
		check := func(name string, ok bool, msg string, args ...any) {
			if !ok {
				bad = append(bad, name)
			}
			detail := fmt.Sprintf(msg, args...)
			t.Run(r.name+"/"+name, func(t *testing.T) {
				require.True(t, ok, "T1 EXPOSED: %s", detail)
			})
		}
		check("spelling_invariant",
			physical(got.k) == physical(want.k) && physical(got.n) == physical(want.n) && physical(got.data) == physical(want.data),
			"same physical dir, different identity: K %q vs %q, N %q vs %q, data %q vs %q",
			short(physical(got.k)), short(physical(want.k)), short(physical(got.n)), short(physical(want.n)), short(physical(got.data)), short(physical(want.data)))
		check("data_dir_inside_project", within(top, physical(got.data)),
			"data dir %q is outside the physical project %q", short(physical(got.data)), short(top))
		check("branch_equals_physical", got.branch == bPhys,
			"branch through %q is %q, physical branch is %q", short(r.input), got.branch, bPhys)
		verdict := "ok"
		if len(bad) > 0 {
			verdict = "LỆCH " + strings.Join(bad, ",")
		}
		table = append(table, fmt.Sprintf("| %s | %s | %s | %s | %s | %q (vật lý %q) | %s | %s | %s |",
			r.name, short(r.input), short(got.k), short(got.n), short(got.data), got.branch, bPhys, short(top), short(launcherRoot(r.input)), verdict))
	}
	t.Log("T1 bảng chân trị resolver danh tính:\n" + strings.Join(table, "\n"))
}
