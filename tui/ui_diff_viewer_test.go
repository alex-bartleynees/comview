package tui

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	vui "git.sr.ht/~rockorager/vaxis/ui"

	"github.com/rockorager/comview/diff"
)

func TestUIDiffViewRendersRowsAsSliverTable(t *testing.T) {
	rows := []diff.Row{
		{Kind: diff.RowContext, Gutter: "1 1   ", Code: "same"},
		{Kind: diff.RowDelete, Gutter: "2     ", Marker: "-", Code: "old"},
		{Kind: diff.RowAdd, Gutter: "  2   ", Marker: "+", Code: "new"},
	}
	app := vui.NewApp(uiDiffView{Rows: rows, Scheme: DefaultColorScheme()})
	app.Pump(vui.Size{Width: 20, Height: 3})
	app.Pump(vui.Size{Width: 20, Height: 3})

	p := vui.NewPainter(vui.Size{Width: 20, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "1" {
		t.Fatalf("old gutter = %q, want 1", got)
	}
	if got := p.Cell(1, 0).Grapheme; got != "1" {
		t.Fatalf("new gutter = %q, want 1", got)
	}
	if got := p.Cell(3, 0).Grapheme; got != "s" {
		t.Fatalf("code start = %q, want s", got)
	}
	if got := p.Cell(2, 1).Grapheme; got != "-" {
		t.Fatalf("delete marker = %q, want -", got)
	}
	if got := p.Cell(2, 2).Grapheme; got != "+" {
		t.Fatalf("add marker = %q, want +", got)
	}
}

func TestUIDiffViewMovesCursorAndRevealsRows(t *testing.T) {
	rows := make([]diff.Row, 20)
	for i := range rows {
		rows[i] = diff.Row{Kind: diff.RowContext, Gutter: "1 1   ", Code: "line"}
	}
	app := vui.NewApp(uiDiffView{Rows: rows, Scheme: DefaultColorScheme()})
	app.Pump(vui.Size{Width: 20, Height: 4})
	app.Pump(vui.Size{Width: 20, Height: 4})

	app.Send(vaxis.Key{Text: "G", Keycode: 'G'})
	app.Pump(vui.Size{Width: 20, Height: 4})
	app.Pump(vui.Size{Width: 20, Height: 4})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 4})
	app.Paint(p)
	if got := p.Cell(0, 3).Background; got != DefaultColorScheme().Selection {
		t.Fatalf("bottom visible row background = %v, want active selection", got)
	}

	app.Send(vaxis.Key{Text: "k", Keycode: 'k'})
	app.Pump(vui.Size{Width: 20, Height: 4})
	p = vui.NewPainter(vui.Size{Width: 20, Height: 4})
	app.Paint(p)
	if got := p.Cell(0, 2).Background; got != DefaultColorScheme().Selection {
		t.Fatalf("row above bottom background = %v, want active selection", got)
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
			app := vui.NewApp(uiDiffView{Rows: rows, Scheme: DefaultColorScheme()})
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
			if got := uiDiffHighlightedScreenRow(p, DefaultColorScheme().Selection); got != tt.wantHighlight {
				t.Fatalf("highlight row = %d, want %d", got, tt.wantHighlight)
			}
		})
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
			wantCol:   4,
			wantGlyph: "b",
		},
		{
			name:      "h moves cursor left",
			keys:      []vaxis.Key{{Text: "l", Keycode: 'l'}, {Text: "l", Keycode: 'l'}, {Text: "h", Keycode: 'h'}},
			wantCol:   4,
			wantGlyph: "b",
		},
		{
			name:      "0 moves to code start",
			keys:      []vaxis.Key{{Text: "l", Keycode: 'l'}, {Text: "l", Keycode: 'l'}, {Text: "0", Keycode: '0'}},
			wantCol:   3,
			wantGlyph: "a",
		},
		{
			name:      "$ moves to code end",
			keys:      []vaxis.Key{{Text: "$", Keycode: '$'}},
			wantCol:   8,
			wantGlyph: "f",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := []diff.Row{{Kind: diff.RowAdd, Gutter: "1 1   ", Marker: "+", Code: "abcdef"}}
			app := vui.NewApp(uiDiffView{Rows: rows, Scheme: DefaultColorScheme()})
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
			if cell.Background != DefaultColorScheme().Yank {
				t.Fatalf("cursor background = %v, want yank", cell.Background)
			}
		})
	}
}

func TestUIDiffViewHorizontalMovementUsesTabStops(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "a\tb"}}
	app := vui.NewApp(uiDiffView{Rows: rows, Scheme: DefaultColorScheme()})
	app.Pump(vui.Size{Width: 20, Height: 3})
	app.Pump(vui.Size{Width: 20, Height: 3})

	app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	app.Pump(vui.Size{Width: 20, Height: 3})
	p := vui.NewPainter(vui.Size{Width: 20, Height: 3})
	app.Paint(p)
	for col := 4; col < 12; col++ {
		if got := p.Cell(col, 0).Background; got != DefaultColorScheme().Yank {
			t.Fatalf("tab cursor background at col %d = %v, want yank", col, got)
		}
	}

	app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
	app.Pump(vui.Size{Width: 20, Height: 3})
	p = vui.NewPainter(vui.Size{Width: 20, Height: 3})
	app.Paint(p)
	if cell := p.Cell(12, 0); cell.Grapheme != "b" || cell.Background != DefaultColorScheme().Yank {
		t.Fatalf("cursor after tab = %q/%v, want b/yank", cell.Grapheme, cell.Background)
	}

	app.Send(vaxis.Key{Text: "h", Keycode: 'h'})
	app.Pump(vui.Size{Width: 20, Height: 3})
	p = vui.NewPainter(vui.Size{Width: 20, Height: 3})
	app.Paint(p)
	for col := 4; col < 12; col++ {
		if got := p.Cell(col, 0).Background; got != DefaultColorScheme().Yank {
			t.Fatalf("tab cursor after h at col %d = %v, want yank", col, got)
		}
	}
}

func TestUIDiffViewHorizontalMovementStopsAtLineEnd(t *testing.T) {
	rows := []diff.Row{{Kind: diff.RowContext, Gutter: "1 1   ", Code: "abc"}}
	app := vui.NewApp(uiDiffView{Rows: rows, Scheme: DefaultColorScheme()})
	app.Pump(vui.Size{Width: 20, Height: 3})
	app.Pump(vui.Size{Width: 20, Height: 3})

	for i := 0; i < 10; i++ {
		app.Send(vaxis.Key{Text: "l", Keycode: 'l'})
		app.Pump(vui.Size{Width: 20, Height: 3})
	}

	p := vui.NewPainter(vui.Size{Width: 20, Height: 3})
	app.Paint(p)
	cell := p.Cell(5, 0)
	if cell.Grapheme != "c" || cell.Background != DefaultColorScheme().Yank {
		t.Fatalf("cursor after repeated l = %q/%v, want c/yank", cell.Grapheme, cell.Background)
	}
	if got := p.Cell(6, 0).Background; got == DefaultColorScheme().Yank {
		t.Fatal("cursor highlighted past end of line")
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

func TestUIDiffViewAltPTogglesProfileOverlay(t *testing.T) {
	app := vui.NewApp(uiDiffView{Rows: []diff.Row{{Kind: diff.RowContext, Code: "line"}}, Scheme: DefaultColorScheme()})
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
	app := vui.NewApp(uiDiffView{Rows: rows, Scheme: DefaultColorScheme(), Wrap: true})
	app.Pump(vui.Size{Width: 8, Height: 4})
	app.Pump(vui.Size{Width: 8, Height: 4})

	p := vui.NewPainter(vui.Size{Width: 8, Height: 4})
	app.Paint(p)
	if got := p.Cell(3, 0).Grapheme; got != "a" {
		t.Fatalf("wrapped first line start = %q, want a", got)
	}
	if got := p.Cell(3, 1).Grapheme; got != "f" {
		t.Fatalf("wrapped second line start = %q, want f", got)
	}
	if got := p.Cell(3, 2).Grapheme; got != "n" {
		t.Fatalf("next row after wrapped row = %q, want n", got)
	}
}
