package store

import "testing"

func TestSplitStatements(t *testing.T) {
	script := `-- a comment with a semicolon; inside
CREATE TABLE a (x TEXT);
CREATE TABLE b (y TEXT DEFAULT ';');
-- trailing comment
`
	stmts := splitStatements(script)
	if len(stmts) != 2 {
		t.Fatalf("statements = %d: %#v", len(stmts), stmts)
	}
	if stmts[0][:6] != "CREATE" || stmts[1][:6] != "CREATE" {
		t.Fatalf("unexpected statements: %#v", stmts)
	}
	if stmts[0] == "" || stmts[1] == "" {
		t.Fatalf("empty statement: %#v", stmts)
	}
}
