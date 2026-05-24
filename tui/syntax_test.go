package tui

import (
	"testing"

	"github.com/alecthomas/chroma/v2"
	"go.rockorager.dev/vaxis"

	"go.rockorager.dev/comview/diff"
)

func TestSyntaxHighlighterHighlightsKnownFileType(t *testing.T) {
	highlighter := NewSyntaxHighlighter()
	base := vaxis.Style{
		Foreground: vaxis.RGBColor(1, 2, 3),
		Background: vaxis.RGBColor(4, 5, 6),
	}

	segments := highlighter.Highlight("main.go", "package main", base)
	if len(segments) < 2 {
		t.Fatalf("segments = %d, want at least 2", len(segments))
	}

	foundSyntaxColor := false
	for _, segment := range segments {
		if segment.Style.Foreground != base.Foreground {
			foundSyntaxColor = true
		}
		if segment.Style.Background != base.Background {
			t.Fatalf("segment background = %v, want %v", segment.Style.Background, base.Background)
		}
	}
	if !foundSyntaxColor {
		t.Fatal("no segment used a syntax foreground color")
	}
}

func TestSyntaxHighlighterFallsBackForUnknownFileType(t *testing.T) {
	highlighter := NewSyntaxHighlighter()
	base := vaxis.Style{
		Foreground: vaxis.RGBColor(1, 2, 3),
		Background: vaxis.RGBColor(4, 5, 6),
	}

	segments := highlighter.Highlight("README.unknown", "plain text", base)
	if len(segments) != 1 {
		t.Fatalf("segments = %d, want 1", len(segments))
	}
	if segments[0].Text != "plain text" {
		t.Fatalf("segment text = %q", segments[0].Text)
	}
	if segments[0].Style != base {
		t.Fatalf("segment style = %+v, want %+v", segments[0].Style, base)
	}
}

func TestSyntaxHighlighterUsesComviewStyle(t *testing.T) {
	highlighter := NewSyntaxHighlighter()
	if highlighter.style.Name != "comview" {
		t.Fatalf("style name = %q, want comview", highlighter.style.Name)
	}
}

func TestSyntaxHighlighterUpdatesWithColorScheme(t *testing.T) {
	scheme := DefaultColorScheme()
	scheme.Hunk = vaxis.RGBColor(11, 22, 33)

	highlighter := NewSyntaxHighlighter()
	highlighter.SetColorScheme(scheme)

	style := highlighter.styleFor(chroma.Keyword, vaxis.Style{})
	if style.Foreground != scheme.Hunk {
		t.Fatalf("keyword foreground = %v, want %v", style.Foreground, scheme.Hunk)
	}
}

func TestSyntaxHighlighterColorsAdditionalTokenKinds(t *testing.T) {
	scheme := DefaultColorScheme()
	highlighter := NewSyntaxHighlighterWithScheme(scheme)
	base := vaxis.Style{Foreground: vaxis.RGBColor(1, 2, 3)}

	tests := []struct {
		tokenType chroma.TokenType
		want      vaxis.Color
	}{
		{chroma.NameProperty, scheme.Cyan()},
		{chroma.NameNamespace, scheme.Blue},
		{chroma.NameTag, scheme.Blue},
		{chroma.NameDecorator, scheme.Magenta()},
		{chroma.NameVariableClass, scheme.Cyan()},
		{chroma.LiteralOther, scheme.Cyan()},
		{chroma.LiteralDate, scheme.Yellow},
		{chroma.TextPunctuation, scheme.Muted},
		{chroma.TextSymbol, scheme.Yellow},
	}

	for _, tt := range tests {
		style := highlighter.styleFor(tt.tokenType, base)
		if style.Foreground != tt.want {
			t.Fatalf("%s foreground = %v, want %v", tt.tokenType, style.Foreground, tt.want)
		}
	}
}

func TestSyntaxHighlighterStylesAdditionalGenericTokenKinds(t *testing.T) {
	scheme := DefaultColorScheme()
	highlighter := NewSyntaxHighlighterWithScheme(scheme)

	style := highlighter.styleFor(chroma.GenericError, vaxis.Style{})
	if style.Foreground != scheme.Delete || style.Attribute&vaxis.AttrBold == 0 {
		t.Fatalf("generic error style = %+v, want delete bold", style)
	}
	style = highlighter.styleFor(chroma.GenericEmph, vaxis.Style{})
	if style.Foreground != scheme.Foreground || style.Attribute&vaxis.AttrItalic == 0 {
		t.Fatalf("generic emph style = %+v, want italic foreground", style)
	}
	style = highlighter.styleFor(chroma.GenericUnderline, vaxis.Style{})
	if style.Foreground != scheme.Foreground || style.UnderlineStyle != vaxis.UnderlineSingle {
		t.Fatalf("generic underline style = %+v, want underlined foreground", style)
	}
}

func TestSyntaxHighlighterPreservesMultilineRawStringStateAcrossRows(t *testing.T) {
	scheme := DefaultColorScheme()
	highlighter := NewSyntaxHighlighterWithScheme(scheme)
	base := vaxis.Style{
		Foreground: vaxis.RGBColor(1, 2, 3),
		Background: vaxis.RGBColor(4, 5, 6),
	}
	rows := []diff.Row{
		{Kind: diff.RowHunk, FileName: "inline_test.go", Text: "@@ -1 +1 @@"},
		{Kind: diff.RowAdd, FileName: "inline_test.go", Code: "doc, err := Parse(`diff --git a/main.go b/main.go"},
		{Kind: diff.RowAdd, FileName: "inline_test.go", Code: "--- a/main.go"},
		{Kind: diff.RowAdd, FileName: "inline_test.go", Code: "+++ b/main.go"},
		{Kind: diff.RowAdd, FileName: "inline_test.go", Code: "`)"},
	}

	segments := highlighter.HighlightRows(rows, func(diff.RowKind) vaxis.Style {
		return base
	})

	rawStringLine := segments[2]
	if len(rawStringLine) != 1 {
		t.Fatalf("raw string line segments = %+v, want one segment", rawStringLine)
	}
	if rawStringLine[0].Text != "--- a/main.go" {
		t.Fatalf("raw string line text = %q", rawStringLine[0].Text)
	}
	if rawStringLine[0].Style.Foreground != scheme.Green() {
		t.Fatalf("raw string line foreground = %v, want string color %v", rawStringLine[0].Style.Foreground, scheme.Green())
	}
	if rawStringLine[0].Style.Background != base.Background {
		t.Fatalf("raw string line background = %v, want %v", rawStringLine[0].Style.Background, base.Background)
	}
}
