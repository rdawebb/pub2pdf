package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestFindPubFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.pub", "B.PUB", "notes.txt", "sub/c.pub"} {
		path := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, nil, 0o644)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Skip("symlinks unavailable:", err)
	}

	for _, tc := range []struct {
		root    string
		recurse bool
		want    []string
	}{
		{dir, false, []string{"B.PUB", "a.pub"}},
		{dir, true, []string{"B.PUB", "a.pub", "sub/c.pub"}},
		{link, false, []string{"B.PUB", "a.pub"}},
	} {
		got, err := findPubFiles(tc.root, tc.recurse)
		if err != nil {
			t.Fatal(err)
		}
		for i := range got {
			got[i], _ = filepath.Rel(tc.root, got[i])
			got[i] = filepath.ToSlash(got[i])
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("findPubFiles(%s, %v) = %v, want %v", tc.root, tc.recurse, got, tc.want)
		}
	}
}

func TestSummaryListsFailuresBeforeOverflow(t *testing.T) {
	var r result
	for i := range 12 {
		r.failures = append(r.failures, string(rune('a'+i))+".pub")
	}
	msg := r.summary()
	if !strings.HasSuffix(msg, "j.pub\n  ...and 2 more") || strings.Contains(msg, "k.pub") {
		t.Errorf("unexpected summary:\n%s", msg)
	}
}
