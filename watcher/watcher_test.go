package watcher_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"auto_ballchasing/watcher"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "auto-ballchasing-test-*")
	if err != nil {
		t.Fatalf("could not create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("could not write file: %v", err)
	}
	return path
}

func startWatcher(t *testing.T, dir string) *watcher.Watcher {
	t.Helper()
	w, err := watcher.New(dir, nil)
	if err != nil {
		t.Fatalf("could not create watcher: %v", err)
	}
	w.Start()
	t.Cleanup(w.Stop)
	return w
}

func TestDetectsReplayFile(t *testing.T) {
	dir := tempDir(t)
	w := startWatcher(t, dir)

	writeFile(t, dir, "match.replay", "fake replay content")

	select {
	case result := <-w.Results:
		if result.Filename != "match.replay" {
			t.Errorf("expected 'match.replay', got '%s'", result.Filename)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestIgnoresNonReplayFiles(t *testing.T) {
	dir := tempDir(t)
	w := startWatcher(t, dir)

	writeFile(t, dir, "notes.txt", "not a replay")

	select {
	case result := <-w.Results:
		t.Errorf("should have ignored .txt file, got result for: %s", result.Filename)
	case <-time.After(2 * time.Second):
		// Good — nothing fired
	}
}

func TestDetectsMultipleFiles(t *testing.T) {
	dir := tempDir(t)
	w := startWatcher(t, dir)

	files := []string{"game1.replay", "game2.replay", "game3.replay"}
	for _, f := range files {
		writeFile(t, dir, f, "fake content")
	}

	detected := make(map[string]bool)
	timeout := time.After(10 * time.Second)

	for len(detected) < len(files) {
		select {
		case result := <-w.Results:
			detected[result.Filename] = true
		case <-timeout:
			t.Fatalf("timed out — detected %d of %d: %v", len(detected), len(files), detected)
		}
	}
}

func TestRejectsNonExistentFolder(t *testing.T) {
	_, err := watcher.New("/this/folder/does/not/exist", nil)
	if err == nil {
		t.Error("expected error for non-existent folder, got nil")
	}
}

func TestStopsCleanly(t *testing.T) {
	dir := tempDir(t)
	w, err := watcher.New(dir, nil)
	if err != nil {
		t.Fatalf("could not create watcher: %v", err)
	}
	w.Start()

	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() blocked for too long")
	}
}

func TestFullFlow(t *testing.T) {
	dir := "test_replays"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("could not create test folder: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	t.Logf("created folder: %s", dir)

	w := startWatcher(t, dir)
	t.Log("watcher started...")

	path := filepath.Join(dir, "test_match.replay")
	if err := os.WriteFile(path, []byte("fake replay content"), 0644); err != nil {
		t.Fatalf("could not write replay file: %v", err)
	}
	t.Logf("wrote file: %s", path)

	select {
	case result := <-w.Results:
		t.Logf("detected: %s", result.Filename)
		if result.Filename != "test_match.replay" {
			t.Errorf("expected 'test_match.replay', got '%s'", result.Filename)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out — file was not detected")
	}
}
