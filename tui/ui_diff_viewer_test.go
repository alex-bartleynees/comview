package tui

import (
	"strings"
	"testing"

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
	theme := uiThemeFromBaseColors(base)
	return vui.NewApp(uiDiffRoot(rows, wrap, drafts), vui.WithTheme(theme))
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
	if got := p.Cell(0, 3).Background; got != uiDiffTestTheme().Selection {
		t.Fatalf("bottom visible row background = %v, want active selection", got)
	}

	app.Send(vaxis.Key{Text: "k", Keycode: 'k'})
	app.Pump(vui.Size{Width: 20, Height: 4})
	p = vui.NewPainter(vui.Size{Width: 20, Height: 4})
	app.Paint(p)
	if got := p.Cell(0, 2).Background; got != uiDiffTestTheme().Selection {
		t.Fatalf("row above bottom background = %v, want active selection", got)
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
		if got := uiDiffHighlightedScreenRow(p, uiDiffTestTheme().Selection); got == -1 {
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
		if got := p.Cell(col, 0).Background; got != uiDiffTestTheme().Selection {
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
	if got := p.Cell(19, 0).Background; got != theme.Selection {
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
	if style := uiGutterStyle(diff.RowContext, true, theme); style.Background != theme.Selection {
		t.Fatalf("active gutter background = %v, want %v", style.Background, theme.Selection)
	}
	if style := uiGutterStyle(diff.RowContext, false, theme); style.Background != theme.Background {
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
	if cell.Background != theme.Selection {
		t.Fatalf("keyword background = %v, want selection %v", cell.Background, theme.Selection)
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
	if got := uiGutterStyle(diff.RowAdd, false, theme).Foreground; got != theme.Palette.Green.Tone400 {
		t.Fatalf("add marker foreground = %v, want green tone400 %v", got, theme.Palette.Green.Tone400)
	}
	if got := uiGutterStyle(diff.RowDelete, false, theme).Foreground; got != theme.Palette.Red.Tone400 {
		t.Fatalf("delete marker foreground = %v, want red tone400 %v", got, theme.Palette.Red.Tone400)
	}
}

func TestUIDiffViewAddLineNumberGutterUsesSoftGreenForeground(t *testing.T) {
	theme := uiDiffTestTheme()
	if got := uiLineNumberGutterStyle(diff.RowAdd, false, theme).Foreground; got != theme.Palette.Green.Tone300 {
		t.Fatalf("add line number foreground = %v, want soft green %v", got, theme.Palette.Green.Tone300)
	}
	if got := uiGutterStyle(diff.RowAdd, false, theme).Foreground; got != theme.Palette.Green.Tone400 {
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
			if got := uiDiffHighlightedScreenRow(p, uiDiffTestTheme().Selection); got != tt.wantHighlight {
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
	if got := uiDiffHighlightedScreenRow(p, uiDiffTestTheme().Selection); got != 1 {
		t.Fatalf("]c highlight row = %d, want 1", got)
	}

	app.Send(vaxis.Key{Text: "]", Keycode: ']'})
	app.Send(vaxis.Key{Text: "c", Keycode: 'c'})
	app.Pump(vui.Size{Width: 24, Height: 5})
	p = vui.NewPainter(vui.Size{Width: 24, Height: 5})
	app.Paint(p)
	if got := uiDiffHighlightedScreenRow(p, uiDiffTestTheme().Selection); got != 4 {
		t.Fatalf("second ]c highlight row = %d, want 4", got)
	}

	app.Send(vaxis.Key{Text: "[", Keycode: '['})
	app.Send(vaxis.Key{Text: "c", Keycode: 'c'})
	app.Pump(vui.Size{Width: 24, Height: 5})
	p = vui.NewPainter(vui.Size{Width: 24, Height: 5})
	app.Paint(p)
	if got := uiDiffHighlightedScreenRow(p, uiDiffTestTheme().Selection); got != 1 {
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
	if got := uiDiffHighlightedScreenRow(p, uiDiffTestTheme().Selection); got != 1 {
		t.Fatalf("]n highlight row = %d, want 1", got)
	}

	app.Send(vaxis.Key{Text: "]", Keycode: ']'})
	app.Send(vaxis.Key{Text: "n", Keycode: 'n'})
	app.Pump(vui.Size{Width: 24, Height: 3})
	p = vui.NewPainter(vui.Size{Width: 24, Height: 3})
	app.Paint(p)
	if got := uiDiffHighlightedScreenRow(p, uiDiffTestTheme().Selection); got != 2 {
		t.Fatalf("second ]n highlight row = %d, want 2", got)
	}

	app.Send(vaxis.Key{Text: "[", Keycode: '['})
	app.Send(vaxis.Key{Text: "n", Keycode: 'n'})
	app.Pump(vui.Size{Width: 24, Height: 3})
	p = vui.NewPainter(vui.Size{Width: 24, Height: 3})
	app.Paint(p)
	if got := uiDiffHighlightedScreenRow(p, uiDiffTestTheme().Selection); got != 1 {
		t.Fatalf("[n highlight row = %d, want 1", got)
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
	for col := 7; col < 15; col++ {
		if got := p.Cell(col, 0).Background; got != uiDiffCursorBackground(uiDiffTestTheme()) {
			t.Fatalf("tab cursor background at col %d = %v, want yank", col, got)
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
	for col := 7; col < 15; col++ {
		if got := p.Cell(col, 0).Background; got != uiDiffCursorBackground(uiDiffTestTheme()) {
			t.Fatalf("tab cursor after h at col %d = %v, want yank", col, got)
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
	if got := uiDiffHighlightedScreenRow(p, uiDiffTestTheme().Selection); got != 0 {
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
	if got := uiDiffHighlightedScreenRow(p, uiDiffTestTheme().Selection); got != 2 {
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
