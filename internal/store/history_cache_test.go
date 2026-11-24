package store

import (
	"context"
	"testing"
	"time"
)

func TestHistoryCache(t *testing.T) {
	st, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	if err := st.SetHistoryCache(context.Background(), "local", `[{"id":"a"}]`, now); err != nil {
		t.Fatal(err)
	}
	raw, scanned, err := st.GetHistoryCache(context.Background(), "local")
	if err != nil || raw != `[{"id":"a"}]` || !scanned.Equal(now) {
		t.Fatalf("raw=%q scanned=%v err=%v", raw, scanned, err)
	}
}
