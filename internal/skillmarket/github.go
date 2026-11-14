package skillmarket

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	skillslib "github.com/OpenSlash/tandemesh/internal/skills"
)

const cacheTTL = 5 * time.Minute

var cache struct {
	sync.Mutex
	key     string
	at      time.Time
	entries []Entry
}

// Fetch merges the optional JSON catalog URL and GitHub repositories, caching
// the result briefly to stay inside GitHub's anonymous rate limit.
func Fetch(ctx context.Context, catalogURL string, repos []string, token string) ([]Entry, error) {
	key := catalogURL + "|" + strings.Join(repos, ",") + "|" + fmt.Sprintf("%t", token != "")
	cache.Lock()
	if cache.key == key && time.Since(cache.at) < cacheTTL {
		entries := append([]Entry(nil), cache.entries...)
		cache.Unlock()
		return entries, nil
	}
	cache.Unlock()

	client := &http.Client{Timeout: 20 * time.Second}
	entries := []Entry{}
	if strings.TrimSpace(catalogURL) != "" {
		remote, err := fetchCatalog(ctx, client, catalogURL)
		if err != nil {
			return nil, err
		}
		entries = append(entries, remote...)
	}
	for _, repo := range repos {
		repo = strings.TrimSpace(repo)
		if repo == "" {
			continue
		}
		remote, err := fetchGitHubRepo(ctx, client, repo, token)
		if err != nil {
			continue
		}
		entries = append(entries, remote...)
	}

	cache.Lock()
	cache.key, cache.at, cache.entries = key, time.Now(), entries
	cache.Unlock()
	return append([]Entry(nil), entries...), nil
}

func fetchCatalog(ctx context.Context, client *http.Client, catalogURL string) ([]Entry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, catalogURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("catalog returned %s", resp.Status)
	}
	var entries []Entry
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&entries); err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].Source == "" {
			entries[i].Source = "url"
		}
		if entries[i].SourceURL == "" {
			entries[i].SourceURL = catalogURL
		}
	}
	return entries, nil
}

func fetchGitHubRepo(ctx context.Context, client *http.Client, repo, token string) ([]Entry, error) {
	var meta struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := githubJSON(ctx, client, "https://api.github.com/repos/"+repo, token, &meta); err != nil {
		return nil, err
	}
	if meta.DefaultBranch == "" {
		return nil, fmt.Errorf("no default branch for %s", repo)
	}
	var tree struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
	}
	endpoint := "https://api.github.com/repos/" + repo + "/git/trees/" + url.PathEscape(meta.DefaultBranch) + "?recursive=1"
	if err := githubJSON(ctx, client, endpoint, token, &tree); err != nil {
		return nil, err
	}

	entries := []Entry{}
	for _, item := range tree.Tree {
		if item.Type != "blob" || !strings.EqualFold(path.Base(item.Path), "SKILL.md") {
			continue
		}
		raw, err := fetchRaw(ctx, client, repo, meta.DefaultBranch, item.Path)
		if err != nil {
			continue
		}
		skill, err := skillslib.Parse(raw)
		if err != nil || strings.TrimSpace(skill.Name) == "" {
			continue
		}
		sum := sha1.Sum([]byte(repo + "/" + meta.DefaultBranch + "/" + item.Path))
		entries = append(entries, Entry{
			ID:          "gh_" + hex.EncodeToString(sum[:8]),
			Name:        skill.Name,
			Description: skill.Description,
			Version:     skill.Version,
			Author:      repo,
			Tags:        skill.Tags,
			Body:        skill.Body,
			Source:      "github",
			SourceURL:   rawURL(repo, meta.DefaultBranch, item.Path),
		})
		if len(entries) >= 100 {
			break
		}
	}
	return entries, nil
}

func githubJSON(ctx context.Context, client *http.Client, endpoint, token string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "orca-skill-market")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github returned %s", resp.Status)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(out)
}

func fetchRaw(ctx context.Context, client *http.Client, repo, branch, filePath string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL(repo, branch, filePath), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("raw returned %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 512<<10))
}

func rawURL(repo, branch, filePath string) string {
	segments := strings.Split(filePath, "/")
	for i := range segments {
		segments[i] = url.PathEscape(segments[i])
	}
	return "https://raw.githubusercontent.com/" + repo + "/" + url.PathEscape(branch) + "/" + strings.Join(segments, "/")
}
