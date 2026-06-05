package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"go.rockorager.dev/comview/internal/terminal"
	"go.rockorager.dev/comview/tui"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "pr" {
		owner, repo, prNumber, err := prCommand(os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "comview: %v\n", err)
			os.Exit(1)
		}
		if err := tui.RunPR(owner, repo, prNumber); err != nil {
			fmt.Fprintf(os.Stderr, "comview: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "watch" {
		command, err := watchCommand(os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "comview: %v\n", err)
			os.Exit(1)
		}
		if err := tui.RunWatch(command); err != nil {
			fmt.Fprintf(os.Stderr, "comview: %v\n", err)
			os.Exit(1)
		}
		return
	}

	input, err := readPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "comview: %v\n", err)
		os.Exit(1)
	}

	if err := tui.Run(input); err != nil {
		fmt.Fprintf(os.Stderr, "comview: %v\n", err)
		os.Exit(1)
	}
}

func prCommand(args []string) (owner, repo string, prNumber int, err error) {
	switch len(args) {
	case 0:
		return "", "", 0, fmt.Errorf("usage: comview pr [<owner/repo>] <pr-number>")
	case 1:
		// Only PR number given — infer owner/repo from git remote origin.
		owner, repo, err = ownerRepoFromRemote()
		if err != nil {
			return "", "", 0, err
		}
		prNumber, err = parsePRNumber(args[0])
		if err != nil {
			return "", "", 0, err
		}
	default:
		parts := strings.SplitN(args[0], "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", 0, fmt.Errorf("invalid repo %q: expected owner/repo", args[0])
		}
		owner, repo = parts[0], parts[1]
		prNumber, err = parsePRNumber(args[1])
		if err != nil {
			return "", "", 0, err
		}
	}
	return owner, repo, prNumber, nil
}

func parsePRNumber(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid PR number %q: expected a positive integer", s)
	}
	return n, nil
}

func ownerRepoFromRemote() (owner, repo string, err error) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", fmt.Errorf("could not read git remote origin (run inside a git repo or pass <owner/repo> explicitly)")
	}
	owner, repo, err = parseRemoteURL(strings.TrimSpace(string(out)))
	if err != nil {
		return "", "", fmt.Errorf("could not parse owner/repo from remote URL %q", strings.TrimSpace(string(out)))
	}
	return owner, repo, nil
}

func parseRemoteURL(url string) (owner, repo string, err error) {
	// SSH: git@github.com:owner/repo.git
	if strings.HasPrefix(url, "git@") {
		url = strings.TrimPrefix(url, "git@")
		if i := strings.Index(url, ":"); i >= 0 {
			url = url[i+1:]
		}
	} else {
		// HTTPS: https://github.com/owner/repo.git
		if i := strings.Index(url, "://"); i >= 0 {
			url = url[i+3:]
		}
		if i := strings.Index(url, "/"); i >= 0 {
			url = url[i+1:]
		}
	}
	url = strings.TrimSuffix(url, ".git")

	parts := strings.SplitN(url, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected owner/repo, got %q", url)
	}
	return parts[0], parts[1], nil
}

func watchCommand(args []string) ([]string, error) {
	if len(args) == 0 {
		return []string{"git", "diff"}, nil
	}
	if args[0] == "--" {
		if len(args) == 1 {
			return nil, fmt.Errorf("watch command after -- is required")
		}
		return append([]string{}, args[1:]...), nil
	}
	command := []string{"git", "diff"}
	command = append(command, args...)
	return command, nil
}

func readPipe() (string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}
	if stat.Mode()&os.ModeCharDevice != 0 {
		return "", nil
	}

	return terminal.PrintableANSIOutput(os.Stdin), nil
}
