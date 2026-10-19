package agent

import (
	"testing"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/stretchr/testify/require"
)

// disabled_tools của người dùng phải áp cả cho tool bên trong agentic_fetch:
// audit g5 thấy `disabled_tools: [web_fetch, web_search, sourcegraph…]` không chặn
// được agentic_fetch vì bộ tool bên trong cố định.
func TestFilterDisabledToolsAppliesToInnerTools(t *testing.T) {
	t.Parallel()
	inner := []fantasy.AgentTool{namedTool(tools.WebFetchToolName), namedTool(tools.WebSearchToolName),
		namedTool(tools.SourcegraphToolName), namedTool(tools.ViewToolName)}
	got := filterDisabledTools(inner, []string{"web_search", "sourcegraph", "download"})
	var names []string
	for _, tl := range got {
		names = append(names, tl.Info().Name)
	}
	require.Equal(t, []string{tools.WebFetchToolName, tools.ViewToolName}, names)
}

// Không còn tool web nào thì agentic_fetch không được đưa cho model.
func TestAgenticFetchUsableNeedsAWebTool(t *testing.T) {
	t.Parallel()
	require.True(t, agenticFetchUsable(nil))
	require.True(t, agenticFetchUsable([]string{"web_search"}))
	require.False(t, agenticFetchUsable([]string{"sourcegraph", "web_search", "web_fetch", "fetch", "download"}), "the user's real disabled_tools")
}

// L5: danh sách tool người dùng thấy chỉ có "fetch"; tắt "fetch" cũng tắt web_fetch
// của agentic_fetch.
func TestDisablingFetchCountsAsDisablingWebFetch(t *testing.T) {
	t.Parallel()
	require.False(t, agenticFetchUsable([]string{tools.FetchToolName, tools.WebSearchToolName}))
	require.True(t, agenticFetchUsable([]string{tools.FetchToolName}), "web_search still works")
	require.Contains(t, agenticFetchDisabled([]string{tools.FetchToolName}), tools.WebFetchToolName)
}
