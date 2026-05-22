package tui

import (
	vui "git.sr.ht/~rockorager/vaxis/ui"

	"github.com/rockorager/comview/diff"
)

func runUIDiff(rows []diff.Row) error {
	cfg := loadConfig()
	if cfg.Theme != "" {
		if t, ok := ThemeByName(cfg.Theme); ok {
			theme := uiThemeFromBaseColors(t.Colors)
			return vui.Run(uiDiffRoot(rows, cfg.Wrap), vui.WithTheme(theme))
		}
	}
	return vui.Run(uiDiffRoot(rows, cfg.Wrap))
}
