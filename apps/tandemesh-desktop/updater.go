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
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var version = "0.1.0"

type updateManifest struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Notes   string `json:"notes"`
}

func updateManifestURL() string {
	return strings.TrimSpace(os.Getenv("TANDEMESH_UPDATE_URL"))
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

func fetchUpdate(ctx context.Context) (*updateManifest, error) {
	url := updateManifestURL()
	if url == "" {
		return nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update manifest: %s", resp.Status)
	}
	var manifest updateManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, err
	}
	if manifest.Version == "" || manifest.URL == "" || manifest.SHA256 == "" {
		return nil, fmt.Errorf("update manifest is incomplete")
	}
	if compareVersions(manifest.Version, version) <= 0 {
		return nil, nil
	}
	return &manifest, nil
}

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
	target := filepath.Join(os.TempDir(), "Tandemesh-Setup-"+manifest.Version+".exe")
	file, err := os.Create(target)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(file, hash), resp.Body); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	sum := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(sum, manifest.SHA256) {
		_ = os.Remove(target)
		return "", fmt.Errorf("checksum mismatch")
	}
	return target, nil
}

func installUpdate(installer string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("installer updates are only wired for Windows")
	}
	cmd := exec.Command("cmd.exe", "/c", "ping 127.0.0.1 -n 3 >nul & \""+installer+"\" /S /UPDATE")
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

func checkForUpdates(app *application.App) {
	if updateManifestURL() == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	manifest, err := fetchUpdate(ctx)
	if err != nil {
		log.Printf("update check: %v", err)
		return
	}
	if manifest == nil {
		return
	}
	question := app.Dialog.Question().
		SetTitle("Tandemesh " + manifest.Version).
		SetMessage("A new version is available.\n\n" + manifest.Notes)
	question.AddButton("Update and restart").OnClick(func() {
		go func() {
			installer, err := downloadUpdate(context.Background(), manifest)
			if err != nil {
				app.Dialog.Error().SetTitle("Update failed").SetMessage(err.Error()).Show()
				return
			}
			if err := installUpdate(installer); err != nil {
				app.Dialog.Error().SetTitle("Install failed").SetMessage(err.Error()).Show()
				return
			}
			app.Quit()
		}()
	})
	question.AddButton("Later")
	question.Show()

}
