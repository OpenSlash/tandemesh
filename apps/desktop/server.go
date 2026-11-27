package main

import (
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
)

type desktopServer struct {
	listener net.Listener
	server   *http.Server
}

// startDesktopServer serves the built web UI and proxies API traffic to the
// local plane. It keeps the browser-facing origin same-origin, so the existing
// relative /api, /auth and /ws URLs keep working.
func startDesktopServer(planeURL string, webFS fs.FS, localToken string) (*desktopServer, string, error) {
	target, err := url.Parse(planeURL)
	if err != nil {
		return nil, "", err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	if localToken != "" {
		director := proxy.Director
		proxy.Director = func(req *http.Request) {
			director(req)
			req.Header.Set("Authorization", "Bearer "+localToken)
		}
	}
	files := http.FileServer(http.FS(webFS))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/"), r.URL.Path == "/auth/me", strings.HasPrefix(r.URL.Path, "/ws"):
			proxy.ServeHTTP(w, r)
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" {
			if info, err := fs.Stat(webFS, name); err == nil && !info.IsDir() {
				files.ServeHTTP(w, r)
				return
			}
		}
		index, err := fs.ReadFile(webFS, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	})

	addr := "127.0.0.1:0"
	if port := os.Getenv("TANDEMESH_DESKTOP_PORT"); port != "" {
		addr = "127.0.0.1:" + port
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, "", err
	}
	server := &http.Server{Handler: handler}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("desktop web server: %v", err)
		}
	}()
	return &desktopServer{listener: listener, server: server}, "http://" + listener.Addr().String(), nil
}
