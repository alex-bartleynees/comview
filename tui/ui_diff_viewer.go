package tui

import (
	"fmt"
	"strings"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	vui "git.sr.ht/~rockorager/vaxis/ui"
	"github.com/rockorager/go-uucode"

	"github.com/rockorager/comview/diff"
	"github.com/rockorager/comview/review"
)

type uiDiffView struct {
	Rows         []diff.Row
	Wrap         bool
	ReviewDrafts []review.CommentDraft
	ShowStatus   bool
}

func uiDiffRoot(rows []diff.Row, wrap bool, drafts []review.CommentDraft) vui.Widget {
	return uiDiffRootWithStatus(rows, wrap, drafts, true)
}

func uiDiffRootWithStatus(rows []diff.Row, wrap bool, drafts []review.CommentDraft, showStatus bool) vui.Widget {
	return uiDiffView{Rows: rows, Wrap: wrap, ReviewDrafts: drafts, ShowStatus: showStatus}
}

func uiThemeFromBaseColors(base BaseColors) vui.Theme {
	return vui.ThemeFromPalette(vui.PaletteFromBaseColors(vui.BaseColors{
		Black:   base.Background,
		Red:     base.Red,
		Green:   base.Green,
		Yellow:  base.Yellow,
		Blue:    base.Blue,
		Magenta: base.Magenta,
		Cyan:    base.Cyan,
		White:   base.Foreground,
	}), vui.DarkTheme)
}

func (w uiDiffView) CreateState() vui.State {
	return &uiDiffViewState{}
}

type uiDiffViewState struct {
	vui.StateBase
	scroll                  vui.ScrollController
	list                    vui.SliverListController
	cursor                  selectionPoint
	cursorCol               int
	pendingG                bool
	pendingBracket          rune
	pendingSpace            bool
	fileFinder              bool
	textObject              textObjectState
	commentEditorActive     bool
	commentEditorFocused    bool
	commentEditorInsert     bool
	commentEditorRow        int
	commentEditorBody       string
	reviewDrafts            []review.CommentDraft
	selectionAnchor         selectionPoint
	selectionActive         bool
	selectionLinewise       bool
	selectionInitialNewline bool
	selectionFinalNewline   bool
	selectionSideFiltered   bool
	selectionSide           diffSide
	mouseSelecting          bool
	mouseAnchor             selectionPoint
	mouseHasAnchor          bool
	mouseStartRow           int
	clicks                  clickState
	yankAnchor              selectionPoint
	yankCursor              selectionPoint
	yankActive              bool
	yankLinewise            bool
	yankUntil               time.Time
	searchMode              bool
	searchQuery             string
	searchMatches           []searchMatch
	searchIndex             int
	searchStart             selectionPoint
	syntaxTheme             vui.Theme
	highlighter             *SyntaxHighlighter
	highlightedTheme        vui.Theme
	highlightedRows         map[int][]vaxis.Segment
}

func (s *uiDiffViewState) Build(ctx vui.BuildContext) vui.Widget {
	w := s.Widget().(uiDiffView)
	s.clearExpiredYank(time.Now())
	s.clampCursor(w.Rows)
	theme := vui.MustDepend[vui.Theme](ctx)
	highlightedRows := s.highlightedCodeRows(w.Rows, theme)
	drafts := s.allReviewDrafts(w.ReviewDrafts)
	scrollView := vui.CustomScrollView{
		Controller: &s.scroll,
		Slivers: []vui.Widget{
			vui.SliverListBuilder{
				Controller:          &s.list,
				Count:               len(w.Rows),
				ItemExtent:          uiDiffItemExtent(w.Wrap || s.commentEditorActive || len(s.reviewDrafts) > 0),
				EstimatedItemExtent: 1,
				Overscan:            8,
				Builder: func(ctx vui.BuildContext, row int) vui.Widget {
					return s.buildItem(w.Rows, row, theme, highlightedRows, drafts, w.Wrap)
				},
			},
		},
	}
	if !w.ShowStatus {
		return scrollView
	}
	content := vui.Flex{
		Axis:               vui.Vertical,
		CrossAxisAlignment: vui.CrossAxisStretch,
		Children: []vui.Widget{
			vui.Expanded(scrollView),
			s.buildStatusBar(w.Rows, theme),
		},
	}
	entries := []vui.OverlayEntry{}
	if s.fileFinder {
		entries = append(entries, vui.OverlayEntry{Modal: true, Child: s.buildFileFinder(w.Rows, theme)})
	}
	return vui.Overlay{Child: content, Entries: entries}
}

func (s *uiDiffViewState) allReviewDrafts(base []review.CommentDraft) []review.CommentDraft {
	if len(s.reviewDrafts) == 0 {
		return base
	}
	drafts := make([]review.CommentDraft, 0, len(base)+len(s.reviewDrafts))
	drafts = append(drafts, base...)
	drafts = append(drafts, s.reviewDrafts...)
	return drafts
}

type uiDiffFileItem struct {
	Label  string
	Detail string
	Row    int
}

func (s *uiDiffViewState) buildFileFinder(rows []diff.Row, theme vui.Theme) vui.Widget {
	return vui.FuzzySelect[uiDiffFileItem]{
		Items:          uiDiffFileFinderItems(rows),
		Item:           func(item uiDiffFileItem) vui.FuzzySelectItem { return uiDiffFileSelectItem(item, theme) },
		Placeholder:    "Find file…",
		EmptyText:      "No matching files",
		MaxVisibleRows: 8,
		RowStyle:       vui.FuzzySelectOneLine,
		OnDismiss: func(vui.EventContext) {
			s.fileFinder = false
			s.SetState(func() {})
		},
		OnSelected: func(_ vui.EventContext, item uiDiffFileItem) {
			s.fileFinder = false
			s.setCursorRowAtStart(rows, item.Row)
		},
	}
}

func uiDiffFileSelectItem(item uiDiffFileItem, theme vui.Theme) vui.FuzzySelectItem {
	return vui.FuzzySelectItem{
		Title:       item.Label,
		Description: item.Detail,
		Aliases:     []string{item.Label},
		Trailing:    uiDiffFileStatWidget(item.Detail, theme),
	}
}

func uiDiffFileStatWidget(detail string, theme vui.Theme) vui.Widget {
	if detail == "" {
		return nil
	}
	parts := strings.Split(detail, " ")
	if len(parts) != 2 {
		return vui.Text{Value: detail, Style: vui.Style{Foreground: theme.MutedForeground}}
	}
	return vui.RichText{Spans: []vui.TextSpan{
		{Text: parts[0], Style: vui.Style{Foreground: theme.Palette.Green.Tone500}},
		{Text: " ", Style: vui.Style{Foreground: theme.MutedForeground}},
		{Text: parts[1], Style: vui.Style{Foreground: theme.Palette.Red.Tone500}},
	}, MaxLines: 1, Overflow: vui.TextOverflowEllipsis}
}

func uiDiffFileFinderItems(rows []diff.Row) []uiDiffFileItem {
	items := make([]uiDiffFileItem, 0)
	for rowIndex, row := range rows {
		switch row.Kind {
		case diff.RowFile:
			items = append(items, uiDiffFileItem{Label: row.Text, Detail: uiDiffFileStatsFromRow(rows, rowIndex).String(), Row: rowIndex})
		case diff.RowDiffStat:
			if row.FileName != "" {
				items = append(items, uiDiffFileItem{Label: row.FileName, Detail: statDetail(row.Stat), Row: rowIndex})
			}
		}
	}
	return items
}

func uiDiffFileStatsFromRow(rows []diff.Row, fileRow int) statusStats {
	if fileRow < 0 || fileRow >= len(rows) || rows[fileRow].Kind != diff.RowFile {
		return statusStats{}
	}
	fileEnd := len(rows)
	for rowIndex := fileRow + 1; rowIndex < len(rows); rowIndex++ {
		switch rows[rowIndex].Kind {
		case diff.RowFile, diff.RowCommitHeader:
			fileEnd = rowIndex
		}
		if fileEnd == rowIndex {
			break
		}
	}
	return rowsStats(rows[fileRow:fileEnd])
}

func (s *uiDiffViewState) buildStatusBar(rows []diff.Row, theme vui.Theme) vui.Widget {
	style := uiDiffStatusStyle(theme)
	if s.searchMode {
		return uiDiffStatusText("/"+s.searchQuery, style)
	}
	return uiDiffStatusSegments(s.statusSegments(rows, theme), style)
}

func (s *uiDiffViewState) statusSegments(rows []diff.Row, theme vui.Theme) []vaxis.Segment {
	leftSegments := s.statusLeftSegments(rows, theme)
	separatorBackground := uiDiffStatusBackground(theme)
	if len(leftSegments) > 0 {
		separatorBackground = leftSegments[0].Style.Background
	}
	segments := []vaxis.Segment{
		{Text: " " + s.statusModeLabel() + " ", Style: uiDiffStatusModeStyle(theme)},
		{Text: "", Style: vaxis.Style{Foreground: theme.Primary, Background: separatorBackground}},
	}
	segments = append(segments, leftSegments...)
	return segments
}

func (s *uiDiffViewState) statusModeLabel() string {
	if s.commentEditorActive {
		if s.commentEditorInsert {
			return "INSERT"
		}
		return "NORMAL"
	}
	if !s.selectionActive {
		return "NORMAL"
	}
	if s.selectionLinewise {
		return "V-LINE"
	}
	return "VISUAL"
}

func (s *uiDiffViewState) statusLeftSegments(rows []diff.Row, theme vui.Theme) []vaxis.Segment {
	context := s.statusContext(rows)
	sections := make([]uiDiffStatusSection, 0, 2)
	if context.Commits > 0 && context.CommitIndex > 0 && context.Commit != "" {
		commit := context.Commit
		if len(commit) > 12 {
			commit = commit[:12]
		}
		if context.Commits > 1 {
			commit = fmt.Sprintf("%d/%d %s", context.CommitIndex, context.Commits, commit)
		}
		sections = append(sections, uiDiffStatusSection{Text: commit, Foreground: theme.Palette.Blue.Tone500, Background: uiDiffStatusCommitBackground(theme)})
	}
	if context.Files > 0 && context.File != "" {
		file := context.File
		if context.Files > 1 {
			file = fmt.Sprintf("%d/%d %s", context.FileIndex, context.Files, file)
		}
		sections = append(sections, uiDiffStatusSection{Text: file, Foreground: theme.Foreground, Background: uiDiffStatusBackground(theme), Separator: "", PathBase: true})
	}
	segments := uiDiffStatusSectionSegments(sections, theme)
	if context.Files > 0 && context.File != "" {
		segments = append(segments, uiDiffStatusStatsSegments(context.FileStats, theme)...)
	}
	return segments
}

type uiDiffStatusSection struct {
	Text       string
	Foreground vaxis.Color
	Background vaxis.Color
	Separator  string
	PathBase   bool
}

func uiDiffStatusText(text string, style vaxis.Style) vui.Widget {
	return vui.DecoratedBox(
		vui.Decoration{Style: style},
		vui.SizedBox{Height: 1, Child: vui.Text{Value: text, Style: style}},
	)
}

func uiDiffStatusSegments(segments []vaxis.Segment, fillStyle vaxis.Style) vui.Widget {
	return vui.DecoratedBox(
		vui.Decoration{Style: fillStyle},
		vui.SizedBox{Height: 1, Child: vui.RichText{Spans: uiTextSpans(segments)}},
	)
}

func uiDiffStatusStyle(theme vui.Theme) vaxis.Style {
	return vaxis.Style{Foreground: theme.Foreground, Background: uiDiffStatusBackground(theme)}
}

func uiDiffStatusModeStyle(theme vui.Theme) vaxis.Style {
	return vaxis.Style{Foreground: uiDiffStatusBackground(theme), Background: theme.Primary, Attribute: vaxis.AttrBold}
}

func uiDiffStatusAddStyle(theme vui.Theme) vaxis.Style {
	return vaxis.Style{Foreground: theme.Success, Background: uiDiffStatusBackground(theme), Attribute: vaxis.AttrBold}
}

func uiDiffStatusDeleteStyle(theme vui.Theme) vaxis.Style {
	return vaxis.Style{Foreground: theme.Danger, Background: uiDiffStatusBackground(theme), Attribute: vaxis.AttrBold}
}

func uiDiffStatusBackground(theme vui.Theme) vaxis.Color {
	return theme.Surface
}

func uiDiffStatusCommitBackground(theme vui.Theme) vaxis.Color {
	if theme.Mode == vui.LightTheme {
		return theme.Palette.Blue.Tone100
	}
	return theme.Palette.Blue.Tone950
}

func uiDiffStatusSectionSegments(sections []uiDiffStatusSection, theme vui.Theme) []vaxis.Segment {
	segments := make([]vaxis.Segment, 0, len(sections)*2)
	for index, section := range sections {
		if section.Text == "" {
			continue
		}
		nextBackground := uiDiffStatusBackground(theme)
		if index+1 < len(sections) {
			nextBackground = sections[index+1].Background
		}
		separator := section.Separator
		if separator == "" {
			separator = ""
		}
		separatorStyle := vaxis.Style{Foreground: section.Background, Background: nextBackground}
		if separator == "" {
			separatorStyle = vaxis.Style{Foreground: section.Foreground, Background: section.Background}
		}
		segments = append(segments, uiDiffStatusSectionTextSegments(section)...)
		segments = append(segments, vaxis.Segment{Text: separator, Style: separatorStyle})
	}
	return segments
}

func uiDiffStatusSectionTextSegments(section uiDiffStatusSection) []vaxis.Segment {
	style := vaxis.Style{Foreground: section.Foreground, Background: section.Background, Attribute: vaxis.AttrBold}
	if !section.PathBase {
		return []vaxis.Segment{{Text: " " + section.Text + " ", Style: style}}
	}
	regularStyle := style
	regularStyle.Attribute = 0
	prefix, base := splitStatusPathBase(section.Text)
	segments := make([]vaxis.Segment, 0, 3)
	segments = append(segments, vaxis.Segment{Text: " " + prefix, Style: regularStyle})
	segments = append(segments, vaxis.Segment{Text: base, Style: style})
	segments = append(segments, vaxis.Segment{Text: " ", Style: regularStyle})
	return segments
}

func uiDiffStatusStatsSegments(stats statusStats, theme vui.Theme) []vaxis.Segment {
	return []vaxis.Segment{
		{Text: " ", Style: uiDiffStatusStyle(theme)},
		{Text: fmt.Sprintf("+%d", stats.Adds), Style: uiDiffStatusAddStyle(theme)},
		{Text: " ", Style: uiDiffStatusStyle(theme)},
		{Text: fmt.Sprintf("-%d", stats.Deletes), Style: uiDiffStatusDeleteStyle(theme)},
	}
}

func (s *uiDiffViewState) statusContext(rows []diff.Row) statusContext {
	var context statusContext
	context.Commits = uiDiffCountRows(rows, diff.RowCommitHeader)
	context.Files = uiDiffCountFiles(rows)
	context.TotalStats = rowsStats(rows)
	context.CommitIndex, context.Commit = uiDiffCurrentCommitContext(rows, s.cursor.Row)
	context.FileIndex, context.File, context.FileStats = uiDiffCurrentFileContext(rows, s.cursor.Row)
	return context
}

func uiDiffCountRows(rows []diff.Row, kind diff.RowKind) int {
	count := 0
	for _, row := range rows {
		if row.Kind == kind {
			count++
		}
	}
	return count
}

func uiDiffCountFiles(rows []diff.Row) int {
	count := 0
	for _, row := range rows {
		switch row.Kind {
		case diff.RowFile, diff.RowDiffStat:
			count++
		}
	}
	return count
}

func uiDiffCurrentCommitContext(rows []diff.Row, cursorRow int) (int, string) {
	index := 0
	currentIndex := 0
	currentCommit := ""
	for rowIndex, row := range rows {
		if row.Kind != diff.RowCommitHeader {
			continue
		}
		index++
		if rowIndex <= cursorRow {
			currentIndex = index
			currentCommit = row.Code
			if currentCommit == "" {
				currentCommit = strings.TrimPrefix(row.Text, "commit ")
			}
		}
	}
	return currentIndex, currentCommit
}

func uiDiffCurrentFileContext(rows []diff.Row, cursorRow int) (int, string, statusStats) {
	fileStart := -1
	fileIndex := 0
	currentIndex := 0
	fileName := ""
	for rowIndex, row := range rows {
		switch row.Kind {
		case diff.RowCommitHeader:
			if rowIndex <= cursorRow {
				fileStart = -1
				currentIndex = 0
				fileName = ""
			}
		case diff.RowFile:
			fileIndex++
			if rowIndex <= cursorRow {
				fileStart = rowIndex
				currentIndex = fileIndex
				fileName = row.Text
			}
		case diff.RowDiffStat:
			fileIndex++
			if rowIndex <= cursorRow {
				fileStart = rowIndex
				currentIndex = fileIndex
				fileName = row.FileName
			}
		}
	}
	if fileStart < 0 {
		return 0, "", statusStats{}
	}
	fileEnd := len(rows)
	for rowIndex := fileStart + 1; rowIndex < len(rows); rowIndex++ {
		switch rows[rowIndex].Kind {
		case diff.RowFile, diff.RowDiffStat, diff.RowCommitHeader:
			fileEnd = rowIndex
		}
		if fileEnd == rowIndex {
			break
		}
	}
	return currentIndex, fileName, rowsStats(rows[fileStart:fileEnd])
}

func uiDiffItemExtent(wrap bool) int {
	if wrap {
		return 0
	}
	return 1
}

func (s *uiDiffViewState) DidUpdateWidget(old vui.Widget) {
	s.highlightedRows = nil
}

func (s *uiDiffViewState) syntaxHighlighter(theme vui.Theme) *SyntaxHighlighter {
	if s.highlighter == nil || s.syntaxTheme != theme {
		s.syntaxTheme = theme
		s.highlighter = NewSyntaxHighlighterWithUITheme(uiSyntaxTheme{Theme: theme})
	}
	return s.highlighter
}

func (s *uiDiffViewState) highlightedCodeRows(rows []diff.Row, theme vui.Theme) map[int][]vaxis.Segment {
	if s.highlightedRows != nil && s.highlightedTheme == theme {
		return s.highlightedRows
	}
	highlighter := s.syntaxHighlighter(theme)
	s.highlightedTheme = theme
	s.highlightedRows = highlighter.HighlightRows(rows, func(kind diff.RowKind) vaxis.Style {
		return uiStyleForDiffRow(kind, theme)
	})
	return s.highlightedRows
}

type uiSyntaxTheme struct {
	Theme vui.Theme
}

func (t uiSyntaxTheme) uiThemeColors() uiThemeColors {
	if t.Theme.Mode == vui.LightTheme {
		return uiThemeColors{
			Foreground: t.Theme.Foreground,
			Muted:      t.Theme.MutedForeground,
			Blue:       t.Theme.Palette.Blue.Tone700,
			Cyan:       t.Theme.Palette.Cyan.Tone700,
			Green:      t.Theme.Palette.Green.Tone700,
			Magenta:    t.Theme.Palette.Magenta.Tone700,
			Yellow:     t.Theme.Palette.Yellow.Tone800,
			Red:        t.Theme.Palette.Red.Tone700,
			Header:     t.Theme.Primary,
			Hunk:       t.Theme.Accent,
		}
	}
	return uiThemeColors{
		Foreground: t.Theme.Foreground,
		Muted:      t.Theme.MutedForeground,
		Blue:       t.Theme.Palette.Blue.Tone500,
		Cyan:       t.Theme.Palette.Cyan.Tone500,
		Green:      t.Theme.Palette.Green.Tone500,
		Magenta:    t.Theme.Palette.Magenta.Tone500,
		Yellow:     t.Theme.Palette.Yellow.Tone500,
		Red:        t.Theme.Palette.Red.Tone500,
		Header:     t.Theme.Primary,
		Hunk:       t.Theme.Accent,
	}
}

func (s *uiDiffViewState) HandleEvent(ctx vui.EventContext, ev vui.Event) vui.EventResult {
	if ctx.Phase() != vui.CapturePhase {
		return vui.EventIgnored
	}
	if mouse, ok := ev.(vaxis.Mouse); ok {
		return s.handleMouse(ctx, mouse)
	}
	key, ok := ev.(vaxis.Key)
	if !ok || key.EventType == vaxis.EventRelease || pureModifierKey(key) {
		return vui.EventIgnored
	}
	if s.fileFinder {
		return vui.EventIgnored
	}
	w := s.Widget().(uiDiffView)
	rows := w.Rows
	if len(rows) == 0 {
		return vui.EventIgnored
	}
	if key.MatchString("Ctrl+c") {
		ctx.Quit()
		return vui.EventHandled
	}
	if s.commentEditorActive && (s.commentEditorFocused || s.commentEditorInsert) {
		return s.handleCommentEditorKey(rows, key)
	}
	if s.searchMode {
		return s.handleSearchKey(rows, key)
	}
	s.clearExpiredTextObject(time.Now())
	if s.textObject.active {
		return s.handleTextObjectKey(rows, key)
	}
	switch {
	case key.MatchString("Alt+p"):
		ctx.ToggleProfileOverlay()
		return vui.EventHandled
	case key.Matches('q'):
		ctx.Quit()
		return vui.EventHandled
	case key.Matches('c') && s.pendingBracket == ']':
		s.clearPendingKeys()
		s.jumpChange(rows, 1)
		return vui.EventHandled
	case key.Matches('c') && s.pendingBracket == '[':
		s.clearPendingKeys()
		s.jumpChange(rows, -1)
		return vui.EventHandled
	case key.Matches('n') && s.pendingBracket == ']':
		s.clearPendingKeys()
		s.jumpNote(w.Rows, w.ReviewDrafts, 1)
		return vui.EventHandled
	case key.Matches('n') && s.pendingBracket == '[':
		s.clearPendingKeys()
		s.jumpNote(w.Rows, w.ReviewDrafts, -1)
		return vui.EventHandled
	case key.Matches('/'):
		s.clearPendingKeys()
		s.enterSearchMode()
		return vui.EventHandled
	case key.Matches('v'):
		s.clearPendingKeys()
		s.toggleSelection(false)
		return vui.EventHandled
	case key.Matches('a') && s.selectionActive:
		s.clearPendingKeys()
		s.startTextObject(textObjectAround)
		return vui.EventHandled
	case key.Matches('V'):
		s.clearPendingKeys()
		s.toggleSelection(true)
		return vui.EventHandled
	case key.Matches('i') && s.selectionActive && !s.selectionLinewise:
		s.clearPendingKeys()
		s.startTextObject(textObjectInner)
		return vui.EventHandled
	case key.Matches('i'):
		s.clearPendingKeys()
		s.openCommentEditor(rows)
		return vui.EventHandled
	case key.Matches('y'):
		s.clearPendingKeys()
		s.yankSelection(ctx, rows)
		return vui.EventHandled
	case key.Matches(vaxis.KeySpace):
		s.pendingG = false
		s.pendingBracket = 0
		s.pendingSpace = true
		return vui.EventHandled
	case key.Matches('e') && s.pendingSpace:
		s.clearPendingKeys()
		if len(uiDiffFileFinderItems(rows)) == 0 {
			return vui.EventHandled
		}
		s.fileFinder = true
		s.SetState(func() {})
		return vui.EventHandled
	case key.Matches('n'):
		s.clearPendingKeys()
		s.moveSearchMatch(rows, 1)
		return vui.EventHandled
	case key.Matches('N'):
		s.clearPendingKeys()
		s.moveSearchMatch(rows, -1)
		return vui.EventHandled
	case key.Matches(vaxis.KeyEsc):
		s.clearPendingKeys()
		if s.selectionActive {
			s.clearLineSelection()
			s.SetState(func() {})
			return vui.EventHandled
		}
		if s.searchQuery != "" || len(s.searchMatches) > 0 || s.searchIndex != -1 {
			s.clearSearch()
			s.SetState(func() {})
			return vui.EventHandled
		}
		return vui.EventIgnored
	case key.Matches(']'):
		s.pendingG = false
		s.pendingBracket = ']'
		return vui.EventHandled
	case key.Matches('['):
		s.pendingG = false
		s.pendingBracket = '['
		return vui.EventHandled
	case key.Matches('g') && s.pendingG:
		s.clearPendingKeys()
		s.setCursorRow(rows, 0)
		return vui.EventHandled
	case key.Matches('g'):
		s.pendingBracket = 0
		s.pendingG = true
		return vui.EventHandled
	case key.Matches(vaxis.KeyHome):
		s.clearPendingKeys()
		s.setCursorRow(rows, 0)
		return vui.EventHandled
	case key.Matches('G'), key.Matches(vaxis.KeyEnd):
		s.clearPendingKeys()
		s.setCursorRow(rows, len(rows)-1)
		return vui.EventHandled
	case key.MatchString("Ctrl+d"), key.Matches(vaxis.KeyPgDown):
		s.clearPendingKeys()
		s.moveCursorRows(rows, s.halfPageRows())
		return vui.EventHandled
	case key.MatchString("Ctrl+u"), key.Matches(vaxis.KeyPgUp):
		s.clearPendingKeys()
		s.moveCursorRows(rows, -s.halfPageRows())
		return vui.EventHandled
	case key.Matches('J'):
		s.clearPendingKeys()
		s.jumpCommit(rows, 1)
		return vui.EventHandled
	case key.Matches('K'):
		s.clearPendingKeys()
		s.jumpCommit(rows, -1)
		return vui.EventHandled
	case key.Matches('j'), key.Matches(vaxis.KeyDown), key.MatchString("Down"):
		s.clearPendingKeys()
		if s.moveIntoCommentEditor(rows, 1) {
			return vui.EventHandled
		}
		s.moveCursorRows(rows, 1)
		return vui.EventHandled
	case key.Matches('k'), key.Matches(vaxis.KeyUp), key.MatchString("Up"):
		s.clearPendingKeys()
		if s.moveIntoCommentEditor(rows, -1) {
			return vui.EventHandled
		}
		s.moveCursorRows(rows, -1)
		return vui.EventHandled
	case key.Matches('h'), key.Matches(vaxis.KeyLeft), key.MatchString("Left"):
		s.clearPendingKeys()
		s.moveCursorCols(rows, -1)
		return vui.EventHandled
	case key.Matches('l'), key.Matches(vaxis.KeyRight), key.MatchString("Right"):
		s.clearPendingKeys()
		s.moveCursorCols(rows, 1)
		return vui.EventHandled
	case key.Matches('0'):
		s.clearPendingKeys()
		s.moveCursorLineStart(rows)
		return vui.EventHandled
	case key.Matches('$'):
		s.clearPendingKeys()
		s.moveCursorLineEnd(rows)
		return vui.EventHandled
	default:
		s.clearPendingKeys()
		return vui.EventIgnored
	}
}

func (s *uiDiffViewState) handleMouse(_ vui.EventContext, mouse vaxis.Mouse) vui.EventResult {
	if s.fileFinder {
		return vui.EventIgnored
	}
	w := s.Widget().(uiDiffView)
	rows := w.Rows
	if len(rows) == 0 {
		return vui.EventIgnored
	}
	if mouseWheelButton(mouse.Button) {
		return s.handleMouseWheel(mouse)
	}
	switch mouse.EventType {
	case vaxis.EventPress:
		if mouse.Button != vaxis.MouseLeftButton {
			return vui.EventIgnored
		}
		point, ok := s.selectionPointForMouse(rows, mouse)
		if !ok {
			s.mouseSelecting = true
			s.mouseHasAnchor = false
			s.mouseStartRow = s.documentRowForMouse(mouse)
			s.clearLineSelection()
			s.clearPendingKeys()
			s.SetState(func() {})
			return vui.EventHandled
		}
		s.setCursorPointWithoutReveal(rows, point)
		s.clearLineSelection()
		s.yankActive = false
		s.clearPendingKeys()
		switch s.registerClick(point, time.Now()) {
		case 2:
			s.mouseSelecting = false
			s.selectTokenAt(rows, point)
			s.SetState(func() {})
			return vui.EventHandled
		case 3:
			s.mouseSelecting = false
			s.selectRowAt(rows, point)
			s.SetState(func() {})
			return vui.EventHandled
		}
		s.mouseSelecting = true
		s.mouseAnchor = point
		s.mouseHasAnchor = true
		s.mouseStartRow = point.Row
		s.SetState(func() {})
		return vui.EventHandled
	case vaxis.EventMotion:
		if !s.mouseSelecting || mouse.Button == vaxis.MouseNoButton {
			return vui.EventIgnored
		}
		point, ok := s.selectionPointForMouse(rows, mouse)
		if !ok {
			return vui.EventIgnored
		}
		if !s.mouseHasAnchor {
			s.mouseAnchor = s.dragAnchor(rows, point)
			s.mouseHasAnchor = true
		}
		if !s.selectionActive && point == s.mouseAnchor {
			return vui.EventIgnored
		}
		if !s.selectionActive {
			s.selectionActive = true
			s.selectionLinewise = false
			s.selectionAnchor = s.mouseAnchor
		}
		s.setCursorPointWithoutReveal(rows, point)
		s.SetState(func() {})
		return vui.EventHandled
	case vaxis.EventRelease:
		if !s.mouseSelecting {
			return vui.EventIgnored
		}
		s.mouseSelecting = false
		s.mouseHasAnchor = false
		if point, ok := s.selectionPointForMouse(rows, mouse); ok {
			s.setCursorPointWithoutReveal(rows, point)
		}
		s.SetState(func() {})
		return vui.EventHandled
	default:
		return vui.EventIgnored
	}
}

func (s *uiDiffViewState) handleMouseWheel(mouse vaxis.Mouse) vui.EventResult {
	if !s.mouseSelecting || !s.selectionActive {
		return vui.EventIgnored
	}
	var lines int
	switch mouse.Button {
	case vaxis.MouseWheelDown:
		lines = mouseWheelScrollLines
	case vaxis.MouseWheelUp:
		lines = -mouseWheelScrollLines
	default:
		return vui.EventIgnored
	}
	if lines == 0 {
		return vui.EventIgnored
	}
	s.scroll.ScrollByLines(lines)
	w := s.Widget().(uiDiffView)
	if point, ok := s.selectionPointForMouse(w.Rows, mouse); ok {
		s.setCursorPointWithoutReveal(w.Rows, point)
	}
	s.SetState(func() {})
	return vui.EventHandled
}

func (s *uiDiffViewState) dragAnchor(rows []diff.Row, point selectionPoint) selectionPoint {
	_, end, ok := uiDiffCodeRange(rows, point.Row)
	if !ok {
		return point
	}
	switch {
	case point.Row > s.mouseStartRow:
		return selectionPoint{Row: point.Row, Col: 0}
	case point.Row < s.mouseStartRow:
		return selectionPoint{Row: point.Row, Col: maxInt(0, end-1)}
	default:
		return point
	}
}

func (s *uiDiffViewState) registerClick(point selectionPoint, now time.Time) int {
	if s.clicks.Point == point && now.Sub(s.clicks.At) <= multiClickTimeout {
		s.clicks.Count++
	} else {
		s.clicks.Count = 1
	}
	if s.clicks.Count > 3 {
		s.clicks.Count = 1
	}
	s.clicks.Point = point
	s.clicks.At = now
	return s.clicks.Count
}

func (s *uiDiffViewState) selectTokenAt(rows []diff.Row, point selectionPoint) {
	if point.Row < 0 || point.Row >= len(rows) || !selectableDiffRow(rows[point.Row].Kind) {
		return
	}
	row := rows[point.Row]
	start, end := tokenRangeAtWithTabWidth(row.Code, point.Col, tabWidthForFile(row.FileName))
	_, codeEnd, ok := uiDiffCodeRange(rows, point.Row)
	if !ok {
		return
	}
	start = clampUIDiffInt(start, 0, maxInt(0, codeEnd-1))
	end = clampUIDiffInt(end, start+1, codeEnd)
	s.selectionActive = true
	s.selectionLinewise = false
	s.selectionAnchor = selectionPoint{Row: point.Row, Col: start}
	s.setCursorPointWithoutReveal(rows, selectionPoint{Row: point.Row, Col: maxInt(start, end-1)})
}

func (s *uiDiffViewState) selectRowAt(rows []diff.Row, point selectionPoint) {
	if point.Row < 0 || point.Row >= len(rows) || !selectableDiffRow(rows[point.Row].Kind) {
		return
	}
	_, end, ok := uiDiffCodeRange(rows, point.Row)
	if !ok {
		return
	}
	s.selectionActive = true
	s.selectionLinewise = true
	s.selectionAnchor = selectionPoint{Row: point.Row, Col: 0}
	s.setCursorPointWithoutReveal(rows, selectionPoint{Row: point.Row, Col: maxInt(0, end-1)})
}

func (s *uiDiffViewState) selectionPointForMouse(rows []diff.Row, mouse vaxis.Mouse) (selectionPoint, bool) {
	rowIndex, ok := s.rowForMouse(mouse)
	if !ok || rowIndex < 0 || rowIndex >= len(rows) || !selectableDiffRow(rows[rowIndex].Kind) {
		return selectionPoint{}, false
	}
	col := mouse.Col - uiDiffCodeOffset(rows)
	_, end, ok := uiDiffCodeRange(rows, rowIndex)
	if !ok {
		return selectionPoint{}, false
	}
	col = clampUIDiffInt(col, 0, maxInt(0, end-1))
	return selectionPoint{Row: rowIndex, Col: col}, true
}

func (s *uiDiffViewState) rowForMouse(mouse vaxis.Mouse) (int, bool) {
	first, last, ok := s.list.VisibleRange()
	if !ok || mouse.Row < 0 {
		return 0, false
	}
	if s.scroll.Attached() {
		metrics := s.scroll.Metrics()
		if mouse.Row >= metrics.ViewportHeight {
			return 0, false
		}
		logicalY := metrics.ScrollOffset + mouse.Row
		for row := first; row < last; row++ {
			offset, ok := s.list.OffsetForIndex(row)
			if !ok {
				continue
			}
			nextOffset := offset + 1
			if row+1 < last {
				if next, ok := s.list.OffsetForIndex(row + 1); ok {
					nextOffset = next
				}
			}
			if logicalY >= offset && logicalY < nextOffset {
				return row, true
			}
		}
		return 0, false
	}
	row := first + mouse.Row
	if row < first || row >= last {
		return 0, false
	}
	return row, true
}

func (s *uiDiffViewState) documentRowForMouse(mouse vaxis.Mouse) int {
	row, ok := s.rowForMouse(mouse)
	if !ok {
		return -1
	}
	return row
}

func uiDiffCodeOffset(rows []diff.Row) int {
	oldWidth, newWidth := uiDiffGutterWidths(rows)
	if oldWidth == 0 && newWidth == 0 {
		return 0
	}
	return oldWidth + 1 + newWidth + 1 + 1 + 1
}

func (s *uiDiffViewState) clearPendingKeys() {
	s.pendingG = false
	s.pendingBracket = 0
	s.pendingSpace = false
}

func (s *uiDiffViewState) openCommentEditor(rows []diff.Row) {
	if s.cursor.Row < 0 || s.cursor.Row >= len(rows) || !reviewAnchorValid(rows[s.cursor.Row].Review) {
		return
	}
	if s.commentEditorActive && s.commentEditorRow == s.cursor.Row {
		s.commentEditorFocused = true
		s.commentEditorInsert = true
		s.clearLineSelection()
		s.SetState(func() {})
		return
	}
	s.commentEditorActive = true
	s.commentEditorFocused = true
	s.commentEditorInsert = true
	s.commentEditorRow = s.cursor.Row
	s.commentEditorBody = ""
	s.clearLineSelection()
	s.SetState(func() {})
}

func (s *uiDiffViewState) handleCommentEditorKey(rows []diff.Row, key vaxis.Key) vui.EventResult {
	if !s.commentEditorInsert {
		switch {
		case key.Matches(vaxis.KeyEsc):
			if strings.TrimSpace(s.commentEditorBody) == "" {
				s.closeCommentEditor()
				s.SetState(func() {})
				return vui.EventHandled
			}
			s.commentEditorFocused = false
			s.SetState(func() {})
			return vui.EventHandled
		case key.Matches('i'):
			s.commentEditorInsert = true
			s.SetState(func() {})
			return vui.EventHandled
		case key.Matches('j'), key.Matches(vaxis.KeyDown), key.MatchString("Down"):
			s.commentEditorFocused = false
			s.setCursorRow(rows, s.commentEditorRow+1)
			return vui.EventHandled
		case key.Matches('k'), key.Matches(vaxis.KeyUp), key.MatchString("Up"):
			s.commentEditorFocused = false
			s.setCursorRow(rows, s.commentEditorRow)
			return vui.EventHandled
		default:
			return vui.EventIgnored
		}
	}
	switch {
	case key.Matches(vaxis.KeyEsc):
		if strings.TrimSpace(s.commentEditorBody) == "" {
			s.closeCommentEditor()
			s.SetState(func() {})
			return vui.EventHandled
		}
		s.commentEditorFocused = false
		s.commentEditorInsert = false
		s.SetState(func() {})
		return vui.EventHandled
	case key.MatchString("Ctrl+s"):
		s.submitCommentEditor(rows)
		s.SetState(func() {})
		return vui.EventHandled
	default:
		return vui.EventIgnored
	}
}

func (s *uiDiffViewState) submitCommentEditor(rows []diff.Row) {
	if !s.commentEditorActive || s.commentEditorRow < 0 || s.commentEditorRow >= len(rows) {
		return
	}
	body := strings.TrimSpace(s.commentEditorBody)
	if body != "" {
		anchor := rows[s.commentEditorRow].Review
		s.reviewDrafts = append(s.reviewDrafts, review.CommentDraft{
			Path:             anchor.Path,
			Line:             anchor.Line,
			Side:             anchor.Side,
			CommitID:         anchor.CommitID,
			OriginalCommitID: anchor.OriginalCommitID,
			Body:             s.commentEditorBody,
		})
	}
	s.commentEditorActive = false
	s.closeCommentEditor()
}

func (s *uiDiffViewState) closeCommentEditor() {
	s.commentEditorActive = false
	s.commentEditorFocused = false
	s.commentEditorInsert = false
	s.commentEditorBody = ""
}

func (s *uiDiffViewState) moveIntoCommentEditor(rows []diff.Row, delta int) bool {
	if !s.commentEditorActive || s.commentEditorInsert || s.commentEditorFocused {
		return false
	}
	if delta > 0 && s.cursor.Row == s.commentEditorRow {
		s.commentEditorFocused = true
		s.SetState(func() {})
		return true
	}
	if delta < 0 && s.cursor.Row == uiDiffCursorTargetRow(rows, s.commentEditorRow+1, 1) {
		s.cursor.Row = s.commentEditorRow
		s.cursor.Col = s.clampCursorCol(rows, s.cursor.Row, s.cursorCol)
		s.cursorCol = s.cursor.Col
		s.commentEditorFocused = true
		s.revealCursorRow()
		s.SetState(func() {})
		return true
	}
	return false
}

func (s *uiDiffViewState) startTextObject(kind textObjectKind) {
	s.textObject = textObjectState{active: true, kind: kind, at: time.Now()}
	s.SetState(func() {})
}

func (s *uiDiffViewState) clearExpiredTextObject(now time.Time) {
	if s.textObject.active && now.Sub(s.textObject.at) > pendingKeyTimeout {
		s.textObject = textObjectState{}
	}
}

func (s *uiDiffViewState) handleTextObjectKey(rows []diff.Row, key vaxis.Key) vui.EventResult {
	state := s.textObject
	s.textObject = textObjectState{}
	if key.Matches(vaxis.KeyEsc) {
		s.SetState(func() {})
		return vui.EventHandled
	}
	object := textObjectKeyRune(key)
	if object == 'w' {
		if s.selectWordTextObject(rows, state.kind) {
			s.SetState(func() {})
			return vui.EventHandled
		}
		s.SetState(func() {})
		return vui.EventHandled
	}
	open, close, ok := textObjectDelimiters(object)
	if !ok || !s.selectDelimitedTextObject(rows, state.kind, open, close) {
		s.SetState(func() {})
		return vui.EventHandled
	}
	s.SetState(func() {})
	return vui.EventHandled
}

func (s *uiDiffViewState) selectWordTextObject(rows []diff.Row, kind textObjectKind) bool {
	if s.cursor.Row < 0 || s.cursor.Row >= len(rows) || !selectableDiffRow(rows[s.cursor.Row].Kind) {
		return false
	}
	row := rows[s.cursor.Row]
	start, end := tokenRangeAtWithTabWidth(row.Code, s.cursor.Col, tabWidthForFile(row.FileName))
	_, codeEnd, ok := uiDiffCodeRange(rows, s.cursor.Row)
	if !ok {
		return false
	}
	start = clampUIDiffInt(start, 0, maxInt(0, codeEnd-1))
	end = clampUIDiffInt(end, start+1, codeEnd)
	if kind == textObjectAround {
		end = uiDiffExtendAroundWord(row, start, end, codeEnd)
		if end == start {
			end = minInt(codeEnd, start+1)
		}
	}
	s.selectionActive = true
	s.selectionLinewise = false
	s.selectionInitialNewline = false
	s.selectionFinalNewline = false
	s.selectionSideFiltered = false
	s.selectionAnchor = selectionPoint{Row: s.cursor.Row, Col: start}
	s.setCursorPointWithoutReveal(rows, selectionPoint{Row: s.cursor.Row, Col: maxInt(start, end-1)})
	return true
}

func uiDiffExtendAroundWord(row diff.Row, start int, end int, codeEnd int) int {
	for end < codeEnd && isSpaceRune(runeAtCellWithTabWidth(row.Code, end, tabWidthForFile(row.FileName))) {
		end++
	}
	if end == codeEnd {
		for start > 0 && isSpaceRune(runeAtCellWithTabWidth(row.Code, start-1, tabWidthForFile(row.FileName))) {
			start--
		}
	}
	return end
}

func (s *uiDiffViewState) selectDelimitedTextObject(rows []diff.Row, kind textObjectKind, open rune, close rune) bool {
	bounds, ok := s.textObjectSearchBounds(rows)
	if !ok {
		return false
	}
	cursor := textObjectPosition{Row: s.cursor.Row, Col: s.cursor.Col - bounds.CodeStart[s.cursor.Row]}
	if cursor.Col < 0 {
		cursor.Col = 0
	}
	if width := bounds.CodeWidth[s.cursor.Row]; cursor.Col >= width {
		cursor.Col = maxInt(0, width-1)
	}
	openPos, closePos, ok := findDelimitedTextObject(bounds, cursor, open, close)
	if !ok {
		return false
	}
	start := openPos
	end := closePos
	includeInitialNewline := false
	includeFinalNewline := false
	if kind == textObjectInner {
		start = advanceTextObjectPosition(bounds, openPos)
		end = previousTextObjectPosition(bounds, closePos)
		includeInitialNewline = openPos.Row != closePos.Row && start.Row > openPos.Row
		includeFinalNewline = openPos.Row != closePos.Row && end.Row < closePos.Row
	}
	if textObjectPositionLess(end, start) {
		return false
	}
	anchor := selectionPoint{Row: start.Row, Col: bounds.CodeStart[start.Row] + start.Col}
	if includeInitialNewline {
		anchor = selectionPoint{Row: openPos.Row, Col: bounds.CodeStart[openPos.Row] + bounds.CodeWidth[openPos.Row]}
	}
	s.selectionActive = true
	s.selectionLinewise = false
	s.selectionInitialNewline = includeInitialNewline
	s.selectionFinalNewline = includeFinalNewline
	s.selectionSideFiltered = false
	if start.Row != end.Row && s.cursor.Row >= 0 && s.cursor.Row < len(rows) {
		s.selectionSideFiltered = true
		s.selectionSide = sideForRow(rows[s.cursor.Row])
	}
	s.selectionAnchor = anchor
	s.setCursorPointWithoutReveal(rows, selectionPoint{Row: end.Row, Col: bounds.CodeStart[end.Row] + end.Col})
	return true
}

func (s *uiDiffViewState) textObjectSearchBounds(rows []diff.Row) (textObjectBounds, bool) {
	if s.cursor.Row < 0 || s.cursor.Row >= len(rows) || !selectableDiffRow(rows[s.cursor.Row].Kind) {
		return textObjectBounds{}, false
	}
	side := sideForRow(rows[s.cursor.Row])
	start := s.cursor.Row
	for start > 0 && textObjectRowsContiguous(rows[start-1]) {
		start--
	}
	end := s.cursor.Row
	for end+1 < len(rows) && textObjectRowsContiguous(rows[end+1]) {
		end++
	}
	bounds := textObjectBounds{
		Start:     start,
		End:       end,
		Side:      side,
		Code:      make(map[int]string, end-start+1),
		CodeStart: make(map[int]int, end-start+1),
		CodeWidth: make(map[int]int, end-start+1),
		TabWidth:  make(map[int]int, end-start+1),
	}
	for rowIndex := start; rowIndex <= end; rowIndex++ {
		if !rowOnTextObjectSide(rows[rowIndex], side) {
			continue
		}
		bounds.Code[rowIndex] = rows[rowIndex].Code
		bounds.CodeStart[rowIndex] = 0
		bounds.CodeWidth[rowIndex] = codeCellWidth(rows[rowIndex])
		bounds.TabWidth[rowIndex] = tabWidthForFile(rows[rowIndex].FileName)
	}
	if _, ok := bounds.CodeStart[s.cursor.Row]; !ok {
		return textObjectBounds{}, false
	}
	return bounds, true
}

func (s *uiDiffViewState) toggleSelection(linewise bool) {
	if s.selectionActive {
		s.clearLineSelection()
	} else {
		w := s.Widget().(uiDiffView)
		if s.cursor.Row < 0 || s.cursor.Row >= len(w.Rows) || !selectableDiffRow(w.Rows[s.cursor.Row].Kind) {
			return
		}
		s.selectionActive = true
		s.selectionLinewise = linewise
		s.selectionAnchor = s.cursor
		s.yankActive = false
	}
	s.SetState(func() {})
}

func (s *uiDiffViewState) clearLineSelection() {
	s.selectionActive = false
	s.selectionLinewise = false
	s.selectionInitialNewline = false
	s.selectionFinalNewline = false
	s.selectionSideFiltered = false
	s.selectionAnchor = selectionPoint{}
}

func (s *uiDiffViewState) yankSelection(ctx vui.EventContext, rows []diff.Row) {
	if !s.selectionActive {
		return
	}
	text := s.selectionText(rows)
	if text == "" {
		return
	}
	ctx.Copy(text)
	s.yankActive = true
	s.yankLinewise = s.selectionLinewise
	s.yankAnchor = s.selectionAnchor
	s.yankCursor = s.cursor
	s.yankUntil = time.Now().Add(yankHighlightDuration)
	s.clearLineSelection()
	s.SetState(func() {})
	runtime := ctx.Runtime()
	time.AfterFunc(yankHighlightDuration, func() {
		runtime.Dispatch(func() {
			s.SetState(func() { s.clearExpiredYank(time.Now()) })
		})
	})
}

func (s *uiDiffViewState) clearExpiredYank(now time.Time) {
	if s.yankActive && !s.yankUntil.IsZero() && !now.Before(s.yankUntil) {
		s.yankActive = false
		s.yankLinewise = false
		s.yankAnchor = selectionPoint{}
		s.yankCursor = selectionPoint{}
		s.yankUntil = time.Time{}
	}
}

func (s *uiDiffViewState) selectionText(rows []diff.Row) string {
	if !s.selectionActive {
		return ""
	}
	start := s.selectionAnchor
	end := s.cursor
	if selectionPointLess(end, start) {
		start, end = end, start
	}
	var text strings.Builder
	wroteRow := false
	for rowIndex := start.Row; rowIndex <= end.Row && rowIndex < len(rows); rowIndex++ {
		if rowIndex < 0 || !s.selectionIncludesRow(rows[rowIndex]) {
			continue
		}
		row := rows[rowIndex]
		rowStart, rowEnd := 0, codeCellWidth(row)
		if s.selectionInitialNewline && rowIndex == start.Row {
			wroteRow = true
			continue
		}
		if !s.selectionLinewise {
			if rowIndex == start.Row {
				rowStart = maxInt(rowStart, start.Col)
			}
			if rowIndex == end.Row {
				rowEnd = minInt(rowEnd, end.Col+1)
			}
		}
		rowText := cellTextRangeWithTabWidth(row.Code, rowStart, rowEnd, tabWidthForFile(row.FileName))
		if rowText == "" {
			continue
		}
		if wroteRow {
			text.WriteByte('\n')
		}
		text.WriteString(rowText)
		wroteRow = true
	}
	if s.selectionFinalNewline && wroteRow {
		text.WriteByte('\n')
	}
	return text.String()
}

func (s *uiDiffViewState) selectionIncludesRow(row diff.Row) bool {
	if !selectableDiffRow(row.Kind) {
		return false
	}
	return !s.selectionSideFiltered || rowOnTextObjectSide(row, s.selectionSide)
}

func (s *uiDiffViewState) lineSelected(row int) bool {
	if !s.selectionActive || !s.selectionLinewise {
		return false
	}
	return s.lineInRange(row, s.selectionAnchor, s.cursor)
}

func (s *uiDiffViewState) lineYanked(row int) bool {
	if !s.yankActive || !s.yankLinewise {
		return false
	}
	return s.lineInRange(row, s.yankAnchor, s.yankCursor)
}

func (s *uiDiffViewState) lineInRange(row int, anchor selectionPoint, cursor selectionPoint) bool {
	w := s.Widget().(uiDiffView)
	if row < 0 || row >= len(w.Rows) || !selectableDiffRow(w.Rows[row].Kind) {
		return false
	}
	start, end := orderedUIDiffInts(anchor.Row, cursor.Row)
	return row >= start && row <= end
}

func (s *uiDiffViewState) charSelectionRange(rowIndex int, row diff.Row) (int, int, bool) {
	if !s.selectionActive || s.selectionLinewise {
		return 0, 0, false
	}
	if !s.selectionIncludesRow(row) {
		return 0, 0, false
	}
	return uiDiffSelectionRange(rowIndex, row, s.selectionAnchor, s.cursor)
}

func uiDiffSelectionRange(rowIndex int, row diff.Row, anchor selectionPoint, cursor selectionPoint) (int, int, bool) {
	if !selectableDiffRow(row.Kind) {
		return 0, 0, false
	}
	start := anchor
	end := cursor
	if selectionPointLess(end, start) {
		start, end = end, start
	}
	if rowIndex < start.Row || rowIndex > end.Row {
		return 0, 0, false
	}
	_, codeEnd, ok := uiDiffCodeRangeForRow(row)
	if !ok {
		return 0, 0, false
	}
	from, to := 0, maxInt(0, codeEnd-1)
	if rowIndex == start.Row {
		from = start.Col
	}
	if rowIndex == end.Row {
		to = end.Col
	}
	from = clampUIDiffInt(from, 0, maxInt(0, codeEnd-1))
	to = clampUIDiffInt(to, 0, maxInt(0, codeEnd-1))
	if to < from {
		return 0, 0, false
	}
	return from, to + 1, true
}

func (s *uiDiffViewState) charYankRange(rowIndex int, row diff.Row) (int, int, bool) {
	if !s.yankActive || s.yankLinewise {
		return 0, 0, false
	}
	return uiDiffSelectionRange(rowIndex, row, s.yankAnchor, s.yankCursor)
}

func (s *uiDiffViewState) enterSearchMode() {
	s.searchMode = true
	s.searchQuery = ""
	s.searchMatches = nil
	s.searchIndex = -1
	s.searchStart = s.cursor
	s.SetState(func() {})
}

func (s *uiDiffViewState) handleSearchKey(rows []diff.Row, key vaxis.Key) vui.EventResult {
	switch {
	case key.Matches(vaxis.KeyEsc):
		s.searchMode = false
		s.clearSearch()
		s.SetState(func() {})
		return vui.EventHandled
	case key.Matches(vaxis.KeyEnter):
		s.searchMode = false
		s.updateSearchMatches(rows)
		if len(s.searchMatches) == 0 {
			s.SetState(func() {})
			return vui.EventHandled
		}
		s.searchIndex = s.nextSearchIndexFromPoint(s.searchStart, 1)
		s.applySearchMatch(rows)
		s.SetState(func() {})
		return vui.EventHandled
	case key.Matches(vaxis.KeyBackspace), key.Matches('h', vaxis.ModCtrl):
		if s.searchQuery != "" {
			runes := []rune(s.searchQuery)
			s.searchQuery = string(runes[:len(runes)-1])
			s.updateSearchMatches(rows)
			s.SetState(func() {})
		}
		return vui.EventHandled
	case key.Text != "" && key.Modifiers&(vaxis.ModCtrl|vaxis.ModAlt|vaxis.ModSuper) == 0:
		for _, r := range key.Text {
			if r >= ' ' {
				s.searchQuery += string(r)
			}
		}
		s.updateSearchMatches(rows)
		s.SetState(func() {})
		return vui.EventHandled
	default:
		return vui.EventIgnored
	}
}

func (s *uiDiffViewState) clearSearch() {
	s.searchQuery = ""
	s.searchMatches = nil
	s.searchIndex = -1
}

func (s *uiDiffViewState) buildItem(rows []diff.Row, rowIndex int, theme vui.Theme, highlightedRows map[int][]vaxis.Segment, drafts []review.CommentDraft, wrap bool) vui.Widget {
	row := rows[rowIndex]
	active := rowIndex == s.cursor.Row && !s.commentEditorInsert && !s.commentEditorFocused
	selected := s.lineSelected(rowIndex)
	yanked := s.lineYanked(rowIndex)
	var item vui.Widget
	if !uiDiffRowUsesGrid(row) {
		item = uiDiffFullWidthRow(row, rowIndex, uiDiffRowBackground(active, selected, yanked, theme), theme, s.searchMatches, wrap)
	} else {
		item = s.buildRow(rows, row, rowIndex, active, selected, yanked, s.cursor.Col, theme, highlightedRows, s.searchMatches, wrap)
	}
	children := []vui.Widget{item}
	if s.commentEditorActive && s.commentEditorRow == rowIndex {
		children = append(children, s.buildCommentEditor(theme))
	}
	for _, draft := range reviewDraftsForRow(row, drafts) {
		children = append(children, uiDiffReviewDraft(draft, theme))
	}
	if len(children) == 1 {
		return item
	}
	return vui.Column(children...)
}

func (s *uiDiffViewState) buildCommentEditor(theme vui.Theme) vui.Widget {
	background := theme.Surface
	if s.commentEditorFocused || s.commentEditorInsert {
		background = theme.SurfaceHovered
	}
	boxStyle := vaxis.Style{Foreground: theme.Foreground, Background: background}
	if !s.commentEditorInsert {
		body := s.commentEditorBody
		if body == "" {
			body = "Add comment…"
		}
		return uiDiffCommentBox(uiDiffCommentColumn(
			uiDiffCommentHalfBlock("▄", background, theme),
			vui.DecoratedBox(
				vui.Decoration{Style: boxStyle},
				vui.Padding(vui.Symmetric(2, 0), vui.RichText{Spans: []vui.TextSpan{{Text: body, Style: boxStyle}}, SoftWrap: true}),
			),
			uiDiffCommentHalfBlock("▀", background, theme),
		))
	}
	return uiDiffCommentBox(uiDiffCommentColumn(
		uiDiffCommentHalfBlock("▄", background, theme),
		vui.DecoratedBox(
			vui.Decoration{Style: boxStyle},
			vui.FocusScope{AutoFocus: true, Child: vui.TextArea{
				Value:       s.commentEditorBody,
				Placeholder: "Add comment…",
				MinHeight:   1,
				SoftWrap:    true,
				Padding:     vui.Symmetric(2, 0),
				OnChanged: func(_ vui.EventContext, value string) {
					s.commentEditorBody = value
					s.SetState(func() {})
				},
			}},
		),
		uiDiffCommentHalfBlock("▀", background, theme),
	))
}

func uiDiffCommentColumn(children ...vui.Widget) vui.Widget {
	return vui.Flex{Axis: vui.Vertical, CrossAxisAlignment: vui.CrossAxisStretch, Children: children}
}

func uiDiffCommentHalfBlock(block string, foreground vaxis.Color, theme vui.Theme) vui.Widget {
	style := vaxis.Style{Foreground: foreground, Background: theme.Background}
	return vui.Text{Value: strings.Repeat(block, 72), Style: style, Overflow: vui.TextOverflowClip}
}

func uiDiffReviewDraft(draft review.CommentDraft, theme vui.Theme) vui.Widget {
	style := vaxis.Style{Foreground: theme.Foreground, Background: theme.Surface}
	return uiDiffCommentBox(vui.DecoratedBox(
		vui.Decoration{Style: style, Border: vui.BorderLine(theme.Primary)},
		vui.Padding(vui.All(1), vui.RichText{Spans: []vui.TextSpan{{Text: draft.Body, Style: style}}, SoftWrap: true}),
	))
}

func uiDiffCommentBox(child vui.Widget) vui.Widget {
	return vui.ConstrainedBox{Constraints: vui.Constraints{MinWidth: 72, MaxWidth: 72}, Child: child}
}

func reviewDraftsForRow(row diff.Row, drafts []review.CommentDraft) []review.CommentDraft {
	if !reviewAnchorValid(row.Review) {
		return nil
	}
	matches := make([]review.CommentDraft, 0, 1)
	for _, draft := range drafts {
		if reviewDraftEndsAt(draft, row.Review) {
			matches = append(matches, draft)
		}
	}
	return matches
}

func uiDiffRowBackground(active bool, selected bool, yanked bool, theme vui.Theme) vaxis.Color {
	if selected {
		return theme.Selection
	}
	if yanked {
		return uiDiffYankBackground(theme)
	}
	if active {
		return uiDiffCursorRowBackground(theme)
	}
	return 0
}

func uiDiffFullWidthRow(row diff.Row, rowIndex int, background vaxis.Color, theme vui.Theme, searchMatches []searchMatch, wrap bool) vui.Widget {
	if segments, ok := uiDiffStructuredSegments(row, theme); ok {
		if background != 0 {
			segments = uiDiffApplyBackground(segments, background)
		}
		segments = uiDiffSearchSegments(rowIndex, row, segments, searchMatches, theme)
		return vui.RichText{Spans: uiTextSpans(segments), SoftWrap: wrap}
	}
	style := uiStyleForDiffRow(row.Kind, theme)
	if background != 0 {
		style.Background = background
	}
	segments := []vaxis.Segment{{Text: uiDiffRowCode(row), Style: style}}
	segments = uiDiffSearchSegments(rowIndex, row, segments, searchMatches, theme)
	return vui.RichText{Spans: uiTextSpans(segments), SoftWrap: wrap}
}

func uiDiffStructuredSegments(row diff.Row, theme vui.Theme) ([]vaxis.Segment, bool) {
	switch row.Kind {
	case diff.RowHunk:
		if row.Prefix == "" && row.Code == "" {
			return nil, false
		}
		return []vaxis.Segment{
			{Text: row.Prefix, Style: uiStyleForDiffRow(diff.RowHunk, theme)},
			{Text: row.Code, Style: uiDimStyle(theme)},
		}, true
	case diff.RowCommitHeader:
		if row.Prefix == "" && row.Code == "" {
			return nil, false
		}
		return []vaxis.Segment{
			{Text: row.Prefix, Style: uiDimStyle(theme)},
			{Text: row.Code, Style: uiCommitHashStyle(theme)},
		}, true
	case diff.RowCommitMeta:
		if row.Prefix == "" && row.Code == "" {
			return nil, false
		}
		return []vaxis.Segment{
			{Text: row.Prefix, Style: uiCommitLabelStyle(theme)},
			{Text: row.Code, Style: uiCommitMetaStyle(theme)},
		}, true
	case diff.RowCommitTrailer:
		if row.Prefix == "" && row.Code == "" {
			return nil, false
		}
		return []vaxis.Segment{
			{Text: row.Prefix, Style: uiCommitTrailerLabelStyle(theme)},
			{Text: row.Code, Style: uiCommitTrailerValueStyle(theme)},
		}, true
	case diff.RowDiffStat:
		return uiDiffStatSegments(row, theme), true
	case diff.RowDiffStatSummary:
		return uiDiffStatSummarySegments(row, theme), true
	default:
		return nil, false
	}
}

func uiDiffApplyBackground(segments []vaxis.Segment, background vaxis.Color) []vaxis.Segment {
	styled := make([]vaxis.Segment, len(segments))
	for i, segment := range segments {
		segment.Style.Background = background
		styled[i] = segment
	}
	return styled
}

func uiDiffSearchSegments(rowIndex int, row diff.Row, segments []vaxis.Segment, matches []searchMatch, theme vui.Theme) []vaxis.Segment {
	if len(matches) == 0 || rowIndex < 0 {
		return segments
	}
	style := uiDiffSearchHighlightStyle(theme)
	for _, match := range matches {
		if match.Row != rowIndex {
			continue
		}
		start := match.Start
		end := match.End
		if row.Code != "" && uiDiffRowUsesGrid(row) {
			segments = styleSegmentsRangeFullWithTabWidth(segments, start, end, style, tabWidthForFile(row.FileName))
		} else {
			segments = styleSegmentsRangeFull(segments, start, end, style)
		}
	}
	return segments
}

func uiDiffSearchHighlightStyle(theme vui.Theme) vaxis.Style {
	return vaxis.Style{Foreground: theme.Background, Background: theme.Warning}
}

func uiDiffCursorRowBackground(theme vui.Theme) vaxis.Color {
	return theme.SurfaceHovered
}

func uiDiffYankBackground(theme vui.Theme) vaxis.Color {
	return theme.Warning
}

func uiDiffLineBackground(theme vui.Theme, scale vui.ColorScale) vaxis.Color {
	if theme.Mode == vui.LightTheme {
		return scale.Tone50
	}
	return scale.Tone950
}

func uiDiffCursorBackground(theme vui.Theme) vaxis.Color {
	return theme.Foreground
}

func uiDiffCursorForeground(theme vui.Theme) vaxis.Color {
	if theme.Mode == vui.LightTheme {
		return theme.Palette.Neutral.Tone50
	}
	return theme.Palette.Neutral.Tone950
}

func uiDiffChangedGutterForeground(theme vui.Theme, scale vui.ColorScale) vaxis.Color {
	if theme.Mode == vui.LightTheme {
		return scale.Tone700
	}
	return scale.Tone400
}

func uiDiffStatSegments(row diff.Row, theme vui.Theme) []vaxis.Segment {
	baseStyle := vaxis.Style{Foreground: theme.Foreground, Background: theme.Background}
	barStyle := uiDimStyle(theme)
	addStyle := baseStyle
	addStyle.Foreground = theme.Success
	deleteStyle := baseStyle
	deleteStyle.Foreground = theme.Danger

	segments := []vaxis.Segment{
		{Text: " " + row.Stat.Path, Style: baseStyle},
		{Text: " | ", Style: uiDimStyle(theme)},
	}
	if row.Stat.Changed > 0 {
		segments = append(segments, vaxis.Segment{Text: fmt.Sprintf("%d ", row.Stat.Changed), Style: barStyle})
	}
	for _, r := range row.Stat.Bar {
		style := barStyle
		switch r {
		case '+':
			style = addStyle
		case '-':
			style = deleteStyle
		}
		segments = append(segments, vaxis.Segment{Text: string(r), Style: style})
	}
	return segments
}

func uiDiffStatSummarySegments(row diff.Row, theme vui.Theme) []vaxis.Segment {
	baseStyle := uiDimStyle(theme)
	addStyle := baseStyle
	addStyle.Foreground = theme.Success
	deleteStyle := baseStyle
	deleteStyle.Foreground = theme.Danger
	return []vaxis.Segment{
		{Text: fmt.Sprintf(" %d %s changed", row.Stat.Files, pluralize(row.Stat.Files, "file")), Style: baseStyle},
		{Text: fmt.Sprintf(", +%d", row.Stat.Adds), Style: addStyle},
		{Text: fmt.Sprintf(" -%d", row.Stat.Deletes), Style: deleteStyle},
	}
}

func uiDimStyle(theme vui.Theme) vaxis.Style {
	return vaxis.Style{Foreground: theme.DisabledForeground, Background: theme.Background}
}

func uiCommitHashStyle(theme vui.Theme) vaxis.Style {
	return vaxis.Style{Foreground: theme.Warning, Background: theme.Background, Attribute: vaxis.AttrBold}
}

func uiCommitLabelStyle(theme vui.Theme) vaxis.Style {
	return vaxis.Style{Foreground: theme.MutedForeground, Background: theme.Background}
}

func uiCommitMetaStyle(theme vui.Theme) vaxis.Style {
	return vaxis.Style{Foreground: theme.Palette.Cyan.Tone500, Background: theme.Background}
}

func uiCommitTrailerLabelStyle(theme vui.Theme) vaxis.Style {
	return vaxis.Style{Foreground: theme.Primary, Background: theme.Background, Attribute: vaxis.AttrBold}
}

func uiCommitTrailerValueStyle(theme vui.Theme) vaxis.Style {
	return vaxis.Style{Foreground: theme.DisabledForeground, Background: theme.Background}
}

func (s *uiDiffViewState) buildRow(rows []diff.Row, row diff.Row, rowIndex int, active bool, selected bool, yanked bool, cursorCol int, theme vui.Theme, highlightedRows map[int][]vaxis.Segment, searchMatches []searchMatch, wrap bool) vui.Widget {
	style := uiStyleForDiffRow(row.Kind, theme)
	rowBackground := vaxis.Color(0)
	if active {
		rowBackground = uiDiffCursorRowBackground(theme)
	}
	fillBackground := style.Background
	if rowBackground != 0 {
		fillBackground = rowBackground
	}
	textBackground := rowBackground
	if selected {
		textBackground = theme.Selection
	} else if yanked {
		textBackground = uiDiffYankBackground(theme)
	}
	if textBackground != 0 {
		style.Background = textBackground
	}
	code := uiDiffRowCode(row)
	codeSegments := highlightedRows[rowIndex]
	if len(codeSegments) == 0 {
		codeSegments = []vaxis.Segment{{Text: code, Style: style}}
	}
	codeSegments = uiDiffToneCodeSegments(row.Kind, codeSegments, theme)
	if textBackground != 0 {
		codeSegments = uiDiffApplyBackground(codeSegments, textBackground)
	}
	if start, end, ok := s.charSelectionRange(rowIndex, row); ok {
		codeSegments = uiDiffApplySegmentBackgroundRange(codeSegments, start, end, theme.Selection, tabWidthForFile(row.FileName))
	}
	if start, end, ok := s.charYankRange(rowIndex, row); ok {
		codeSegments = uiDiffApplySegmentBackgroundRange(codeSegments, start, end, uiDiffYankBackground(theme), tabWidthForFile(row.FileName))
	}
	codeSegments = uiDiffSearchSegments(rowIndex, row, codeSegments, searchMatches, theme)
	oldLine, newLine, marker := splitDiffGutter(row)
	oldWidth, newWidth := uiDiffGutterWidths(rows)
	gutterStyle := uiGutterStyle(row.Kind, rowBackground, theme)
	lineNumberStyle := uiLineNumberGutterStyle(row.Kind, rowBackground, theme)
	return vui.Row(
		uiDiffFixedCell(oldWidth, lineNumberStyle, vui.Text{Value: oldLine, Style: lineNumberStyle, Align: vui.TextAlignRight}),
		uiDiffFixedCell(1, gutterStyle, vui.Text{Value: " ", Style: gutterStyle}),
		uiDiffFixedCell(newWidth, lineNumberStyle, vui.Text{Value: newLine, Style: lineNumberStyle, Align: vui.TextAlignRight}),
		uiDiffFixedCell(1, gutterStyle, vui.Text{Value: " ", Style: gutterStyle}),
		uiDiffFixedCell(1, gutterStyle, vui.Text{Value: marker, Style: gutterStyle}),
		uiDiffFixedCell(1, gutterStyle, vui.Text{Value: " ", Style: gutterStyle}),
		vui.Expanded(uiDiffCodeWidget(row, code, codeSegments, active, s.selectionActive && !s.selectionLinewise, cursorCol, theme, fillBackground, wrap)),
	)
}

func uiDiffFixedCell(width int, style vaxis.Style, child vui.Widget) vui.Widget {
	return vui.DecoratedBox(
		vui.Decoration{Style: style},
		vui.SizedBox{Width: width, Height: 1, Child: child},
	)
}

func uiDiffCodeWidget(row diff.Row, code string, segments []vaxis.Segment, active bool, cursorTabEnd bool, cursorCol int, theme vui.Theme, background vaxis.Color, wrap bool) vui.Widget {
	if !active || code == "" {
		return vui.DecoratedBox(
			vui.Decoration{Style: vaxis.Style{Background: background}},
			vui.RichText{Spans: uiTextSpans(segments), SoftWrap: wrap},
		)
	}
	cursorStyle := vaxis.Style{Foreground: uiDiffCursorForeground(theme), Background: uiDiffCursorBackground(theme)}
	segments = uiDiffCursorSegments(segments, cursorCol, cursorStyle, tabWidthForFile(row.FileName), cursorTabEnd)
	return vui.DecoratedBox(
		vui.Decoration{Style: vaxis.Style{Background: background}},
		vui.RichText{Spans: uiTextSpans(segments), SoftWrap: wrap},
	)
}

func uiDiffCursorSegments(segments []vaxis.Segment, cursorCol int, cursorStyle vaxis.Style, tabWidth int, cursorTabEnd bool) []vaxis.Segment {
	var styled []vaxis.Segment
	col := 0
	for _, segment := range segments {
		it := uucode.NewGraphemeIterator(segment.Text)
		for g, ok := it.Next(); ok; g, ok = it.Next() {
			grapheme := segment.Text[g.Start:g.End]
			char := characterForGraphemeWithTabWidth(grapheme, tabWidth)
			next := col + char.Width
			if cursorCol >= col && cursorCol < next {
				if grapheme == "\t" && char.Width > 1 {
					if cursorTabEnd {
						styled = appendSegment(styled, vaxis.Segment{Text: strings.Repeat(" ", char.Width-1), Style: segment.Style})
						styled = appendSegment(styled, vaxis.Segment{Text: " ", Style: cursorStyle})
					} else {
						styled = appendSegment(styled, vaxis.Segment{Text: " ", Style: cursorStyle})
						styled = appendSegment(styled, vaxis.Segment{Text: strings.Repeat(" ", char.Width-1), Style: segment.Style})
					}
				} else {
					styled = appendSegment(styled, vaxis.Segment{Text: grapheme, Style: cursorStyle})
				}
			} else {
				styled = appendSegment(styled, vaxis.Segment{Text: grapheme, Style: segment.Style})
			}
			col = next
		}
	}
	return styled
}

func uiDiffApplySegmentBackgroundRange(segments []vaxis.Segment, start int, end int, background vaxis.Color, tabWidth int) []vaxis.Segment {
	if end <= start {
		return segments
	}
	var styled []vaxis.Segment
	col := 0
	for _, segment := range segments {
		it := uucode.NewGraphemeIterator(segment.Text)
		for g, ok := it.Next(); ok; g, ok = it.Next() {
			grapheme := segment.Text[g.Start:g.End]
			char := characterForGraphemeWithTabWidth(grapheme, tabWidth)
			next := col + char.Width
			style := segment.Style
			if next > start && col < end {
				style.Background = background
			}
			styled = appendSegment(styled, vaxis.Segment{Text: grapheme, Style: style})
			col = next
		}
	}
	return styled
}

func uiTextSpans(segments []vaxis.Segment) []vui.TextSpan {
	spans := make([]vui.TextSpan, 0, len(segments))
	for _, segment := range segments {
		spans = append(spans, vui.TextSpan{Text: segment.Text, Style: segment.Style})
	}
	return spans
}

func uiDiffToneCodeSegments(kind diff.RowKind, segments []vaxis.Segment, theme vui.Theme) []vaxis.Segment {
	switch kind {
	case diff.RowContext:
		return uiDiffDimSegments(segments, theme, 1)
	case diff.RowDelete:
		return uiDiffDimSegments(segments, theme, 1)
	default:
		return segments
	}
}

func uiDiffDimSegments(segments []vaxis.Segment, theme vui.Theme, steps int) []vaxis.Segment {
	styled := make([]vaxis.Segment, len(segments))
	for i, segment := range segments {
		if segment.Style.Foreground != vaxis.ColorDefault {
			segment.Style.Foreground = uiDiffDimSyntaxColor(segment.Style.Foreground, theme, steps)
		}
		styled[i] = segment
	}
	return styled
}

func uiDiffDimSyntaxColor(color vaxis.Color, theme vui.Theme, steps int) vaxis.Color {
	if color == theme.Foreground {
		return theme.MutedForeground
	}
	if dimmed, ok := uiDiffDimScaleColor(color, theme.Palette.Blue, theme.Mode, steps); ok {
		return dimmed
	}
	if dimmed, ok := uiDiffDimScaleColor(color, theme.Palette.Cyan, theme.Mode, steps); ok {
		return dimmed
	}
	if dimmed, ok := uiDiffDimScaleColor(color, theme.Palette.Green, theme.Mode, steps); ok {
		return dimmed
	}
	if dimmed, ok := uiDiffDimScaleColor(color, theme.Palette.Magenta, theme.Mode, steps); ok {
		return dimmed
	}
	if dimmed, ok := uiDiffDimScaleColor(color, theme.Palette.Yellow, theme.Mode, steps); ok {
		return dimmed
	}
	if dimmed, ok := uiDiffDimScaleColor(color, theme.Palette.Red, theme.Mode, steps); ok {
		return dimmed
	}
	return color
}

func uiDiffDimScaleColor(color vaxis.Color, scale vui.ColorScale, mode vui.ThemeMode, steps int) (vaxis.Color, bool) {
	tone := []vaxis.Color{
		scale.Tone50,
		scale.Tone100,
		scale.Tone200,
		scale.Tone300,
		scale.Tone400,
		scale.Tone500,
		scale.Tone600,
		scale.Tone700,
		scale.Tone800,
		scale.Tone900,
		scale.Tone950,
	}
	for index, candidate := range tone {
		if color != candidate {
			continue
		}
		if mode == vui.LightTheme {
			return tone[clampUIDiffInt(index-steps, 0, len(tone)-1)], true
		}
		return tone[clampUIDiffInt(index+steps, 0, len(tone)-1)], true
	}
	return color, false
}

func uiDiffGutterWidths(rows []diff.Row) (int, int) {
	oldWidth := 0
	newWidth := 0
	for _, row := range rows {
		if !uiDiffRowUsesGrid(row) {
			continue
		}
		oldLine, newLine, _ := splitDiffGutter(row)
		oldWidth = maxInt(oldWidth, textCellWidth(oldLine))
		newWidth = maxInt(newWidth, textCellWidth(newLine))
	}
	return oldWidth, newWidth
}

func uiDiffRowUsesGrid(row diff.Row) bool {
	return row.Gutter != "" || row.Marker != ""
}

func (s *uiDiffViewState) setCursorRow(rows []diff.Row, row int) {
	if len(rows) == 0 {
		s.cursor = selectionPoint{}
		return
	}
	if s.selectionActive {
		s.cursor.Row = uiDiffSelectableTargetRow(rows, row, signUIDiffInt(row-s.cursor.Row))
	} else {
		s.cursor.Row = uiDiffCursorTargetRow(rows, row, signUIDiffInt(row-s.cursor.Row))
	}
	s.cursor.Col = s.clampCursorCol(rows, s.cursor.Row, s.cursorCol)
	s.cursorCol = s.cursor.Col
	s.revealCursorRow()
	s.SetState(func() {})
}

func (s *uiDiffViewState) moveCursorRows(rows []diff.Row, delta int) {
	if delta == 0 {
		return
	}
	s.setCursorRow(rows, s.cursor.Row+delta)
}

func uiDiffCursorTargetRow(rows []diff.Row, row int, direction int) int {
	if len(rows) == 0 {
		return 0
	}
	row = clampUIDiffInt(row, 0, len(rows)-1)
	if uiDiffCursorableRow(rows[row]) {
		return row
	}
	if direction == 0 {
		direction = 1
	}
	for next := row + direction; next >= 0 && next < len(rows); next += direction {
		if uiDiffCursorableRow(rows[next]) {
			return next
		}
	}
	for next := row - direction; next >= 0 && next < len(rows); next -= direction {
		if uiDiffCursorableRow(rows[next]) {
			return next
		}
	}
	return row
}

func uiDiffSelectableTargetRow(rows []diff.Row, row int, direction int) int {
	if len(rows) == 0 {
		return 0
	}
	row = clampUIDiffInt(row, 0, len(rows)-1)
	if selectableDiffRow(rows[row].Kind) {
		return row
	}
	if direction >= 0 {
		for next := row + 1; next < len(rows); next++ {
			if selectableDiffRow(rows[next].Kind) {
				return next
			}
		}
	}
	if direction <= 0 {
		for next := row - 1; next >= 0; next-- {
			if selectableDiffRow(rows[next].Kind) {
				return next
			}
		}
	}
	return row
}

func uiDiffCursorableRow(row diff.Row) bool {
	return row.Kind != diff.RowBlank
}

func signUIDiffInt(v int) int {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}

func (s *uiDiffViewState) jumpCommit(rows []diff.Row, direction int) {
	if len(rows) == 0 {
		return
	}
	if direction < 0 {
		for row := s.cursor.Row - 1; row >= 0; row-- {
			if rows[row].Kind == diff.RowCommitHeader {
				s.setCursorRowAtStart(rows, row)
				return
			}
		}
		return
	}
	for row := s.cursor.Row + 1; row < len(rows); row++ {
		if rows[row].Kind == diff.RowCommitHeader {
			s.setCursorRowAtStart(rows, row)
			return
		}
	}
}

func (s *uiDiffViewState) setCursorRowAtStart(rows []diff.Row, row int) {
	s.setCursorRow(rows, row)
	s.list.ScrollToIndex(s.cursor.Row, vui.ScrollAlignStart)
}

func (s *uiDiffViewState) jumpChange(rows []diff.Row, direction int) {
	targets := uiDiffChangeTargetRows(rows)
	s.jumpTargetRow(rows, targets, direction)
}

func (s *uiDiffViewState) jumpNote(rows []diff.Row, drafts []review.CommentDraft, direction int) {
	targets := noteTargetRows(rows, drafts)
	s.jumpTargetRow(rows, targets, direction)
}

func (s *uiDiffViewState) jumpTargetRow(rows []diff.Row, targets []int, direction int) {
	if len(targets) == 0 {
		return
	}
	if direction < 0 {
		for index := len(targets) - 1; index >= 0; index-- {
			if targets[index] < s.cursor.Row {
				s.setCursorRow(rows, targets[index])
				return
			}
		}
		return
	}
	for _, target := range targets {
		if target > s.cursor.Row {
			s.setCursorRow(rows, target)
			return
		}
	}
}

func (s *uiDiffViewState) updateSearchMatches(rows []diff.Row) {
	if s.searchQuery == "" {
		s.searchMatches = nil
		s.searchIndex = -1
		return
	}
	query := strings.ToLower(s.searchQuery)
	matches := make([]searchMatch, 0)
	for rowIndex, row := range rows {
		searchText, offset := uiDiffSearchableText(row)
		text := strings.ToLower(searchText)
		for start := 0; ; {
			index := strings.Index(text[start:], query)
			if index < 0 {
				break
			}
			matchStart := start + index
			matchEnd := matchStart + len(query)
			matchStartCol := textCellWidth(searchText[:matchStart])
			matchEndCol := textCellWidth(searchText[:matchEnd])
			if uiDiffRowUsesGrid(row) && row.Code != "" {
				matchStartCol = textCellWidthWithTabWidth(searchText[:matchStart], tabWidthForFile(row.FileName))
				matchEndCol = textCellWidthWithTabWidth(searchText[:matchEnd], tabWidthForFile(row.FileName))
			}
			matches = append(matches, searchMatch{Row: rowIndex, Start: offset + matchStartCol, End: offset + matchEndCol})
			start = matchEnd
		}
	}
	s.searchMatches = matches
	s.searchIndex = -1
}

func uiDiffSearchableText(row diff.Row) (string, int) {
	if uiDiffRowUsesGrid(row) && row.Code != "" {
		return row.Code, 0
	}
	if row.Prefix != "" || row.Code != "" {
		return row.Prefix + row.Code, 0
	}
	return uiDiffRowCode(row), 0
}

func (s *uiDiffViewState) moveSearchMatch(rows []diff.Row, delta int) {
	if len(s.searchMatches) == 0 {
		return
	}
	if s.searchIndex < 0 || s.searchIndex >= len(s.searchMatches) {
		s.searchIndex = s.nextSearchIndexFromPoint(s.cursor, delta)
	} else {
		s.searchIndex = (s.searchIndex + delta + len(s.searchMatches)) % len(s.searchMatches)
	}
	s.applySearchMatch(rows)
	s.SetState(func() {})
}

func (s *uiDiffViewState) nextSearchIndexFromPoint(origin selectionPoint, direction int) int {
	if len(s.searchMatches) == 0 {
		return -1
	}
	if direction < 0 {
		for index := len(s.searchMatches) - 1; index >= 0; index-- {
			point := selectionPoint{Row: s.searchMatches[index].Row, Col: s.searchMatches[index].Start}
			if selectionPointLess(point, origin) {
				return index
			}
		}
		return len(s.searchMatches) - 1
	}
	for index, match := range s.searchMatches {
		point := selectionPoint{Row: match.Row, Col: match.Start}
		if selectionPointLess(origin, point) || origin == point {
			return index
		}
	}
	return 0
}

func (s *uiDiffViewState) applySearchMatch(rows []diff.Row) {
	if s.searchIndex < 0 || s.searchIndex >= len(s.searchMatches) {
		return
	}
	match := s.searchMatches[s.searchIndex]
	if s.searchMode {
		s.setCursorPointWithoutReveal(rows, selectionPoint{Row: match.Row, Col: match.Start})
		return
	}
	s.setCursorPoint(rows, selectionPoint{Row: match.Row, Col: match.Start})
}

func (s *uiDiffViewState) setCursorPoint(rows []diff.Row, point selectionPoint) {
	if len(rows) == 0 {
		return
	}
	s.setCursorPointWithoutReveal(rows, point)
	s.revealCursorRow()
}

func (s *uiDiffViewState) setCursorPointWithoutReveal(rows []diff.Row, point selectionPoint) {
	if len(rows) == 0 {
		return
	}
	s.cursor.Row = clampUIDiffInt(point.Row, 0, len(rows)-1)
	s.cursor.Col = s.clampCursorCol(rows, s.cursor.Row, point.Col)
	s.cursorCol = s.cursor.Col
}

func uiDiffChangeTargetRows(rows []diff.Row) []int {
	targets := make([]int, 0)
	inChange := false
	for index, row := range rows {
		if row.Kind == diff.RowAdd || row.Kind == diff.RowDelete {
			if !inChange {
				targets = append(targets, index)
			}
			inChange = true
			continue
		}
		inChange = false
	}
	return targets
}

func (s *uiDiffViewState) halfPageRows() int {
	first, last, ok := s.list.VisibleRange()
	if !ok || last <= first+1 {
		return 1
	}
	return maxInt(1, (last-first)/2)
}

func (s *uiDiffViewState) revealCursorRow() {
	first, last, ok := s.list.VisibleRange()
	if !ok {
		return
	}
	if s.cursor.Row < first {
		s.list.ScrollToIndex(s.cursor.Row, vui.ScrollAlignStart)
		return
	}
	if s.cursor.Row >= last {
		s.list.ScrollToIndex(s.cursor.Row, vui.ScrollAlignEnd)
	}
}

func (s *uiDiffViewState) moveCursorCols(rows []diff.Row, delta int) {
	if s.cursor.Row < 0 || s.cursor.Row >= len(rows) {
		return
	}
	s.cursor.Col = uiDiffMoveCursorCol(rows[s.cursor.Row], s.cursor.Col, delta)
	s.cursorCol = s.cursor.Col
	s.revealCursorRow()
	s.SetState(func() {})
}

func uiDiffMoveCursorCol(row diff.Row, col int, delta int) int {
	start, end, ok := uiDiffCodeRangeForRow(row)
	if !ok || delta == 0 {
		return col
	}
	col = uiDiffClampCursorCol(row, col, start, end)
	if delta > 0 {
		for ; delta > 0; delta-- {
			col = uiDiffNextCursorCol(row, col, end)
		}
		return col
	}
	for ; delta < 0; delta++ {
		col = uiDiffPrevCursorCol(row, col, start)
	}
	return col
}

func uiDiffNextCursorCol(row diff.Row, col int, end int) int {
	last := minInt(col, end)
	for _, pos := range uiDiffCursorPositions(row) {
		if pos > col {
			return minInt(pos, end)
		}
		last = minInt(pos, end)
	}
	return last
}

func uiDiffPrevCursorCol(row diff.Row, col int, start int) int {
	prev := start
	for _, pos := range uiDiffCursorPositions(row) {
		if pos >= col {
			return prev
		}
		prev = pos
	}
	return prev
}

func uiDiffClampCursorCol(row diff.Row, col int, start int, end int) int {
	positions := uiDiffCursorPositions(row)
	if len(positions) == 0 {
		return start
	}
	if col <= start {
		return start
	}
	last := start
	for _, pos := range positions {
		if pos > col || pos > end {
			return last
		}
		last = pos
	}
	return last
}

func uiDiffCursorPositions(row diff.Row) []int {
	code := uiDiffRowCode(row)
	if code == "" {
		return []int{0}
	}
	positions := make([]int, 0)
	col := 0
	tabWidth := tabWidthForFile(row.FileName)
	it := uucode.NewGraphemeIterator(code)
	for g, ok := it.Next(); ok; g, ok = it.Next() {
		positions = append(positions, col)
		col += graphemeCellWidthWithTabWidth(code[g.Start:g.End], tabWidth)
	}
	return positions
}

func (s *uiDiffViewState) moveCursorLineStart(rows []diff.Row) {
	if start, _, ok := uiDiffCodeRange(rows, s.cursor.Row); ok {
		s.cursor.Col = start
		s.cursorCol = s.cursor.Col
		s.SetState(func() {})
	}
}

func (s *uiDiffViewState) moveCursorLineEnd(rows []diff.Row) {
	if start, end, ok := uiDiffCodeRange(rows, s.cursor.Row); ok {
		s.cursor.Col = maxInt(start, end-1)
		s.cursorCol = s.cursor.Col
		s.SetState(func() {})
	}
}

func (s *uiDiffViewState) clampCursor(rows []diff.Row) {
	if len(rows) == 0 {
		s.cursor = selectionPoint{}
		s.cursorCol = 0
		return
	}
	s.cursor.Row = uiDiffCursorTargetRow(rows, s.cursor.Row, 1)
	s.cursor.Col = s.clampCursorCol(rows, s.cursor.Row, s.cursor.Col)
	s.cursorCol = s.cursor.Col
}

func (s *uiDiffViewState) clampCursorCol(rows []diff.Row, row int, col int) int {
	start, end, ok := uiDiffCodeRange(rows, row)
	if !ok {
		return 0
	}
	return uiDiffClampCursorCol(rows[row], col, start, end)
}

func clampUIDiffInt(v int, lo int, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func orderedUIDiffInts(a int, b int) (int, int) {
	if a <= b {
		return a, b
	}
	return b, a
}

func uiDiffCodeRange(rows []diff.Row, rowIndex int) (int, int, bool) {
	if rowIndex < 0 || rowIndex >= len(rows) {
		return 0, 0, false
	}
	row := rows[rowIndex]
	return uiDiffCodeRangeForRow(row)
}

func uiDiffCodeRangeForRow(row diff.Row) (int, int, bool) {
	if !selectableDiffRow(row.Kind) {
		return 0, textCellWidth(uiDiffRowCode(row)), true
	}
	return 0, codeCellWidth(row), true
}

func uiDiffRowCode(row diff.Row) string {
	if row.Code != "" {
		return row.Code
	}
	return row.Text
}

func splitDiffGutter(row diff.Row) (string, string, string) {
	if row.Gutter == "" && row.Marker == "" {
		return "", "", ""
	}
	fields := stringsFields(row.Gutter)
	marker := row.Marker
	if len(fields) > 0 && isDiffMarker(fields[len(fields)-1]) {
		marker = fields[len(fields)-1]
		fields = fields[:len(fields)-1]
	}
	switch len(fields) {
	case 0:
		return "", "", marker
	case 1:
		if row.Kind == diff.RowDelete {
			return fields[0], "", marker
		}
		return "", fields[0], marker
	default:
		return fields[0], fields[1], marker
	}
}

func isDiffMarker(s string) bool {
	return s == "+" || s == "-"
}

func stringsFields(s string) []string {
	fields := make([]string, 0, 2)
	start := -1
	for i, r := range s {
		if r == ' ' || r == '\t' {
			if start >= 0 {
				fields = append(fields, s[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		fields = append(fields, s[start:])
	}
	return fields
}

func uiGutterStyle(kind diff.RowKind, background vaxis.Color, theme vui.Theme) vaxis.Style {
	style := vaxis.Style{Foreground: theme.MutedForeground, Background: theme.Background}
	if background != 0 {
		style.Background = background
	}
	switch kind {
	case diff.RowAdd:
		style.Foreground = uiDiffChangedGutterForeground(theme, theme.Palette.Green)
	case diff.RowDelete:
		style.Foreground = uiDiffChangedGutterForeground(theme, theme.Palette.Red)
	}
	return style
}

func uiLineNumberGutterStyle(kind diff.RowKind, background vaxis.Color, theme vui.Theme) vaxis.Style {
	style := uiGutterStyle(kind, background, theme)
	if kind == diff.RowAdd {
		style.Foreground = uiDiffAddedLineNumberForeground(theme)
	}
	return style
}

func uiDiffAddedLineNumberForeground(theme vui.Theme) vaxis.Color {
	if theme.Mode == vui.LightTheme {
		return theme.Palette.Green.Tone600
	}
	return theme.Palette.Green.Tone300
}

func uiStyleForDiffRow(kind diff.RowKind, theme vui.Theme) vaxis.Style {
	switch kind {
	case diff.RowFile:
		return vaxis.Style{Foreground: theme.Primary, Background: theme.Background, Attribute: vaxis.AttrBold}
	case diff.RowHunk:
		return vaxis.Style{Foreground: theme.Accent, Background: theme.Background}
	case diff.RowAdd:
		return vaxis.Style{Foreground: theme.Success, Background: theme.Surface}
	case diff.RowDelete:
		return vaxis.Style{Foreground: uiDiffDimChangedForeground(theme, theme.Palette.Red), Background: uiDiffLineBackground(theme, theme.Palette.Red)}
	case diff.RowMeta, diff.RowPreamble, diff.RowNoNewline:
		return vaxis.Style{Foreground: theme.MutedForeground, Background: theme.Background}
	case diff.RowCommitHeader:
		return vaxis.Style{Foreground: theme.Warning, Background: theme.Background, Attribute: vaxis.AttrBold}
	case diff.RowCommitMeta:
		return vaxis.Style{Foreground: theme.Palette.Cyan.Tone500, Background: theme.Background}
	default:
		return vaxis.Style{Foreground: theme.MutedForeground, Background: theme.Background}
	}
}

func uiDiffDimChangedForeground(theme vui.Theme, scale vui.ColorScale) vaxis.Color {
	if theme.Mode == vui.LightTheme {
		return scale.Tone800
	}
	return scale.Tone500
}
