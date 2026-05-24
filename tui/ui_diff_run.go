package tui

import (
	vui "go.rockorager.dev/vaxis/ui"

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
			return vui.Run(uiDiffRootWithReviewFile(rows, cfg.Wrap, commentFile.Comments, commentPath, true), vui.WithTheme(theme))
		}
	}
	return vui.Run(uiDiffRootWithReviewFile(rows, cfg.Wrap, commentFile.Comments, commentPath, true))
}
