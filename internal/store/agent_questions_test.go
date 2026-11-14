package store

import (
	"context"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestAgentQuestionExpiry(t *testing.T) {
	st, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	expired := time.Now().UTC().Add(-time.Minute)
	question, err := st.CreateAgentQuestion(context.Background(), &domain.AgentQuestion{
		SessionID: "ses_test", Question: "expired?", ExpiresAt: &expired,
	})
	if err != nil {
		t.Fatal(err)
	}
	items, err := st.ListAgentQuestions(context.Background(), "ses_test", "")
	if err != nil || len(items) != 1 || items[0].Status != "expired" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	answered, err := st.AnswerAgentQuestion(context.Background(), question.ID, "late")
	if err != nil || answered != nil {
		t.Fatalf("answered=%+v err=%v", answered, err)
	}
}
