package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestResolveCwdMakesRelativeFlagAbsolute: `-c ..` phải thành đường tuyệt đối
// theo cwd của chính lệnh. Chuỗi tương đối gửi cho server bị ghép theo cwd
// CỦA SERVER, và config ghép thêm lần nữa sau chdir.
func TestResolveCwdMakesRelativeFlagAbsolute(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	repo := filepath.Join(root, "repo")
	sub := filepath.Join(repo, "internal")
	require.NoError(t, os.MkdirAll(sub, 0o755))

	for name, flag := range map[string]string{"relative": "..", "absolute": repo} {
		t.Run(name, func(t *testing.T) {
			t.Chdir(sub)
			cmd := &cobra.Command{}
			cmd.Flags().StringP("cwd", "c", "", "")
			require.NoError(t, cmd.Flags().Set("cwd", flag))

			got, err := ResolveCwd(cmd)
			require.NoError(t, err)
			require.Equal(t, repo, got)

			wd, err := os.Getwd()
			require.NoError(t, err)
			require.Equal(t, repo, wd, "ResolveCwd must still chdir")

			// `loop` gọi lần nữa qua setupLocalWorkspace/connectToServer, lúc
			// này cwd đã là repo: không được đi tiếp lên thư mục cha.
			got, err = ResolveCwd(cmd)
			require.NoError(t, err)
			require.Equal(t, repo, got, "a second call must not resolve the flag again")
		})
	}
}

// TestResolveCwdFollowsLogicalPwd ghim cách hiểu `..` theo $PWD (như `cd ..`
// của shell và của launcher), không theo cha vật lý của symlink.
func TestResolveCwdFollowsLogicalPwd(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	sub := filepath.Join(root, "real", "repo", "internal")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	link := filepath.Join(root, "link")
	require.NoError(t, os.Symlink(sub, link))
	t.Chdir(link) // Đặt luôn PWD=link.

	cmd := &cobra.Command{}
	cmd.Flags().StringP("cwd", "c", "", "")
	require.NoError(t, cmd.Flags().Set("cwd", ".."))

	got, err := ResolveCwd(cmd)
	require.NoError(t, err)
	require.Equal(t, root, got)
}
