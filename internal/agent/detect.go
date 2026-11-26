package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// Launcher sources, highest precedence first.
const (
	SourceFlag     = "flag"
	SourceCustom   = "custom"
	SourceRegistry = "registry"
	SourcePath     = "path"
)

// launcherFallbacks lists older adapter commands that still work when the
// default launcher is not installed.
var launcherFallbacks = map[string][]string{
	"claude": {"claude-code-acp"},
}

// DetectOptions configures DetectRuntimes. Zero values use the real system.
type DetectOptions struct {
	// Installs are ACP Registry installs, oldest first.
	Installs []ACPInstall
	// Custom holds launchers from an operator-supplied backends file.
	Custom map[string]ACPBackendSpec
	// Overrides holds launchers set explicitly on the command line.
	Overrides map[string]ACPBackendSpec
	LookPath  func(string) (string, error)
	Version   func(ctx context.Context, path string) string
}

// Detection is what a host can run and why.
type Detection struct {
	Statuses []domain.RuntimeStatus
	// Backends holds the resolved launcher of every ready backend.
	Backends map[string]ACPBackendSpec
}

type launcherCandidate struct {
	spec     ACPBackendSpec
	source   string
	registry *ACPInstall
}

// DetectRuntimes reports every known backend. A backend is ready only when
// the ACP launcher that a run would start resolves; an installed CLI without
// its adapter is reported separately, because runs would fail.
func DetectRuntimes(ctx context.Context, opts DetectOptions) Detection {
	lookPath := opts.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	version := opts.Version
	if version == nil {
		version = cliVersion
	}
	installs := map[string]ACPInstall{}
	for _, install := range opts.Installs {
		installs[backendForRegistry(install.ID)] = install
	}

	det := Detection{Backends: map[string]ACPBackendSpec{}}
	cliPaths := map[int]string{}
	known := map[string]bool{}
	for _, runtime := range RuntimeCatalog {
		known[runtime.ID] = true
		st := domain.RuntimeStatus{ID: runtime.ID, Name: runtime.Name}
		cliName := runtime.Command
		if cliName == "" {
			cliName = runtime.ID
		}
		cliPath, cliErr := lookPath(cliName)
		if cliErr == nil {
			st.CLI = cliPath
		}
		if !runtime.ACP {
			st.Status = domain.RuntimeMissing
			if cliErr == nil {
				st.Status = domain.RuntimeUnsupported
			}
		} else {
			candidates := launcherCandidates(runtime.ID, opts, installs)
			if spec, c, ok := firstResolved(candidates, lookPath); ok {
				st.Status = domain.RuntimeReady
				st.Launcher = spec.Command
				st.Source = c.source
				if c.registry != nil {
					st.RegistryID, st.RegistryVersion = c.registry.ID, c.registry.Version
				}
				det.Backends[runtime.ID] = spec
			} else {
				st.Status = domain.RuntimeMissing
				if cliErr == nil {
					st.Status = domain.RuntimeAdapterMissing
				}
				// Name the launcher users are expected to install.
				st.Launcher = DefaultACPBackends[runtime.ID].Command
			}
		}
		if st.CLI != "" {
			cliPaths[len(det.Statuses)] = st.CLI
		}
		det.Statuses = append(det.Statuses, st)
	}

	// Backends outside the catalog: custom launchers, explicit flags and
	// registry agents that do not alias a catalog entry.
	extras := map[string]launcherCandidate{}
	for id, spec := range opts.Custom {
		extras[id] = launcherCandidate{spec: spec, source: SourceCustom}
	}
	for id, install := range installs {
		if _, ok := extras[id]; !ok {
			install := install
			extras[id] = launcherCandidate{spec: ACPBackendSpec{Command: install.Command, Args: install.Args}, source: SourceRegistry, registry: &install}
		}
	}
	for id, spec := range opts.Overrides {
		extras[id] = launcherCandidate{spec: spec, source: SourceFlag}
	}
	ids := make([]string, 0, len(extras))
	for id := range extras {
		if !known[id] {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		c := extras[id]
		st := domain.RuntimeStatus{ID: id, Name: id, Status: domain.RuntimeMissing, Launcher: c.spec.Command, Source: c.source}
		if c.registry != nil {
			st.RegistryID, st.RegistryVersion, st.Version = c.registry.ID, c.registry.Version, c.registry.Version
		}
		if spec, _, ok := firstResolved([]launcherCandidate{c}, lookPath); ok {
			st.Status = domain.RuntimeReady
			st.Launcher = spec.Command
			det.Backends[id] = spec
		}
		det.Statuses = append(det.Statuses, st)
	}

	// --version can take a second per CLI; probe them in parallel.
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i, path := range cliPaths {
		wg.Add(1)
		go func(i int, path string) {
			defer wg.Done()
			v := version(ctx, path)
			mu.Lock()
			det.Statuses[i].Version = v
			mu.Unlock()
		}(i, path)
	}
	wg.Wait()
	// Without a CLI, the registry install's version is the best we have.
	for i := range det.Statuses {
		if det.Statuses[i].Version == "" {
			det.Statuses[i].Version = det.Statuses[i].RegistryVersion
		}
	}
	return det
}

// launcherCandidates lists the launchers for a catalog backend in the order
// they are tried: explicit flag, registry install, built-in default, then
// legacy adapter names.
func launcherCandidates(id string, opts DetectOptions, installs map[string]ACPInstall) []launcherCandidate {
	var out []launcherCandidate
	if spec, ok := opts.Overrides[id]; ok {
		out = append(out, launcherCandidate{spec: spec, source: SourceFlag})
	}
	if install, ok := installs[id]; ok {
		install := install
		out = append(out, launcherCandidate{spec: ACPBackendSpec{Command: install.Command, Args: install.Args}, source: SourceRegistry, registry: &install})
	}
	if spec, ok := DefaultACPBackends[id]; ok {
		out = append(out, launcherCandidate{spec: spec, source: SourcePath})
	}
	for _, command := range launcherFallbacks[id] {
		out = append(out, launcherCandidate{spec: ACPBackendSpec{Command: command}, source: SourcePath})
	}
	return out
}

// firstResolved returns the first candidate whose command exists, with the
// command replaced by its absolute path.
func firstResolved(candidates []launcherCandidate, lookPath func(string) (string, error)) (ACPBackendSpec, launcherCandidate, bool) {
	for _, c := range candidates {
		if path, ok := resolveCommand(c.spec.Command, lookPath); ok {
			spec := c.spec
			spec.Command = path
			return spec, c, true
		}
	}
	return ACPBackendSpec{}, launcherCandidate{}, false
}

func resolveCommand(command string, lookPath func(string) (string, error)) (string, bool) {
	if command == "" {
		return "", false
	}
	if filepath.IsAbs(command) {
		info, err := os.Stat(command)
		return command, err == nil && !info.IsDir()
	}
	path, err := lookPath(command)
	return path, err == nil
}

func backendForRegistry(id string) string {
	if backend, ok := RegistryAliases[id]; ok {
		return backend
	}
	return id
}

func cliVersion(ctx context.Context, path string) string {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	// A CLI that cannot report its version is still installed.
	raw, _ := exec.CommandContext(ctx, path, "--version").Output()
	version := strings.TrimSpace(strings.SplitN(string(raw), "\n", 2)[0])
	if version == "" {
		version = "installed"
	}
	return version
}
