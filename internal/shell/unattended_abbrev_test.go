package shell

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// git chấp nhận tiền tố duy nhất của option dài (`--har` = `--hard`). Guard so
// khớp nguyên văn nên lượt soát độc lập chạy được `git branch --delet`,
// `reset --har`… qua danh sách trắng. Tiền tố của option nguy hiểm phải bị coi
// như chính option đó.
func TestUnattendedGitDeniesAbbreviatedLongOptions(t *testing.T) {
	t.Parallel()
	g := unattendedGuardForTest(false)
	for _, args := range [][]string{
		{"git", "reset", "--har", "HEAD~1"},
		{"git", "reset", "--ha", "HEAD~1"},
		{"git", "reset", "--mer"},
		{"git", "branch", "--delet", "main"},
		{"git", "branch", "--del", "main"},
		{"git", "branch", "--mov", "a", "b"},
		{"git", "tag", "--dele", "v1"},
		{"git", "checkout", "-b", "x", "--forc"},
		{"git", "switch", "-c", "x", "--force-c"},
		{"git", "restore", "--staged", "--worktre", "f"},
		{"git", "restore", "--staged", "--wor", "f"},
	} {
		reason, ok := g.Blocked(args)
		require.Truef(t, ok, "%q must be blocked", args)
		require.NotEmpty(t, reason)
	}
	for _, args := range [][]string{
		{"git", "reset", "--soft", "HEAD~1"},
		{"git", "branch", "--list"},
		{"git", "restore", "--staged", "f"},
		{"git", "checkout", "-b", "feature"},
	} {
		_, ok := g.Blocked(args)
		require.Falsef(t, ok, "%q must pass", args)
	}
}
