package pipeline

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestTileFilesWritesPyramidPaths(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	w, err := newFileWorldTileWriter(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := []byte("webp tile")
	if err := w.add(7, -12, 34, want); err != nil {
		t.Fatal(err)
	}
	w.addMany(0, [][2]int{{0, 0}, {1, -1}}, want)
	if err := w.close(); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		filepath.Join(dir, "7", "-12", "34.webp"),
		filepath.Join(dir, "0", "0", "0.webp"),
		filepath.Join(dir, "0", "1", "-1.webp"),
	} {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
}

func TestTileFilesReportsWriteFailureAtClose(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	blocked := filepath.Join(root, "blocked")
	if err := os.WriteFile(blocked, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	w := &tileFiles{dir: blocked}
	if err := w.add(1, 2, 3, []byte("tile")); err == nil {
		t.Fatal("add succeeded through a non-directory root")
	}
	if err := w.close(); err == nil {
		t.Fatal("close did not report the prior write failure")
	}
}
