package main

import "testing"

func TestCompareVersions(t *testing.T) {
	if compareVersions("0.1.0", "0.2.0") != -1 || compareVersions("1.0.0", "1.0.0") != 0 || compareVersions("2.0.0", "1.9.9") != 1 {
		t.Fatal("version comparison failed")
	}
}
