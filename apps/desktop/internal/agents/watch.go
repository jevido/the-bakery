package agents

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// debounce is how long the watcher waits for an agent's files to settle
// before reporting it: editors save in several steps.
const debounce = 750 * time.Millisecond

// Watch reports, per agent slug, that its folder changed, until ctx ends.
// fsnotify does not watch recursively, so every directory is added, and new
// ones as they appear. The engine's own files (.sync.json, .state.json,
// .conflicts, .trash, half-written folders) and editor leftovers are
// ignored.
func Watch(ctx context.Context, dir string, changed func(slug string)) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	addTree := func(root string) {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if path != dir && strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				_ = w.Add(path)
			}
			return nil
		})
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		w.Close()
		return err
	}
	addTree(dir)

	var mu sync.Mutex
	timers := map[string]*time.Timer{}
	go func() {
		defer w.Close()
		for {
			select {
			case <-ctx.Done():
				mu.Lock()
				for _, t := range timers {
					t.Stop()
				}
				mu.Unlock()
				return
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				rel, err := filepath.Rel(dir, ev.Name)
				if err != nil || rel == "." {
					continue
				}
				parts := strings.Split(filepath.ToSlash(rel), "/")
				slug := parts[0]
				if strings.HasPrefix(slug, ".") || ignored(filepath.Base(ev.Name)) {
					continue
				}
				// fsnotify keeps a watch under the name it was added with, so a
				// renamed directory would go on reporting its old name (and lose
				// its events once something new takes that name). Drop the old
				// watches; the new name's Create adds fresh ones.
				if ev.Has(fsnotify.Rename) {
					for _, p := range w.WatchList() {
						if p == ev.Name || strings.HasPrefix(p, ev.Name+string(filepath.Separator)) {
							_ = w.Remove(p)
						}
					}
				}
				if ev.Has(fsnotify.Create) {
					if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
						addTree(ev.Name)
					}
				}
				mu.Lock()
				if t, ok := timers[slug]; ok {
					t.Reset(debounce)
				} else {
					timers[slug] = time.AfterFunc(debounce, func() {
						mu.Lock()
						delete(timers, slug)
						mu.Unlock()
						changed(slug)
					})
				}
				mu.Unlock()
			case <-w.Errors:
				// A lost event is caught by the next focus or the 5-minute sync.
			}
		}
	}()
	return nil
}
