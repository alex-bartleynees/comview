package tui

import (
	"strings"
	"testing"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	vui "git.sr.ht/~rockorager/vaxis/ui"

	"github.com/rockorager/comview/diff"
	"github.com/rockorager/comview/review"
)

func uiDiffTestTheme() vui.Theme {
	return uiThemeFromBaseColors(DefaultBaseColors())
}

func newUIDiffTestApp(rows []diff.Row, wrap bool) *vui.App {
	base := DefaultBaseColors()
	return newUIDiffTestAppWithBase(rows, base, wrap)
}

func newUIDiffTestAppWithBase(rows []diff.Row, base BaseColors, wrap bool) *vui.App {
	return newUIDiffTestAppWithBaseAndDrafts(rows, base, wrap, nil)
}

func newUIDiffTestAppWithBaseAndDrafts(rows []diff.Row, base BaseColors, wrap bool, drafts []review.CommentDraft) *vui.App {
	return newUIDiffTestAppWithBaseDraftsAndStatus(rows, base, wrap, drafts, false)
}

func newUIDiffTestAppWithBaseDraftsAndStatus(rows []diff.Row, base BaseColors, wrap bool, drafts []review.CommentDraft, showStatus bool) *vui.App {
	theme := uiThemeFromBaseColors(base)
	return vui.NewApp(uiDiffRootWithStatus(rows, wrap, drafts, showStatus), vui.WithTheme(theme))
}

func TestUIDiffViewRendersRowsAsSliverTable(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "same"},
		{Kind: diff.RowDelete, Gutter: "2     ", Marker: "-", Code: "old"},
		{Kind: diff.RowAdd, Gutter: "  2   ", Marker: "+", Code: "new"},
	}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 20, Height: 3})
	app.Pump(vui.Size{Width: 20, Height: 3})

	p := vui.NewPainter(vui.Size{Width: 20, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "1" {
		t.Fatalf("old gutter = %q, want 1", got)
	}
	if got := p.Cell(2, 0).Grapheme; got != "1" {
		t.Fatalf("new gutter = %q, want 1", got)
	}
	if got := p.Cell(6, 0).Grapheme; got != "s" {
		t.Fatalf("code start = %q, want s", got)
	}
	if got := p.Cell(4, 1).Grapheme; got != "-" {
		t.Fatalf("delete marker = %q, want -", got)
	}
	if got := p.Cell(4, 2).Grapheme; got != "+" {
		t.Fatalf("add marker = %q, want +", got)
	}
}

func TestUIDiffViewStatusBar(t *testing.T) {
	app := newUIDiffTestAppWithBaseDraftsAndStatus([]diff.Row{{Kind: diff.RowContext, Text: "line"}}, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 20, Height: 2})
	app.Pump(vui.Size{Width: 20, Height: 2})

	p := vui.NewPainter(vui.Size{Width: 20, Height: 2})
	app.Paint(p)
	if got := uiDiffPainterText(p, 1); got != " NORMAL " {
		t.Fatalf("status bar = %q, want NORMAL segment", got)
	}
}

func TestUIDiffViewStatusBarShowsFileAndStats(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowFile, Text: "src/main.go"},
		{Kind: diff.RowAdd, Gutter: "1 1   ", Code: "new"},
		{Kind: diff.RowDelete, Gutter: "2 2   ", Code: "old"},
	}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 40, Height: 3})
	app.Pump(vui.Size{Width: 40, Height: 3})

	p := vui.NewPainter(vui.Size{Width: 40, Height: 3})
	app.Paint(p)
	if got := uiDiffPainterText(p, 2); got != " NORMAL  src/main.go  +1 -1" {
		t.Fatalf("status bar = %q, want file context", got)
	}
	if got := p.Cell(8, 2).Background; got != uiDiffStatusBackground(uiDiffTestTheme()) {
		t.Fatalf("mode separator background = %v, want following status background", got)
	}
}

func TestUIDiffViewFileFinderItemsIncludeDiffStatFiles(t *testing.T) {
	rows, err := rowsForInput(` README.md        |  1 +
 tui/app.go       | 12 ++++++------
 2 files changed, 7 insertions(+), 6 deletions(-)
`)
	if err != nil {
		t.Fatal(err)
	}

	items := uiDiffFileFinderItems(rows)
	if len(items) != 2 {
		t.Fatalf("items = %+v, want 2", items)
	}
	if items[1].Label != "tui/app.go" || items[1].Detail != "+6 -6" {
		t.Fatalf("second item = %+v", items[1])
	}
}

func TestUIDiffViewFileFinderJumpsToSelectedFile(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowFile, Text: "first.go"},
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "first"},
		{Kind: diff.RowFile, Text: "second.go"},
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "second"},
	}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 60, Height: 10})
	app.Pump(vui.Size{Width: 60, Height: 10})

	app.Send(vaxis.Key{Text: " ", Keycode: vaxis.KeySpace})
	app.Send(vaxis.Key{Text: "e", Keycode: 'e'})
	app.Pump(vui.Size{Width: 60, Height: 10})
	p := vui.NewPainter(vui.Size{Width: 60, Height: 10})
	app.Paint(p)
	if _, _, ok := uiDiffFindText(p, "Find file…"); !ok {
		t.Fatal("file finder did not open")
	}

	app.Send(vaxis.Key{Text: "second"})
	app.Pump(vui.Size{Width: 60, Height: 10})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Pump(vui.Size{Width: 60, Height: 10})
	app.Pump(vui.Size{Width: 60, Height: 10})
	p = vui.NewPainter(vui.Size{Width: 60, Height: 10})
	app.Paint(p)
	if got := uiDiffPainterText(p, 2); got != "second.go" {
		t.Fatalf("selected row = %q, want second.go", got)
	}
	if got := uiDiffPainterText(p, 9); got != " NORMAL  2/2 second.go  +0 -0" {
		t.Fatalf("status row = %q, want second file status", got)
	}
}

func TestUIDiffViewFileFinderStatsAreColorized(t *testing.T) {
	theme := uiDiffTestTheme()
	app := vui.NewApp(vui.Provider[vui.Theme]{Value: theme, Child: uiDiffFileStatWidget("+1 -1", theme)})
	app.Pump(vui.Size{Width: 10, Height: 1})
	p := vui.NewPainter(vui.Size{Width: 10, Height: 1})
	app.Paint(p)

	if got := uiDiffPainterText(p, 0); got != "+1 -1" {
		t.Fatalf("stat text = %q, want +1 -1", got)
	}
	if got := p.Cell(0, 0).Foreground; got != theme.Palette.Green.Tone500 {
		t.Fatalf("add stat foreground = %v, want green tone500 %v", got, theme.Palette.Green.Tone500)
	}
	if got := p.Cell(3, 0).Foreground; got != theme.Palette.Red.Tone500 {
		t.Fatalf("delete stat foreground = %v, want red tone500 %v", got, theme.Palette.Red.Tone500)
	}
}

func TestUIDiffViewMovesCursorAndRevealsRows(t *testing.T) {
	rows := make([]diff.Row, 20)
	for i := range rows {
		rows[i] = diff.Row{Kind: diff.RowContext, Gutter: "1 1   ", Code: "line"}
	}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 20, Height: 4})
	app.Pump(vui.Size{Width: 20, Height: 4})

	app.Send(vaxis.Key{Text: "G", Keycode: 'G'})
	app.Pump(vui.Size{Width: 20, Height: 4})
	app.Pump(vui.Size{Width: 20, Height: 4})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 4})
	app.Paint(p)
	if got := p.Cell(0, 3).Background; got != uiDiffCursorRowBackground(uiDiffTestTheme()) {
		t.Fatalf("bottom visible row background = %v, want active selection", got)
	}

	app.Send(vaxis.Key{Text: "k", Keycode: 'k'})
	app.Pump(vui.Size{Width: 20, Height: 4})
	p = vui.NewPainter(vui.Size{Width: 20, Height: 4})
	app.Paint(p)
	if got := p.Cell(0, 2).Background; got != uiDiffCursorRowBackground(uiDiffTestTheme()) {
		t.Fatalf("row above bottom background = %v, want active selection", got)
	}
}

func TestUIDiffViewSkipsBlankRowsWhenMovingCursor(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "first"},
		{Kind: diff.RowBlank},
		{Kind: diff.RowContext, Gutter: "2 2   ", Code: "second"},
	}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 20, Height: 3})
	app.Pump(vui.Size{Width: 20, Height: 3})

	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Pump(vui.Size{Width: 20, Height: 3})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 3})
	app.Paint(p)
	if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != 2 {
		t.Fatalf("highlight row after j = %d, want 2", got)
	}

	app.Send(vaxis.Key{Text: "k", Keycode: 'k'})
	app.Pump(vui.Size{Width: 20, Height: 3})
	p = vui.NewPainter(vui.Size{Width: 20, Height: 3})
	app.Paint(p)
	if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != 0 {
		t.Fatalf("highlight row after k = %d, want 0", got)
	}
}

func TestUIDiffViewKeepsCursorVisibleWhenMovingDown(t *testing.T) {
	rows := make([]diff.Row, 12)
	for i := range rows {
		rows[i] = diff.Row{Kind: diff.RowContext, Gutter: "1 1   ", Code: "line"}
	}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 20, Height: 3})
	app.Pump(vui.Size{Width: 20, Height: 3})

	for i := 0; i < 8; i++ {
		app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
		app.Pump(vui.Size{Width: 20, Height: 3})
		app.Pump(vui.Size{Width: 20, Height: 3})
		p := vui.NewPainter(vui.Size{Width: 20, Height: 3})
		app.Paint(p)
		if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got == -1 {
			t.Fatalf("cursor not visible after %d j presses", i+1)
		}
	}
}

func TestUIDiffViewActiveCodeRowHighlightsToRightEdge(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "short"}}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 20, Height: 1})
	app.Pump(vui.Size{Width: 20, Height: 1})

	p := vui.NewPainter(vui.Size{Width: 20, Height: 1})
	app.Paint(p)
	for col := 0; col < 20; col++ {
		if col == 6 {
			continue
		}
		if got := p.Cell(col, 0).Background; got != uiDiffCursorRowBackground(uiDiffTestTheme()) {
			t.Fatalf("active row background at col %d = %v, want selection", col, got)
		}
	}
}

func TestUIDiffViewChangedRowsHighlightToRightEdge(t *testing.T) {
	theme := uiDiffTestTheme()
	rows := []diff.Row{
		{Kind: diff.RowDelete, Gutter: "1     - ", Code: "old"},
		{Kind: diff.RowAdd, Gutter: "    1 + ", Code: "new"},
	}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 20, Height: 2})
	app.Pump(vui.Size{Width: 20, Height: 2})

	p := vui.NewPainter(vui.Size{Width: 20, Height: 2})
	app.Paint(p)
	if got := p.Cell(19, 0).Background; got != uiDiffCursorRowBackground(theme) {
		t.Fatalf("active delete row right edge background = %v, want selection", got)
	}
	if got := p.Cell(19, 1).Background; got != theme.Surface {
		t.Fatalf("add row right edge background = %v, want surface", got)
	}
}

func TestUIDiffViewUsesStableFixedGutterColumns(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "one"},
		{Kind: diff.RowContext, Gutter: "100 100   ", Code: "hundred"},
	}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 20, Height: 2})
	app.Pump(vui.Size{Width: 20, Height: 2})

	p := vui.NewPainter(vui.Size{Width: 20, Height: 2})
	app.Paint(p)
	if got := p.Cell(10, 0).Grapheme; got != "o" {
		t.Fatalf("first row code start = %q, want o at stable col 10", got)
	}
	if got := p.Cell(10, 1).Grapheme; got != "h" {
		t.Fatalf("second row code start = %q, want h at stable col 10", got)
	}
}

func TestUIDiffViewRendersMetadataOutsideDiffGrid(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowCommitHeader, Text: "commit abc123"},
		{Kind: diff.RowContext, Gutter: "12 34   ", Code: "code"},
	}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 20, Height: 2})
	app.Pump(vui.Size{Width: 20, Height: 2})

	p := vui.NewPainter(vui.Size{Width: 20, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "c" {
		t.Fatalf("metadata row starts at col 0 with %q, want c", got)
	}
	if got := p.Cell(0, 1).Grapheme; got != "1" {
		t.Fatalf("diff row old gutter starts at col 0 with %q, want 1", got)
	}
	if got := p.Cell(8, 1).Grapheme; got != "c" {
		t.Fatalf("diff row code starts after gutter with %q, want c", got)
	}
}

func TestUIDiffViewRendersStructuredFullWidthRows(t *testing.T) {
	base := DefaultBaseColors()
	theme := uiThemeFromBaseColors(base)
	rows := []diff.Row{
		{Kind: diff.RowCommitHeader, Prefix: "commit ", Code: "abc123"},
		{Kind: diff.RowCommitMeta, Prefix: "Author: ", Code: "Example"},
		{Kind: diff.RowHunk, Prefix: "@@ -1 +1 @@", Code: " func"},
	}
	app := newUIDiffTestAppWithBase(rows, base, false)
	app.Pump(vui.Size{Width: 24, Height: 3})
	app.Pump(vui.Size{Width: 24, Height: 3})

	p := vui.NewPainter(vui.Size{Width: 24, Height: 3})
	app.Paint(p)
	if cell := p.Cell(0, 0); cell.Grapheme != "c" || cell.Foreground != theme.DisabledForeground {
		t.Fatalf("commit prefix cell = %q/%v, want c/dim", cell.Grapheme, cell.Foreground)
	}
	if cell := p.Cell(7, 0); cell.Grapheme != "a" || cell.Foreground != theme.Warning {
		t.Fatalf("commit hash cell = %q/%v, want a/yellow", cell.Grapheme, cell.Foreground)
	}
	if cell := p.Cell(0, 1); cell.Grapheme != "A" || cell.Foreground != theme.MutedForeground {
		t.Fatalf("commit meta label = %q/%v, want A/muted", cell.Grapheme, cell.Foreground)
	}
	if cell := p.Cell(8, 1); cell.Grapheme != "E" || cell.Foreground != theme.Palette.Cyan.Tone500 {
		t.Fatalf("commit meta value = %q/%v, want E/cyan", cell.Grapheme, cell.Foreground)
	}
	if cell := p.Cell(0, 2); cell.Grapheme != "@" || cell.Foreground != theme.Accent {
		t.Fatalf("hunk prefix = %q/%v, want @/hunk", cell.Grapheme, cell.Foreground)
	}
	if cell := p.Cell(11, 2); cell.Grapheme != " " || cell.Foreground != theme.DisabledForeground {
		t.Fatalf("hunk suffix = %q/%v, want space/dim", cell.Grapheme, cell.Foreground)
	}
}

func TestUIDiffViewDerivesDiffColorsFromUITheme(t *testing.T) {
	theme := uiThemeFromBaseColors(DefaultBaseColors())
	if style := uiStyleForDiffRow(diff.RowContext, theme); style.Background != theme.Background {
		t.Fatalf("context background = %v, want %v", style.Background, theme.Background)
	}
	if style := uiStyleForDiffRow(diff.RowContext, theme); style.Foreground != theme.MutedForeground {
		t.Fatalf("context foreground = %v, want muted %v", style.Foreground, theme.MutedForeground)
	}
	if style := uiGutterStyle(diff.RowContext, uiDiffCursorRowBackground(theme), theme); style.Background != uiDiffCursorRowBackground(theme) {
		t.Fatalf("active gutter background = %v, want %v", style.Background, uiDiffCursorRowBackground(theme))
	}
	if style := uiGutterStyle(diff.RowContext, 0, theme); style.Background != theme.Background {
		t.Fatalf("gutter background = %v, want %v", style.Background, theme.Background)
	}
	if style := uiStyleForDiffRow(diff.RowAdd, theme); style.Foreground != theme.Success {
		t.Fatalf("add foreground = %v, want %v", style.Foreground, theme.Success)
	}
	if style := uiStyleForDiffRow(diff.RowAdd, theme); style.Background != theme.Surface {
		t.Fatalf("add background = %v, want surface %v", style.Background, theme.Surface)
	}
	if style := uiStyleForDiffRow(diff.RowDelete, theme); style.Foreground != theme.Palette.Red.Tone500 {
		t.Fatalf("delete foreground = %v, want dim red %v", style.Foreground, theme.Palette.Red.Tone500)
	}
	if style := uiStyleForDiffRow(diff.RowDelete, theme); style.Background != theme.Palette.Red.Tone950 {
		t.Fatalf("delete background = %v, want dark red tone", style.Background)
	}
}

func TestUIDiffViewHighlightsCodeWithChroma(t *testing.T) {
	theme := uiDiffTestTheme()
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "    1 + ", Code: "package main", FileName: "main.go"}}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 24, Height: 1})
	app.Pump(vui.Size{Width: 24, Height: 1})

	p := vui.NewPainter(vui.Size{Width: 24, Height: 1})
	app.Paint(p)
	cell := p.Cell(6, 0)
	if cell.Grapheme != "a" {
		t.Fatalf("code glyph = %q, want a", cell.Grapheme)
	}
	if cell.Foreground != theme.Palette.Magenta.Tone500 {
		t.Fatalf("keyword foreground = %v, want magenta tone500 %v", cell.Foreground, theme.Palette.Magenta.Tone500)
	}
	if cell.Background != uiDiffCursorRowBackground(theme) {
		t.Fatalf("keyword background = %v, want selection %v", cell.Background, uiDiffCursorRowBackground(theme))
	}
}

func TestUIDiffViewDimsChromaCodeForContextAndDeletes(t *testing.T) {
	theme := uiDiffTestTheme()
	base := []vaxis.Segment{{Text: "package", Style: vaxis.Style{Foreground: theme.Palette.Magenta.Tone500}}}
	contextSegments := uiDiffToneCodeSegments(diff.RowContext, base, theme)
	deleteSegments := uiDiffToneCodeSegments(diff.RowDelete, base, theme)
	addSegments := uiDiffToneCodeSegments(diff.RowAdd, base, theme)
	if contextSegments[0].Style.Foreground != theme.Palette.Magenta.Tone600 {
		t.Fatalf("context syntax foreground = %v, want magenta tone600", contextSegments[0].Style.Foreground)
	}
	if deleteSegments[0].Style.Foreground != theme.Palette.Magenta.Tone600 {
		t.Fatalf("delete syntax foreground = %v, want magenta tone600", deleteSegments[0].Style.Foreground)
	}
	if addSegments[0].Style.Foreground != base[0].Style.Foreground {
		t.Fatal("add syntax foreground should remain unchanged")
	}
}

func TestUIDiffViewSyntaxColorsUseBrighterPaletteTones(t *testing.T) {
	theme := uiDiffTestTheme()
	colors := (uiSyntaxTheme{Theme: theme}).uiThemeColors()
	if colors.Magenta != theme.Palette.Magenta.Tone500 {
		t.Fatalf("syntax magenta = %v, want tone500 %v", colors.Magenta, theme.Palette.Magenta.Tone500)
	}
	if colors.Blue != theme.Palette.Blue.Tone500 {
		t.Fatalf("syntax blue = %v, want tone500 %v", colors.Blue, theme.Palette.Blue.Tone500)
	}
	if colors.Green != theme.Palette.Green.Tone500 {
		t.Fatalf("syntax green = %v, want tone500 %v", colors.Green, theme.Palette.Green.Tone500)
	}
}

func TestUIDiffViewCursorUsesThemeForeground(t *testing.T) {
	theme := uiDiffTestTheme()
	if got := uiDiffCursorBackground(theme); got != theme.Foreground {
		t.Fatalf("cursor background = %v, want foreground %v", got, theme.Foreground)
	}
	if got := uiDiffCursorForeground(theme); got != theme.Palette.Neutral.Tone950 {
		t.Fatalf("cursor foreground = %v, want dark neutral %v", got, theme.Palette.Neutral.Tone950)
	}
}

func TestUIDiffViewChangedGutterUsesBrighterTone(t *testing.T) {
	theme := uiDiffTestTheme()
	if got := uiGutterStyle(diff.RowAdd, 0, theme).Foreground; got != theme.Palette.Green.Tone400 {
		t.Fatalf("add marker foreground = %v, want green tone400 %v", got, theme.Palette.Green.Tone400)
	}
	if got := uiGutterStyle(diff.RowDelete, 0, theme).Foreground; got != theme.Palette.Red.Tone400 {
		t.Fatalf("delete marker foreground = %v, want red tone400 %v", got, theme.Palette.Red.Tone400)
	}
}

func TestUIDiffViewAddLineNumberGutterUsesSoftGreenForeground(t *testing.T) {
	theme := uiDiffTestTheme()
	if got := uiLineNumberGutterStyle(diff.RowAdd, 0, theme).Foreground; got != theme.Palette.Green.Tone300 {
		t.Fatalf("add line number foreground = %v, want soft green %v", got, theme.Palette.Green.Tone300)
	}
	if got := uiGutterStyle(diff.RowAdd, 0, theme).Foreground; got != theme.Palette.Green.Tone400 {
		t.Fatalf("add marker foreground = %v, want green tone400 %v", got, theme.Palette.Green.Tone400)
	}
}

func TestUIDiffViewVimNavigationKeys(t *testing.T) {
	tests := []struct {
		name          string
		prime         []vaxis.Key
		key           vaxis.Key
		wantHighlight int
	}{
		{
			name:          "Home moves cursor to top",
			prime:         []vaxis.Key{{Text: "G", Keycode: 'G'}},
			key:           vaxis.Key{Keycode: vaxis.KeyHome},
			wantHighlight: 0,
		},
		{
			name:          "End moves cursor to bottom",
			key:           vaxis.Key{Keycode: vaxis.KeyEnd},
			wantHighlight: 3,
		},
		{
			name:          "gg moves cursor to top",
			prime:         []vaxis.Key{{Text: "G", Keycode: 'G'}},
			key:           vaxis.Key{Text: "g", Keycode: 'g'},
			wantHighlight: 0,
		},
		{
			name:          "Ctrl+d moves cursor down half page",
			key:           vaxis.Key{Text: "d", Keycode: 'd', Modifiers: vaxis.ModCtrl},
			wantHighlight: 2,
		},
		{
			name:          "Page Down moves cursor down half page",
			key:           vaxis.Key{Keycode: vaxis.KeyPgDown},
			wantHighlight: 2,
		},
		{
			name:          "Ctrl+u moves cursor up half page",
			prime:         []vaxis.Key{{Text: "G", Keycode: 'G'}},
			key:           vaxis.Key{Text: "u", Keycode: 'u', Modifiers: vaxis.ModCtrl},
			wantHighlight: 1,
		},
		{
			name:          "Page Up moves cursor up half page",
			prime:         []vaxis.Key{{Text: "G", Keycode: 'G'}},
			key:           vaxis.Key{Keycode: vaxis.KeyPgUp},
			wantHighlight: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := make([]diff.Row, 20)
			for i := range rows {
				rows[i] = diff.Row{Kind: diff.RowContext, Gutter: "1 1   ", Code: "line"}
			}
			app := newUIDiffTestApp(rows, false)
			app.Pump(vui.Size{Width: 20, Height: 4})
			app.Pump(vui.Size{Width: 20, Height: 4})
			for _, key := range tt.prime {
				app.Send(key)
				app.Pump(vui.Size{Width: 20, Height: 4})
				app.Pump(vui.Size{Width: 20, Height: 4})
			}
			app.Send(tt.key)
			if tt.key.Text == "g" {
				app.Send(tt.key)
			}
			app.Pump(vui.Size{Width: 20, Height: 4})
			app.Pump(vui.Size{Width: 20, Height: 4})

			p := vui.NewPainter(vui.Size{Width: 20, Height: 4})
			app.Paint(p)
			if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != tt.wantHighlight {
				t.Fatalf("highlight row = %d, want %d", got, tt.wantHighlight)
			}
		})
	}
}

func TestUIDiffViewBracketCJumpsBetweenChanges(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "same"},
		{Kind: diff.RowDelete, Gutter: "2     - ", Code: "old"},
		{Kind: diff.RowAdd, Gutter: "    2 + ", Code: "new"},
		{Kind: diff.RowContext, Gutter: "3 3   ", Code: "same"},
		{Kind: diff.RowAdd, Gutter: "    4 + ", Code: "other"},
	}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 24, Height: 5})
	app.Pump(vui.Size{Width: 24, Height: 5})

	app.Send(vaxis.Key{Text: "]", Keycode: ']'})
	app.Pump(vui.Size{Width: 24, Height: 5})
	app.Send(vaxis.Key{Text: "c", Keycode: 'c'})
	app.Pump(vui.Size{Width: 24, Height: 5})
	p := vui.NewPainter(vui.Size{Width: 24, Height: 5})
	app.Paint(p)
	if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != 1 {
		t.Fatalf("]c highlight row = %d, want 1", got)
	}

	app.Send(vaxis.Key{Text: "]", Keycode: ']'})
	app.Send(vaxis.Key{Text: "c", Keycode: 'c'})
	app.Pump(vui.Size{Width: 24, Height: 5})
	p = vui.NewPainter(vui.Size{Width: 24, Height: 5})
	app.Paint(p)
	if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != 4 {
		t.Fatalf("second ]c highlight row = %d, want 4", got)
	}

	app.Send(vaxis.Key{Text: "[", Keycode: '['})
	app.Send(vaxis.Key{Text: "c", Keycode: 'c'})
	app.Pump(vui.Size{Width: 24, Height: 5})
	p = vui.NewPainter(vui.Size{Width: 24, Height: 5})
	app.Paint(p)
	if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != 1 {
		t.Fatalf("[c highlight row = %d, want 1", got)
	}
}

func TestUIDiffViewBracketNJumpsBetweenNotes(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowAdd, Gutter: "1 1   ", Code: "one", Review: review.Anchor{Path: "main.go", Line: 1, Side: review.SideRight}},
		{Kind: diff.RowAdd, Gutter: "2 2   ", Code: "two", Review: review.Anchor{Path: "main.go", Line: 2, Side: review.SideRight}},
		{Kind: diff.RowAdd, Gutter: "3 3   ", Code: "three", Review: review.Anchor{Path: "main.go", Line: 3, Side: review.SideRight}},
	}
	drafts := []review.CommentDraft{
		{Path: "main.go", Line: 2, Side: review.SideRight, Body: "two"},
		{Path: "main.go", Line: 3, Side: review.SideRight, Body: "three"},
	}
	app := newUIDiffTestAppWithBaseAndDrafts(rows, DefaultBaseColors(), false, drafts)
	app.Pump(vui.Size{Width: 24, Height: 3})
	app.Pump(vui.Size{Width: 24, Height: 3})

	app.Send(vaxis.Key{Text: "]", Keycode: ']'})
	app.Pump(vui.Size{Width: 24, Height: 3})
	app.Send(vaxis.Key{Text: "n", Keycode: 'n'})
	app.Pump(vui.Size{Width: 24, Height: 3})
	p := vui.NewPainter(vui.Size{Width: 24, Height: 3})
	app.Paint(p)
	if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != 1 {
		t.Fatalf("]n highlight row = %d, want 1", got)
	}

	app.Send(vaxis.Key{Text: "]", Keycode: ']'})
	app.Send(vaxis.Key{Text: "n", Keycode: 'n'})
	app.Pump(vui.Size{Width: 24, Height: 3})
	p = vui.NewPainter(vui.Size{Width: 24, Height: 3})
	app.Paint(p)
	if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != 2 {
		t.Fatalf("second ]n highlight row = %d, want 2", got)
	}

	app.Send(vaxis.Key{Text: "[", Keycode: '['})
	app.Send(vaxis.Key{Text: "n", Keycode: 'n'})
	app.Pump(vui.Size{Width: 24, Height: 3})
	p = vui.NewPainter(vui.Size{Width: 24, Height: 3})
	app.Paint(p)
	if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != 1 {
		t.Fatalf("[n highlight row = %d, want 1", got)
	}
}

func TestUIDiffViewSlashSearchMovesToMatch(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "alpha"},
		{Kind: diff.RowContext, Gutter: "2 2   ", Code: "needle"},
	}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 20, Height: 3})
	app.Pump(vui.Size{Width: 20, Height: 3})

	app.Send(vaxis.Key{Text: "/", Keycode: '/'})
	app.Send(vaxis.Key{Text: "needle"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Pump(vui.Size{Width: 20, Height: 3})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 3})
	app.Paint(p)
	if cell := p.Cell(6, 1); cell.Grapheme != "n" || cell.Background != uiDiffCursorBackground(uiDiffTestTheme()) {
		t.Fatalf("search cursor = %q/%v, want n/cursor", cell.Grapheme, cell.Background)
	}
	if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != 1 {
		t.Fatalf("search highlight row = %d, want 1", got)
	}
}

func TestUIDiffViewSearchModeUsesStatusBar(t *testing.T) {
	app := newUIDiffTestAppWithBaseDraftsAndStatus([]diff.Row{{Kind: diff.RowContext, Text: "needle"}}, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 20, Height: 2})
	app.Pump(vui.Size{Width: 20, Height: 2})

	app.Send(vaxis.Key{Text: "/", Keycode: '/'})
	app.Pump(vui.Size{Width: 20, Height: 2})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 2})
	app.Paint(p)
	if got := uiDiffPainterText(p, 1); got != "/" {
		t.Fatalf("empty search status = %q, want /", got)
	}

	app.Send(vaxis.Key{Text: "needle"})
	app.Pump(vui.Size{Width: 20, Height: 2})
	p = vui.NewPainter(vui.Size{Width: 20, Height: 2})
	app.Paint(p)
	if got := uiDiffPainterText(p, 1); got != "/needle" {
		t.Fatalf("search status = %q, want /needle", got)
	}
}

func TestUIDiffViewIncrementalSearchHighlightsMatches(t *testing.T) {
	theme := uiDiffTestTheme()
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "needle"},
		{Kind: diff.RowHunk, Prefix: "@@ -10 +10 @@", Code: " func"},
	}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 24, Height: 2})
	app.Pump(vui.Size{Width: 24, Height: 2})

	app.Send(vaxis.Key{Text: "/", Keycode: '/'})
	app.Send(vaxis.Key{Text: "needle"})
	app.Pump(vui.Size{Width: 24, Height: 2})
	p := vui.NewPainter(vui.Size{Width: 24, Height: 2})
	app.Paint(p)
	if got := p.Cell(7, 0).Background; got != uiDiffSearchHighlightStyle(theme).Background {
		t.Fatalf("code search highlight background = %v, want warning", got)
	}

	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Send(vaxis.Key{Text: "/", Keycode: '/'})
	app.Send(vaxis.Key{Text: "-10"})
	app.Pump(vui.Size{Width: 24, Height: 2})
	p = vui.NewPainter(vui.Size{Width: 24, Height: 2})
	app.Paint(p)
	if got := p.Cell(3, 1).Background; got != uiDiffSearchHighlightStyle(theme).Background {
		t.Fatalf("structured search highlight background = %v, want warning", got)
	}
}

func TestUIDiffViewSearchesStructuredRows(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "alpha"},
		{Kind: diff.RowHunk, Prefix: "@@ -10 +10 @@", Code: " func"},
		{Kind: diff.RowCommitHeader, Prefix: "commit ", Code: "abc123"},
	}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 24, Height: 3})
	app.Pump(vui.Size{Width: 24, Height: 3})

	app.Send(vaxis.Key{Text: "/", Keycode: '/'})
	app.Send(vaxis.Key{Text: "-10"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Pump(vui.Size{Width: 24, Height: 3})
	p := vui.NewPainter(vui.Size{Width: 24, Height: 3})
	app.Paint(p)
	if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != 1 {
		t.Fatalf("structured hunk search highlight row = %d, want 1", got)
	}

	app.Send(vaxis.Key{Text: "/", Keycode: '/'})
	app.Send(vaxis.Key{Text: "commit"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Pump(vui.Size{Width: 24, Height: 3})
	p = vui.NewPainter(vui.Size{Width: 24, Height: 3})
	app.Paint(p)
	if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != 2 {
		t.Fatalf("structured commit search highlight row = %d, want 2", got)
	}
}

func TestUIDiffViewIncrementalSearchUsesNextMatchFromStart(t *testing.T) {
	rows := make([]diff.Row, 10)
	for i := range rows {
		rows[i] = diff.Row{Kind: diff.RowContext, Text: "line"}
	}
	rows[2].Text = "alpha"
	rows[8].Text = "alpha"
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 20, Height: 4})
	app.Pump(vui.Size{Width: 20, Height: 4})
	for i := 0; i < 5; i++ {
		app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
		app.Pump(vui.Size{Width: 20, Height: 4})
	}

	app.Send(vaxis.Key{Text: "/", Keycode: '/'})
	app.Send(vaxis.Key{Text: "alpha"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Pump(vui.Size{Width: 20, Height: 4})
	app.Pump(vui.Size{Width: 20, Height: 4})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 4})
	app.Paint(p)
	if got := uiDiffPainterText(p, 3); got != "alpha" {
		t.Fatalf("search target text = %q, want alpha", got)
	}
	if got := p.Cell(1, 3).Background; got != uiDiffSearchHighlightStyle(uiDiffTestTheme()).Background {
		t.Fatalf("search target background = %v, want search highlight", got)
	}
}

func TestUIDiffViewSearchNextPrevious(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Text: "one needle"},
		{Kind: diff.RowContext, Text: "two needle"},
	}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 20, Height: 2})
	app.Pump(vui.Size{Width: 20, Height: 2})
	app.Send(vaxis.Key{Text: "/", Keycode: '/'})
	app.Send(vaxis.Key{Text: "needle"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Pump(vui.Size{Width: 20, Height: 2})

	app.Send(vaxis.Key{Text: "n", Keycode: 'n'})
	app.Pump(vui.Size{Width: 20, Height: 2})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 2})
	app.Paint(p)
	if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != 1 {
		t.Fatalf("n highlight row = %d, want 1", got)
	}

	app.Send(vaxis.Key{Text: "N", Keycode: 'N'})
	app.Pump(vui.Size{Width: 20, Height: 2})
	p = vui.NewPainter(vui.Size{Width: 20, Height: 2})
	app.Paint(p)
	if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != 0 {
		t.Fatalf("N highlight row = %d, want 0", got)
	}
}

func TestUIDiffViewEscapeClearsSearch(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Text: "needle"}}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 20, Height: 1})
	app.Pump(vui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Text: "/", Keycode: '/'})
	app.Send(vaxis.Key{Text: "needle"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Pump(vui.Size{Width: 20, Height: 1})

	app.Send(vaxis.Key{Text: "n", Keycode: 'n'})
	app.Pump(vui.Size{Width: 20, Height: 1})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if cell := p.Cell(0, 0); cell.Background != uiDiffCursorRowBackground(uiDiffTestTheme()) {
		t.Fatalf("cursor moved after escaped search: first cell bg = %v, want selection", cell.Background)
	}
}

func TestUIDiffViewEscapeClearsAcceptedSearch(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Text: "needle"}}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 20, Height: 1})
	app.Pump(vui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Text: "/", Keycode: '/'})
	app.Send(vaxis.Key{Text: "needle"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Pump(vui.Size{Width: 20, Height: 1})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Background; got != uiDiffSearchHighlightStyle(uiDiffTestTheme()).Background {
		t.Fatalf("search highlight background = %v, want search highlight", got)
	}

	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Pump(vui.Size{Width: 20, Height: 1})
	p = vui.NewPainter(vui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Background; got == uiDiffSearchHighlightStyle(uiDiffTestTheme()).Background {
		t.Fatal("search highlight still visible after escape")
	}
}

func TestUIDiffViewLinewiseSelectionExtendsWithCursor(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "one"},
		{Kind: diff.RowContext, Gutter: "2 2   ", Code: "two"},
		{Kind: diff.RowContext, Gutter: "3 3   ", Code: "three"},
	}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 30, Height: 3})
	app.Pump(vui.Size{Width: 30, Height: 3})

	app.Send(vaxis.Key{Text: "V", Keycode: 'V'})
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Pump(vui.Size{Width: 30, Height: 3})
	p := vui.NewPainter(vui.Size{Width: 30, Height: 3})
	app.Paint(p)
	theme := uiDiffTestTheme()
	if got := p.Cell(6, 0).Background; got != theme.Selection {
		t.Fatalf("anchor text background = %v, want selection", got)
	}
	if got := p.Cell(7, 1).Background; got != theme.Selection {
		t.Fatalf("cursor text background = %v, want selection", got)
	}
	if got := p.Cell(29, 0).Background; got == theme.Selection {
		t.Fatal("anchor row selection extends past text")
	}
	if got := p.Cell(29, 1).Background; got == theme.Selection {
		t.Fatal("cursor row selection extends past text")
	}
	if got := p.Cell(29, 2).Background; got == theme.Selection {
		t.Fatal("unselected row has selection background")
	}
}

func TestUIDiffViewVisualLineSkipsHunkRows(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "one"},
		{Kind: diff.RowHunk, Text: "@@ -2 +2 @@"},
		{Kind: diff.RowContext, Gutter: "2 2   ", Code: "two"},
	}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 30, Height: 3})
	app.Pump(vui.Size{Width: 30, Height: 3})

	app.Send(vaxis.Key{Text: "V", Keycode: 'V'})
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Pump(vui.Size{Width: 30, Height: 3})
	p := vui.NewPainter(vui.Size{Width: 30, Height: 3})
	app.Paint(p)
	theme := uiDiffTestTheme()
	if got := p.Cell(6, 0).Background; got != theme.Selection {
		t.Fatalf("first code row background = %v, want selection", got)
	}
	if got := p.Cell(0, 1).Background; got == theme.Selection {
		t.Fatal("hunk row was selected")
	}
	if got := p.Cell(7, 2).Background; got != theme.Selection {
		t.Fatalf("second code row background = %v, want selection", got)
	}
}

func TestUIDiffViewSelectionTextLinewiseSkipsHunkRows(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "one"},
		{Kind: diff.RowHunk, Text: "@@ -2 +2 @@"},
		{Kind: diff.RowContext, Gutter: "2 2   ", Code: "two"},
	}
	state := &uiDiffViewState{
		selectionActive:   true,
		selectionLinewise: true,
		selectionAnchor:   selectionPoint{Row: 0},
		cursor:            selectionPoint{Row: 2},
	}
	if got, want := state.selectionText(rows), "one\ntwo"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestUIDiffViewSelectionTextSkipsNonSelectableRowsWithoutExtraNewline(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Code: "hello", Text: "hello"},
		{Kind: diff.RowHunk, Text: "@@ -1 +1 @@ func main()", Prefix: "@@ -1 +1 @@", Code: " func main()"},
		{Kind: diff.RowContext, Code: "world", Text: "world"},
	}
	state := &uiDiffViewState{
		selectionActive: true,
		selectionAnchor: selectionPoint{Row: 0, Col: 0},
		cursor:          selectionPoint{Row: 1, Col: textCellWidth(rows[1].Text) - 1},
	}
	if got, want := state.selectionText(rows), "hello"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}

	state.cursor = selectionPoint{Row: 2, Col: textCellWidth(rows[2].Text) - 1}
	if got, want := state.selectionText(rows), "hello\nworld"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestUIDiffViewSelectionTextCharacterwise(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "abcd"},
		{Kind: diff.RowContext, Gutter: "2 2   ", Code: "wxyz"},
	}
	state := &uiDiffViewState{
		selectionActive: true,
		selectionAnchor: selectionPoint{Row: 0, Col: 1},
		cursor:          selectionPoint{Row: 1, Col: 2},
	}
	if got, want := state.selectionText(rows), "bcd\nwxy"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestUIDiffViewMouseDragSelectsCode(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "abcde"},
		{Kind: diff.RowContext, Gutter: "2 2   ", Code: "fghij"},
	}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 30, Height: 3})
	app.Pump(vui.Size{Width: 30, Height: 3})
	codeOffset := uiDiffCodeOffset(rows)

	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 0, Col: codeOffset + 1})
	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 1, Col: codeOffset + 2})
	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: 1, Col: codeOffset + 2})
	app.Pump(vui.Size{Width: 30, Height: 3})

	p := vui.NewPainter(vui.Size{Width: 30, Height: 3})
	app.Paint(p)
	theme := uiDiffTestTheme()
	if got := p.Cell(codeOffset+1, 0).Background; got != theme.Selection {
		t.Fatalf("first selected cell background = %v, want selection", got)
	}
	if got := p.Cell(codeOffset+2, 1).Background; got != uiDiffCursorBackground(theme) {
		t.Fatalf("cursor cell background = %v, want cursor", got)
	}
	if got := p.Cell(0, 0).Background; got == theme.Selection {
		t.Fatal("gutter was selected")
	}
}

func TestUIDiffViewMouseClickDoesNotSelect(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "abcde"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 30, Height: 3})
	app.Pump(vui.Size{Width: 30, Height: 3})
	codeOffset := uiDiffCodeOffset(rows)

	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 0, Col: codeOffset + 2})
	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: 0, Col: codeOffset + 2})
	app.Pump(vui.Size{Width: 30, Height: 3})

	p := vui.NewPainter(vui.Size{Width: 30, Height: 3})
	app.Paint(p)
	if got := p.Cell(codeOffset+2, 0).Background; got == uiDiffTestTheme().Selection {
		t.Fatal("single click selected text")
	}
	if got := uiDiffPainterText(p, 2); !strings.HasPrefix(got, " NORMAL ") {
		t.Fatalf("status bar = %q, want NORMAL", got)
	}
}

func TestUIDiffViewDoubleClickSelectsToken(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "foo bar.baz"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 30, Height: 3})
	app.Pump(vui.Size{Width: 30, Height: 3})
	codeOffset := uiDiffCodeOffset(rows)

	for i := 0; i < 2; i++ {
		app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 0, Col: codeOffset + 5})
		app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: 0, Col: codeOffset + 5})
	}
	app.Pump(vui.Size{Width: 30, Height: 3})

	p := vui.NewPainter(vui.Size{Width: 30, Height: 3})
	app.Paint(p)
	theme := uiDiffTestTheme()
	for col := codeOffset + 4; col <= codeOffset+6; col++ {
		if got := p.Cell(col, 0).Background; got != theme.Selection && got != uiDiffCursorBackground(theme) {
			t.Fatalf("token cell %d background = %v, want selected/cursor", col, got)
		}
	}
	if got := p.Cell(codeOffset+7, 0).Background; got == theme.Selection {
		t.Fatal("punctuation after token was selected")
	}
	if got := uiDiffPainterText(p, 2); !strings.HasPrefix(got, " VISUAL ") {
		t.Fatalf("status bar = %q, want VISUAL", got)
	}
}

func TestUIDiffViewTripleClickSelectsCodeRow(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "hello"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 30, Height: 3})
	app.Pump(vui.Size{Width: 30, Height: 3})
	codeOffset := uiDiffCodeOffset(rows)

	for i := 0; i < 3; i++ {
		app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 0, Col: codeOffset + 1})
		app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: 0, Col: codeOffset + 1})
	}
	app.Pump(vui.Size{Width: 30, Height: 3})

	p := vui.NewPainter(vui.Size{Width: 30, Height: 3})
	app.Paint(p)
	for col := codeOffset; col < codeOffset+len("hello"); col++ {
		if got := p.Cell(col, 0).Background; got != uiDiffTestTheme().Selection && got != uiDiffCursorBackground(uiDiffTestTheme()) {
			t.Fatalf("row cell %d background = %v, want selected/cursor", col, got)
		}
	}
	if got := p.Cell(0, 0).Background; got == uiDiffTestTheme().Selection {
		t.Fatal("triple click selected gutter")
	}
	if got := uiDiffPainterText(p, 2); !strings.HasPrefix(got, " V-LINE ") {
		t.Fatalf("status bar = %q, want V-LINE", got)
	}
}

func TestUIDiffViewMouseSelectionSkipsHunkRows(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "hello"},
		{Kind: diff.RowHunk, Text: "@@ -1 +1 @@", Prefix: "@@ -1 +1 @@"},
		{Kind: diff.RowContext, Gutter: "2 2   ", Code: "world"},
	}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 30, Height: 4})
	app.Pump(vui.Size{Width: 30, Height: 4})
	codeOffset := uiDiffCodeOffset(rows)

	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 0, Col: codeOffset})
	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 1, Col: codeOffset + 2})
	app.Pump(vui.Size{Width: 30, Height: 4})
	p := vui.NewPainter(vui.Size{Width: 30, Height: 4})
	app.Paint(p)
	if got := p.Cell(0, 1).Background; got == uiDiffTestTheme().Selection {
		t.Fatal("hunk row was selected")
	}

	app.Pump(vui.Size{Width: 30, Height: 4})

	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 2, Col: codeOffset + 1})
	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: 2, Col: codeOffset + 1})
	app.Pump(vui.Size{Width: 30, Height: 4})
	p = vui.NewPainter(vui.Size{Width: 30, Height: 4})
	app.Paint(p)
	if got := p.Cell(codeOffset, 2).Background; got != uiDiffTestTheme().Selection {
		t.Fatalf("second code row selected cell background = %v, want selection", got)
	}
}

func TestUIDiffViewDragFromHunkIntoCodeStartsAtLineStart(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowHunk, Text: "@@ -1 +1 @@", Prefix: "@@ -1 +1 @@"},
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "abcdef"},
	}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 30, Height: 4})
	app.Pump(vui.Size{Width: 30, Height: 4})
	codeOffset := uiDiffCodeOffset(rows)

	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 0, Col: 2})
	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 1, Col: codeOffset + 1})
	app.Pump(vui.Size{Width: 30, Height: 4})

	p := vui.NewPainter(vui.Size{Width: 30, Height: 4})
	app.Paint(p)
	theme := uiDiffTestTheme()
	if got := p.Cell(codeOffset, 1).Background; got != theme.Selection {
		t.Fatalf("line-start selected background = %v, want selection", got)
	}
	if got := p.Cell(codeOffset+1, 1).Background; got != uiDiffCursorBackground(theme) {
		t.Fatalf("drag cursor background = %v, want cursor", got)
	}
	if got := p.Cell(0, 0).Background; got == theme.Selection {
		t.Fatal("hunk row was selected")
	}
}

func TestUIDiffViewDragFromHunkUpIntoCodeStartsAtLineEnd(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "abcdef"},
		{Kind: diff.RowHunk, Text: "@@ -1 +1 @@", Prefix: "@@ -1 +1 @@"},
	}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 30, Height: 4})
	app.Pump(vui.Size{Width: 30, Height: 4})
	codeOffset := uiDiffCodeOffset(rows)

	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 1, Col: 2})
	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 0, Col: codeOffset + 4})
	app.Pump(vui.Size{Width: 30, Height: 4})

	p := vui.NewPainter(vui.Size{Width: 30, Height: 4})
	app.Paint(p)
	theme := uiDiffTestTheme()
	if got := p.Cell(codeOffset+4, 0).Background; got != uiDiffCursorBackground(theme) {
		t.Fatalf("drag cursor background = %v, want cursor", got)
	}
	if got := p.Cell(codeOffset+5, 0).Background; got != theme.Selection {
		t.Fatalf("line-end selected background = %v, want selection", got)
	}
	if got := p.Cell(codeOffset+3, 0).Background; got == theme.Selection {
		t.Fatal("drag selection started before target cell")
	}
}

func TestUIDiffViewMouseWheelExtendsDraggingSelection(t *testing.T) {
	rows := make([]diff.Row, 12)
	for i := range rows {
		rows[i] = diff.Row{Kind: diff.RowContext, Gutter: "1 1   ", Code: "abcdef"}
	}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 30, Height: 4})
	app.Pump(vui.Size{Width: 30, Height: 4})
	codeOffset := uiDiffCodeOffset(rows)

	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 0, Col: codeOffset})
	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 1, Col: codeOffset})
	app.Send(vaxis.Mouse{Button: vaxis.MouseWheelDown, EventType: vaxis.EventPress, Row: 1, Col: codeOffset + 2})
	app.Pump(vui.Size{Width: 30, Height: 4})
	app.Pump(vui.Size{Width: 30, Height: 4})

	p := vui.NewPainter(vui.Size{Width: 30, Height: 4})
	app.Paint(p)
	theme := uiDiffTestTheme()
	if got := p.Cell(codeOffset+2, 1).Background; got != uiDiffCursorBackground(theme) {
		t.Fatalf("cursor after wheel drag background = %v, want cursor", got)
	}
	if got := p.Cell(codeOffset, 0).Background; got != theme.Selection {
		t.Fatalf("scrolled selection background = %v, want selection", got)
	}
}

func TestUIDiffViewMouseWheelDoesNotExtendFinishedSelection(t *testing.T) {
	rows := make([]diff.Row, 12)
	for i := range rows {
		rows[i] = diff.Row{Kind: diff.RowContext, Gutter: "1 1   ", Code: "abcdef"}
	}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 30, Height: 4})
	app.Pump(vui.Size{Width: 30, Height: 4})
	codeOffset := uiDiffCodeOffset(rows)

	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 0, Col: codeOffset})
	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 1, Col: codeOffset})
	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease, Row: 1, Col: codeOffset})
	app.Send(vaxis.Mouse{Button: vaxis.MouseWheelDown, EventType: vaxis.EventPress, Row: 1, Col: codeOffset + 2})
	app.Pump(vui.Size{Width: 30, Height: 4})
	app.Pump(vui.Size{Width: 30, Height: 4})

	p := vui.NewPainter(vui.Size{Width: 30, Height: 4})
	app.Paint(p)
	if got := p.Cell(codeOffset+2, 1).Background; got == uiDiffCursorBackground(uiDiffTestTheme()) {
		t.Fatal("finished selection cursor moved to mouse position after wheel")
	}
}

func TestUIDiffViewSelectionTextPreservesTabs(t *testing.T) {
	row := diff.Row{Kind: diff.RowContext, Code: "a\tb", Text: "a\tb"}
	state := &uiDiffViewState{
		selectionActive: true,
		selectionAnchor: selectionPoint{Row: 0, Col: 1},
		cursor:          selectionPoint{Row: 0, Col: 1},
	}
	if got, want := state.selectionText([]diff.Row{row}), "\t"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestUIDiffViewSelectionTextSkipsCommitRows(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Code: "selectable", Text: "selectable"},
		{Kind: diff.RowCommitHeader, Text: "commit abc123"},
		{Kind: diff.RowCommitMeta, Text: "Author: Example"},
	}
	state := &uiDiffViewState{
		selectionActive: true,
		selectionAnchor: selectionPoint{Row: 0, Col: 0},
		cursor:          selectionPoint{Row: 2, Col: textCellWidth(rows[2].Text) - 1},
	}
	if got, want := state.selectionText(rows), "selectable"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestUIDiffViewYankClearsSelectionAndHighlightsText(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "one"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 30, Height: 3})
	app.Pump(vui.Size{Width: 30, Height: 3})

	app.Send(vaxis.Key{Text: "V", Keycode: 'V'})
	app.Send(vaxis.Key{Text: "y", Keycode: 'y'})
	app.Pump(vui.Size{Width: 30, Height: 3})
	p := vui.NewPainter(vui.Size{Width: 30, Height: 3})
	app.Paint(p)
	if got := uiDiffPainterText(p, 2); !strings.HasPrefix(got, " NORMAL ") {
		t.Fatalf("status bar = %q, want NORMAL after yank", got)
	}
	if got := p.Cell(7, 0).Background; got != uiDiffYankBackground(uiDiffTestTheme()) {
		t.Fatalf("yank background = %v, want yank", got)
	}
	if got := p.Cell(29, 0).Background; got == uiDiffYankBackground(uiDiffTestTheme()) {
		t.Fatal("yank highlight extends past text")
	}
}

func TestUIDiffViewYankHighlightExpires(t *testing.T) {
	state := &uiDiffViewState{
		yankActive:   true,
		yankLinewise: true,
		yankAnchor:   selectionPoint{Row: 0},
		yankCursor:   selectionPoint{Row: 1},
		yankUntil:    time.Unix(10, 0),
	}
	state.clearExpiredYank(time.Unix(9, 0))
	if !state.yankActive {
		t.Fatal("yank highlight expired early")
	}
	state.clearExpiredYank(time.Unix(10, 0))
	if state.yankActive || state.yankLinewise || !state.yankUntil.IsZero() {
		t.Fatalf("yank highlight still active: active=%v linewise=%v until=%v", state.yankActive, state.yankLinewise, state.yankUntil)
	}
}

func TestUIDiffViewVisualLineStatusMode(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "one"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 40, Height: 3})
	app.Pump(vui.Size{Width: 40, Height: 3})

	app.Send(vaxis.Key{Text: "V", Keycode: 'V'})
	app.Pump(vui.Size{Width: 40, Height: 3})
	p := vui.NewPainter(vui.Size{Width: 40, Height: 3})
	app.Paint(p)
	if got := uiDiffPainterText(p, 2); !strings.HasPrefix(got, " V-LINE ") {
		t.Fatalf("status bar = %q, want V-LINE mode", got)
	}
}

func TestUIDiffViewVisualCharDoesNotEnterLinewiseSelection(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "one"},
		{Kind: diff.RowContext, Gutter: "2 2   ", Code: "two"},
	}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 30, Height: 4})
	app.Pump(vui.Size{Width: 30, Height: 4})

	app.Send(vaxis.Key{Text: "v", Keycode: 'v'})
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Pump(vui.Size{Width: 30, Height: 4})
	p := vui.NewPainter(vui.Size{Width: 30, Height: 4})
	app.Paint(p)
	theme := uiDiffTestTheme()
	if got := p.Cell(29, 0).Background; got == theme.Selection {
		t.Fatal("visual character mode selected the full anchor row")
	}
	if got := uiDiffPainterText(p, 3); !strings.HasPrefix(got, " VISUAL ") {
		t.Fatalf("status bar = %q, want VISUAL mode", got)
	}
}

func TestUIDiffViewVisualCharSelectsSameLineRange(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "abcdef"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 20, Height: 3})
	app.Pump(vui.Size{Width: 20, Height: 3})

	app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	app.Send(vaxis.Key{Text: "v", Keycode: 'v'})
	app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	app.Pump(vui.Size{Width: 20, Height: 3})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 3})
	app.Paint(p)
	theme := uiDiffTestTheme()
	for col := 7; col <= 9; col++ {
		if col == 9 {
			continue
		}
		if got := p.Cell(col, 0).Background; got != theme.Selection {
			t.Fatalf("selected char col %d background = %v, want selection", col, got)
		}
	}
	if got := p.Cell(9, 0).Background; got != uiDiffCursorBackground(theme) {
		t.Fatalf("selected cursor background = %v, want cursor", got)
	}
	if got := p.Cell(6, 0).Background; got == theme.Selection {
		t.Fatal("character before selection is highlighted")
	}
	if got := p.Cell(10, 0).Background; got == theme.Selection {
		t.Fatal("character after selection is highlighted")
	}
}

func TestUIDiffViewVisualCharStillRendersCursor(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "abcdef"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 20, Height: 3})
	app.Pump(vui.Size{Width: 20, Height: 3})

	app.Send(vaxis.Key{Text: "v", Keycode: 'v'})
	app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	app.Pump(vui.Size{Width: 20, Height: 3})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 3})
	app.Paint(p)
	cell := p.Cell(7, 0)
	if cell.Grapheme != "b" {
		t.Fatalf("cursor grapheme = %q, want b", cell.Grapheme)
	}
	if cell.Background != uiDiffCursorBackground(uiDiffTestTheme()) {
		t.Fatalf("cursor background = %v, want cursor", cell.Background)
	}
	if cell.Foreground != uiDiffCursorForeground(uiDiffTestTheme()) {
		t.Fatalf("cursor foreground = %v, want cursor contrast", cell.Foreground)
	}
}

func TestUIDiffViewVisualCharTabCursorUsesLastTabCell(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "a\tb"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 20, Height: 3})
	app.Pump(vui.Size{Width: 20, Height: 3})

	app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	app.Send(vaxis.Key{Text: "v", Keycode: 'v'})
	app.Pump(vui.Size{Width: 20, Height: 3})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 3})
	app.Paint(p)
	for col := 7; col < 14; col++ {
		if got := p.Cell(col, 0).Background; got == uiDiffCursorBackground(uiDiffTestTheme()) {
			t.Fatalf("tab cursor rendered too early at col %d", col)
		}
	}
	if got := p.Cell(14, 0).Background; got != uiDiffCursorBackground(uiDiffTestTheme()) {
		t.Fatalf("tab cursor background = %v, want cursor on last tab cell", got)
	}
}

func TestUIDiffViewVisualCharSelectsPartialCrossLineRange(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "abcd"},
		{Kind: diff.RowContext, Gutter: "2 2   ", Code: "wxyz"},
	}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 20, Height: 4})
	app.Pump(vui.Size{Width: 20, Height: 4})

	app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	app.Send(vaxis.Key{Text: "v", Keycode: 'v'})
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	app.Pump(vui.Size{Width: 20, Height: 4})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 4})
	app.Paint(p)
	theme := uiDiffTestTheme()
	for col := 7; col <= 9; col++ {
		if got := p.Cell(col, 0).Background; got != theme.Selection {
			t.Fatalf("first row selected col %d background = %v, want selection", col, got)
		}
	}
	if got := p.Cell(6, 0).Background; got == theme.Selection {
		t.Fatal("first row character before selection is highlighted")
	}
	for col := 6; col <= 8; col++ {
		if col == 8 {
			continue
		}
		if got := p.Cell(col, 1).Background; got != theme.Selection {
			t.Fatalf("second row selected col %d background = %v, want selection", col, got)
		}
	}
	if got := p.Cell(8, 1).Background; got != uiDiffCursorBackground(theme) {
		t.Fatalf("second row cursor background = %v, want cursor", got)
	}
	if got := p.Cell(9, 1).Background; got == theme.Selection {
		t.Fatal("second row character after selection is highlighted")
	}
}

func TestUIDiffViewEscapeClearsLinewiseSelection(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "one"},
		{Kind: diff.RowContext, Gutter: "2 2   ", Code: "two"},
	}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 30, Height: 2})
	app.Pump(vui.Size{Width: 30, Height: 2})

	app.Send(vaxis.Key{Text: "V", Keycode: 'V'})
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Pump(vui.Size{Width: 30, Height: 2})
	p := vui.NewPainter(vui.Size{Width: 30, Height: 2})
	app.Paint(p)
	theme := uiDiffTestTheme()
	if got := p.Cell(29, 0).Background; got == theme.Selection {
		t.Fatal("anchor row still selected after escape")
	}
	if got := p.Cell(29, 1).Background; got != uiDiffCursorRowBackground(theme) {
		t.Fatalf("cursor row background = %v, want selection", got)
	}
}

func TestUIDiffViewLineBoundaryKeys(t *testing.T) {
	tests := []struct {
		name      string
		keys      []vaxis.Key
		wantCol   int
		wantGlyph string
	}{
		{
			name:      "l moves cursor right",
			keys:      []vaxis.Key{{Text: "l", Keycode: 'l'}},
			wantCol:   7,
			wantGlyph: "b",
		},
		{
			name:      "h moves cursor left",
			keys:      []vaxis.Key{{Text: "l", Keycode: 'l'}, {Text: "l", Keycode: 'l'}, {Text: "h", Keycode: 'h'}},
			wantCol:   7,
			wantGlyph: "b",
		},
		{
			name:      "0 moves to code start",
			keys:      []vaxis.Key{{Text: "l", Keycode: 'l'}, {Text: "l", Keycode: 'l'}, {Text: "0", Keycode: '0'}},
			wantCol:   6,
			wantGlyph: "a",
		},
		{
			name:      "$ moves to code end",
			keys:      []vaxis.Key{{Text: "$", Keycode: '$'}},
			wantCol:   11,
			wantGlyph: "f",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1   ", Marker: "+", Code: "abcdef"}}
			app := newUIDiffTestApp(rows, false)
			app.Pump(vui.Size{Width: 20, Height: 3})
			app.Pump(vui.Size{Width: 20, Height: 3})
			for _, key := range tt.keys {
				app.Send(key)
				app.Pump(vui.Size{Width: 20, Height: 3})
			}

			p := vui.NewPainter(vui.Size{Width: 20, Height: 3})
			app.Paint(p)
			cell := p.Cell(tt.wantCol, 0)
			if cell.Grapheme != tt.wantGlyph {
				t.Fatalf("cursor glyph = %q, want %q", cell.Grapheme, tt.wantGlyph)
			}
			if cell.Background != uiDiffCursorBackground(uiDiffTestTheme()) {
				t.Fatalf("cursor background = %v, want yank", cell.Background)
			}
			if cell.Foreground != uiDiffCursorForeground(uiDiffTestTheme()) {
				t.Fatalf("cursor foreground = %v, want contrast foreground", cell.Foreground)
			}
		})
	}
}

func TestUIDiffViewHorizontalMovementUsesTabStops(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "a\tb"}}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 20, Height: 3})
	app.Pump(vui.Size{Width: 20, Height: 3})

	app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	app.Pump(vui.Size{Width: 20, Height: 3})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 3})
	app.Paint(p)
	if got := p.Cell(7, 0).Background; got != uiDiffCursorBackground(uiDiffTestTheme()) {
		t.Fatalf("tab cursor first cell background = %v, want cursor", got)
	}
	for col := 8; col < 15; col++ {
		if got := p.Cell(col, 0).Background; got != uiDiffCursorRowBackground(uiDiffTestTheme()) {
			t.Fatalf("tab cursor remainder background at col %d = %v, want row selection", col, got)
		}
	}

	app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	app.Pump(vui.Size{Width: 20, Height: 3})
	p = vui.NewPainter(vui.Size{Width: 20, Height: 3})
	app.Paint(p)
	if cell := p.Cell(15, 0); cell.Grapheme != "b" || cell.Background != uiDiffCursorBackground(uiDiffTestTheme()) {
		t.Fatalf("cursor after tab = %q/%v, want b/yank", cell.Grapheme, cell.Background)
	}

	app.Send(vaxis.Key{Text: "h", Keycode: 'h'})
	app.Pump(vui.Size{Width: 20, Height: 3})
	p = vui.NewPainter(vui.Size{Width: 20, Height: 3})
	app.Paint(p)
	if got := p.Cell(7, 0).Background; got != uiDiffCursorBackground(uiDiffTestTheme()) {
		t.Fatalf("tab cursor after h first cell background = %v, want cursor", got)
	}
	for col := 8; col < 15; col++ {
		if got := p.Cell(col, 0).Background; got != uiDiffCursorRowBackground(uiDiffTestTheme()) {
			t.Fatalf("tab cursor after h remainder background at col %d = %v, want row selection", col, got)
		}
	}
}

func TestUIDiffViewHorizontalMovementStopsAtLineEnd(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "abc"}}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 20, Height: 3})
	app.Pump(vui.Size{Width: 20, Height: 3})

	for i := 0; i < 10; i++ {
		app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
		app.Pump(vui.Size{Width: 20, Height: 3})
	}

	p := vui.NewPainter(vui.Size{Width: 20, Height: 3})
	app.Paint(p)
	cell := p.Cell(8, 0)
	if cell.Grapheme != "c" || cell.Background != uiDiffCursorBackground(uiDiffTestTheme()) {
		t.Fatalf("cursor after repeated l = %q/%v, want c/yank", cell.Grapheme, cell.Background)
	}
	if got := p.Cell(9, 0).Background; got == uiDiffCursorBackground(uiDiffTestTheme()) {
		t.Fatal("cursor highlighted past end of line")
	}
}

func TestUIDiffViewJumpCommitScrollsTargetToTop(t *testing.T) {
	rows := make([]diff.Row, 30)
	for i := range rows {
		rows[i] = diff.Row{Kind: diff.RowContext, Gutter: "1 1   ", Code: "line"}
	}
	rows[0] = diff.Row{Kind: diff.RowCommitHeader, Text: "commit one"}
	rows[12] = diff.Row{Kind: diff.RowCommitHeader, Text: "commit two"}
	rows[28] = diff.Row{Kind: diff.RowCommitHeader, Text: "commit three"}
	app := newUIDiffTestApp(rows, false)
	app.Pump(vui.Size{Width: 20, Height: 4})
	app.Pump(vui.Size{Width: 20, Height: 4})

	app.Send(vaxis.Key{Text: "J", Keycode: 'J'})
	app.Pump(vui.Size{Width: 20, Height: 4})
	app.Pump(vui.Size{Width: 20, Height: 4})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 4})
	app.Paint(p)
	if got := uiDiffPainterText(p, 0); got != "commit two" {
		t.Fatalf("top row after J = %q, want commit two", got)
	}
	if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != 0 {
		t.Fatalf("highlight row after J = %d, want 0", got)
	}

	app.Send(vaxis.Key{Text: "J", Keycode: 'J'})
	app.Pump(vui.Size{Width: 20, Height: 4})
	app.Pump(vui.Size{Width: 20, Height: 4})
	p = vui.NewPainter(vui.Size{Width: 20, Height: 4})
	app.Paint(p)
	if got := uiDiffPainterText(p, 2); got != "commit three" {
		t.Fatalf("visible row after final J = %q, want commit three", got)
	}
	if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != 2 {
		t.Fatalf("highlight row after final J = %d, want 2", got)
	}

	app.Send(vaxis.Key{Text: "K", Keycode: 'K'})
	app.Pump(vui.Size{Width: 20, Height: 4})
	app.Pump(vui.Size{Width: 20, Height: 4})
	p = vui.NewPainter(vui.Size{Width: 20, Height: 4})
	app.Paint(p)
	if got := uiDiffPainterText(p, 0); got != "commit two" {
		t.Fatalf("top row after K = %q, want commit two", got)
	}
}

func uiDiffHighlightedScreenRow(p *vui.Painter, bg vaxis.Color) int {
	size := p.Size()
	for row := 0; row < size.Height; row++ {
		for col := 0; col < size.Width; col++ {
			if p.Cell(col, row).Background == bg {
				return row
			}
		}
	}
	return -1
}

func uiDiffPainterText(p *vui.Painter, row int) string {
	size := p.Size()
	text := ""
	for col := 0; col < size.Width; col++ {
		text += p.Cell(col, row).Grapheme
	}
	return strings.TrimRight(text, " ")
}

func uiDiffFindText(p *vui.Painter, text string) (int, int, bool) {
	size := p.Size()
	for row := 0; row < size.Height; row++ {
		if col, ok := uiDiffFindTextOnRow(p, row, text); ok {
			return col, row, true
		}
	}
	return 0, 0, false
}

func uiDiffFindTextOnRow(p *vui.Painter, row int, text string) (int, bool) {
	size := p.Size()
	line := ""
	for col := 0; col < size.Width; col++ {
		line += p.Cell(col, row).Grapheme
	}
	col := strings.Index(line, text)
	return col, col >= 0
}

func TestUIDiffViewAltPTogglesProfileOverlay(t *testing.T) {
	app := newUIDiffTestApp([]diff.Row{{Kind: diff.RowContext, Code: "line"}}, false)
	app.Pump(vui.Size{Width: 20, Height: 4})
	app.Send(vaxis.Key{Text: "p", Keycode: 'p', Modifiers: vaxis.ModAlt})
	if !app.ProfileOverlay() {
		t.Fatal("profile overlay not enabled")
	}
	app.Send(vaxis.Key{Text: "p", Keycode: 'p', Modifiers: vaxis.ModAlt})
	if app.ProfileOverlay() {
		t.Fatal("profile overlay not disabled")
	}
}

func TestUIDiffViewWrapsCodeThroughMeasuredRows(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "abcdefghij"},
		{Kind: diff.RowContext, Gutter: "2 2   ", Code: "next"},
	}
	app := newUIDiffTestApp(rows, true)
	app.Pump(vui.Size{Width: 11, Height: 4})
	app.Pump(vui.Size{Width: 11, Height: 4})

	p := vui.NewPainter(vui.Size{Width: 11, Height: 4})
	app.Paint(p)
	if got := p.Cell(6, 0).Grapheme; got != "a" {
		t.Fatalf("wrapped first line start = %q, want a", got)
	}
	if got := p.Cell(6, 1).Grapheme; got != "f" {
		t.Fatalf("wrapped second line start = %q, want f", got)
	}
	if got := p.Cell(6, 2).Grapheme; got != "n" {
		t.Fatalf("next row after wrapped row = %q, want n", got)
	}
}
