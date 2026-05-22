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
	if got := p.Cell(3, 3).Background; got != DefaultColorScheme().Selection {
		t.Fatalf("bottom visible row background = %v, want active selection", got)
	}

	app.Send(vaxis.Key{Text: "k", Keycode: 'k'})
	app.Pump(vui.Size{Width: 20, Height: 4})
	p = vui.NewPainter(vui.Size{Width: 20, Height: 4})
	app.Paint(p)
	if got := p.Cell(3, 2).Background; got != DefaultColorScheme().Selection {
		t.Fatalf("row above bottom background = %v, want active selection", got)
	}
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
