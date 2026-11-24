package main

import (
	"embed"
	"io/fs"
	"os"
)

// The release build copies web/dist here. The placeholder keeps `go build`
// working in a clean checkout; runtime falls back to ../../web/dist when the
// embedded directory has no index.html.
//
//go:embed all:webdist
var embeddedWeb embed.FS

func desktopWebFS() fs.FS {
	if value := os.Getenv("TANDEMESH_WEB_DIST"); value != "" {
		return os.DirFS(value)
	}
	if sub, err := fs.Sub(embeddedWeb, "webdist"); err == nil {
		if _, err := fs.Stat(sub, "index.html"); err == nil {
			return sub
		}
	}
	fallback := os.Getenv("TANDEMESH_WEB_DIST_FALLBACK")
	if fallback == "" {
		fallback = "../../web/dist"
	}
	return os.DirFS(fallback)
}
