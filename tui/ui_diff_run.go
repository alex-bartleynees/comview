package tui

import (
	vui "git.sr.ht/~rockorager/vaxis/ui"

	"github.com/rockorager/comview/diff"
	"github.com/rockorager/comview/review"
)

func runUIDiff(rows []diff.Row) error {
	cfg := loadConfig()
	commentPath := cfg.CommentFile
	if commentPath == "" {
		commentPath = review.DefaultFilePath
	}
	commentFile, err := review.LoadFile(commentPath)
	if err != nil {
		return err
	}
	if cfg.Theme != "" {
		if t, ok := ThemeByName(cfg.Theme); ok {
			theme := uiThemeFromBaseColors(t.Colors)
			return vui.Run(uiDiffRoot(rows, cfg.Wrap, commentFile.Comments), vui.WithTheme(theme))
		}
	}
	return vui.Run(uiDiffRoot(rows, cfg.Wrap, commentFile.Comments))
}
