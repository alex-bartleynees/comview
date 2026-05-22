package tui

import (
	vui "git.sr.ht/~rockorager/vaxis/ui"

	"github.com/rockorager/comview/diff"
)

func runUIDiff(rows []diff.Row) error {
	cfg := loadConfig()
	scheme := DefaultColorScheme()
	if cfg.Theme != "" {
		if t, ok := ThemeByName(cfg.Theme); ok {
			scheme = NewColorScheme(t.Colors)
		}
	}
	return vui.Run(uiDiffView{Rows: rows, Scheme: scheme, Wrap: cfg.Wrap})
}
