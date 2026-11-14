package integration

import (
	"errors"
	"testing"
)

func TestIsAuthError(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want bool
	}{
		{errors.New("plane: status 401: unauthorized"), true},
		{errors.New("status 403: forbidden"), true},
		{errors.New("invalid token"), true},
		{errors.New("connection refused"), false},
		{nil, false},
	} {
		if got := IsAuthError(tc.err); got != tc.want {
			t.Fatalf("IsAuthError(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}
