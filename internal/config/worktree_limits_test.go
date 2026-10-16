package config_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/stretchr/testify/require"
)

// Giới hạn duyệt tệp (ls depth/items) áp khi THƯ MỤC WORKSPACE không nằm trong
// một git work tree, không phụ thuộc cwd của tiến trình. Server agent-cli thừa
// hưởng cwd của client đầu tiên (vd $HOME), nên đọc cwd làm sai mọi repo mở sau.
// Test chạy với cwd = thư mục gói (nằm trong repo git) để lộ lỗi đó.
func TestLoad_FileWalkLimitsFollowWorkspaceNotProcessCwd(t *testing.T) {
	isolated := t.TempDir()
	t.Setenv("HOME", isolated)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(isolated, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(isolated, ".local", "share"))
	t.Setenv("CRUSH_GLOBAL_CONFIG", filepath.Join(isolated, ".config", "crush"))
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(isolated, ".local", "share", "crush"))

	gitRepo := t.TempDir()
	out, err := exec.Command("git", "init", "-q", gitRepo).CombinedOutput()
	require.NoError(t, err, string(out))
	plain := t.TempDir()

	inRepo, err := config.Load(gitRepo, t.TempDir(), false)
	require.NoError(t, err)
	require.Nil(t, inRepo.Config().Tools.Ls.MaxDepth, "workspace là repo git: không giới hạn")

	notRepo, err := config.Load(plain, t.TempDir(), false)
	require.NoError(t, err)
	require.NotNil(t, notRepo.Config().Tools.Ls.MaxDepth, "workspace không phải git: phải giới hạn dù cwd tiến trình ở trong repo")
	require.Equal(t, 2, *notRepo.Config().Tools.Ls.MaxDepth)
	require.NotNil(t, notRepo.Config().Tools.Ls.MaxItems)
	require.Equal(t, 100, *notRepo.Config().Tools.Ls.MaxItems)
}
