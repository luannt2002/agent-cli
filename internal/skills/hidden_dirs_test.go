package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Thư mục ẩn trong cây skill (`.trash` của trình quản lý skill, `.git`…) chứa bản
// đã xoá hoặc không dành cho agent. Audit g5 đo trên máy thật: 8 skill người dùng
// đã xoá vẫn nằm trong system prompt vì walk đệ quy không bỏ thư mục ẩn.
func TestDiscoverSkipsHiddenDirectories(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), ".claude", "skills")
	write := func(rel, name string) {
		dir := filepath.Join(base, rel)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, SkillFileName),
			[]byte("---\nname: "+name+"\ndescription: test skill.\n---\n# x\n"), 0o644))
	}
	write("kept", "kept")
	write(filepath.Join(".trash", "1791857989703-3946780-h98KIX", "canvas-design"), "canvas-design")
	write(filepath.Join("group", ".hidden", "secret-skill"), "secret-skill")
	// Symlink tên ẩn trỏ tới một thư mục skill ở nơi khác: vẫn là thư mục ẩn.
	outside := filepath.Join(t.TempDir(), "elsewhere")
	require.NoError(t, os.MkdirAll(filepath.Join(outside, "linked-skill"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(outside, "linked-skill", SkillFileName),
		[]byte("---\nname: linked-skill\ndescription: test skill.\n---\n# x\n"), 0o644))
	require.NoError(t, os.Symlink(outside, filepath.Join(base, ".old-link")))

	skills, _ := DiscoverWithStates([]string{base})
	var names []string
	for _, s := range skills {
		names = append(names, s.Name)
	}
	require.Equal(t, []string{"kept"}, names, "only the visible skill; a hidden parent dir in the base path itself is fine")
}

// Gốc cấu hình tên ẩn viết kèm "/" cuối (skills_paths: ["~/.myskills/"]): fastwalk
// bỏ "/" khỏi gốc, nên so chuỗi với base không nhận ra gốc và bỏ cả cây.
func TestDiscoverKeepsAHiddenBaseWrittenWithATrailingSlash(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), ".myskills")
	dir := filepath.Join(base, "mine")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, SkillFileName),
		[]byte("---\nname: mine\ndescription: test skill.\n---\n# x\n"), 0o644))

	for _, b := range []string{base, base + "/", filepath.Join(base, "..", ".myskills")} {
		skills, _ := DiscoverWithStates([]string{b})
		require.Len(t, skills, 1, "base %q", b)
	}
}
