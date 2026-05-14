package diff

import "testing"

func TestRowsWithOptionsAddsInlineSpansForReplacementLines(t *testing.T) {
	doc, err := Parse(`diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1 @@
-foo := oldValue + 1
+foo := newValue + 1
`)
	if err != nil {
		t.Fatal(err)
	}

	rows := doc.Rows()
	var deleteRow, addRow Row
	for _, row := range rows {
		switch row.Kind {
		case RowDelete:
			deleteRow = row
		case RowAdd:
			addRow = row
		}
	}

	assertSpan(t, deleteRow.InlineSpans, 7, 15)
	assertSpan(t, addRow.InlineSpans, 7, 15)
}

func TestRowsWithOptionsDoesNotHighlightEqualReplacementLines(t *testing.T) {
	doc, err := Parse(`diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1 @@
-foo := value
+foo := value
`)
	if err != nil {
		t.Fatal(err)
	}

	for _, row := range doc.Rows() {
		if len(row.InlineSpans) != 0 {
			t.Fatalf("row %q has inline spans %+v", row.Text, row.InlineSpans)
		}
	}
}

func TestRowsWithOptionsDoesNotPairUnrelatedReplacementLines(t *testing.T) {
	doc, err := Parse(`diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1 @@
-func oldThing() {
+type User struct {
`)
	if err != nil {
		t.Fatal(err)
	}

	for _, row := range doc.Rows() {
		if len(row.InlineSpans) != 0 {
			t.Fatalf("row %q has inline spans %+v", row.Text, row.InlineSpans)
		}
	}
}

func TestRowsWithOptionsDoesNotPairLinesSharingOnlyPunctuation(t *testing.T) {
	doc, err := Parse(`diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1 @@
-foo.Bar()
+baz.Qux()
`)
	if err != nil {
		t.Fatal(err)
	}

	for _, row := range doc.Rows() {
		if len(row.InlineSpans) != 0 {
			t.Fatalf("row %q has inline spans %+v", row.Text, row.InlineSpans)
		}
	}
}

func TestRowsWithOptionsPairsShiftedSimilarLines(t *testing.T) {
	doc, err := Parse(`diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
-foo := oldValue + 1
-keep()
+inserted()
+foo := newValue + 1
+keep()
`)
	if err != nil {
		t.Fatal(err)
	}

	var deleteRows []Row
	var addRows []Row
	for _, row := range doc.Rows() {
		switch row.Kind {
		case RowDelete:
			deleteRows = append(deleteRows, row)
		case RowAdd:
			addRows = append(addRows, row)
		}
	}

	if len(deleteRows) != 2 || len(addRows) != 3 {
		t.Fatalf("got %d delete rows and %d add rows", len(deleteRows), len(addRows))
	}
	assertSpan(t, deleteRows[0].InlineSpans, 7, 15)
	if len(addRows[0].InlineSpans) != 0 {
		t.Fatalf("inserted row has inline spans %+v", addRows[0].InlineSpans)
	}
	assertSpan(t, addRows[1].InlineSpans, 7, 15)
	if len(deleteRows[1].InlineSpans) != 0 || len(addRows[2].InlineSpans) != 0 {
		t.Fatalf("equal shifted rows have inline spans delete=%+v add=%+v", deleteRows[1].InlineSpans, addRows[2].InlineSpans)
	}
}

func TestRowsWithOptionsHighlightsReplacedAppendArgumentAsOneSpan(t *testing.T) {
	doc, err := Parse(`diff --git a/render.go b/render.go
--- a/render.go
+++ b/render.go
@@ -65 +65 @@
-rows = append(rows, Row{Kind: RowHunk, Text: hunk.Header, FileName: syntaxName})
+rows = append(rows, renderHunkHeaderRow(syntaxName, hunk))
`)
	if err != nil {
		t.Fatal(err)
	}

	var deleteRow, addRow Row
	for _, row := range doc.Rows() {
		switch row.Kind {
		case RowDelete:
			deleteRow = row
		case RowAdd:
			addRow = row
		}
	}

	const unchangedPrefix = "rows = append(rows, "
	assertSpan(t, deleteRow.InlineSpans, len(unchangedPrefix), len(deleteRow.Code)-1)
	assertSpan(t, addRow.InlineSpans, len(unchangedPrefix), len(addRow.Code)-1)
}

func TestInlineSpansCoalesceAcrossWhitespace(t *testing.T) {
	oldSpans, newSpans := inlineSpans("old value", "new thing")

	assertSpan(t, oldSpans, 0, len("old value"))
	assertSpan(t, newSpans, 0, len("new thing"))
}

func assertSpan(t *testing.T, spans []InlineSpan, start int, end int) {
	t.Helper()
	if len(spans) != 1 {
		t.Fatalf("spans = %+v, want one span", spans)
	}
	if spans[0].Start != start || spans[0].End != end {
		t.Fatalf("span = %+v, want %d:%d", spans[0], start, end)
	}
}
