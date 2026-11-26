package main

import (
	"net/http"
	"net/url"
	"time"
)

// planeAuthTransport attaches the plane's local token to requests bound for
// the plane. A desktop-managed plane rejects machine and approval routes
// without it; other hosts must never see the token.
type planeAuthTransport struct {
	host  string
	token string
	base  http.RoundTripper
}

func (t *planeAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.token != "" && req.URL.Host == t.host && req.Header.Get("Authorization") == "" {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.base.RoundTrip(req)
}

func newPlaneClient(plane, token string, timeout time.Duration) *http.Client {
	client := &http.Client{Timeout: timeout}
	if token == "" {
		return client
	}
	host := ""
	if u, err := url.Parse(plane); err == nil {
		host = u.Host
	}
	client.Transport = &planeAuthTransport{host: host, token: token, base: http.DefaultTransport}
	return client
}
