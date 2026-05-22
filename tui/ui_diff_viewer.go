package tui

import (
	"fmt"

	"git.sr.ht/~rockorager/vaxis"
	vui "git.sr.ht/~rockorager/vaxis/ui"
	"github.com/rockorager/go-uucode"

	"github.com/rockorager/comview/diff"
)

type uiDiffView struct {
	Rows []diff.Row
	Wrap bool
}

func uiDiffRoot(rows []diff.Row, wrap bool) vui.Widget {
	return uiDiffView{Rows: rows, Wrap: wrap}
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
	list             vui.SliverListController
	cursor           selectionPoint
	cursorCol        int
	pendingG         bool
	pendingBracket   rune
	syntaxTheme      vui.Theme
	highlighter      *SyntaxHighlighter
	highlightedTheme vui.Theme
	highlightedRows  map[int][]vaxis.Segment
}

func (s *uiDiffViewState) Build(ctx vui.BuildContext) vui.Widget {
	w := s.Widget().(uiDiffView)
	s.clampCursor(w.Rows)
	theme := vui.MustDepend[vui.Theme](ctx)
	highlightedRows := s.highlightedCodeRows(w.Rows, theme)
	return vui.CustomScrollView{
		Slivers: []vui.Widget{
			vui.SliverListBuilder{
				Controller:          &s.list,
				Count:               len(w.Rows),
				ItemExtent:          uiDiffItemExtent(w.Wrap),
				EstimatedItemExtent: 1,
				Overscan:            8,
				Builder: func(ctx vui.BuildContext, row int) vui.Widget {
					return s.buildItem(w.Rows, row, theme, highlightedRows, w.Wrap)
				},
			},
		},
	}
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
	key, ok := ev.(vaxis.Key)
	if !ok || key.EventType == vaxis.EventRelease || pureModifierKey(key) {
		return vui.EventIgnored
	}
	rows := s.Widget().(uiDiffView).Rows
	if len(rows) == 0 {
		return vui.EventIgnored
	}
	switch {
	case key.MatchString("Alt+p"):
		ctx.ToggleProfileOverlay()
		return vui.EventHandled
	case key.Matches('q'), key.MatchString("Ctrl+c"):
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
	case key.Matches('j'), key.Matches(vaxis.KeyDown), key.MatchString("Down"):
		s.clearPendingKeys()
		s.moveCursorRows(rows, 1)
		return vui.EventHandled
	case key.Matches('k'), key.Matches(vaxis.KeyUp), key.MatchString("Up"):
		s.clearPendingKeys()
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

func (s *uiDiffViewState) clearPendingKeys() {
	s.pendingG = false
	s.pendingBracket = 0
}

func (s *uiDiffViewState) buildItem(rows []diff.Row, rowIndex int, theme vui.Theme, highlightedRows map[int][]vaxis.Segment, wrap bool) vui.Widget {
	row := rows[rowIndex]
	active := rowIndex == s.cursor.Row
	if !uiDiffRowUsesGrid(row) {
		return uiDiffFullWidthRow(row, active, theme, wrap)
	}
	return s.buildRow(rows, row, rowIndex, active, s.cursor.Col, theme, highlightedRows, wrap)
}

func uiDiffFullWidthRow(row diff.Row, active bool, theme vui.Theme, wrap bool) vui.Widget {
	if segments, ok := uiDiffStructuredSegments(row, theme); ok {
		if active {
			segments = uiDiffApplyBackground(segments, theme.Selection)
		}
		return vui.RichText{Spans: uiTextSpans(segments), SoftWrap: wrap}
	}
	style := uiStyleForDiffRow(row.Kind, theme)
	if active {
		style.Background = theme.Selection
	}
	return vui.Text{Value: uiDiffRowCode(row), Style: style, SoftWrap: wrap}
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

func (s *uiDiffViewState) buildRow(rows []diff.Row, row diff.Row, rowIndex int, active bool, cursorCol int, theme vui.Theme, highlightedRows map[int][]vaxis.Segment, wrap bool) vui.Widget {
	style := uiStyleForDiffRow(row.Kind, theme)
	if active {
		style.Background = theme.Selection
	}
	code := uiDiffRowCode(row)
	codeSegments := highlightedRows[rowIndex]
	if len(codeSegments) == 0 {
		codeSegments = []vaxis.Segment{{Text: code, Style: style}}
	}
	codeSegments = uiDiffToneCodeSegments(row.Kind, codeSegments, theme)
	if active {
		codeSegments = uiDiffApplyBackground(codeSegments, theme.Selection)
	}
	oldLine, newLine, marker := splitDiffGutter(row)
	oldWidth, newWidth := uiDiffGutterWidths(rows)
	gutterStyle := uiGutterStyle(row.Kind, active, theme)
	return vui.Row(
		uiDiffFixedCell(oldWidth, gutterStyle, vui.Text{Value: oldLine, Style: gutterStyle, Align: vui.TextAlignRight}),
		uiDiffFixedCell(1, gutterStyle, vui.Text{Value: " ", Style: gutterStyle}),
		uiDiffFixedCell(newWidth, gutterStyle, vui.Text{Value: newLine, Style: gutterStyle, Align: vui.TextAlignRight}),
		uiDiffFixedCell(1, gutterStyle, vui.Text{Value: " ", Style: gutterStyle}),
		uiDiffFixedCell(1, gutterStyle, vui.Text{Value: marker, Style: gutterStyle}),
		uiDiffFixedCell(1, gutterStyle, vui.Text{Value: " ", Style: gutterStyle}),
		vui.Expanded(uiDiffCodeWidget(row, code, codeSegments, active, cursorCol, theme, style.Background, wrap)),
	)
}

func uiDiffFixedCell(width int, style vaxis.Style, child vui.Widget) vui.Widget {
	return vui.DecoratedBox(
		vui.Decoration{Style: style},
		vui.SizedBox{Width: width, Height: 1, Child: child},
	)
}

func uiDiffCodeWidget(row diff.Row, code string, segments []vaxis.Segment, active bool, cursorCol int, theme vui.Theme, background vaxis.Color, wrap bool) vui.Widget {
	if !active || code == "" {
		return vui.DecoratedBox(
			vui.Decoration{Style: vaxis.Style{Background: background}},
			vui.RichText{Spans: uiTextSpans(segments), SoftWrap: wrap},
		)
	}
	cursorStyle := vaxis.Style{Foreground: uiDiffCursorForeground(theme), Background: uiDiffCursorBackground(theme)}
	segments = styleSegmentsRangeFullWithTabWidth(
		segments,
		cursorCol,
		cursorCol+1,
		cursorStyle,
		tabWidthForFile(row.FileName),
	)
	return vui.DecoratedBox(
		vui.Decoration{Style: vaxis.Style{Background: background}},
		vui.RichText{Spans: uiTextSpans(segments), SoftWrap: wrap},
	)
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
	s.cursor.Row = clampUIDiffInt(row, 0, len(rows)-1)
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

func (s *uiDiffViewState) jumpChange(rows []diff.Row, direction int) {
	targets := uiDiffChangeTargetRows(rows)
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
	s.cursor.Row = clampUIDiffInt(s.cursor.Row, 0, len(rows)-1)
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

func uiGutterStyle(kind diff.RowKind, active bool, theme vui.Theme) vaxis.Style {
	style := vaxis.Style{Foreground: theme.MutedForeground, Background: theme.Background}
	if active {
		style.Background = theme.Selection
	}
	switch kind {
	case diff.RowAdd:
		style.Foreground = uiDiffChangedGutterForeground(theme, theme.Palette.Green)
	case diff.RowDelete:
		style.Foreground = uiDiffChangedGutterForeground(theme, theme.Palette.Red)
	}
	return style
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
