// Package gitops runs local git operations for a session worktree.
package gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Service executes git against a local directory.
type Service struct {
	Git           string
	GH            string
	Glab          string
	BitbucketUser string
	BitbucketPass string
	BitbucketAPI  string
}

type Commit struct {
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
	Author  string `json:"author"`
	At      string `json:"at"`
}

// Status describes the current worktree state.
type Status struct {
	Branch     string   `json:"branch"`
	Clean      bool     `json:"clean"`
	Changes    []string `json:"changes"`
	LastCommit string   `json:"lastCommit,omitempty"`
	Commits    []Commit `json:"commits"`
}

// New resolves the git executable from TANDEMESH_GIT or PATH.
func New() *Service {
	git := os.Getenv("TANDEMESH_GIT")
	if git == "" {
		git = "git"
	}
	gh := os.Getenv("TANDEMESH_GH")
	if gh == "" {
		gh = "gh"
	}
	glab := os.Getenv("TANDEMESH_GLAB")
	if glab == "" {
		glab = "glab"
	}
	api := os.Getenv("BITBUCKET_API")
	if api == "" {
		api = "https://api.bitbucket.org/2.0"
	}
	return &Service{
		Git: git, GH: gh, Glab: glab,
		BitbucketUser: os.Getenv("BITBUCKET_USERNAME"),
		BitbucketPass: os.Getenv("BITBUCKET_APP_PASSWORD"),
		BitbucketAPI:  api,
	}
}

func (s *Service) run(ctx context.Context, dir string, args ...string) (string, error) {
	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, s.Git, append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// Inspect returns branch, changed paths and the last commit.
func (s *Service) Inspect(ctx context.Context, dir string) (Status, error) {
	status := Status{Changes: []string{}}
	branch, err := s.run(ctx, dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return status, err
	}
	status.Branch = strings.TrimSpace(branch)
	raw, err := s.run(ctx, dir, "status", "--porcelain")
	if err != nil {
		return status, err
	}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		status.Changes = append(status.Changes, strings.TrimSpace(line))
	}
	status.Clean = len(status.Changes) == 0
	if log, err := s.run(ctx, dir, "log", "-1", "--oneline"); err == nil {
		status.LastCommit = strings.TrimSpace(log)
	}
	if raw, err := s.run(ctx, dir, "log", "-10", "--pretty=format:%h%x1f%s%x1f%an%x1f%aI"); err == nil {
		status.Commits = parseCommits(raw)
	}
	return status, nil
}

func parseCommits(raw string) []Commit {
	out := []Commit{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		parts := strings.Split(line, "\x1f")
		if len(parts) != 4 {
			continue
		}
		out = append(out, Commit{Hash: parts[0], Subject: parts[1], Author: parts[2], At: parts[3]})
	}
	return out
}

// PRState describes a GitHub pull request.
type PRState struct {
	URL   string `json:"url"`
	State string `json:"state"`
	Draft bool   `json:"draft"`
	Title string `json:"title,omitempty"`
}

// PRStatus reads the pull request state for the current branch.
func (s *Service) PRStatus(ctx context.Context, dir string) (PRState, error) {
	if remote, _ := s.remoteURL(ctx, dir); isGitLabRemote(remote) {
		return s.gitLabMRStatus(ctx, dir)
	} else if isBitbucketRemote(remote) {
		return s.bitbucketPRStatus(ctx, dir)
	}
	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, s.GH, "pr", "view", "--json", "url,state,isDraft,title")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return PRState{}, fmt.Errorf("gh pr view: %w: %s", err, strings.TrimSpace(string(out)))
	}
	var state PRState
	if err := json.Unmarshal(out, &state); err != nil {
		return PRState{}, err
	}
	return state, nil
}

// CommitAll stages everything and creates a commit.
func (s *Service) CommitAll(ctx context.Context, dir, message string) (string, error) {
	if strings.TrimSpace(message) == "" {
		return "", fmt.Errorf("commit message is required")
	}
	if _, err := s.run(ctx, dir, "add", "-A"); err != nil {
		return "", err
	}
	if _, err := s.run(ctx, dir, "commit", "-m", message); err != nil {
		return "", err
	}
	out, err := s.run(ctx, dir, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Push pushes the current branch to origin.
func (s *Service) Push(ctx context.Context, dir string) error {
	_, err := s.run(ctx, dir, "push", "-u", "origin", "HEAD")
	return err
}

// Ship commits when needed and pushes the branch.
func (s *Service) Ship(ctx context.Context, dir, message string) (Status, error) {
	status, err := s.Inspect(ctx, dir)
	if err != nil {
		return status, err
	}
	if !status.Clean {
		if _, err := s.CommitAll(ctx, dir, message); err != nil {
			return status, err
		}
	}
	if err := s.Push(ctx, dir); err != nil {
		return status, err
	}
	return s.Inspect(ctx, dir)
}

// CreatePR opens a pull request for the current branch with gh.
func (s *Service) CreatePR(ctx context.Context, dir string) (string, error) {
	status, err := s.Inspect(ctx, dir)
	if err != nil {
		return "", err
	}
	if remote, _ := s.remoteURL(ctx, dir); isGitLabRemote(remote) {
		return s.createGitLabMR(ctx, dir, status.Branch)
	} else if isBitbucketRemote(remote) {
		return s.createBitbucketPR(ctx, dir, status.Branch)
	}
	runCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, s.GH, "pr", "create", "--fill", "--head", status.Branch)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh pr create: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func (s *Service) remoteURL(ctx context.Context, dir string) (string, error) {
	out, err := s.run(ctx, dir, "remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func isGitLabRemote(remote string) bool {
	return strings.Contains(strings.ToLower(remote), "gitlab")
}

func (s *Service) createGitLabMR(ctx context.Context, dir, branch string) (string, error) {
	runCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, s.Glab, "mr", "create", "--fill", "--yes", "--source-branch", branch)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("glab mr create: %w: %s", err, strings.TrimSpace(string(out)))
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.Contains(line, "http") {
			return line, nil
		}
	}
	return strings.TrimSpace(string(out)), nil
}

func (s *Service) gitLabMRStatus(ctx context.Context, dir string) (PRState, error) {
	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, s.Glab, "mr", "view", "--output", "json")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return PRState{}, fmt.Errorf("glab mr view: %w: %s", err, strings.TrimSpace(string(out)))
	}
	var raw struct {
		URL   string `json:"web_url"`
		State string `json:"state"`
		Draft bool   `json:"draft"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return PRState{}, err
	}
	return PRState{URL: raw.URL, State: strings.ToLower(raw.State), Draft: raw.Draft, Title: raw.Title}, nil
}

func isBitbucketRemote(remote string) bool {
	return strings.Contains(strings.ToLower(remote), "bitbucket.org")
}

func bitbucketRepo(remote string) (string, string, bool) {
	if !isBitbucketRemote(remote) {
		return "", "", false
	}
	value := strings.TrimSpace(remote)
	for _, prefix := range []string{"git@bitbucket.org:", "https://bitbucket.org/", "http://bitbucket.org/", "ssh://git@bitbucket.org/"} {
		if strings.HasPrefix(value, prefix) {
			value = strings.TrimPrefix(value, prefix)
			break
		}
	}
	value = strings.TrimSuffix(value, ".git")
	parts := strings.Split(strings.Trim(value, "/"), "/")
	if len(parts) < 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func (s *Service) createBitbucketPR(ctx context.Context, dir, branch string) (string, error) {
	workspace, repo, ok := bitbucketRepo(mustRemote(ctx, s, dir))
	if !ok {
		return "", fmt.Errorf("could not parse Bitbucket repository")
	}
	if s.BitbucketUser == "" || s.BitbucketPass == "" {
		return "", fmt.Errorf("Bitbucket MR requires BITBUCKET_USERNAME and BITBUCKET_APP_PASSWORD")
	}
	body, _ := json.Marshal(map[string]any{
		"title":       "Merge " + branch,
		"description": "Created by Orca from session worktree.",
		"source":      map[string]any{"branch": map[string]string{"name": branch}},
		"destination": map[string]any{"branch": map[string]string{"name": "main"}},
	})
	endpoint := strings.TrimRight(s.BitbucketAPI, "/") + "/repositories/" + workspace + "/" + repo + "/pullrequests"
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(s.BitbucketUser, s.BitbucketPass)
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		Links struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("bitbucket pull request: status %d", resp.StatusCode)
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result.Links.HTML.Href == "" {
		return "", fmt.Errorf("bitbucket pull request returned no URL")
	}
	return result.Links.HTML.Href, nil
}

func (s *Service) bitbucketPRStatus(ctx context.Context, dir string) (PRState, error) {
	status, err := s.Inspect(ctx, dir)
	if err != nil {
		return PRState{}, err
	}
	workspace, repo, ok := bitbucketRepo(mustRemote(ctx, s, dir))
	if !ok {
		return PRState{}, fmt.Errorf("could not parse Bitbucket repository")
	}
	if s.BitbucketUser == "" || s.BitbucketPass == "" {
		return PRState{}, fmt.Errorf("Bitbucket MR requires BITBUCKET_USERNAME and BITBUCKET_APP_PASSWORD")
	}
	query := fmt.Sprintf(`source.branch.name="%s"`, status.Branch)
	endpoint := strings.TrimRight(s.BitbucketAPI, "/") + "/repositories/" + workspace + "/" + repo + "/pullrequests?q=" + url.QueryEscape(query)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.SetBasicAuth(s.BitbucketUser, s.BitbucketPass)
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return PRState{}, err
	}
	defer resp.Body.Close()
	var result struct {
		Values []struct {
			Title string `json:"title"`
			State string `json:"state"`
			Links struct {
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
			} `json:"links"`
		} `json:"values"`
	}
	if resp.StatusCode >= 300 {
		return PRState{}, fmt.Errorf("bitbucket pull requests: status %d", resp.StatusCode)
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Values) == 0 {
		return PRState{}, fmt.Errorf("no Bitbucket pull request for %s", status.Branch)
	}
	pr := result.Values[0]
	return PRState{URL: pr.Links.HTML.Href, State: strings.ToLower(pr.State), Title: pr.Title}, nil
}

func mustRemote(ctx context.Context, s *Service, dir string) string {
	remote, _ := s.remoteURL(ctx, dir)
	return remote
}
