package api

import (
	"testing"
	"time"
)

func TestAcquireImportConnSerializes(t *testing.T) {
	s := &Server{importConns: map[string]chan struct{}{}}
	release := s.acquireImportConn("conn_1")
	acquired := make(chan struct{})
	go func() {
		release2 := s.acquireImportConn("conn_1")
		close(acquired)
		release2()
	}()
	select {
	case <-acquired:
		t.Fatal("second import should wait")
	case <-time.After(20 * time.Millisecond):
	}
	release()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second import did not acquire")
	}
}
