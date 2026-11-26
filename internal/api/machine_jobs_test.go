package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func doJSON(t *testing.T, method, url string, body any) (*http.Response, map[string]any) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out := map[string]any{}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

func TestMachineJobFlow(t *testing.T) {
	ts, st := newTestServerWithStore(t)
	if err := st.UpsertMachine(t.Context(), &domain.Machine{ID: "mch_a", Name: "a", Status: "online"}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertMachine(t.Context(), &domain.Machine{ID: "mch_off", Name: "off", Status: "offline"}); err != nil {
		t.Fatal(err)
	}

	queue := []struct {
		name   string
		method string
		path   string
		body   any
		want   int
	}{
		{"runtime check", http.MethodPost, "/api/v1/machines/mch_a/runtime-checks", nil, http.StatusAccepted},
		{"install", http.MethodPost, "/api/v1/machines/mch_a/acp-installs", map[string]string{"registryId": "cursor"}, http.StatusAccepted},
		{"install path traversal", http.MethodPost, "/api/v1/machines/mch_a/acp-installs", map[string]string{"registryId": "../x"}, http.StatusBadRequest},
		{"uninstall bad version", http.MethodDelete, "/api/v1/machines/mch_a/acp-installs?registryId=cursor&version=../1", nil, http.StatusBadRequest},
		{"offline machine", http.MethodPost, "/api/v1/machines/mch_off/runtime-checks", nil, http.StatusConflict},
		{"unknown machine", http.MethodPost, "/api/v1/machines/nope/runtime-checks", nil, http.StatusNotFound},
	}
	var checkID string
	for _, tc := range queue {
		resp, body := doJSON(t, tc.method, ts.URL+tc.path, tc.body)
		if resp.StatusCode != tc.want {
			t.Fatalf("%s: status %d, want %d (%v)", tc.name, resp.StatusCode, tc.want, body)
		}
		if tc.name == "runtime check" {
			checkID, _ = body["id"].(string)
		}
	}

	resp, body := doJSON(t, http.MethodGet, ts.URL+"/api/v1/agentd/machine-jobs/mch_a", nil)
	if items, _ := body["items"].([]any); resp.StatusCode != http.StatusOK || len(items) != 2 {
		t.Fatalf("claim: %d %v", resp.StatusCode, body)
	}

	result := MachineJobResult{
		Status:         "done",
		Runtimes:       map[string]string{"codex": "1.0"},
		RuntimeDetails: []domain.RuntimeStatus{{ID: "codex", Status: domain.RuntimeReady}},
		Agents:         []string{"codex"},
	}
	if resp, body := doJSON(t, http.MethodPost, ts.URL+"/api/v1/agentd/machine-jobs/mch_off/"+checkID, result); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("other machine finished the job: %d %v", resp.StatusCode, body)
	}
	if resp, body := doJSON(t, http.MethodPost, ts.URL+"/api/v1/agentd/machine-jobs/mch_a/"+checkID, result); resp.StatusCode != http.StatusOK {
		t.Fatalf("finish: %d %v", resp.StatusCode, body)
	}

	if _, body := doJSON(t, http.MethodGet, ts.URL+"/api/v1/machine-jobs/"+checkID, nil); body["status"] != "done" {
		t.Fatalf("job = %v", body)
	}
	m, err := st.GetMachine(t.Context(), "mch_a")
	if err != nil || len(m.RuntimeDetails) != 1 || m.Agents[0] != "codex" {
		t.Fatalf("machine runtimes not updated from job result: %+v err=%v", m, err)
	}
}

func TestDeleteMachineEndpoint(t *testing.T) {
	ts, st := newTestServerWithStore(t)
	for _, m := range []*domain.Machine{
		{ID: "mch_on", Name: "on", Status: "online"},
		{ID: "mch_off", Name: "off", Status: "offline"},
	} {
		if err := st.UpsertMachine(t.Context(), m); err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		name string
		id   string
		want int
	}{
		{"online is refused", "mch_on", http.StatusConflict},
		{"offline is deleted", "mch_off", http.StatusNoContent},
		{"already gone", "mch_off", http.StatusNotFound},
	}
	for _, tc := range tests {
		if resp, body := doJSON(t, http.MethodDelete, ts.URL+"/api/v1/machines/"+tc.id, nil); resp.StatusCode != tc.want {
			t.Fatalf("%s: status %d, want %d (%v)", tc.name, resp.StatusCode, tc.want, body)
		}
	}
}
