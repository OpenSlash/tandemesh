package pgdriver

import "testing"

func TestRebind(t *testing.T) {
	cases := []struct{ in, want string }{
		{"SELECT 1", "SELECT 1"},
		{"SELECT * FROM runs WHERE id = ?", "SELECT * FROM runs WHERE id = $1"},
		{"INSERT INTO runs (a,b) VALUES (?,?)", "INSERT INTO runs (a,b) VALUES ($1,$2)"},
		{"SELECT '?' AS q, id FROM runs WHERE id = ?", "SELECT '?' AS q, id FROM runs WHERE id = $1"},
		{"SELECT * FROM runs WHERE a = ? AND b = ?", "SELECT * FROM runs WHERE a = $1 AND b = $2"},
	}
	for _, c := range cases {
		if got := Rebind(c.in); got != c.want {
			t.Fatalf("Rebind(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
