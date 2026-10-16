package styles

import (
	"encoding/json"
	"strings"
	"testing"

	"charm.land/glamour/v2/ansi"
	"github.com/stretchr/testify/require"
)

// Màu của một theme tuỳ chỉnh thật (github-dark-dim): primary là xanh nhạt và
// warning_subtle là cam, hai màu gần cùng độ sáng. Kiểu H1 gốc của quickStyle
// (chữ warning_subtle trên nền primary) cho 1,00:1 nên tiêu đề vô hình.
const darkDimPaletteJSON = `{
  "primary": "#79c0ff", "secondary": "#d2a8ff", "accent": "#4ec9b0", "keyword": "#ff7b72",
  "fg_base": "#c9d1d9", "fg_subtle": "#b1bac4", "fg_more_subtle": "#8b949e", "fg_most_subtle": "#7d8590",
  "bg_base": "#121314", "bg_least_visible": "#181a1b", "bg_less_visible": "#202325", "bg_most_visible": "#2f3437",
  "on_primary": "#121314", "separator": "#202325",
  "destructive": "#ffa198", "error": "#f85149", "warning": "#d29922", "warning_subtle": "#ffa657",
  "info": "#79c0ff", "success": "#7ee787", "success_most_subtle": "#56d364"
}`

// Theme tuỳ chỉnh không được rơi về markdown nhiều màu của quickStyle: tiêu đề,
// inline code là chữ trung tính (fg_base) và đọc được; không dùng màu trạng thái
// (info/destructive/warning) hay nền thương hiệu (primary) cho văn bản thường.
func TestPaletteTheme_MarkdownStaysNeutral(t *testing.T) {
	t.Parallel()

	var p Palette
	require.NoError(t, json.Unmarshal([]byte(darkDimPaletteJSON), &p))
	s, err := LoadPaletteTheme("", p)
	require.NoError(t, err)

	fgBase := strings.ToLower(p.FgBase)
	forbiddenFg := []string{p.Info, p.Destructive, p.WarningSubtle, p.Warning, p.SuccessMostSubtle, p.Primary}
	for name, md := range map[string]ansi.StyleConfig{"markdown": s.Markdown, "plan": s.PlanMarkdown} {
		for part, blk := range map[string]ansi.StyleBlock{"H1": md.H1, "Heading": md.Heading, "Code": md.Code} {
			require.NotNil(t, blk.Color, "%s %s", name, part)
			require.Equal(t, fgBase, strings.ToLower(*blk.Color), "%s %s phải là chữ trung tính", name, part)
			for _, bad := range forbiddenFg {
				require.NotEqual(t, strings.ToLower(bad), strings.ToLower(*blk.Color), "%s %s dùng màu ngữ nghĩa", name, part)
			}
		}
		require.NotNil(t, md.H1.BackgroundColor)
		require.NotEqual(t, strings.ToLower(p.Primary), strings.ToLower(*md.H1.BackgroundColor), "%s: H1 không nền thương hiệu", name)
		require.GreaterOrEqual(t, contrastRatio(*md.H1.Color, *md.H1.BackgroundColor), 4.5, "%s H1", name)
		require.GreaterOrEqual(t, contrastRatio(*md.Code.Color, *md.Code.BackgroundColor), 4.5, "%s Code", name)
	}
}

// Vế ngược: theme dựng sẵn nguyên trạng giữ nguyên kiểu của nó (Charmtone là
// mặc định của upstream: H1 vẫn là chữ ấm trên nền primary).
func TestBuiltinThemeMarkdownUnchangedByNeutralPass(t *testing.T) {
	t.Parallel()

	s := CharmtonePantera()
	require.NotNil(t, s.Markdown.H1.BackgroundColor)
	require.Equal(t, *hex(charmtoneOpts().primary), *s.Markdown.H1.BackgroundColor)
}
