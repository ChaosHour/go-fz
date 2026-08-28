package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// mustWriteFile creates path (and any missing parent dirs) containing size
// bytes of arbitrary content.
func mustWriteFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), size), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
}

// buildTree lays out:
//
//	root/root_file.txt   (10 bytes)
//	root/a/big.bin        (2000 bytes)
//	root/a/b/mid.bin       (500 bytes)
//	root/a/b/c/small.bin    (50 bytes)
//	root/a/b/empty/         (empty dir)
func buildTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "root_file.txt"), 10)
	mustWriteFile(t, filepath.Join(root, "a", "big.bin"), 2000)
	mustWriteFile(t, filepath.Join(root, "a", "b", "mid.bin"), 500)
	mustWriteFile(t, filepath.Join(root, "a", "b", "c", "small.bin"), 50)
	mustMkdir(t, filepath.Join(root, "a", "b", "empty"))
	return root
}

func TestScanAggregatesSizesBottomUp(t *testing.T) {
	root := buildTree(t)

	files, dirs, err := scan(root, nil)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if len(files) != 4 {
		t.Fatalf("got %d files, want 4", len(files))
	}

	wantSizes := map[string]int64{
		root:                                   2560, // everything
		filepath.Join(root, "a"):               2550, // big + mid + small
		filepath.Join(root, "a", "b"):          550,  // mid + small
		filepath.Join(root, "a", "b", "c"):     50,
		filepath.Join(root, "a", "b", "empty"): 0,
	}
	for path, want := range wantSizes {
		node, ok := dirs[path]
		if !ok {
			t.Errorf("missing dir node for %q", path)
			continue
		}
		if node.size != want {
			t.Errorf("size(%q) = %d, want %d", path, node.size, want)
		}
	}
}

func TestScanComputesDepthRelativeToRoot(t *testing.T) {
	root := buildTree(t)

	_, dirs, err := scan(root, nil)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	wantDepths := map[string]int{
		root:                                   0,
		filepath.Join(root, "a"):               1,
		filepath.Join(root, "a", "b"):          2,
		filepath.Join(root, "a", "b", "c"):     3,
		filepath.Join(root, "a", "b", "empty"): 3,
	}
	for path, want := range wantDepths {
		node, ok := dirs[path]
		if !ok {
			t.Errorf("missing dir node for %q", path)
			continue
		}
		if node.depth != want {
			t.Errorf("depth(%q) = %d, want %d", path, node.depth, want)
		}
	}
}

func TestScanExcludeSkipsSubtreeEntirely(t *testing.T) {
	root := buildTree(t)

	files, dirs, err := scan(root, map[string]bool{"c": true})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if _, ok := dirs[filepath.Join(root, "a", "b", "c")]; ok {
		t.Errorf("excluded dir %q should not appear in dirs", filepath.Join(root, "a", "b", "c"))
	}

	for _, f := range files {
		if filepath.Base(f.path) == "small.bin" {
			t.Errorf("excluded file %q should not appear in files", f.path)
		}
	}
	if len(files) != 3 {
		t.Errorf("got %d files, want 3 (small.bin excluded)", len(files))
	}

	// Sizes above the excluded subtree must not include what was skipped.
	wantSizes := map[string]int64{
		root:                          2510, // root_file + big + mid (no small)
		filepath.Join(root, "a"):      2500,
		filepath.Join(root, "a", "b"): 500,
	}
	for path, want := range wantSizes {
		node, ok := dirs[path]
		if !ok {
			t.Fatalf("missing dir node for %q", path)
		}
		if node.size != want {
			t.Errorf("size(%q) = %d, want %d", path, node.size, want)
		}
	}
}

func TestScanExcludeMatchesExactNameOnly(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "cached", "file.bin"), 100)

	files, dirs, err := scan(root, map[string]bool{"cache": true})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if _, ok := dirs[filepath.Join(root, "cached")]; !ok {
		t.Error(`dir "cached" should not be excluded by pattern "cache" (must match exact base name)`)
	}
	if len(files) != 1 {
		t.Errorf("got %d files, want 1", len(files))
	}
}

func TestScanExcludeDoesNotSkipRootEvenIfNameMatches(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "cache")
	mustWriteFile(t, filepath.Join(root, "file.bin"), 100)

	files, dirs, err := scan(root, map[string]bool{"cache": true})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if _, ok := dirs[root]; !ok {
		t.Error("root itself must never be excluded, even if its name matches an exclude pattern")
	}
	if len(files) != 1 {
		t.Errorf("got %d files, want 1", len(files))
	}
}

func TestScanEmptyDirectoryHasZeroSize(t *testing.T) {
	root := t.TempDir()

	_, dirs, err := scan(root, nil)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	node, ok := dirs[root]
	if !ok {
		t.Fatal("missing root dir node")
	}
	if node.size != 0 {
		t.Errorf("size(root) = %d, want 0", node.size)
	}
}

func TestScanNonexistentRootReturnsError(t *testing.T) {
	root := filepath.Join(t.TempDir(), "does-not-exist")

	_, _, err := scan(root, nil)
	if err == nil {
		t.Fatal("scan of a nonexistent root should return an error")
	}
}

func TestHumanSize(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0B"},
		{1, "1B"},
		{1023, "1023B"},
		{1024, "1.0KiB"},
		{1536, "1.5KiB"},
		{1024 * 1024, "1.0MiB"},
		{1024 * 1024 * 1024, "1.0GiB"},
		{1024 * 1024 * 1024 * 1024, "1.0TiB"},
	}
	for _, tt := range tests {
		if got := humanSize(tt.in); got != tt.want {
			t.Errorf("humanSize(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
