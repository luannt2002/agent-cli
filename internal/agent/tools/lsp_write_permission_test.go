package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/stretchr/testify/require"
)

// lspWriteFixture: dự án, một thư mục ngoài, và một symlink trong dự án trỏ ra
// ngoài. Permission service ở YOLO và tự duyệt phiên, như routine không người trực.
func lspWriteFixture(t *testing.T) (svc permission.Service, project, outside string) {
	t.Helper()
	project, outside = t.TempDir(), t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(project, "main.go"), []byte("package main\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(outside, "crushrc"), nil, 0o644))
	require.NoError(t, os.Symlink(outside, filepath.Join(project, "link")))
	execpolicy.SetUnattended(true, false)
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	svc = permission.NewPermissionService(project, true, nil)
	svc.AutoApproveSession("s")
	return svc, project, outside
}

// lsp_replace_symbol ghi params.FilePath bằng os.WriteFile: không người trực thì
// không được ghi ra ngoài thư mục làm việc, kể cả qua symlink.
func TestReplaceSymbolRefusedOutsideWhileUnattended(t *testing.T) {
	svc, project, outside := lspWriteFixture(t)
	for _, path := range []string{filepath.Join(outside, "crushrc"), filepath.Join(project, "link", "crushrc")} {
		ok, err := svc.Request(t.Context(), replaceSymbolPermission("s", "replace", ReplaceSymbolParams{Symbol: "x", FilePath: path}, "", "y"))
		require.NoError(t, err)
		require.Falsef(t, ok, "%s must be refused while unattended", path)
	}
	ok, err := svc.Request(t.Context(), replaceSymbolPermission("s", "replace", ReplaceSymbolParams{Symbol: "x", FilePath: filepath.Join(project, "main.go")}, "", "y"))
	require.NoError(t, err)
	require.True(t, ok, "a file inside the working dir still passes")
}

// lsp_rename áp workspace edit lên mọi tệp bị ảnh hưởng: một tệp ngoài thư mục
// làm việc là đủ để từ chối cả lượt khi không người trực.
func TestRenameRefusedWhenAnyFileOutsideWhileUnattended(t *testing.T) {
	svc, project, outside := lspWriteFixture(t)
	inside := filepath.Join(project, "main.go")
	for _, files := range [][]string{
		{inside, filepath.Join(outside, "crushrc")},
		{inside, filepath.Join(project, "link", "crushrc")},
	} {
		ok, err := authorizeRenameWrites(t.Context(), svc, "s", "call-1", files)
		require.NoError(t, err)
		require.Falsef(t, ok, "%v must be refused while unattended", files)
	}
	ok, err := authorizeRenameWrites(t.Context(), svc, "s", "call-1", []string{inside})
	require.NoError(t, err)
	require.True(t, ok, "files inside the working dir still pass")
}

// Có người trực thì lsp_rename giữ một hộp thoại duy nhất, không hỏi từng tệp.
func TestRenameAttendedAsksNoPerFilePermission(t *testing.T) {
	rec := newRecordingPermissions(true)
	ok, err := authorizeRenameWrites(t.Context(), rec, "s", "call-1", []string{"/a.go", "/b.go"})
	require.NoError(t, err)
	require.True(t, ok)
	require.Empty(t, rec.seen())
}
