package shell

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Các nhánh của leavesWorkspace mà chưa test nào chạm (audit 22/10): đúng "..", đuôi
// "/..", và từ động trong script của `sh -c` ($(…), backtick, $VAR — guard không biết
// giá trị nên coi như ra ngoài). Mỗi nhánh có ca chỉ nó bắt được.
func TestUnattendedLeavesWorkspaceBranches(t *testing.T) {
	t.Parallel()
	g := unattendedGuardForTest(false)
	for _, args := range [][]string{
		// p == ".."
		{"rm", "-rf", ".."},
		{"git", "-C", "..", "rm", "-q", "precious.txt"},
		{"git", "--git-dir=..", "status"},
		// đuôi "/.."
		{"rm", "-rf", "sub/.."},
		{"git", "-C", "sub/..", "checkout", "-b", "x"},
		{"git", "--work-tree", "a/b/..", "add", "."},
		// từ động trong script của sh -c
		{"sh", "-c", "rm -rf $(pwd)/x"},
		{"sh", "-c", "rm -rf `pwd`/x"},
		{"sh", "-c", "rm -f $TARGET"},
		{"bash", "-c", `git -C "$(git rev-parse --show-toplevel)/../main" rm -q precious.txt`},
		{"sh", "-c", "git -C `dirname $PWD`/main status"},
		{"sh", "-c", "git -C $REPO add ."},
		// `$` còn nguyên trong argv (chuỗi đã quote)
		{"git", "-C", "$REPO", "status"},
		// soát 22/10 #3: `\"` trong nháy kép của script lồng, bash bỏ dấu `\`
		{"sh", "-c", `bash -c "rm -rf \"/tmp/x\""`},
		{"sh", "-c", `bash -c "git -C \"../main\" rm -q precious.txt"`},
	} {
		reason, ok := g.Blocked(args)
		require.Truef(t, ok, "%q must be blocked", args)
		require.Containsf(t, reason, "outside the workspace", "%q", args)
	}
	// Vế ngược: tên chỉ CHỨA hai dấu chấm vẫn ở trong workspace.
	for _, args := range [][]string{
		{"rm", "-rf", "..cache"},
		{"rm", "-f", "notes..txt"},
		{"rm", "-rf", "sub/..d"},
		{"git", "-C", "..hidden", "status"},
		{"sh", "-c", "rm -rf build/x"},
	} {
		_, ok := g.Blocked(args)
		require.Falsef(t, ok, "%q stays in the workspace", args)
	}
}
