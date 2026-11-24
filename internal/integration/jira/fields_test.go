package jira

import "testing"

func TestProjectFields(t *testing.T) {
	got := projectFields([]string{"bug"}, "High", []string{"API"}, []string{"1.2"})
	want := []string{"bug", "priority:High", "component:API", "fixVersion:1.2"}
	if len(got) != len(want) {
		t.Fatalf("got = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got = %v, want = %v", got, want)
		}
	}
}

func TestParseProjectsJiraFields(t *testing.T) {
	body := []byte(`{"webhookEvent":"jira:issue_updated","issue":{"key":"ENG-1","self":"https://acme.atlassian.net/rest/api/2/issue/1","fields":{"summary":"Fix","updated":"2026-01-01T00:00:00.000+0000","labels":["bug"],"priority":{"name":"High"},"components":[{"name":"API"}],"fixVersions":[{"name":"1.2"}]}},"user":{"displayName":"Alice"}}`)
	ev, err := (&Adapter{}).Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(ev.Labels) != 4 || ev.Labels[1] != "priority:High" {
		t.Fatalf("labels = %v", ev.Labels)
	}
}

func TestParseAttachments(t *testing.T) {
	body := []byte(`{"webhookEvent":"jira:issue_updated","issue":{"key":"ENG-1","self":"https://acme.atlassian.net/rest/api/2/issue/1","fields":{"summary":"Fix","updated":"2026-01-01T00:00:00.000+0000","attachment":[{"id":"10001","filename":"spec.pdf","content":"https://example.test/spec.pdf","mimeType":"application/pdf","size":42,"author":{"displayName":"Alice"}}]}}}`)
	ev, err := (&Adapter{}).Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(ev.Attachments) != 1 || ev.Attachments[0].Name != "spec.pdf" || ev.Attachments[0].Size != 42 {
		t.Fatalf("attachments = %+v", ev.Attachments)
	}
}
