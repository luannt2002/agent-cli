package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Lệnh tự duyệt không được đọc ra ngoài workspace: đối số là đường dẫn tuyệt
// đối, ~, .., biến $VAR, hoặc cờ làm git so sánh tệp tuỳ ý (--no-index) đều
// phải hỏi quyền. Nhóm "vẫn tự duyệt" chặn việc siết quá tay làm hỏi oan.
func TestIsSafeCommand_StaysInsideWorkspace(t *testing.T) {
	t.Parallel()

	for _, cmd := range []string{
		"ls", "ls -la", "ls -R src", "pwd", "echo hi", `echo "a/b"`, "date +%s", "du -sh .",
		"git status", "git diff", "git diff --stat", "git diff HEAD~1", "git diff HEAD~1 HEAD",
		"git log --oneline -1", "git log origin/main..HEAD", "git log --format=%H",
		"git show HEAD:README.md", "git branch -a",
		"git ls-files src", "git grep -n foo",
		"echo 'hello world'", `echo "a b"`,
	} {
		require.Truef(t, isSafeCommand(cmd), "%q phải vẫn tự duyệt", cmd)
	}

	for _, cmd := range []string{
		// X-01: git so sánh tệp ngoài repo.
		"git diff --no-index /dev/null /etc/hostname",
		"git diff --no-index /dev/null ~/.config/agent-cli/env",
		"git diff --no-index a b",
		"git diff --no-ind a b",
		// đường dẫn tuyệt đối / ~ / .. trong đối số
		"ls /etc", `ls "/etc"`, "ls ~", "ls -la ~/.ssh", "ls ..", "ls ../x", "ls a/../b", "ls -la /",
		"du -sh /", "git log -- /etc/passwd", "git ls-files /etc", "git show --stat=/x",
		"git diff --git-dir=/etc",
		// shell ghép chuỗi trước khi chạy: nháy, \ và {a,b} dựng được đường dẫn mà
		// token không bắt đầu bằng "/" hay "..".
		`ls \/etc`, "ls {/etc,}", "ls {/etc,/tmp}", "ls .''./etc", `ls ."."/etc`, `ls "/"etc`, "ls ''/etc", `ls /e\tc`,
		// lượt soát độc lập 16/10: giá trị dính liền cờ ngắn, đọc cấu hình git, ps hiện env.
		"date -f/etc/hostname", "git grep -f/etc/x foo", "git blame -S/etc/x main.go", "git ls-files -X/etc/x",
		"du --files0-from=/etc/x", "git config --get-regexp .", "git config --get user.name",
		"git config --get-regexp -f/etc/x key", "ps eww", "ps auxe", "ps aux",
		// đổ biến môi trường
		"printenv", "printenv HOME", "echo $HOME", `echo "$AWS_SECRET_ACCESS_KEY"`, "echo ${HOME}",
		// nhắc lại: cấu hình có thể chứa token trong URL remote
		"git config --list",
	} {
		require.Falsef(t, isSafeCommand(cmd), "%q phải hỏi quyền", cmd)
	}
}

// Ở mức tool: lệnh đọc ngoài workspace (hoặc chạy với working_dir ngoài) phải đi
// qua permission service; từ chối thì nội dung tệp không được lọt vào kết quả.
func TestBashTool_ReadsOutsideWorkspaceRequirePermission(t *testing.T) {
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	require.NoError(t, os.WriteFile(secret, []byte("SECRET-CANARY-7f3a"), 0o644))
	t.Setenv("AUDIT_ENV_CANARY", "ENV-CANARY-9c1e")

	for _, tc := range []struct {
		name   string
		params BashParams
	}{
		{"git diff --no-index", BashParams{Command: "git diff --no-index /dev/null " + secret}},
		{"ls thư mục ngoài", BashParams{Command: "ls " + outside}},
		{"echo $VAR", BashParams{Command: "echo $AUDIT_ENV_CANARY"}},
		{"printenv", BashParams{Command: "printenv AUDIT_ENV_CANARY"}},
		{"working_dir ngoài", BashParams{Command: "ls", WorkingDir: outside}},
		{"date -f<tệp ngoài>", BashParams{Command: "date -f" + secret}},
		{"ps hiện env", BashParams{Command: "ps eww -o args= -p 1"}},
		{"git config --get-regexp", BashParams{Command: "git config --get-regexp ."}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tool, perms := newBashToolWithRecordingPerms(t.TempDir(), false)
			ctx := context.WithValue(t.Context(), SessionIDContextKey, "s")
			tc.params.Description = "outside read"

			resp := runBashTool(t, tool, ctx, tc.params)

			require.Equal(t, 1, perms.requestCount, "phải hỏi quyền")
			require.Contains(t, resp.Content, "User denied permission")
			require.False(t, strings.Contains(resp.Content, "SECRET-CANARY-7f3a") || strings.Contains(resp.Content, "ENV-CANARY-9c1e"),
				"từ chối thì không được lộ nội dung")
		})
	}
}

// working_dir nằm trong workspace thì vẫn tự duyệt lệnh chỉ-đọc.
func TestBashTool_WorkingDirInsideWorkspaceStillAutoApproves(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	require.NoError(t, os.MkdirAll(sub, 0o755))

	tool, perms := newBashToolWithRecordingPerms(root, false)
	ctx := context.WithValue(t.Context(), SessionIDContextKey, "s")
	resp := runBashTool(t, tool, ctx, BashParams{Description: "inside", Command: "ls", WorkingDir: sub})

	require.Equal(t, 0, perms.requestCount)
	require.NotContains(t, resp.Content, "User denied permission")
}
