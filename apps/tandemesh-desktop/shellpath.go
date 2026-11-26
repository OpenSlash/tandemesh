package main

import (
	"os"
	"path/filepath"
	"strings"
)

// mergePathLists keeps primary's order, then appends entries of extra that
// primary lacks.
func mergePathLists(primary, extra string) string {
	seen := map[string]bool{}
	var out []string
	for _, list := range []string{primary, extra} {
		for _, dir := range filepath.SplitList(list) {
			if dir == "" || seen[dir] {
				continue
			}
			seen[dir] = true
			out = append(out, dir)
		}
	}
	return strings.Join(out, string(os.PathListSeparator))
}
