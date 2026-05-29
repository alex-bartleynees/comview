package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.rockorager.dev/vaxis"

	"go.rockorager.dev/comview/diff"
	"go.rockorager.dev/comview/review"
)

func TestRunWatchCommandDropsANSISequences(t *testing.T) {
	commandPath := filepath.Join(t.TempDir(), "ansi-diff")
	commandSource := "#!/bin/sh\nprintf 'diff --git a/main.go b/main.go\\n\\033[31m-old\\033[0m\\n\\033[32m+new\\033[0m\\n'\n"
	if err := os.WriteFile(commandPath, []byte(commandSource), 0o755); err != nil {
		t.Fatal(err)
	}

	output, err := runWatchCommand(context.Background(), []string{commandPath})
	if err != nil {
		t.Fatalf("runWatchCommand() error = %v", err)
	}

	want := "diff --git a/main.go b/main.go\n-old\n+new\n"
	if output != want {
		t.Fatalf("runWatchCommand() = %q, want %q", output, want)
	}
}

func TestUIWatchApplyResultRendersDiffUpdate(t *testing.T) {
	state := &uiWatchViewState{}
	now := time.Date(2026, 5, 25, 12, 34, 56, 0, time.UTC)
	changed := state.applyWatchResult(`diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1 @@
-old
+new
`, nil, now)
	if !changed {
		t.Fatal("applyWatchResult changed = false, want true")
	}
	if len(state.rows) == 0 {
		t.Fatal("rows are empty after diff update")
	}
	if got, want := state.message, "Updated 12:34:56"; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
}

func TestUIWatchApplyResultRendersNoChanges(t *testing.T) {
	state := &uiWatchViewState{}
	now := time.Date(2026, 5, 25, 12, 34, 56, 0, time.UTC)
	changed := state.applyWatchResult("", nil, now)
	if !changed {
		t.Fatal("applyWatchResult changed = false, want true")
	}
	if len(state.rows) != 0 {
		t.Fatalf("rows = %d, want empty", len(state.rows))
	}
	if got, want := state.message, "No changes 12:34:56"; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
}

func TestUIWatchApplyResultErrorPreservesRows(t *testing.T) {
	state := &uiWatchViewState{rows: []diff.Row{{Kind: diff.RowContext, Text: "existing"}}}
	changed := state.applyWatchResult("", errors.New("boom"), time.Date(2026, 5, 25, 12, 34, 56, 0, time.UTC))
	if !changed {
		t.Fatal("applyWatchResult changed = false, want true")
	}
	if len(state.rows) != 1 || state.rows[0].Text != "existing" {
		t.Fatalf("rows = %+v, want existing row preserved", state.rows)
	}
	if got := state.message; !strings.Contains(got, "Watch command failed: boom") {
		t.Fatalf("message = %q, want watch error", got)
	}
}

func TestUIWatchApplyResultParseErrorPreservesRows(t *testing.T) {
	state := &uiWatchViewState{rows: []diff.Row{{Kind: diff.RowContext, Text: "existing"}}}
	changed := state.applyWatchResult(strings.Repeat("x", 1024*1024*9), nil, time.Date(2026, 5, 25, 12, 34, 56, 0, time.UTC))
	if !changed {
		t.Fatal("applyWatchResult changed = false, want true")
	}
	if len(state.rows) != 1 || state.rows[0].Text != "existing" {
		t.Fatalf("rows = %+v, want existing row preserved", state.rows)
	}
	if got := state.message; !strings.Contains(got, "Could not parse diff:") {
		t.Fatalf("message = %q, want parse error", got)
	}
}

func TestUIWatchApplyResultSkipsDuplicateOutput(t *testing.T) {
	state := &uiWatchViewState{}
	now := time.Date(2026, 5, 25, 12, 34, 56, 0, time.UTC)
	if !state.applyWatchResult("", nil, now) {
		t.Fatal("first applyWatchResult changed = false, want true")
	}
	if state.applyWatchResult("", nil, now.Add(time.Minute)) {
		t.Fatal("second applyWatchResult changed = true, want false for duplicate output")
	}
	if got, want := state.message, "No changes 12:34:56"; got != want {
		t.Fatalf("message = %q, want original duplicate message %q", got, want)
	}
}

func TestUIWatchDiffRootPreservesConfigAndReviewState(t *testing.T) {
	draft := review.CommentDraft{Path: "main.go", Line: 1, Side: review.SideRight, Body: "watch note"}
	state := &uiWatchViewState{
		rows:    []diff.Row{{Kind: diff.RowAdd, Text: "+new"}},
		message: "Watching: custom diff",
	}
	root := state.diffRoot(uiWatchView{
		Command: []string{"custom", "diff"},
		Config: Config{
			Wrap:        true,
			Keybindings: map[string][]string{"cursor_down": {"ctrl+n"}},
		},
		Review: review.CommentFile{Comments: []review.CommentDraft{draft}},
		File:   "/tmp/comments.json",
	})

	if !root.Wrap {
		t.Fatal("wrap = false, want true")
	}
	if got := root.ReviewDrafts; len(got) != 1 || got[0] != draft {
		t.Fatalf("review drafts = %+v, want draft", got)
	}
	if got, want := root.ReviewFile, "/tmp/comments.json"; got != want {
		t.Fatalf("review file = %q, want %q", got, want)
	}
	if got, want := root.EmptyMessage, "No changes."; got != want {
		t.Fatalf("empty message = %q, want %q", got, want)
	}
	if got, want := root.EmptyHint, "Watching: custom diff"; got != want {
		t.Fatalf("empty hint = %q, want %q", got, want)
	}
	if got, want := root.InitialStatus, "Watching: custom diff"; got != want {
		t.Fatalf("initial status = %q, want %q", got, want)
	}
	if !root.Binds.Matches(vaxis.Key{Text: "n", Keycode: 'n', Modifiers: vaxis.ModCtrl}, "cursor_down") {
		t.Fatal("custom cursor_down binding was not preserved")
	}
}
