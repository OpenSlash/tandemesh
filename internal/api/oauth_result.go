package api

import (
	"fmt"
	"html"
	"net/http"
)

// oauthResult renders a small standalone result page for browser OAuth flows.
func (s *Server) oauthResult(w http.ResponseWriter, r *http.Request) {
	provider := html.EscapeString(r.URL.Query().Get("provider"))
	status := r.URL.Query().Get("status")
	message := html.EscapeString(r.URL.Query().Get("message"))
	ok := status != "error"
	title := "Authorization complete"
	color := "#4ade80"
	if !ok {
		title = "Authorization failed"
		color = "#f87171"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>%s</title>
<style>body{background:#0b0b0c;color:#e7e7e9;font:15px/1.5 system-ui,sans-serif;display:grid;place-items:center;height:100vh;margin:0}
main{max-width:460px;padding:32px;border:1px solid #26262a;border-radius:12px;background:#121214}
h1{font-size:18px;margin:0 0 12px;color:%s}p{color:#a1a1aa;margin:0 0 20px}
a{color:#e7e7e9;border:1px solid #333;padding:8px 14px;border-radius:8px;text-decoration:none}</style></head>
<body><main><h1>%s</h1><p>%s %s</p><a href="/providers">Back to providers</a></main></body></html>`,
		title, color, title, provider, message)
}
