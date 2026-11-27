package api

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

type fakeAccountProbe struct{ account integration.Account }

func (f fakeAccountProbe) Probe(context.Context, integration.Credential) error { return nil }
func (f fakeAccountProbe) CurrentAccount(context.Context, integration.Credential) (integration.Account, error) {
	return f.account, nil
}

func TestConnectRemembersMyAccount(t *testing.T) {
	_, st := newTestServerWithStore(t)
	s := &Server{store: st, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	ad := &integration.Integration{Probe: fakeAccountProbe{integration.Account{ID: "u-1", Login: "Alice Zhang", Email: "a@example.test"}}}
	cred := integration.Credential{Provider: "plane", AccessToken: "token"}
	s.rememberMyAccount(t.Context(), ad, "ws_default", "me", cred)
	s.rememberMyAccount(t.Context(), ad, "ws_default", "me", cred) // reconnecting must not duplicate it
	items, err := st.ListMemberIdentities(t.Context(), "ws_default", "")
	if err != nil || len(items) != 1 || items[0].Login != "Alice Zhang" || items[0].ExternalID != "u-1" {
		t.Fatalf("member identities = %+v err=%v", items, err)
	}

	tasks := []*domain.Task{
		{WorkspaceID: "ws_default", Provider: "plane", Assignee: "Alice Zhang"},
		{WorkspaceID: "ws_default", Provider: "plane", Assignee: "Bob"},
	}
	s.enrichTaskAssignees(t.Context(), tasks)
	if tasks[0].AssigneeMemberID != items[0].MemberID || tasks[1].AssigneeMemberID != "" {
		t.Fatalf("assignees = %q, %q", tasks[0].AssigneeMemberID, tasks[1].AssigneeMemberID)
	}
}
