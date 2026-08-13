package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/dgravesa/go-parallel/parallel"
)

// tileFiles writes each pyramid tile as an ordinary WebP file. It records the
// first write failure so callers that deliberately tolerate individual missing
// tiles still receive an error when the writer closes.
type tileFiles struct {
	dir string

	dirMu sync.Mutex
	dirs  map[string]bool

	errMu sync.Mutex
	err   error
}

func newFileWorldTileWriter(dir string) (worldTileWriter, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &tileFiles{dir: dir, dirs: make(map[string]bool)}, nil
}

func (w *tileFiles) add(z, x, y int, data []byte) error {
	dir := filepath.Join(w.dir, strconv.Itoa(z), strconv.Itoa(x))
	if err := w.ensureDir(dir); err != nil {
		return w.remember(fmt.Errorf("create tile directory: %w", err))
	}
	path := filepath.Join(dir, strconv.Itoa(y)+".webp")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return w.remember(fmt.Errorf("write tile %d/%d/%d: %w", z, x, y, err))
	}
	return nil
}

func (w *tileFiles) ensureDir(dir string) error {
	w.dirMu.Lock()
	defer w.dirMu.Unlock()
	if w.dirs[dir] {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if w.dirs == nil {
		w.dirs = make(map[string]bool)
	}
	w.dirs[dir] = true
	return nil
}

func (w *tileFiles) addMany(z int, coords [][2]int, data []byte) {
	parallel.For(len(coords), func(i, _ int) {
		_ = w.add(z, coords[i][0], coords[i][1], data)
	})
}

func (w *tileFiles) close() error {
	w.errMu.Lock()
	defer w.errMu.Unlock()
	return w.err
}

func (w *tileFiles) remember(err error) error {
	w.errMu.Lock()
	if w.err == nil {
		w.err = err
	}
	w.errMu.Unlock()
	return err
}
