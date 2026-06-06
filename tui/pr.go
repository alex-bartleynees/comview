package tui

import (
	vui "go.rockorager.dev/vaxis/ui"

	"go.rockorager.dev/comview/review"
)

// RunPR fetches a GitHub PR diff and its review comments, then launches the TUI.
func RunPR(owner, repo string, prNumber int) error {
	diffStr, err := review.FetchPRDiff(owner, repo, prNumber)
	if err != nil {
		return err
	}
	rows, err := rowsForInput(diffStr)
	if err != nil {
		return err
	}

	ghComments, err := review.FetchPRComments(owner, repo, prNumber)
	if err != nil {
		return err
	}

	headSHA, err := review.FetchPRHeadSHA(owner, repo, prNumber)
	if err != nil {
		return err
	}

	cfg := loadConfig()
	commentPath := cfg.CommentFile
	if commentPath == "" {
		commentPath = review.DefaultFilePath
	}
	localFile, err := review.LoadFile(commentPath)
	if err != nil {
		return err
	}

	mergedDrafts := mergeDrafts(review.FromGitHubComments(ghComments), localFile.Comments)

	source := review.CommentSource{
		Provider:   "github",
		Owner:      owner,
		Repo:       repo,
		PullNumber: prNumber,
		HeadSHA:    headSHA,
	}

	root := uiDiffView{
		Rows:          rows,
		Wrap:          cfg.Wrap,
		ReviewDrafts:  mergedDrafts,
		ReviewFile:    commentPath,
		CommentSource: source,
		WorkTreeRoot:  gitWorkTreeRoot(),
		ShowStatus:    true,
		Binds:         newBindings(cfg.Keybindings),
		Theme:         cfg.Theme,
	}

	if cfg.Theme != "" {
		if t, ok := ThemeByName(cfg.Theme); ok {
			theme := uiThemeFromBaseColors(t.Colors)
			return vui.Run(root, vui.WithTheme(theme))
		}
	}
	return vui.Run(root)
}

// mergeDrafts combines GitHub-fetched drafts with locally saved drafts.
// Local drafts take precedence for matching GitHubIDs (user may have edited them).
func mergeDrafts(ghDrafts, localDrafts []review.CommentDraft) []review.CommentDraft {
	localByGHID := make(map[int64]review.CommentDraft, len(localDrafts))
	for _, d := range localDrafts {
		if d.GitHubID != 0 {
			localByGHID[d.GitHubID] = d
		}
	}

	result := make([]review.CommentDraft, 0, len(ghDrafts)+len(localDrafts))
	seenGHIDs := make(map[int64]bool, len(ghDrafts))

	for _, d := range ghDrafts {
		if local, ok := localByGHID[d.GitHubID]; ok {
			result = append(result, local)
		} else {
			result = append(result, d)
		}
		seenGHIDs[d.GitHubID] = true
	}

	// Append local-only drafts (new comments not yet submitted).
	for _, d := range localDrafts {
		if d.GitHubID == 0 || !seenGHIDs[d.GitHubID] {
			result = append(result, d)
		}
	}

	return result
}
