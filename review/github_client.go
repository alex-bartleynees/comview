package review

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func checkGH() error {
	_, err := exec.LookPath("gh")
	if err != nil {
		return fmt.Errorf("gh CLI not found in PATH; install from https://cli.github.com")
	}
	return nil
}

func FetchPRDiff(owner, repo string, prNumber int) (string, error) {
	if err := checkGH(); err != nil {
		return "", err
	}
	out, err := exec.Command("gh", "pr", "diff", strconv.Itoa(prNumber), "--repo", owner+"/"+repo).Output()
	if err != nil {
		return "", ghError("fetch PR diff", err)
	}
	return string(out), nil
}

// FetchPRComments returns all review comments on the given PR.
// Uses --paginate --jq '.[]' to stream items across pages and decode them one by one.
func FetchPRComments(owner, repo string, prNumber int) ([]GitHubComment, error) {
	if err := checkGH(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/comments", owner, repo, prNumber)
	out, err := exec.Command("gh", "api", "--paginate", path, "--jq", ".[]").Output()
	if err != nil {
		return nil, ghError("fetch PR comments", err)
	}
	var comments []GitHubComment
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var c GitHubComment
		if err := dec.Decode(&c); err != nil {
			return nil, fmt.Errorf("parse PR comment: %w", err)
		}
		comments = append(comments, c)
	}
	return comments, nil
}

func FetchPRHeadSHA(owner, repo string, prNumber int) (string, error) {
	if err := checkGH(); err != nil {
		return "", err
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, prNumber)
	out, err := exec.Command("gh", "api", path, "--jq", ".head.sha").Output()
	if err != nil {
		return "", ghError("fetch PR head SHA", err)
	}
	return strings.TrimSpace(string(out)), nil
}

type createCommentRequest struct {
	Body      string `json:"body"`
	CommitID  string `json:"commit_id"`
	Path      string `json:"path"`
	Line      int    `json:"line"`
	Side      Side   `json:"side"`
	StartLine int    `json:"start_line,omitempty"`
	StartSide Side   `json:"start_side,omitempty"`
}

// CreateComment posts a new review comment on the PR.
// headSHA is used as commit_id if draft.CommitID is empty.
func CreateComment(owner, repo string, prNumber int, draft CommentDraft, headSHA string) (GitHubComment, error) {
	if err := checkGH(); err != nil {
		return GitHubComment{}, err
	}
	commitID := draft.CommitID
	if commitID == "" {
		commitID = headSHA
	}
	req := createCommentRequest{
		Body:      draft.Body,
		CommitID:  commitID,
		Path:      draft.Path,
		Line:      draft.Line,
		Side:      draft.Side,
		StartLine: draft.StartLine,
		StartSide: draft.StartSide,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return GitHubComment{}, fmt.Errorf("marshal comment: %w", err)
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/comments", owner, repo, prNumber)
	cmd := exec.Command("gh", "api", path, "--method", "POST", "--input", "-")
	cmd.Stdin = bytes.NewReader(body)
	out, err := cmd.Output()
	if err != nil {
		return GitHubComment{}, ghError("create comment", err)
	}
	var created GitHubComment
	if err := json.Unmarshal(out, &created); err != nil {
		return GitHubComment{}, fmt.Errorf("parse created comment: %w", err)
	}
	return created, nil
}

type updateCommentRequest struct {
	Body string `json:"body"`
}

func UpdateComment(owner, repo string, commentID int64, body string) error {
	if err := checkGH(); err != nil {
		return err
	}
	req := updateCommentRequest{Body: body}
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal comment update: %w", err)
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls/comments/%d", owner, repo, commentID)
	cmd := exec.Command("gh", "api", path, "--method", "PATCH", "--input", "-")
	cmd.Stdin = bytes.NewReader(data)
	if _, err := cmd.Output(); err != nil {
		return ghError("update comment", err)
	}
	return nil
}

func DeleteComment(owner, repo string, commentID int64) error {
	if err := checkGH(); err != nil {
		return err
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls/comments/%d", owner, repo, commentID)
	if _, err := exec.Command("gh", "api", path, "--method", "DELETE").Output(); err != nil {
		return ghError("delete comment", err)
	}
	return nil
}

func ghError(op string, err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
		return fmt.Errorf("%s: %s", op, strings.TrimSpace(string(exitErr.Stderr)))
	}
	return fmt.Errorf("%s: %w", op, err)
}
