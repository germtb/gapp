package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchGoFiles watches for .go file changes under dir and calls onChange after
// debouncing. Returns the watcher so the caller can close it.
func WatchGoFiles(dir string, debounce time.Duration, onChange func()) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Recursively add all directories
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
			return filepath.SkipDir
		}
		return watcher.Add(path)
	})
	if err != nil {
		watcher.Close()
		return nil, err
	}

	var mu sync.Mutex
	var timer *time.Timer

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if !strings.HasSuffix(event.Name, ".go") {
					continue
				}
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) == 0 {
					continue
				}
				mu.Lock()
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounce, onChange)
				mu.Unlock()

			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	return watcher, nil
}
