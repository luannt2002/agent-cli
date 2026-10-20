package shell

// Test phơi nhiễm V3 C9 (reports/v3-research/experiments/00_EXPOSURE_TESTS.md):
// R14 leavesWorkspace (shell/unattended.go:249-253), lexical theo thiết kế. Bất biến mong muốn: symlink trỏ ra ngoài và symlink treo đều là
// "ngoài workspace". Hai gói kia chạy cùng fixture; bảng ghép ở báo cáo.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestV3Exposure_C9_ShellLeavesWorkspace(t *testing.T) {
	_, cases := v3C9Fixture(t)
	for _, c := range cases {
		outside := leavesWorkspace(c.rel)
		t.Logf("C9 R14 %-16s %-16s outside=%v want=%v", c.name, c.rel, outside, c.wantOutside)
		t.Run(c.name, func(t *testing.T) {
			if outside != c.wantOutside {
				t.Fatalf("C9 EXPOSED: R14 says outside=%v for %q, want %v", outside, c.rel, c.wantOutside)
			}
		})
	}
}

// v3C9Fixture dựng CÙNG một fixture cho ba test C9 (permission, tools, shell;
// khối này được chép nguyên văn sang cả ba gói):
//
//	W/inner/f.txt           tệp thường trong workspace
//	W/link-out -> O         symlink trong workspace trỏ RA NGOÀI
//	W/dang -> G/missing     symlink treo (đích không tồn tại, ngoài workspace)
//	O/f.txt                 thư mục ngoài workspace
//
// Mỗi ca: đường tương đối như tool/lệnh viết, và câu trả lời mong muốn
// (true = ngoài workspace). Symlink treo phải tính là ngoài (fail closed).
func v3C9Fixture(t *testing.T) (string, []v3C9Case) {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	w, o := filepath.Join(base, "W"), filepath.Join(base, "O")
	for _, d := range []string{filepath.Join(w, "inner"), o} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{filepath.Join(w, "inner", "f.txt"), filepath.Join(o, "f.txt")} {
		if err := os.WriteFile(f, []byte("x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(o, filepath.Join(w, "link-out")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(base, "G", "missing"), filepath.Join(w, "dang")); err != nil {
		t.Fatal(err)
	}
	return w, []v3C9Case{
		{"inside-plain", "inner/f.txt", false},
		{"symlink-out", "link-out/f.txt", true},
		{"dangling-symlink", "dang", true},
		{"dotdot", "../O/f.txt", true},
	}
}

type v3C9Case struct {
	name, rel   string
	wantOutside bool
}
