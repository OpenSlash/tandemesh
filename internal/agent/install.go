package agent

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type ACPInstall struct {
	ID       string   `json:"id"`
	Version  string   `json:"version"`
	Dir      string   `json:"dir"`
	Command  string   `json:"command"`
	Args     []string `json:"args,omitempty"`
	Verified bool     `json:"verified"`
}
type acpManifest struct {
	ID          string   `json:"id"`
	Version     string   `json:"version"`
	Command     string   `json:"command"`
	Args        []string `json:"args,omitempty"`
	Verified    bool     `json:"verified"`
	InstalledAt string   `json:"installedAt"`
}

func platformKey() string {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	}
	return runtime.GOOS + "-" + arch
}

// InstallACP installs a registry agent into an isolated home. Binary
// distributions are checksum-verified; npx and uvx distributions are installed
// into a private prefix with npm or uv.
func InstallACP(ctx context.Context, id, home string) (*ACPInstall, error) {
	snapshot, err := FetchACPRegistry(ctx)
	if err != nil {
		return nil, err
	}
	var entry *ACPRegistryEntry
	for i := range snapshot.Items {
		if snapshot.Items[i].ID == id {
			entry = &snapshot.Items[i]
			break
		}
	}
	if entry == nil {
		return nil, fmt.Errorf("registry agent %q not found", id)
	}
	if npx, ok := entry.Distribution["npx"].(map[string]any); ok {
		return installNPM(ctx, id, entry.Version, npx, home)
	}
	if uvx, ok := entry.Distribution["uvx"].(map[string]any); ok {
		return installUV(ctx, id, entry.Version, uvx, home)
	}
	binary, ok := entry.Distribution["binary"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("agent %q has no binary or npx distribution", id)
	}
	raw, ok := binary[platformKey()].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("agent %q has no %s binary", id, platformKey())
	}
	archiveURL, _ := raw["archive"].(string)
	command, _ := raw["cmd"].(string)
	checksum, _ := raw["sha256"].(string)
	if archiveURL == "" || command == "" {
		return nil, fmt.Errorf("registry agent %q has an incomplete binary entry", id)
	}
	if !strings.HasPrefix(archiveURL, "https://") {
		return nil, fmt.Errorf("registry archive must use https")
	}
	if home == "" {
		userHome, _ := os.UserHomeDir()
		home = filepath.Join(userHome, ".tandemesh", "acp")
	}
	target := filepath.Join(home, id, entry.Version)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp("", "tandemesh-acp-*")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	client := &http.Client{Timeout: 5 * time.Minute}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: %s", resp.Status)
	}
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hash), resp.Body); err != nil {
		return nil, err
	}
	tmp.Close()
	verified := checksum != ""
	if verified {
		sum := hex.EncodeToString(hash.Sum(nil))
		if !strings.EqualFold(sum, checksum) {
			return nil, fmt.Errorf("checksum mismatch: got %s", sum)
		}
	}
	if err := extractArchive(tmpPath, archiveURL, target); err != nil {
		return nil, err
	}
	commandPath := filepath.Join(target, filepath.FromSlash(strings.TrimPrefix(command, "./")))
	if _, err := os.Stat(commandPath); err != nil {
		return nil, fmt.Errorf("installed command missing: %s", commandPath)
	}
	result := &ACPInstall{ID: id, Version: entry.Version, Dir: target, Command: commandPath, Verified: verified}
	writeManifest(target, result)
	return result, nil
}

func installUV(ctx context.Context, id, version string, uvx map[string]any, home string) (*ACPInstall, error) {
	pkg, _ := uvx["package"].(string)
	if pkg == "" {
		return nil, fmt.Errorf("agent %q has no Python package", id)
	}
	if home == "" {
		userHome, _ := os.UserHomeDir()
		home = filepath.Join(userHome, ".tandemesh", "acp")
	}
	target := filepath.Join(home, id, version)
	venv := filepath.Join(target, ".venv")
	if err := os.MkdirAll(target, 0o755); err != nil {
		return nil, err
	}
	uv, err := exec.LookPath("uv")
	if err != nil {
		return nil, fmt.Errorf("uv is required for uvx distribution: %w", err)
	}
	run := func(args ...string) error {
		cmd := exec.CommandContext(ctx, uv, args...)
		cmd.Env = os.Environ()
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("uv %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	if err := run("venv", venv); err != nil {
		return nil, err
	}
	python := filepath.Join(venv, "bin", "python")
	binDir := filepath.Join(venv, "bin")
	if runtime.GOOS == "windows" {
		python = filepath.Join(venv, "Scripts", "python.exe")
		binDir = filepath.Join(venv, "Scripts")
	}
	if err := run("pip", "install", "--python", python, pipRequirement(pkg, version)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return nil, err
	}
	command := ""
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if strings.HasPrefix(name, "python") || strings.HasPrefix(name, "activate") || strings.HasSuffix(name, ".py") {
			continue
		}
		command = filepath.Join(binDir, entry.Name())
		break
	}
	if command == "" {
		return nil, fmt.Errorf("uv install produced no executable")
	}
	args := make([]string, 0)
	if raw, ok := uvx["args"].([]any); ok {
		for _, item := range raw {
			if value, ok := item.(string); ok {
				args = append(args, value)
			}
		}
	}
	result := &ACPInstall{ID: id, Version: version, Dir: target, Command: command, Args: args}
	writeManifest(target, result)
	return result, nil
}

func installNPM(ctx context.Context, id, version string, npx map[string]any, home string) (*ACPInstall, error) {
	pkg, _ := npx["package"].(string)
	if pkg == "" {
		return nil, fmt.Errorf("agent %q has no npm package", id)
	}
	if home == "" {
		userHome, _ := os.UserHomeDir()
		home = filepath.Join(userHome, ".tandemesh", "acp")
	}
	target := filepath.Join(home, id, version)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return nil, err
	}
	npm, err := exec.LookPath("npm")
	if err != nil {
		return nil, fmt.Errorf("npm is required for npx distribution: %w", err)
	}
	spec := npmSpec(pkg, version)
	cmd := exec.CommandContext(ctx, npm, "install", "--prefix", target, "--no-save", "--ignore-scripts", "--no-audit", "--no-fund", spec)
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("npm install %s: %w: %s", spec, err, strings.TrimSpace(string(out)))
	}
	bin, err := npmBin(target, npmPackageName(spec))
	if err != nil {
		return nil, err
	}
	command := filepath.Join(target, "node_modules", ".bin", bin)
	args := make([]string, 0)
	if raw, ok := npx["args"].([]any); ok {
		for _, item := range raw {
			if value, ok := item.(string); ok {
				args = append(args, value)
			}
		}
	}
	result := &ACPInstall{ID: id, Version: version, Dir: target, Command: command, Args: args}
	writeManifest(target, result)
	return result, nil
}

// npmPackageName strips a version from an npm spec: "@scope/pkg@1.0" and
// "pkg@1.0" become "@scope/pkg" and "pkg".
func npmPackageName(spec string) string {
	if i := strings.LastIndex(spec, "@"); i > 0 {
		return spec[:i]
	}
	return spec
}

// npmSpec pins pkg to version unless the registry already did.
func npmSpec(pkg, version string) string {
	if npmPackageName(pkg) != pkg || version == "" {
		return pkg
	}
	return pkg + "@" + version
}

// pipRequirement turns a uvx package ("pkg", "pkg@1.0" or "pkg==1.0") into
// a pinned pip requirement.
func pipRequirement(pkg, version string) string {
	if strings.Contains(pkg, "==") {
		return pkg
	}
	if i := strings.LastIndex(pkg, "@"); i > 0 {
		return pkg[:i] + "==" + pkg[i+1:]
	}
	if version == "" {
		return pkg
	}
	return pkg + "==" + version
}

// npmBin returns the executable an npm package declares. Dependencies can
// add their own entries to node_modules/.bin, so the directory alone is not
// enough to pick one.
func npmBin(prefix, name string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(prefix, "node_modules", filepath.FromSlash(name), "package.json"))
	if err != nil {
		return "", fmt.Errorf("read %s package.json: %w", name, err)
	}
	var manifest struct {
		Name string          `json:"name"`
		Bin  json.RawMessage `json:"bin"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return "", fmt.Errorf("parse %s package.json: %w", name, err)
	}
	base := manifest.Name
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	var single string
	if json.Unmarshal(manifest.Bin, &single) == nil && single != "" {
		// A string bin is installed under the unscoped package name.
		return base, nil
	}
	var bins map[string]string
	if err := json.Unmarshal(manifest.Bin, &bins); err != nil || len(bins) == 0 {
		return "", fmt.Errorf("package %s declares no executable", name)
	}
	if _, ok := bins[base]; ok {
		return base, nil
	}
	names := make([]string, 0, len(bins))
	for key := range bins {
		names = append(names, key)
	}
	sort.Strings(names)
	return names[0], nil
}

func extractArchive(path, source, target string) error {
	if strings.HasSuffix(strings.ToLower(source), ".zip") {
		reader, err := zip.OpenReader(path)
		if err != nil {
			return err
		}
		defer reader.Close()
		for _, file := range reader.File {
			if err := extractPath(target, file.Name, file.Mode(), file.Open); err != nil {
				return err
			}
		}
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	tarReader := tar.NewReader(gz)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := extractPath(target, header.Name, os.FileMode(header.Mode), func() (io.ReadCloser, error) { return io.NopCloser(tarReader), nil }); err != nil {
			return err
		}
	}
}

func extractPath(root, name string, mode os.FileMode, open func() (io.ReadCloser, error)) error {
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return fmt.Errorf("unsafe archive path: %s", name)
	}
	dest := filepath.Join(root, clean)
	if !strings.HasPrefix(dest, filepath.Clean(root)+string(os.PathSeparator)) && dest != filepath.Clean(root) {
		return fmt.Errorf("archive path escapes target: %s", name)
	}
	if mode.IsDir() {
		return os.MkdirAll(dest, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := open()
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
func writeManifest(dir string, result *ACPInstall) {
	manifest := acpManifest{ID: result.ID, Version: result.Version, Command: result.Command, Args: result.Args, Verified: result.Verified, InstalledAt: time.Now().UTC().Format(time.RFC3339)}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err == nil {
		_ = os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0o644)
	}
}

func acpHome(home string) string {
	if home != "" {
		return home
	}
	userHome, _ := os.UserHomeDir()
	return filepath.Join(userHome, ".tandemesh", "acp")
}

func UninstallACP(id, version, home string) error {
	if id == "" || version == "" {
		return fmt.Errorf("id and version are required")
	}
	target := filepath.Join(acpHome(home), id, version)
	clean := filepath.Clean(target)
	base := filepath.Clean(acpHome(home))
	if !strings.HasPrefix(clean, base+string(os.PathSeparator)) {
		return fmt.Errorf("invalid install path")
	}
	if err := os.RemoveAll(clean); err != nil {
		return err
	}
	// Drop the agent directory once its last version is gone; os.Remove
	// fails harmlessly while other versions remain.
	_ = os.Remove(filepath.Dir(clean))
	return nil
}
