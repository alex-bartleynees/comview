package tui

import (
	vui "git.sr.ht/~rockorager/vaxis/ui"

	"github.com/rockorager/comview/diff"
)

func runUIDiff(rows []diff.Row) error {
	cfg := loadConfig()
	base := DefaultBaseColors()
	if cfg.Theme != "" {
		if t, ok := ThemeByName(cfg.Theme); ok {
			base = t.Colors
		}
	}
	theme := uiThemeFromBaseColors(base)
	return vui.Run(uiDiffRoot(rows, cfg.Wrap), vui.WithTheme(theme))
}
