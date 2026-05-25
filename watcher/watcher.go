package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"auto_ballchasing/uploader"

	"github.com/fsnotify/fsnotify"
)

type Result struct {
	Filename  string
	Success   bool
	Duplicate bool
	Error     string
	Time      time.Time
}

type Watcher struct {
	fsw     *fsnotify.Watcher
	folder  string
	upload  *uploader.Uploader
	Results chan Result
	done    chan struct{}
}

func New(folder string, u *uploader.Uploader) (*Watcher, error) {
	if _, err := os.Stat(folder); err != nil {
		return nil, fmt.Errorf("folder does not exist: %w", err)
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("could not create watcher: %w", err)
	}

	if err := fsw.Add(folder); err != nil {
		fsw.Close()
		return nil, fmt.Errorf("could not watch folder: %w", err)
	}

	return &Watcher{
		fsw:     fsw,
		folder:  folder,
		upload:  u,
		Results: make(chan Result, 20),
		done:    make(chan struct{}),
	}, nil
}

func (w *Watcher) Start() {
	go func() {
		for {
			select {
			case event, ok := <-w.fsw.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Create) {
					if filepath.Ext(event.Name) != ".replay" {
						continue
					}
					go w.handleFile(event.Name)
				}
			case _, ok := <-w.fsw.Errors:
				if !ok {
					return
				}
			case <-w.done:
				return
			}
		}
	}()
}

func (w *Watcher) Stop() {
	close(w.done)
	w.fsw.Close()
}

func (w *Watcher) handleFile(path string) {
	if err := waitUntilStable(path, 3*time.Second); err != nil {
		w.Results <- Result{
			Filename: filepath.Base(path),
			Error:    fmt.Sprintf("file never stabilised: %v", err),
			Time:     time.Now(),
		}
		return
	}

	if w.upload == nil {
		w.Results <- Result{
			Filename: filepath.Base(path),
			Success:  true,
			Time:     time.Now(),
		}
		return
	}

	result := w.upload.Upload(path)
	w.Results <- Result{
		Filename:  result.Filename,
		Success:   result.Success,
		Duplicate: result.Duplicate,
		Error:     result.Error,
		Time:      result.Time,
	}
}

func waitUntilStable(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastSize int64 = -1

	for time.Now().Before(deadline) {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Size() == lastSize {
			return nil
		}
		lastSize = info.Size()
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timed out after %s", timeout)
}
