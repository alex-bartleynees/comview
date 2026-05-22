package tui

import (
	"git.sr.ht/~rockorager/vaxis"
	vui "git.sr.ht/~rockorager/vaxis/ui"
	"github.com/rockorager/go-uucode"

	"github.com/rockorager/comview/diff"
)

type uiDiffView struct {
	Rows   []diff.Row
	Scheme ColorScheme
	Wrap   bool
}

func (w uiDiffView) CreateState() vui.State {
	return &uiDiffViewState{}
}

type uiDiffViewState struct {
	vui.StateBase
	table     vui.SliverTableController
	cursor    selectionPoint
	cursorCol int
	pendingG  bool
}

func (s *uiDiffViewState) Build(ctx vui.BuildContext) vui.Widget {
	w := s.Widget().(uiDiffView)
	s.clampCursor(w.Rows)
	scheme := w.Scheme
	if scheme.Foreground == vaxis.ColorDefault {
		scheme = DefaultColorScheme()
	}
	return vui.CustomScrollView{
		Slivers: []vui.Widget{
			vui.SliverTableBuilder{
				Controller: &s.table,
				Columns: []vui.TableColumn{
					vui.IntrinsicColumn(),
					vui.IntrinsicColumn(),
					vui.FixedColumn(1),
					vui.FlexColumn(1),
				},
				RowCount:           len(w.Rows),
				EstimatedRowExtent: 1,
				Overscan:           8,
				Builder: func(ctx vui.BuildContext, row int) vui.TableRow {
					return s.buildRow(w.Rows[row], row == s.cursor.Row, s.cursor.Col, scheme, w.Wrap)
				},
			},
		},
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
	case key.Matches('g') && s.pendingG:
		s.pendingG = false
		s.setCursorRow(rows, 0)
		return vui.EventHandled
	case key.Matches('g'):
		s.pendingG = true
		return vui.EventHandled
	case key.Matches(vaxis.KeyHome):
		s.pendingG = false
		s.setCursorRow(rows, 0)
		return vui.EventHandled
	case key.Matches('G'), key.Matches(vaxis.KeyEnd):
		s.pendingG = false
		s.setCursorRow(rows, len(rows)-1)
		return vui.EventHandled
	case key.MatchString("Ctrl+d"), key.Matches(vaxis.KeyPgDown):
		s.pendingG = false
		s.moveCursorRows(rows, s.halfPageRows())
		return vui.EventHandled
	case key.MatchString("Ctrl+u"), key.Matches(vaxis.KeyPgUp):
		s.pendingG = false
		s.moveCursorRows(rows, -s.halfPageRows())
		return vui.EventHandled
	case key.Matches('j'), key.Matches(vaxis.KeyDown), key.MatchString("Down"):
		s.pendingG = false
		s.moveCursorRows(rows, 1)
		return vui.EventHandled
	case key.Matches('k'), key.Matches(vaxis.KeyUp), key.MatchString("Up"):
		s.pendingG = false
		s.moveCursorRows(rows, -1)
		return vui.EventHandled
	case key.Matches('h'), key.Matches(vaxis.KeyLeft), key.MatchString("Left"):
		s.pendingG = false
		s.moveCursorCols(rows, -1)
		return vui.EventHandled
	case key.Matches('l'), key.Matches(vaxis.KeyRight), key.MatchString("Right"):
		s.pendingG = false
		s.moveCursorCols(rows, 1)
		return vui.EventHandled
	case key.Matches('0'):
		s.pendingG = false
		s.moveCursorLineStart(rows)
		return vui.EventHandled
	case key.Matches('$'):
		s.pendingG = false
		s.moveCursorLineEnd(rows)
		return vui.EventHandled
	default:
		s.pendingG = false
		return vui.EventIgnored
	}
}

func (s *uiDiffViewState) buildRow(row diff.Row, active bool, cursorCol int, scheme ColorScheme, wrap bool) vui.TableRow {
	style := uiStyleForDiffRow(row.Kind, scheme)
	if active {
		style.Background = scheme.Selection
	}
	oldLine, newLine, marker := splitDiffGutter(row)
	return vui.TableRow{Children: []vui.Widget{
		vui.Text{Value: oldLine, Style: uiGutterStyle(row.Kind, active, scheme), Align: vui.TextAlignRight},
		vui.Text{Value: newLine, Style: uiGutterStyle(row.Kind, active, scheme), Align: vui.TextAlignRight},
		vui.Text{Value: marker, Style: uiGutterStyle(row.Kind, active, scheme)},
		uiDiffCodeWidget(row, active, cursorCol, style, scheme, wrap),
	}}
}

func uiDiffCodeWidget(row diff.Row, active bool, cursorCol int, style vaxis.Style, scheme ColorScheme, wrap bool) vui.Widget {
	code := uiDiffRowCode(row)
	if !active || code == "" {
		return vui.Text{Value: code, Style: style, SoftWrap: wrap}
	}
	cursorStyle := vaxis.Style{Background: scheme.Yank}
	segments := styleSegmentsRangeFullWithTabWidth(
		[]vaxis.Segment{{Text: code, Style: style}},
		cursorCol,
		cursorCol+1,
		cursorStyle,
		tabWidthForFile(row.FileName),
	)
	return vui.RichText{Spans: uiTextSpans(segments), SoftWrap: wrap}
}

func uiTextSpans(segments []vaxis.Segment) []vui.TextSpan {
	spans := make([]vui.TextSpan, 0, len(segments))
	for _, segment := range segments {
		spans = append(spans, vui.TextSpan{Text: segment.Text, Style: segment.Style})
	}
	return spans
}

func (s *uiDiffViewState) setCursorRow(rows []diff.Row, row int) {
	if len(rows) == 0 {
		s.cursor = selectionPoint{}
		return
	}
	s.cursor.Row = clampUIDiffInt(row, 0, len(rows)-1)
	s.cursor.Col = s.clampCursorCol(rows, s.cursor.Row, s.cursorCol)
	s.cursorCol = s.cursor.Col
	s.table.RevealRow(s.cursor.Row)
	s.SetState(func() {})
}

func (s *uiDiffViewState) moveCursorRows(rows []diff.Row, delta int) {
	if delta == 0 {
		return
	}
	s.setCursorRow(rows, s.cursor.Row+delta)
}

func (s *uiDiffViewState) halfPageRows() int {
	first, last, ok := s.table.VisibleRange()
	if !ok || last <= first+1 {
		return 1
	}
	return maxInt(1, (last-first)/2)
}

func (s *uiDiffViewState) moveCursorCols(rows []diff.Row, delta int) {
	if s.cursor.Row < 0 || s.cursor.Row >= len(rows) {
		return
	}
	s.cursor.Col = uiDiffMoveCursorCol(rows[s.cursor.Row], s.cursor.Col, delta)
	s.cursorCol = s.cursor.Col
	s.table.RevealRow(s.cursor.Row)
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
	switch len(fields) {
	case 0:
		return "", "", row.Marker
	case 1:
		if row.Kind == diff.RowDelete {
			return fields[0], "", row.Marker
		}
		return "", fields[0], row.Marker
	default:
		return fields[0], fields[1], row.Marker
	}
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

func uiGutterStyle(kind diff.RowKind, active bool, scheme ColorScheme) vaxis.Style {
	style := vaxis.Style{Foreground: scheme.Muted, Background: scheme.Gutter}
	if active {
		style.Background = scheme.Selection
	}
	switch kind {
	case diff.RowAdd:
		style.Foreground = scheme.Add
	case diff.RowDelete:
		style.Foreground = scheme.Delete
	}
	return style
}

func uiStyleForDiffRow(kind diff.RowKind, scheme ColorScheme) vaxis.Style {
	switch kind {
	case diff.RowFile:
		return vaxis.Style{Foreground: scheme.Header, Background: scheme.Background, Attribute: vaxis.AttrBold}
	case diff.RowHunk:
		return vaxis.Style{Foreground: scheme.Hunk, Background: scheme.Background}
	case diff.RowAdd:
		return vaxis.Style{Foreground: scheme.Add, Background: scheme.AddLine}
	case diff.RowDelete:
		return vaxis.Style{Foreground: scheme.Delete, Background: scheme.DeleteLine}
	case diff.RowMeta, diff.RowPreamble, diff.RowNoNewline:
		return vaxis.Style{Foreground: scheme.Muted, Background: scheme.Background}
	case diff.RowCommitHeader:
		return vaxis.Style{Foreground: scheme.Yellow, Background: scheme.Background, Attribute: vaxis.AttrBold}
	case diff.RowCommitMeta:
		return vaxis.Style{Foreground: scheme.Base.Cyan, Background: scheme.Background}
	default:
		return vaxis.Style{Foreground: scheme.Foreground, Background: scheme.Background}
	}
}
