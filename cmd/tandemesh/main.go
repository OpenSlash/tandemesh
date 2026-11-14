// Command orca is the thin CLI over the Orca control-plane HTTP API.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const version = "0.1.0-dev"

type run struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	AgentID   string `json:"agentId"`
	MachineID string `json:"machineId"`
	Repo      string `json:"repo"`
	Work      struct {
		URL string `json:"url"`
	} `json:"work"`
	CreatedAt time.Time `json:"createdAt"`
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		return
	}
	server := env("ORCA_BASE_URL", "http://127.0.0.1:8080")
	if strings.HasPrefix(args[0], "-server=") {
		server = strings.TrimPrefix(args[0], "-server=")
		args = args[1:]
	}

	switch args[0] {
	case "version":
		fmt.Println("orca", version)
	case "runs":
		listRuns(server)
	case "machines":
		listMachines(server)
	case "run":
		createRun(server, args[1:])
	case "skills":
		skillsCommand(server, args[1:])
	case "session":
		sessionCommand(server, args[1:])
	case "worktree":
		worktreeCommand(server, args[1:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", args[0])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Print(`orca - Agent execution plane CLI

Usage:
  orca version
  orca runs [--status=queued]
  orca machines
  orca run --repo=acme/api --prompt="fix the flaky test" [--agent=claude] [--work-url=...]
  orca skills list [--workspace=ws_default]
  orca skills get --name=<name> | --id=<id>
  orca skills install --file=<path> | --url=<url> [--name=]
  orca skills export --dir=./skills
  orca session list [--workspace=ws_default]
  orca session new --title="Refactor" [--agent=claude] [--mode=chat] [--task=<id>]
  orca session send --session=<id> --prompt="..."
  orca session close --session=<id>
  orca worktree current --run=<runId> [--json]
  orca worktree set --run=<runId> --comment="..." [--workspace-status=in-progress]

Environment:
  ORCA_BASE_URL   control plane URL (default http://127.0.0.1:8080)
  ORCA_TOKEN      API token (reserved for M1)
`)
}

func listRuns(server string) {
	var out struct {
		Items []run `json:"items"`
	}
	getJSON(server+"/api/v1/runs", &out)
	if len(out.Items) == 0 {
		fmt.Println("no runs")
		return
	}
	fmt.Printf("%-28s %-10s %-8s %-10s %s\n", "ID", "STATUS", "AGENT", "MACHINE", "WORK")
	for _, r := range out.Items {
		fmt.Printf("%-28s %-10s %-8s %-10s %s\n", r.ID, r.Status, r.AgentID, r.MachineID, r.Work.URL)
	}
}

func listMachines(server string) {
	var out struct {
		Items []struct {
			ID          string    `json:"id"`
			Name        string    `json:"name"`
			Status      string    `json:"status"`
			Concurrency int       `json:"concurrency"`
			LastSeen    time.Time `json:"lastSeen"`
		} `json:"items"`
	}
	getJSON(server+"/api/v1/machines", &out)
	if len(out.Items) == 0 {
		fmt.Println("no machines")
		return
	}
	fmt.Printf("%-28s %-20s %-8s %-6s %s\n", "ID", "NAME", "STATUS", "SLOTS", "LAST SEEN")
	for _, m := range out.Items {
		fmt.Printf("%-28s %-20s %-8s %-6d %s\n", m.ID, m.Name, m.Status, m.Concurrency, m.LastSeen.Format(time.RFC3339))
	}
}

func createRun(server string, args []string) {
	req := map[string]any{
		"work":    map[string]any{"provider": "manual", "instance": "local", "type": "task", "id": fmt.Sprintf("%d", time.Now().UnixNano())},
		"agentId": "claude",
	}
	for _, a := range args {
		key, val, ok := strings.Cut(strings.TrimPrefix(a, "--"), "=")
		if !ok {
			continue
		}
		switch key {
		case "repo":
			req["repo"] = val
		case "prompt":
			req["prompt"] = val
		case "agent":
			req["agentId"] = val
		case "base":
			req["base"] = val
		case "work-url":
			req["work"].(map[string]any)["url"] = val
		case "work-id":
			req["work"].(map[string]any)["id"] = val
		}
	}
	raw, _ := json.Marshal(req)
	resp, err := http.Post(server+"/api/v1/runs", "application/json", bytes.NewReader(raw))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
	if resp.StatusCode >= 300 {
		os.Exit(1)
	}
}

func getJSON(url string, v any) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "status %d: %s\n", resp.StatusCode, body)
		os.Exit(1)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		fmt.Fprintln(os.Stderr, "decode:", err)
		os.Exit(1)
	}
}

type cliWorktree struct {
	ID                string `json:"id"`
	RunID             string `json:"runId"`
	TaskID            string `json:"taskId"`
	Branch            string `json:"branch"`
	Path              string `json:"path"`
	Status            string `json:"status"`
	CheckpointComment string `json:"checkpointComment"`
	WorkspaceStatus   string `json:"workspaceStatus"`
}

func worktreeCommand(server string, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: orca worktree current|set --run=<runId>")
		os.Exit(2)
	}
	runID := ""
	comment := ""
	workspaceStatus := ""
	asJSON := false
	for _, a := range args[1:] {
		if a == "--json" {
			asJSON = true
			continue
		}
		key, val, ok := strings.Cut(strings.TrimPrefix(a, "--"), "=")
		if !ok {
			continue
		}
		switch key {
		case "run":
			runID = val
		case "comment":
			comment = val
		case "workspace-status":
			workspaceStatus = val
		}
	}
	if runID == "" {
		fmt.Fprintln(os.Stderr, "error: --run is required")
		os.Exit(2)
	}
	switch args[0] {
	case "current":
		var wt cliWorktree
		getJSON(server+"/api/v1/runs/"+runID+"/worktree", &wt)
		if asJSON {
			raw, _ := json.Marshal(wt)
			fmt.Println(string(raw))
			return
		}
		fmt.Printf("%s\n  branch: %s\n  path:   %s\n  status: %s\n  note:   %s\n", wt.ID, wt.Branch, wt.Path, wt.Status, wt.CheckpointComment)
	case "set":
		raw, _ := json.Marshal(map[string]string{"comment": comment, "workspaceStatus": workspaceStatus})
		resp, err := http.Post(server+"/api/v1/runs/"+runID+"/worktree/checkpoint", "application/json", bytes.NewReader(raw))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 300 {
			fmt.Fprintf(os.Stderr, "status %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		fmt.Println(string(body))
	default:
		fmt.Fprintf(os.Stderr, "unknown worktree command %q\n", args[0])
		os.Exit(2)
	}
}

type cliSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Source      string `json:"source"`
	Enabled     bool   `json:"enabled"`
}

func skillsCommand(server string, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: orca skills list|get|install|export")
		os.Exit(2)
	}
	opts := map[string]string{}
	for _, a := range args[1:] {
		key, val, ok := strings.Cut(strings.TrimPrefix(a, "--"), "=")
		if ok {
			opts[key] = val
		}
	}
	ws := envDefault(opts["workspace"], "ws_default")
	switch args[0] {
	case "list":
		var out struct {
			Items []cliSkill `json:"items"`
		}
		getJSON(server+"/api/v1/skills?workspace="+ws, &out)
		if len(out.Items) == 0 {
			fmt.Println("no skills")
			return
		}
		fmt.Printf("%-30s %-10s %-8s %-8s %s\n", "ID", "VERSION", "SOURCE", "ENABLED", "NAME")
		for _, sk := range out.Items {
			fmt.Printf("%-30s %-10s %-8s %-8v %s\n", sk.ID, sk.Version, sk.Source, sk.Enabled, sk.Name)
		}
	case "get":
		id := opts["id"]
		if id == "" {
			var out struct {
				Items []cliSkill `json:"items"`
			}
			getJSON(server+"/api/v1/skills?workspace="+ws, &out)
			for _, sk := range out.Items {
				if strings.EqualFold(sk.Name, opts["name"]) {
					id = sk.ID
					break
				}
			}
		}
		if id == "" {
			fmt.Fprintln(os.Stderr, "error: skill not found; use --id or --name")
			os.Exit(1)
		}
		resp, err := http.Get(server + "/api/v1/skills/" + id + "/raw")
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			data, _ := io.ReadAll(resp.Body)
			fmt.Fprintf(os.Stderr, "status %d: %s\n", resp.StatusCode, data)
			os.Exit(1)
		}
		body, _ := io.ReadAll(resp.Body)
		fmt.Print(string(body))
	case "install":
		body := map[string]any{
			"workspaceId": ws,
			"name":        opts["name"],
			"version":     opts["version"],
			"sourceUrl":   opts["url"],
		}
		if path := opts["file"]; path != "" {
			data, err := os.ReadFile(path)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			body["body"] = string(data)
			body["source"] = "file"
		}
		var skill cliSkill
		postJSONCLI(server+"/api/v1/skills/install", body, &skill)
		fmt.Printf("installed %s (%s)\n", skill.Name, skill.ID)
	case "export":
		dir := opts["dir"]
		if dir == "" {
			fmt.Fprintln(os.Stderr, "error: --dir is required")
			os.Exit(2)
		}
		var out struct {
			Items []struct {
				Name     string `json:"name"`
				Markdown string `json:"markdown"`
			} `json:"items"`
		}
		getJSON(server+"/api/v1/skills/export?workspace="+ws, &out)
		for _, sk := range out.Items {
			target := filepath.Join(dir, sk.Name)
			if err := os.MkdirAll(target, 0o755); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			if err := os.WriteFile(filepath.Join(target, "SKILL.md"), []byte(sk.Markdown), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
		}
		fmt.Printf("exported %d skills to %s\n", len(out.Items), dir)
	default:
		fmt.Fprintf(os.Stderr, "unknown skills command %q\n", args[0])
		os.Exit(2)
	}
}

type cliSession struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	AgentID   string `json:"agentId"`
	Mode      string `json:"mode"`
	Status    string `json:"status"`
	TaskID    string `json:"taskId"`
	UpdatedAt string `json:"updatedAt"`
}

func sessionCommand(server string, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: orca session list|new|send|close")
		os.Exit(2)
	}
	opts := map[string]string{}
	for _, a := range args[1:] {
		key, val, ok := strings.Cut(strings.TrimPrefix(a, "--"), "=")
		if ok {
			opts[key] = val
		}
	}
	switch args[0] {
	case "list":
		ws := opts["workspace"]
		if ws == "" {
			ws = "ws_default"
		}
		var out struct {
			Sessions []cliSession `json:"sessions"`
		}
		getJSON(server+"/api/v1/sessions?workspaceId="+ws, &out)
		if len(out.Sessions) == 0 {
			fmt.Println("no sessions")
			return
		}
		fmt.Printf("%-28s %-10s %-8s %-8s %s\n", "ID", "STATUS", "MODE", "AGENT", "TITLE")
		for _, s := range out.Sessions {
			fmt.Printf("%-28s %-10s %-8s %-8s %s\n", s.ID, s.Status, s.Mode, s.AgentID, s.Title)
		}
	case "new":
		body := map[string]any{
			"workspaceId": envDefault(opts["workspace"], "ws_default"),
			"title":       opts["title"],
			"agentId":     envDefault(opts["agent"], "claude"),
			"mode":        envDefault(opts["mode"], "chat"),
		}
		if opts["task"] != "" {
			body["taskId"] = opts["task"]
		}
		var sess cliSession
		postJSONCLI(server+"/api/v1/sessions", body, &sess)
		fmt.Printf("%s\t%s\t%s\n", sess.ID, sess.Mode, sess.Title)
	case "send":
		if opts["session"] == "" || opts["prompt"] == "" {
			fmt.Fprintln(os.Stderr, "error: --session and --prompt are required")
			os.Exit(2)
		}
		var run struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		postJSONCLI(server+"/api/v1/sessions/"+opts["session"]+"/runs", map[string]any{"prompt": opts["prompt"]}, &run)
		fmt.Printf("%s\t%s\n", run.ID, run.Status)
	case "close":
		if opts["session"] == "" {
			fmt.Fprintln(os.Stderr, "error: --session is required")
			os.Exit(2)
		}
		req, _ := http.NewRequest(http.MethodPost, server+"/api/v1/sessions/"+opts["session"]+"/close", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			fmt.Fprintf(os.Stderr, "status %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		fmt.Println("closed")
	default:
		fmt.Fprintf(os.Stderr, "unknown session command %q\n", args[0])
		os.Exit(2)
	}
}

func envDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func postJSONCLI(url string, body any, out any) {
	raw, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "status %d: %s\n", resp.StatusCode, data)
		os.Exit(1)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			fmt.Fprintln(os.Stderr, "decode:", err)
			os.Exit(1)
		}
	}
}
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
