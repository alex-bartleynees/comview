package main

import (
	"reflect"
	"testing"
)

func TestWatchCommandDefaultsToGitDiff(t *testing.T) {
	command, err := watchCommand(nil)
	if err != nil {
		t.Fatalf("watchCommand() error = %v", err)
	}
	if want := []string{"git", "diff"}; !reflect.DeepEqual(command, want) {
		t.Fatalf("watchCommand() = %#v, want %#v", command, want)
	}
}

func TestWatchCommandPassesArgsToGitDiff(t *testing.T) {
	command, err := watchCommand([]string{"--staged", "HEAD~1"})
	if err != nil {
		t.Fatalf("watchCommand() error = %v", err)
	}
	if want := []string{"git", "diff", "--staged", "HEAD~1"}; !reflect.DeepEqual(command, want) {
		t.Fatalf("watchCommand() = %#v, want %#v", command, want)
	}
}

func TestWatchCommandAcceptsCustomCommandAfterSeparator(t *testing.T) {
	command, err := watchCommand([]string{"--", "gh", "pr", "diff", "123"})
	if err != nil {
		t.Fatalf("watchCommand() error = %v", err)
	}
	if want := []string{"gh", "pr", "diff", "123"}; !reflect.DeepEqual(command, want) {
		t.Fatalf("watchCommand() = %#v, want %#v", command, want)
	}
}

func TestWatchCommandRequiresCommandAfterSeparator(t *testing.T) {
	if _, err := watchCommand([]string{"--"}); err == nil {
		t.Fatal("watchCommand() error = nil, want error")
	}
}

func TestPRCommandExplicitRepo(t *testing.T) {
	owner, repo, prNumber, err := prCommand([]string{"rockorager/comview", "42"})
	if err != nil {
		t.Fatalf("prCommand() error = %v", err)
	}
	if owner != "rockorager" || repo != "comview" || prNumber != 42 {
		t.Fatalf("prCommand() = %q, %q, %d, want rockorager, comview, 42", owner, repo, prNumber)
	}
}

func TestPRCommandRequiresPRNumber(t *testing.T) {
	if _, _, _, err := prCommand(nil); err == nil {
		t.Fatal("prCommand() error = nil, want error")
	}
}

func TestPRCommandRejectsInvalidRepo(t *testing.T) {
	if _, _, _, err := prCommand([]string{"noslash", "42"}); err == nil {
		t.Fatal("prCommand() error = nil, want error")
	}
}

func TestPRCommandRejectsInvalidPRNumber(t *testing.T) {
	if _, _, _, err := prCommand([]string{"owner/repo", "notanumber"}); err == nil {
		t.Fatal("prCommand() error = nil, want error")
	}
}

func TestOwnerRepoFromRemoteHTTPS(t *testing.T) {
	tests := []struct {
		url        string
		wantOwner  string
		wantRepo   string
	}{
		{"https://github.com/rockorager/comview.git", "rockorager", "comview"},
		{"https://github.com/rockorager/comview", "rockorager", "comview"},
	}
	for _, tt := range tests {
		owner, repo, err := parseRemoteURL(tt.url)
		if err != nil {
			t.Fatalf("parseRemoteURL(%q) error = %v", tt.url, err)
		}
		if owner != tt.wantOwner || repo != tt.wantRepo {
			t.Fatalf("parseRemoteURL(%q) = %q, %q, want %q, %q", tt.url, owner, repo, tt.wantOwner, tt.wantRepo)
		}
	}
}

func TestOwnerRepoFromRemoteSSH(t *testing.T) {
	tests := []struct {
		url       string
		wantOwner string
		wantRepo  string
	}{
		{"git@github.com:rockorager/comview.git", "rockorager", "comview"},
		{"git@github.com:rockorager/comview", "rockorager", "comview"},
	}
	for _, tt := range tests {
		owner, repo, err := parseRemoteURL(tt.url)
		if err != nil {
			t.Fatalf("parseRemoteURL(%q) error = %v", tt.url, err)
		}
		if owner != tt.wantOwner || repo != tt.wantRepo {
			t.Fatalf("parseRemoteURL(%q) = %q, %q, want %q, %q", tt.url, owner, repo, tt.wantOwner, tt.wantRepo)
		}
	}
}
