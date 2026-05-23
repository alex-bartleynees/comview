package tui

import (
	"errors"
	"os"
	"path/filepath"
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

func newUIDiffTestAppWithReviewFile(rows []diff.Row, drafts []review.CommentDraft, reviewFile string) *vui.App {
	return vui.NewApp(uiDiffRootWithReviewFile(rows, false, drafts, reviewFile, true), vui.WithTheme(uiDiffTestTheme()))
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

func TestUIDiffViewQuestionTogglesHelpOverlay(t *testing.T) {
	app := newUIDiffTestAppWithBaseDraftsAndStatus([]diff.Row{{Kind: diff.RowContext, Code: "line"}}, DefaultBaseColors(), false, nil, true)
	size := vui.Size{Width: 80, Height: len(helpKeybinds) + 6}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Key{Text: "/", Keycode: '/', Modifiers: vaxis.ModShift})
	app.Pump(size)
	p := vui.NewPainter(size)
	app.Paint(p)
	if _, _, ok := uiDiffFindText(p, "Keybinds"); !ok {
		t.Fatal("help overlay did not render")
	}
	if _, _, ok := uiDiffFindText(p, "Open cursor location in editor"); !ok {
		t.Fatal("help overlay did not include legacy keybinds")
	}
	if _, _, ok := uiDiffFindText(p, "╭"); ok {
		t.Fatal("help overlay rendered border chrome")
	}

	app.Send(vaxis.Key{Text: "?", Keycode: '?'})
	app.Pump(size)
	p = vui.NewPainter(size)
	app.Paint(p)
	if _, _, ok := uiDiffFindText(p, "Keybinds"); ok {
		t.Fatal("help overlay stayed visible after ?")
	}
}

func TestUIDiffViewHelpOverlayClosesWithEscapeAndQ(t *testing.T) {
	tests := []struct {
		name string
		key  vaxis.Key
	}{
		{name: "escape", key: vaxis.Key{Keycode: vaxis.KeyEsc}},
		{name: "q", key: vaxis.Key{Text: "q", Keycode: 'q'}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newUIDiffTestAppWithBaseDraftsAndStatus([]diff.Row{{Kind: diff.RowContext, Code: "line"}}, DefaultBaseColors(), false, nil, true)
			size := vui.Size{Width: 80, Height: len(helpKeybinds) + 6}
			app.Pump(size)
			app.Send(vaxis.Key{Text: "?", Keycode: '?'})
			app.Pump(size)
			app.Send(tt.key)
			app.Pump(size)

			p := vui.NewPainter(size)
			app.Paint(p)
			if _, _, ok := uiDiffFindText(p, "Keybinds"); ok {
				t.Fatal("help overlay stayed visible")
			}
		})
	}
}

func TestUIDiffViewThemePickerSelectsTheme(t *testing.T) {
	app := newUIDiffTestAppWithBaseDraftsAndStatus([]diff.Row{{Kind: diff.RowContext, Code: "line"}}, DefaultBaseColors(), false, nil, true)
	size := vui.Size{Width: 80, Height: 12}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Key{Text: "t", Keycode: 't'})
	app.Pump(size)
	p := vui.NewPainter(size)
	app.Paint(p)
	if _, _, ok := uiDiffFindText(p, "Choose theme"); !ok {
		t.Fatal("theme picker did not render")
	}

	app.Send(vaxis.Key{Text: "latte"})
	app.Pump(size)
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Pump(size)
	app.Pump(size)
	p = vui.NewPainter(size)
	app.Paint(p)
	if got := uiDiffPainterText(p, size.Height-1); !strings.Contains(got, "Theme: Catppuccin Latte") {
		t.Fatalf("status = %q, want selected theme", got)
	}
	selected := uiThemeFromBaseColors(Themes[2].Colors)
	if got := p.Cell(0, 0).Background; got != uiDiffCursorRowBackground(selected) {
		t.Fatalf("cursor row background = %v, want selected theme cursor background %v", got, uiDiffCursorRowBackground(selected))
	}
	if got := p.Cell(size.Width-1, size.Height-2).Background; got != selected.Background {
		t.Fatalf("blank area background = %v, want selected theme background %v", got, selected.Background)
	}
	for _, pt := range []vui.Point{{X: size.Width - 1, Y: 0}, {X: size.Width - 1, Y: size.Height - 2}} {
		if got := p.Cell(pt.X, pt.Y).Background; got != selected.Background {
			t.Fatalf("root background at %#v = %v, want selected theme background %v", pt, got, selected.Background)
		}
	}
}

func TestUIDiffViewThemePickerEscapeKeepsTheme(t *testing.T) {
	app := newUIDiffTestAppWithBaseDraftsAndStatus([]diff.Row{{Kind: diff.RowContext, Code: "line"}}, DefaultBaseColors(), false, nil, true)
	size := vui.Size{Width: 80, Height: 12}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Key{Text: "t", Keycode: 't'})
	app.Pump(size)
	app.Send(vaxis.Key{Text: "latte"})
	app.Pump(size)
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Pump(size)
	p := vui.NewPainter(size)
	app.Paint(p)
	if _, _, ok := uiDiffFindText(p, "Choose theme"); ok {
		t.Fatal("theme picker stayed visible after escape")
	}
	if got := p.Cell(0, 0).Background; got != uiDiffCursorRowBackground(uiDiffTestTheme()) {
		t.Fatalf("cursor row background = %v, want original theme", got)
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

func TestUIDiffViewHorizontalScrollKeepsGuttersFixed(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "abcdefghijklmnop"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	size := vui.Size{Width: 14, Height: 3}
	app.Pump(size)
	app.Pump(size)

	for i := 0; i < 9; i++ {
		app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
		app.Pump(size)
	}
	app.Pump(size)
	p := vui.NewPainter(size)
	app.Paint(p)
	if got := uiDiffPainterText(p, 0); !strings.HasPrefix(got, "1 1   ") {
		t.Fatalf("row = %q, want fixed gutter prefix", got)
	}
	if got := uiDiffPainterText(p, 0); !strings.Contains(got, "cdefghij") {
		t.Fatalf("row = %q, want horizontally scrolled code", got)
	}
}

func TestUIDiffViewDollarAndZeroAdjustHorizontalScroll(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "abcdefghijklmnop"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	size := vui.Size{Width: 14, Height: 3}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Key{Text: "$", Keycode: '$'})
	app.Pump(size)
	app.Pump(size)
	p := vui.NewPainter(size)
	app.Paint(p)
	if got := uiDiffPainterText(p, 0); !strings.Contains(got, "ijklmnop") {
		t.Fatalf("row after $ = %q, want line end visible", got)
	}

	app.Send(vaxis.Key{Text: "0", Keycode: '0'})
	app.Pump(size)
	app.Pump(size)
	p = vui.NewPainter(size)
	app.Paint(p)
	if got := uiDiffPainterText(p, 0); !strings.Contains(got, "abcdefgh") {
		t.Fatalf("row after 0 = %q, want line start visible", got)
	}
}

func TestUIDiffViewHorizontalMouseWheelScrollsCode(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "abcdefghijklmnop"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	size := vui.Size{Width: 14, Height: 3}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Mouse{Button: mouseWheelRight, EventType: vaxis.EventPress})
	app.Pump(size)
	p := vui.NewPainter(size)
	app.Paint(p)
	if got := uiDiffPainterText(p, 0); !strings.Contains(got, "bcdefghi") {
		t.Fatalf("row after right wheel = %q, want horizontally scrolled code", got)
	}

	app.Send(vaxis.Mouse{Button: mouseWheelLeft, EventType: vaxis.EventPress})
	app.Pump(size)
	p = vui.NewPainter(size)
	app.Paint(p)
	if got := uiDiffPainterText(p, 0); !strings.Contains(got, "abcdefgh") {
		t.Fatalf("row after left wheel = %q, want line start visible", got)
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

func TestUIDiffViewEditorTargetUsesCursorRow(t *testing.T) {
	rows, err := rowsForInput(`diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -10,2 +10,2 @@
 old context
-old
+new
`)
	if err != nil {
		t.Fatal(err)
	}
	target, ok := uiDiffEditorTarget(rows, selectionPoint{Row: 3, Col: 2})
	if !ok {
		t.Fatal("editor target not found")
	}
	if target.Path != "main.go" || target.Line != 11 || target.Column != 3 {
		t.Fatalf("target = %+v, want main.go:11:3", target)
	}
}

func TestUIDiffViewEditorTargetUsesTextColumnForTabs(t *testing.T) {
	row := diff.Row{
		Kind:     diff.RowAdd,
		FileName: "main.go",
		Code:     "\tfoo",
		Review:   review.Anchor{Line: 12},
	}
	tests := []struct {
		name       string
		cursorCell int
		wantColumn int
	}{
		{name: "inside tab", cursorCell: 4, wantColumn: 1},
		{name: "after tab", cursorCell: 8, wantColumn: 2},
		{name: "after first rune", cursorCell: 9, wantColumn: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, ok := uiDiffEditorTarget([]diff.Row{row}, selectionPoint{Row: 0, Col: tt.cursorCell})
			if !ok {
				t.Fatal("editor target not found")
			}
			if target.Column != tt.wantColumn {
				t.Fatalf("column = %d, want %d", target.Column, tt.wantColumn)
			}
		})
	}
}

func TestUIDiffViewEditorTargetUsesFourCellTabsForNonGoFiles(t *testing.T) {
	row := diff.Row{Kind: diff.RowAdd, FileName: "main.py", Code: "\tfoo", Review: review.Anchor{Line: 12}}
	target, ok := uiDiffEditorTarget([]diff.Row{row}, selectionPoint{Row: 0, Col: 4})
	if !ok {
		t.Fatal("editor target not found")
	}
	if target.Column != 2 {
		t.Fatalf("column = %d, want 2", target.Column)
	}
}

func TestUIDiffViewEditorTargetFallsBackToLineOne(t *testing.T) {
	target, ok := uiDiffEditorTarget([]diff.Row{{Kind: diff.RowFile, FileName: "main.go"}}, selectionPoint{})
	if !ok {
		t.Fatal("editor target not found")
	}
	if target.Path != "main.go" || target.Line != 1 || target.Column != 1 {
		t.Fatalf("target = %+v, want main.go:1:1", target)
	}
}

func TestUIDiffViewOReportsMissingFile(t *testing.T) {
	app := newUIDiffTestAppWithBaseDraftsAndStatus([]diff.Row{{Kind: diff.RowBlank}}, DefaultBaseColors(), false, nil, true)
	size := vui.Size{Width: 40, Height: 3}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Key{Text: "o", Keycode: 'o'})
	app.Pump(size)
	p := vui.NewPainter(size)
	app.Paint(p)
	if got := uiDiffPainterText(p, size.Height-1); !strings.Contains(got, "No file.") {
		t.Fatalf("status = %q, want no file", got)
	}
}

func TestUIDiffViewEditorTerminalReceivesCommandKeys(t *testing.T) {
	t.Setenv("GIT_EDITOR", "true")
	row := diff.Row{Kind: diff.RowAdd, FileName: "main.go", Code: "line", Review: review.Anchor{Line: 12}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus([]diff.Row{row}, DefaultBaseColors(), false, nil, true)
	size := vui.Size{Width: 40, Height: 5}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Key{Text: "o", Keycode: 'o'})
	app.Send(vaxis.Key{Text: ":", Keycode: ':'})
	app.Pump(size)
	p := vui.NewPainter(size)
	app.Paint(p)
	if got := uiDiffPainterText(p, size.Height-1); strings.HasPrefix(got, ":") {
		t.Fatalf("diff command mode intercepted terminal key: status = %q", got)
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

func TestUIDiffViewCommandModeUsesStatusBar(t *testing.T) {
	app := newUIDiffTestAppWithBaseDraftsAndStatus([]diff.Row{{Kind: diff.RowContext, Text: "line"}}, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 20, Height: 2})
	app.Pump(vui.Size{Width: 20, Height: 2})

	app.Send(vaxis.Key{Text: ":", Keycode: ':'})
	app.Send(vaxis.Key{Text: "wq"})
	app.Pump(vui.Size{Width: 20, Height: 2})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 2})
	app.Paint(p)
	if got := uiDiffPainterText(p, 1); got != ":wq" {
		t.Fatalf("command status = %q, want :wq", got)
	}
	cursor, ok := p.Cursor()
	if !ok {
		t.Fatal("command cursor was not rendered")
	}
	if cursor.Col != 3 || cursor.Row != 1 || cursor.Shape != vui.CursorBeam {
		t.Fatalf("command cursor = %+v, want beam at 3,1", cursor)
	}
}

func TestUIDiffViewCommandQQuits(t *testing.T) {
	app := newUIDiffTestAppWithBaseDraftsAndStatus([]diff.Row{{Kind: diff.RowContext, Text: "line"}}, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 20, Height: 2})

	app.Send(vaxis.Key{Text: ":", Keycode: ':'})
	app.Send(vaxis.Key{Text: "q"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if !app.ShouldQuit() {
		t.Fatal(":q did not quit")
	}
}

func TestUIDiffViewCommandQWarnsWithUnsavedComments(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "line", Review: review.Anchor{Path: "main.go", Line: 1, Side: review.SideRight}}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	size := vui.Size{Width: 60, Height: 6}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(size)
	app.Send(vaxis.Key{Text: "draft"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Pump(size)
	app.Send(vaxis.Key{Text: ":", Keycode: ':'})
	app.Send(vaxis.Key{Text: "q"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Pump(size)
	if app.ShouldQuit() {
		t.Fatal(":q quit with unsaved comment")
	}
	p := vui.NewPainter(size)
	app.Paint(p)
	if got := uiDiffPainterText(p, size.Height-1); !strings.Contains(got, "Unsaved comments") {
		t.Fatalf("status message = %q, want unsaved warning", got)
	}
}

func TestUIDiffViewCommandWWritesCommentsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".comview", "comments.json")
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "line", Review: review.Anchor{Path: "main.go", Line: 1, Side: review.SideRight}}}
	app := newUIDiffTestAppWithReviewFile(rows, nil, path)
	size := vui.Size{Width: 60, Height: 6}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(size)
	app.Send(vaxis.Key{Text: "saved"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Pump(size)
	app.Send(vaxis.Key{Text: ":", Keycode: ':'})
	app.Send(vaxis.Key{Text: "w"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Pump(size)
	file, err := review.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Comments) != 1 || file.Comments[0].Body != "saved" {
		t.Fatalf("comments file = %+v", file)
	}
	p := vui.NewPainter(size)
	app.Paint(p)
	if got := uiDiffPainterText(p, size.Height-1); !strings.Contains(got, "Comments saved.") {
		t.Fatalf("status message = %q, want save confirmation", got)
	}
}

func TestUIDiffViewCommandWWithoutCommentsShowsNoopStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".comview", "comments.json")
	app := newUIDiffTestAppWithReviewFile([]diff.Row{{Kind: diff.RowContext, Text: "line"}}, nil, path)
	size := vui.Size{Width: 60, Height: 3}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Key{Text: ":", Keycode: ':'})
	app.Send(vaxis.Key{Text: "w"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Pump(size)
	p := vui.NewPainter(size)
	app.Paint(p)
	if got := uiDiffPainterText(p, size.Height-1); !strings.Contains(got, "No comments to save.") {
		t.Fatalf("status message = %q, want no comments status", got)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("created comments file for no-op save")
	}
}

func TestUIDiffViewCommandWQWritesAndQuits(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".comview", "comments.json")
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "line", Review: review.Anchor{Path: "main.go", Line: 1, Side: review.SideRight}}}
	app := newUIDiffTestAppWithReviewFile(rows, nil, path)
	app.Pump(vui.Size{Width: 60, Height: 6})

	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 60, Height: 6})
	app.Send(vaxis.Key{Text: "saved"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Pump(vui.Size{Width: 60, Height: 6})
	app.Send(vaxis.Key{Text: ":", Keycode: ':'})
	app.Send(vaxis.Key{Text: "wq"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if !app.ShouldQuit() {
		t.Fatal(":wq did not quit")
	}
	file, err := review.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Comments) != 1 || file.Comments[0].Body != "saved" {
		t.Fatalf("comments file = %+v", file)
	}
}

func TestUIDiffViewCommandQBangQuitsWithUnsavedComments(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "line", Review: review.Anchor{Path: "main.go", Line: 1, Side: review.SideRight}}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 60, Height: 6})

	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 60, Height: 6})
	app.Send(vaxis.Key{Text: "draft"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Pump(vui.Size{Width: 60, Height: 6})
	app.Send(vaxis.Key{Text: ":", Keycode: ':'})
	app.Send(vaxis.Key{Text: "q!"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if !app.ShouldQuit() {
		t.Fatal(":q! did not quit")
	}
}

func TestUIDiffViewXDeletesNoteAtCursor(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "line", Review: review.Anchor{Path: "main.go", Line: 12, Side: review.SideRight}}}
	drafts := []review.CommentDraft{{Path: "main.go", Line: 12, Side: review.SideRight, Body: "comment"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, drafts, true)
	size := vui.Size{Width: 60, Height: 6}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Key{Text: "x", Keycode: 'x'})
	app.Pump(size)
	p := vui.NewPainter(size)
	app.Paint(p)
	if uiDiffPainterRowContaining(p, "comment") != -1 {
		t.Fatal("deleted note is still rendered")
	}
	if got := uiDiffPainterText(p, size.Height-1); !strings.Contains(got, "Note deleted.") {
		t.Fatalf("status message = %q, want note deleted", got)
	}

	path := filepath.Join(t.TempDir(), ".comview", "comments.json")
	app = newUIDiffTestAppWithReviewFile(rows, drafts, path)
	app.Pump(size)
	app.Pump(size)
	app.Send(vaxis.Key{Text: "x", Keycode: 'x'})
	app.Send(vaxis.Key{Text: ":", Keycode: ':'})
	app.Send(vaxis.Key{Text: "w"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	file, err := review.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Comments) != 0 {
		t.Fatalf("saved comments = %+v, want none", file.Comments)
	}
}

func TestUIDiffViewDDDeletesNoteAtCursor(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "line", Review: review.Anchor{Path: "main.go", Line: 12, Side: review.SideRight}}}
	drafts := []review.CommentDraft{{Path: "main.go", Line: 12, Side: review.SideRight, Body: "comment"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, drafts, true)
	size := vui.Size{Width: 60, Height: 6}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Key{Text: "d", Keycode: 'd'})
	app.Pump(size)
	p := vui.NewPainter(size)
	app.Paint(p)
	if uiDiffPainterRowContaining(p, "comment") == -1 {
		t.Fatal("first d deleted note")
	}
	app.Send(vaxis.Key{Text: "d", Keycode: 'd'})
	app.Pump(size)
	p = vui.NewPainter(size)
	app.Paint(p)
	if uiDiffPainterRowContaining(p, "comment") != -1 {
		t.Fatal("dd did not delete note")
	}
}

func TestUIDiffViewXDeletesNoteOverlappingSelection(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "hello", Review: review.Anchor{Path: "main.go", Line: 12, Side: review.SideRight}}}
	drafts := []review.CommentDraft{{Path: "main.go", Line: 12, Side: review.SideRight, StartColumn: intPtr(2), EndColumn: intPtr(4), Body: "inline"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, drafts, true)
	size := vui.Size{Width: 60, Height: 6}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Key{Text: "v", Keycode: 'v'})
	app.Send(vaxis.Key{Text: "x", Keycode: 'x'})
	app.Pump(size)
	p := vui.NewPainter(size)
	app.Paint(p)
	if uiDiffPainterRowContaining(p, "inline") != -1 {
		t.Fatal("selection-overlapping note is still rendered")
	}
	if got := uiDiffPainterText(p, size.Height-1); !strings.Contains(got, "Note deleted.") {
		t.Fatalf("status message = %q, want note deleted", got)
	}
}

func TestUIDiffViewXShowsMessageWithoutNote(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "line", Review: review.Anchor{Path: "main.go", Line: 12, Side: review.SideRight}}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	size := vui.Size{Width: 60, Height: 4}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Key{Text: "x", Keycode: 'x'})
	app.Pump(size)
	p := vui.NewPainter(size)
	app.Paint(p)
	if got := uiDiffPainterText(p, size.Height-1); !strings.Contains(got, "No note.") {
		t.Fatalf("status message = %q, want no note", got)
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
	if got := p.Cell(3, 1).Background; got == uiDiffSearchHighlightStyle(theme).Background {
		t.Fatalf("hunk metadata was highlighted by search")
	}
}

func TestUIDiffViewSearchesStructuredRowsExceptHunks(t *testing.T) {
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
	if got := uiDiffHighlightedScreenRow(p, uiDiffCursorRowBackground(uiDiffTestTheme())); got != 0 {
		t.Fatalf("hunk search highlight row = %d, want unchanged cursor row 0", got)
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

func TestUIDiffViewTextObjectSelectsInnerWord(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Code: "foo bar.baz", Text: "foo bar.baz"}}
	state := &uiDiffViewState{
		cursor: selectionPoint{Row: 0, Col: 5},
	}
	if !state.selectWordTextObject(rows, textObjectInner) {
		t.Fatal("inner word text object failed")
	}
	if got, want := state.selectionText(rows), "bar"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
	if got, want := state.cursor, (selectionPoint{Row: 0, Col: 6}); got != want {
		t.Fatalf("cursor = %+v, want %+v", got, want)
	}
}

func TestUIDiffViewTextObjectSelectsAroundWord(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Code: "foo bar baz", Text: "foo bar baz"}}
	state := &uiDiffViewState{
		cursor: selectionPoint{Row: 0, Col: 5},
	}
	if !state.selectWordTextObject(rows, textObjectAround) {
		t.Fatal("around word text object failed")
	}
	if got, want := state.selectionText(rows), "bar "; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestUIDiffViewTextObjectSelectsPunctuationToken(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Code: "foo bar.baz", Text: "foo bar.baz"}}
	state := &uiDiffViewState{
		cursor: selectionPoint{Row: 0, Col: 7},
	}
	if !state.selectWordTextObject(rows, textObjectInner) {
		t.Fatal("punctuation token text object failed")
	}
	if got, want := state.selectionText(rows), "."; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestUIDiffViewTextObjectKeysSelectInnerWord(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "foo bar.baz"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 30, Height: 3})
	app.Pump(vui.Size{Width: 30, Height: 3})

	for i := 0; i < 5; i++ {
		app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	}
	app.Send(vaxis.Key{Text: "v", Keycode: 'v'})
	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Send(vaxis.Key{Text: "w", Keycode: 'w'})
	app.Pump(vui.Size{Width: 30, Height: 3})

	p := vui.NewPainter(vui.Size{Width: 30, Height: 3})
	app.Paint(p)
	codeOffset := uiDiffCodeOffset(rows)
	theme := uiDiffTestTheme()
	for col := codeOffset + 4; col <= codeOffset+6; col++ {
		if got := p.Cell(col, 0).Background; got != theme.Selection && got != uiDiffCursorBackground(theme) {
			t.Fatalf("word cell %d background = %v, want selected/cursor", col, got)
		}
	}
	if got := p.Cell(codeOffset+7, 0).Background; got == theme.Selection {
		t.Fatal("punctuation after word was selected")
	}
}

func TestUIDiffViewTextObjectIgnoresShiftModifierPress(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "foo bar.baz"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 30, Height: 3})
	app.Pump(vui.Size{Width: 30, Height: 3})

	for i := 0; i < 5; i++ {
		app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	}
	app.Send(vaxis.Key{Text: "v", Keycode: 'v'})
	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Send(vaxis.Key{Keycode: vaxis.KeyLeftShift})
	app.Send(vaxis.Key{Text: "w", Keycode: 'w'})
	app.Pump(vui.Size{Width: 30, Height: 3})

	p := vui.NewPainter(vui.Size{Width: 30, Height: 3})
	app.Paint(p)
	codeOffset := uiDiffCodeOffset(rows)
	theme := uiDiffTestTheme()
	for col := codeOffset + 4; col <= codeOffset+6; col++ {
		if got := p.Cell(col, 0).Background; got != theme.Selection && got != uiDiffCursorBackground(theme) {
			t.Fatalf("word cell %d background = %v, want selected/cursor", col, got)
		}
	}
}

func TestUIDiffViewTextObjectUsesShiftedPunctuation(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "call(foo)"}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 30, Height: 3})
	app.Pump(vui.Size{Width: 30, Height: 3})

	for i := 0; i < strings.Index(rows[0].Code, "foo"); i++ {
		app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	}
	app.Send(vaxis.Key{Text: "v", Keycode: 'v'})
	app.Send(vaxis.Key{Text: "a", Keycode: 'a'})
	app.Send(vaxis.Key{Text: ")", Keycode: '0', Modifiers: vaxis.ModShift})
	app.Pump(vui.Size{Width: 30, Height: 3})

	p := vui.NewPainter(vui.Size{Width: 30, Height: 3})
	app.Paint(p)
	codeOffset := uiDiffCodeOffset(rows)
	theme := uiDiffTestTheme()
	if got := p.Cell(codeOffset+4, 0).Background; got != theme.Selection {
		t.Fatalf("opening paren background = %v, want selection", got)
	}
	if got := p.Cell(codeOffset+8, 0).Background; got != uiDiffCursorBackground(theme) {
		t.Fatalf("closing paren cursor background = %v, want cursor", got)
	}
	if got := p.Cell(codeOffset, 0).Background; got == uiDiffCursorBackground(theme) {
		t.Fatal("shifted punctuation fell through to 0 cursor movement")
	}
}

func TestUIDiffViewTextObjectSelectsMultilineBraces(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Code: "func main() {", Text: "func main() {"},
		{Kind: diff.RowContext, Code: "\tcall()", Text: "\tcall()"},
		{Kind: diff.RowContext, Code: "}", Text: "}"},
	}
	state := &uiDiffViewState{cursor: selectionPoint{Row: 1, Col: 2}}
	open, close, ok := textObjectDelimiters('{')
	if !ok {
		t.Fatal("brace delimiter missing")
	}
	if !state.selectDelimitedTextObject(rows, textObjectAround, open, close) {
		t.Fatal("around brace text object failed")
	}
	if got, want := state.selectionText(rows), "{\n\tcall()\n}"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestUIDiffViewInnerTextObjectIncludesBoundaryNewlines(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Code: "foo{", Text: "foo{"},
		{Kind: diff.RowContext, Code: "  foo,", Text: "  foo,"},
		{Kind: diff.RowContext, Code: "}", Text: "}"},
	}
	state := &uiDiffViewState{cursor: selectionPoint{Row: 1, Col: 0}}
	open, close, ok := textObjectDelimiters('{')
	if !ok {
		t.Fatal("brace delimiter missing")
	}
	if !state.selectDelimitedTextObject(rows, textObjectInner, open, close) {
		t.Fatal("inner brace text object failed")
	}
	if got, want := state.selectionText(rows), "\n  foo,\n"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestUIDiffViewTextObjectStopsAtHunkBoundary(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Code: "func main() {", Text: "func main() {"},
		{Kind: diff.RowContext, Code: "\tcall()", Text: "\tcall()"},
		{Kind: diff.RowContext, Code: "}", Text: "}"},
		{Kind: diff.RowHunk, Text: "@@ -10 +10 @@"},
		{Kind: diff.RowContext, Code: "other {}", Text: "other {}"},
	}
	state := &uiDiffViewState{cursor: selectionPoint{Row: 1, Col: 2}}
	open, close, ok := textObjectDelimiters('{')
	if !ok {
		t.Fatal("brace delimiter missing")
	}
	if !state.selectDelimitedTextObject(rows, textObjectAround, open, close) {
		t.Fatal("around brace text object failed")
	}
	if got, want := state.selectionText(rows), "{\n\tcall()\n}"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestUIDiffViewTextObjectSkipsOppositeSideRows(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Code: "if ok {", Text: "if ok {", FileName: "main.go"},
		{Kind: diff.RowDelete, Code: "old()", Text: "old()", FileName: "main.go"},
		{Kind: diff.RowAdd, Code: "new()", Text: "new()", FileName: "main.go"},
		{Kind: diff.RowContext, Code: "}", Text: "}", FileName: "main.go"},
	}
	state := &uiDiffViewState{cursor: selectionPoint{Row: 2, Col: 0}}
	open, close, ok := textObjectDelimiters('{')
	if !ok {
		t.Fatal("brace delimiter missing")
	}
	if !state.selectDelimitedTextObject(rows, textObjectAround, open, close) {
		t.Fatal("around brace text object failed")
	}
	if got, want := state.selectionText(rows), "{\nnew()\n}"; got != want {
		t.Fatalf("selection text = %q, want %q", got, want)
	}
}

func TestUIDiffViewOpensCommentEditor(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "new", Review: review.Anchor{Path: "main.go", Line: 12, Side: review.SideRight}}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 40, Height: 6})
	app.Pump(vui.Size{Width: 40, Height: 6})

	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 40, Height: 6})
	app.Pump(vui.Size{Width: 40, Height: 6})
	p := vui.NewPainter(vui.Size{Width: 40, Height: 6})
	app.Paint(p)
	if got := uiDiffPainterText(p, 1); !strings.Contains(got, "▄") {
		t.Fatalf("editor row = %q, want top half-block padding", got)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 2 || cursor.Row != 2 || cursor.Shape != vui.CursorBeam {
		t.Fatalf("cursor = %+v, want editor row beam cursor after two left padding cells", cursor)
	}
	if got := p.Cell(0, 0).Background; got == uiDiffCursorRowBackground(uiDiffTestTheme()) {
		t.Fatal("diff cursor row is highlighted while textarea is focused")
	}
	if got := uiDiffPainterText(p, 5); !strings.HasPrefix(got, " INSERT ") {
		t.Fatalf("status bar = %q, want INSERT", got)
	}
}

func TestUIDiffViewCommentEditorEscapeReturnsToNormal(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "new", Review: review.Anchor{Path: "main.go", Line: 12, Side: review.SideRight}}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 40, Height: 6})
	app.Pump(vui.Size{Width: 40, Height: 6})

	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 40, Height: 6})
	app.Send(vaxis.Key{Text: "draft"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Pump(vui.Size{Width: 40, Height: 6})
	p := vui.NewPainter(vui.Size{Width: 40, Height: 6})
	app.Paint(p)
	if got := uiDiffPainterText(p, 2); !strings.Contains(got, "draft") {
		t.Fatalf("editor body after escape = %q, want draft", got)
	}
	if got := uiDiffPainterText(p, 5); !strings.HasPrefix(got, " NORMAL ") {
		t.Fatalf("status bar = %q, want NORMAL", got)
	}
	if _, ok := p.Cursor(); ok {
		t.Fatal("textarea cursor is still visible after escape returned focus to diff")
	}
	if got := p.Cell(0, 0).Background; got != uiDiffCursorRowBackground(uiDiffTestTheme()) {
		t.Fatalf("diff cursor row background = %v, want cursor row", got)
	}
}

func TestUIDiffViewCommentEditorEscapeClosesEmptyEditor(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "new", Review: review.Anchor{Path: "main.go", Line: 12, Side: review.SideRight}}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 40, Height: 6})
	app.Pump(vui.Size{Width: 40, Height: 6})

	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 40, Height: 6})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Pump(vui.Size{Width: 40, Height: 6})
	p := vui.NewPainter(vui.Size{Width: 40, Height: 6})
	app.Paint(p)
	if got := uiDiffPainterText(p, 1); strings.Contains(got, "▄") || strings.Contains(got, "Add comment") {
		t.Fatalf("editor row = %q, want empty editor closed", got)
	}
	if _, ok := p.Cursor(); ok {
		t.Fatal("textarea cursor is still visible after empty editor closed")
	}
	if got := p.Cell(0, 0).Background; got != uiDiffCursorRowBackground(uiDiffTestTheme()) {
		t.Fatalf("diff cursor row background = %v, want cursor row", got)
	}
}

func TestUIDiffViewCommentEditorCanCursorInAndOut(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "one", Review: review.Anchor{Path: "main.go", Line: 1, Side: review.SideRight}},
		{Kind: diff.RowAdd, Gutter: "2 2 + ", Code: "two", Review: review.Anchor{Path: "main.go", Line: 2, Side: review.SideRight}},
	}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 40, Height: 8})
	app.Pump(vui.Size{Width: 40, Height: 8})

	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 40, Height: 8})
	app.Send(vaxis.Key{Text: "draft"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Pump(vui.Size{Width: 40, Height: 8})

	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Pump(vui.Size{Width: 40, Height: 8})
	p := vui.NewPainter(vui.Size{Width: 40, Height: 8})
	app.Paint(p)
	if got := p.Cell(39, 0).Background; got == uiDiffCursorRowBackground(uiDiffTestTheme()) {
		t.Fatal("diff row stayed highlighted after cursoring into comment box")
	}
	if got := p.Cell(0, 2).Background; got != uiDiffTestTheme().SurfaceHovered {
		t.Fatalf("focused comment body background = %v, want hovered surface", got)
	}

	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Pump(vui.Size{Width: 40, Height: 8})
	p = vui.NewPainter(vui.Size{Width: 40, Height: 8})
	app.Paint(p)
	secondRow := uiDiffPainterRowContaining(p, "two")
	if secondRow == -1 {
		t.Fatal("second diff row was not rendered")
	}
	if got := p.Cell(39, secondRow).Background; got != uiDiffCursorRowBackground(uiDiffTestTheme()) {
		t.Fatalf("second diff row background = %v, want cursor row after moving out of comment", got)
	}

	app.Send(vaxis.Key{Text: "k", Keycode: 'k'})
	app.Send(vaxis.Key{Text: "k", Keycode: 'k'})
	app.Pump(vui.Size{Width: 40, Height: 8})
	p = vui.NewPainter(vui.Size{Width: 40, Height: 8})
	app.Paint(p)
	if got := p.Cell(39, 0).Background; got != uiDiffCursorRowBackground(uiDiffTestTheme()) {
		t.Fatalf("first diff row background = %v, want cursor row after moving back out of comment", got)
	}
}

func uiDiffPainterRowContaining(p *vui.Painter, text string) int {
	for row := 0; row < p.Size().Height; row++ {
		if strings.Contains(uiDiffPainterText(p, row), text) {
			return row
		}
	}
	return -1
}

func TestUIDiffViewCommentEditorEscapeKeepsEditorOpen(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "new", Review: review.Anchor{Path: "main.go", Line: 12, Side: review.SideRight}}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 40, Height: 6})
	app.Pump(vui.Size{Width: 40, Height: 6})

	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 40, Height: 6})
	app.Send(vaxis.Key{Text: "draft"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Pump(vui.Size{Width: 40, Height: 6})
	p := vui.NewPainter(vui.Size{Width: 40, Height: 6})
	app.Paint(p)
	if got := uiDiffPainterText(p, 1); !strings.Contains(got, "▄") {
		t.Fatalf("editor row = %q, want editor to remain open", got)
	}
	if got := uiDiffPainterText(p, 2); !strings.Contains(got, "draft") {
		t.Fatalf("editor body = %q, want draft", got)
	}
	if got := p.Cell(0, 2).Background; got != uiDiffTestTheme().Surface {
		t.Fatalf("editor body left edge background = %v, want full-width comment surface", got)
	}
	if got := p.Cell(1, 2).Background; got != uiDiffTestTheme().Surface {
		t.Fatalf("editor body second padding background = %v, want full-width comment surface", got)
	}
}

func TestUIDiffViewCommentEditorNormalIReentersInsert(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "new", Review: review.Anchor{Path: "main.go", Line: 12, Side: review.SideRight}}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 40, Height: 6})
	app.Pump(vui.Size{Width: 40, Height: 6})

	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 40, Height: 6})
	app.Send(vaxis.Key{Text: "a"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 40, Height: 6})
	app.Send(vaxis.Key{Text: "b"})
	app.Send(vaxis.Key{Text: "s", Keycode: 's', Modifiers: vaxis.ModCtrl})
	app.Pump(vui.Size{Width: 40, Height: 6})
	p := vui.NewPainter(vui.Size{Width: 40, Height: 6})
	app.Paint(p)
	if got := uiDiffPainterText(p, 2); !strings.Contains(got, "ab") {
		t.Fatalf("draft row = %q, want edited body", got)
	}
}

func TestUIDiffViewCommentEditorSubmitCreatesDraft(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "new", Review: review.Anchor{Path: "main.go", Line: 12, Side: review.SideRight}}}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 40, Height: 6})
	app.Pump(vui.Size{Width: 40, Height: 6})

	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 40, Height: 6})
	app.Send(vaxis.Key{Text: "looks good"})
	app.Send(vaxis.Key{Text: "s", Keycode: 's', Modifiers: vaxis.ModCtrl})
	app.Pump(vui.Size{Width: 40, Height: 6})
	p := vui.NewPainter(vui.Size{Width: 40, Height: 6})
	app.Paint(p)
	if got := uiDiffPainterText(p, 2); !strings.Contains(got, "looks good") {
		t.Fatalf("draft row = %q, want submitted body", got)
	}
	if got := uiDiffPainterText(p, 5); !strings.HasPrefix(got, " NORMAL ") {
		t.Fatalf("status bar = %q, want NORMAL", got)
	}
}

func TestUIDiffViewVisualLineIOpensCommentEditor(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".comview", "comments.json")
	rows := []diff.Row{
		{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "first", Review: review.Anchor{Path: "main.go", Line: 1, Side: review.SideRight}},
		{Kind: diff.RowAdd, Gutter: "2 2 + ", Code: "second", Review: review.Anchor{Path: "main.go", Line: 2, Side: review.SideRight}},
	}
	app := newUIDiffTestAppWithReviewFile(rows, nil, path)
	size := vui.Size{Width: 80, Height: 8}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Key{Text: "V", Keycode: 'V'})
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Send(vaxis.Key{Text: "I", Keycode: 'I'})
	app.Pump(size)
	p := vui.NewPainter(size)
	app.Paint(p)
	if got := uiDiffPainterText(p, size.Height-1); !strings.HasPrefix(got, " INSERT ") {
		t.Fatalf("status = %q, want insert", got)
	}

	app.Send(vaxis.Key{Text: "range"})
	app.Send(vaxis.Key{Text: "s", Keycode: 's', Modifiers: vaxis.ModCtrl})
	app.Send(vaxis.Key{Text: ":", Keycode: ':'})
	app.Send(vaxis.Key{Text: "w"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	file, err := review.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Comments) != 1 {
		t.Fatalf("comments = %+v, want one", file.Comments)
	}
	draft := file.Comments[0]
	if draft.Body != "range" || draft.StartLine != 1 || draft.Line != 2 || draft.StartColumn != nil || draft.EndColumn != nil {
		t.Fatalf("draft = %+v, want line range 1-2 without columns", draft)
	}
}

func TestUIDiffViewVisualIOpensColumnCommentEditor(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".comview", "comments.json")
	rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "hello", Review: review.Anchor{Path: "main.go", Line: 12, Side: review.SideRight}}}
	app := newUIDiffTestAppWithReviewFile(rows, nil, path)
	size := vui.Size{Width: 80, Height: 7}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	app.Send(vaxis.Key{Text: "v", Keycode: 'v'})
	app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	app.Send(vaxis.Key{Text: "I", Keycode: 'I'})
	app.Pump(size)
	p := vui.NewPainter(size)
	app.Paint(p)
	if got := uiDiffPainterText(p, size.Height-1); !strings.HasPrefix(got, " INSERT ") {
		t.Fatalf("status = %q, want insert", got)
	}

	app.Send(vaxis.Key{Text: "inline"})
	app.Send(vaxis.Key{Text: "s", Keycode: 's', Modifiers: vaxis.ModCtrl})
	app.Send(vaxis.Key{Text: ":", Keycode: ':'})
	app.Send(vaxis.Key{Text: "w"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	file, err := review.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Comments) != 1 {
		t.Fatalf("comments = %+v, want one", file.Comments)
	}
	draft := file.Comments[0]
	if draft.Body != "inline" || draft.Line != 12 || draft.StartLine != 0 || draft.StartColumn == nil || draft.EndColumn == nil || *draft.StartColumn != 2 || *draft.EndColumn != 3 {
		t.Fatalf("draft = %+v, want columns 2-3", draft)
	}
}

func TestUIDiffViewOpeningSecondCommentKeepsFirstDraft(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "one", Review: review.Anchor{Path: "main.go", Line: 1, Side: review.SideRight}},
		{Kind: diff.RowAdd, Gutter: "2 2 + ", Code: "two", Review: review.Anchor{Path: "main.go", Line: 2, Side: review.SideRight}},
	}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 80, Height: 12})
	app.Pump(vui.Size{Width: 80, Height: 12})

	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 80, Height: 12})
	app.Send(vaxis.Key{Text: "first"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Pump(vui.Size{Width: 80, Height: 12})
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 80, Height: 12})
	p := vui.NewPainter(vui.Size{Width: 80, Height: 12})
	app.Paint(p)
	if uiDiffPainterRowContaining(p, "first") == -1 {
		t.Fatal("first draft disappeared after opening second comment editor")
	}
	if uiDiffPainterRowContaining(p, "Add comment") == -1 {
		t.Fatal("second comment editor was not opened")
	}
}

func TestUIDiffViewCursorUpStopsAtAdjacentStoredComments(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "one", Review: review.Anchor{Path: "main.go", Line: 1, Side: review.SideRight}},
		{Kind: diff.RowAdd, Gutter: "2 2 + ", Code: "two", Review: review.Anchor{Path: "main.go", Line: 2, Side: review.SideRight}},
		{Kind: diff.RowAdd, Gutter: "3 3 + ", Code: "three", Review: review.Anchor{Path: "main.go", Line: 3, Side: review.SideRight}},
	}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, false)
	size := vui.Size{Width: 80, Height: 12}
	app.Pump(size)
	app.Pump(size)

	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(size)
	app.Send(vaxis.Key{Text: "comment 1"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Pump(size)
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Pump(size)
	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(size)
	app.Send(vaxis.Key{Text: "comment 2"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Pump(size)
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Pump(size)

	for range 3 {
		app.Send(vaxis.Key{Text: "k", Keycode: 'k'})
		app.Pump(size)
	}
	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(size)
	app.Send(vaxis.Key{Text: "!"})
	app.Pump(size)
	p := vui.NewPainter(size)
	app.Paint(p)
	if uiDiffPainterRowContaining(p, "!comment 1") == -1 {
		t.Fatal("cursor skipped first adjacent comment while moving up")
	}
}

func TestUIDiffViewCursorStopsAtSubmittedDraftComments(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "one", Review: review.Anchor{Path: "main.go", Line: 1, Side: review.SideRight}},
		{Kind: diff.RowAdd, Gutter: "2 2 + ", Code: "two", Review: review.Anchor{Path: "main.go", Line: 2, Side: review.SideRight}},
		{Kind: diff.RowAdd, Gutter: "3 3 + ", Code: "three", Review: review.Anchor{Path: "main.go", Line: 3, Side: review.SideRight}},
	}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 80, Height: 14})
	app.Pump(vui.Size{Width: 80, Height: 14})

	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 80, Height: 14})
	app.Send(vaxis.Key{Text: "submitted 1"})
	app.Send(vaxis.Key{Text: "s", Keycode: 's', Modifiers: vaxis.ModCtrl})
	app.Pump(vui.Size{Width: 80, Height: 14})
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 80, Height: 14})
	app.Send(vaxis.Key{Text: "submitted 2"})
	app.Send(vaxis.Key{Text: "s", Keycode: 's', Modifiers: vaxis.ModCtrl})
	app.Pump(vui.Size{Width: 80, Height: 14})
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Pump(vui.Size{Width: 80, Height: 14})

	app.Send(vaxis.Key{Text: "k", Keycode: 'k'})
	app.Send(vaxis.Key{Text: "k", Keycode: 'k'})
	app.Send(vaxis.Key{Text: "k", Keycode: 'k'})
	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 80, Height: 14})
	app.Send(vaxis.Key{Text: "!"})
	app.Pump(vui.Size{Width: 80, Height: 14})
	p := vui.NewPainter(vui.Size{Width: 80, Height: 14})
	app.Paint(p)
	if uiDiffPainterRowContaining(p, "!submitted 1") == -1 {
		t.Fatal("moving up through submitted draft comments skipped the first draft")
	}
}

func TestUIDiffViewCursorStopsAtProvidedDraftComments(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "one", Review: review.Anchor{Path: "main.go", Line: 1, Side: review.SideRight}},
		{Kind: diff.RowAdd, Gutter: "2 2 + ", Code: "two", Review: review.Anchor{Path: "main.go", Line: 2, Side: review.SideRight}},
		{Kind: diff.RowAdd, Gutter: "3 3 + ", Code: "three", Review: review.Anchor{Path: "main.go", Line: 3, Side: review.SideRight}},
	}
	drafts := []review.CommentDraft{
		{Path: "main.go", Line: 1, Side: review.SideRight, Body: "provided 1"},
		{Path: "main.go", Line: 2, Side: review.SideRight, Body: "provided 2"},
	}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, drafts, true)
	app.Pump(vui.Size{Width: 80, Height: 14})
	app.Pump(vui.Size{Width: 80, Height: 14})

	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Pump(vui.Size{Width: 80, Height: 14})
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Pump(vui.Size{Width: 80, Height: 14})
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Pump(vui.Size{Width: 80, Height: 14})
	app.Send(vaxis.Key{Text: "j", Keycode: 'j'})
	app.Pump(vui.Size{Width: 80, Height: 14})
	app.Send(vaxis.Key{Text: "k", Keycode: 'k'})
	app.Pump(vui.Size{Width: 80, Height: 14})
	p := vui.NewPainter(vui.Size{Width: 80, Height: 14})
	app.Paint(p)
	if uiDiffPainterRowContaining(p, "provided 2") == -1 {
		t.Fatal("first k from code 3 did not show provided draft 2")
	}
	app.Send(vaxis.Key{Text: "k", Keycode: 'k'})
	app.Pump(vui.Size{Width: 80, Height: 14})
	p = vui.NewPainter(vui.Size{Width: 80, Height: 14})
	app.Paint(p)
	code2Row := uiDiffPainterRowContaining(p, "2 2 + two")
	if code2Row == -1 || p.Cell(79, code2Row).Background != uiDiffCursorRowBackground(uiDiffTestTheme()) {
		t.Fatal("second k from code 3 did not hover code 2")
	}
	app.Send(vaxis.Key{Text: "k", Keycode: 'k'})
	app.Pump(vui.Size{Width: 80, Height: 14})
	p = vui.NewPainter(vui.Size{Width: 80, Height: 14})
	app.Paint(p)
	if uiDiffPainterRowContaining(p, "provided 1") == -1 {
		t.Fatal("third k from code 3 did not show provided draft 1")
	}
	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 80, Height: 14})
	app.Send(vaxis.Key{Text: "!"})
	app.Pump(vui.Size{Width: 80, Height: 14})
	p = vui.NewPainter(vui.Size{Width: 80, Height: 14})
	app.Paint(p)
	if uiDiffPainterRowContaining(p, "!provided 1") == -1 {
		t.Fatal("moving up through provided draft comments skipped the first draft")
	}
}

func TestUIDiffViewMouseSelectionAccountsForCommentRows(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowAdd, Gutter: "1 1 + ", Code: "one", Review: review.Anchor{Path: "main.go", Line: 1, Side: review.SideRight}},
		{Kind: diff.RowAdd, Gutter: "2 2 + ", Code: "two", Review: review.Anchor{Path: "main.go", Line: 2, Side: review.SideRight}},
	}
	app := newUIDiffTestAppWithBaseDraftsAndStatus(rows, DefaultBaseColors(), false, nil, true)
	app.Pump(vui.Size{Width: 40, Height: 8})
	app.Pump(vui.Size{Width: 40, Height: 8})
	codeOffset := uiDiffCodeOffset(rows)

	app.Send(vaxis.Key{Text: "i", Keycode: 'i'})
	app.Pump(vui.Size{Width: 40, Height: 8})
	app.Send(vaxis.Key{Text: "draft"})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	app.Pump(vui.Size{Width: 40, Height: 8})
	app.Pump(vui.Size{Width: 40, Height: 8})

	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress, Row: 4, Col: codeOffset})
	app.Send(vaxis.Mouse{Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion, Row: 4, Col: codeOffset + 1})
	app.Pump(vui.Size{Width: 40, Height: 8})
	p := vui.NewPainter(vui.Size{Width: 40, Height: 8})
	app.Paint(p)
	if got := p.Cell(codeOffset, 4).Background; got != uiDiffTestTheme().Selection {
		t.Fatalf("second diff row selected background = %v, want selection", got)
	}
	if got := p.Cell(codeOffset, 0).Background; got == uiDiffTestTheme().Selection {
		t.Fatal("mouse selection hit first row instead of row after comment box")
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
