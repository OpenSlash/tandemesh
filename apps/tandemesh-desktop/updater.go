package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"changeme/updatesig"
)

// version is set at build time with -ldflags -X. A plain `go build` reports
// "dev" and never checks for updates on its own.
var version = "dev"

// updatePublicKey verifies release manifests. The matching private key lives
// only in the release pipeline's TANDEMESH_UPDATE_KEY secret.
var updatePublicKey = "VVIEPM9Lw1JOnvNcOkeRRUNMWefUDBQKaCrkg3UVzYU="

// Releases are published on GitHub. The latest/download path always resolves
// to the newest non-prerelease, so prereleases can be tested without reaching
// everyone, and plain file downloads avoid the API's rate limit.
const releaseDownloadBase = "https://github.com/OpenSlash/tandemesh/releases/latest/download/"

const updateInterval = 6 * time.Hour

type updateManifest = updatesig.Manifest

// updateManifestURL returns where to look for updates, or "" when this build
// cannot update itself. TANDEMESH_UPDATE_URL points at a mirror or a test feed.
func updateManifestURL() string {
	if url := strings.TrimSpace(os.Getenv("TANDEMESH_UPDATE_URL")); url != "" {
		return url
	}
	if version == "dev" || !selfUpdateSupported {
		return ""
	}
	return releaseDownloadBase + "update-" + runtime.GOOS + "-" + runtime.GOARCH + ".json"
}

func compareVersions(a, b string) int {
	pa := strings.Split(strings.TrimPrefix(a, "v"), ".")
	pb := strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < 3; i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(strings.Split(pa[i], "-")[0])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(strings.Split(pb[i], "-")[0])
		}
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}

func fetchUpdate(ctx context.Context, url string) (*updateManifest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		// No release carries a manifest for this platform yet.
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update manifest: %s", resp.Status)
	}
	var manifest updateManifest
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode update manifest: %w", err)
	}
	if err := updatesig.Verify(manifest, updatePublicKey); err != nil {
		return nil, err
	}
	if compareVersions(manifest.Version, version) <= 0 {
		return nil, nil
	}
	return &manifest, nil
}

// downloadUpdate saves the package into a fresh temporary directory and
// rejects it unless its SHA-256 matches the manifest.
func downloadUpdate(ctx context.Context, manifest *updateManifest) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifest.URL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download: %s", resp.Status)
	}
	dir, err := os.MkdirTemp("", "tandemesh-update-")
	if err != nil {
		return "", err
	}
	name := path.Base(req.URL.Path)
	if name == "" || name == "." || name == "/" {
		name = "update"
	}
	target := filepath.Join(dir, name)
	file, err := os.Create(target)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(file, hash), resp.Body); err != nil {
		file.Close()
		return "", fmt.Errorf("download: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	if sum := hex.EncodeToString(hash.Sum(nil)); !strings.EqualFold(sum, manifest.SHA256) {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("checksum mismatch")
	}
	return target, nil
}

var updateChecking atomic.Bool

// checkForUpdates asks the user to update when a newer release exists. A
// manual check also reports "up to date" and errors; a background check
// stays quiet about both.
func checkForUpdates(app *application.App, manual bool) {
	if !updateChecking.CompareAndSwap(false, true) {
		return
	}
	defer updateChecking.Store(false)
	url := updateManifestURL()
	if url == "" {
		if manual {
			app.Dialog.Info().SetTitle("Tandemesh " + version).SetMessage("This build does not update itself.").Show()
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	manifest, err := fetchUpdate(ctx, url)
	if err != nil {
		log.Printf("update check: %v", err)
		if manual {
			app.Dialog.Error().SetTitle("Update check failed").SetMessage(err.Error()).Show()
		}
		return
	}
	if manifest == nil {
		if manual {
			app.Dialog.Info().SetTitle("Tandemesh " + version).SetMessage("You are on the latest version.").Show()
		}
		return
	}
	message := "Version " + manifest.Version + " is available. You have " + version + "."
	if notes := strings.TrimSpace(manifest.Notes); notes != "" {
		message += "\n\n" + notes
	}
	question := app.Dialog.Question().SetTitle("Update Tandemesh").SetMessage(message)
	question.AddButton("Update and restart").OnClick(func() {
		go func() {
			pkg, err := downloadUpdate(context.Background(), manifest)
			if err == nil {
				err = installUpdate(pkg)
			}
			if err != nil {
				log.Printf("update: %v", err)
				app.Dialog.Error().SetTitle("Update failed").SetMessage(err.Error()).Show()
				return
			}
			app.Quit()
		}()
	})
	question.AddButton("Later")
	question.Show()
}

// runUpdateChecks checks shortly after launch and then every updateInterval.
func runUpdateChecks(app *application.App) {
	time.Sleep(10 * time.Second)
	for {
		checkForUpdates(app, false)
		time.Sleep(updateInterval)
	}
}
