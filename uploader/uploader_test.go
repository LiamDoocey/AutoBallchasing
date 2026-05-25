package uploader_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"auto_ballchasing/uploader"
)

func fakeServer(t *testing.T, status int, body any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != nil {
			json.NewEncoder(w).Encode(body)
		}
	}))
}

func tempReplay(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.replay")
	if err := os.WriteFile(path, []byte("fake replay data"), 0644); err != nil {
		t.Fatalf("could not write replay: %v", err)
	}
	return path
}

func TestPingSuccess(t *testing.T) {
	srv := fakeServer(t, http.StatusOK, map[string]string{"ball": "is life"})
	defer srv.Close()

	u := uploader.NewWithURLs("test-key", uploader.VisibilityPublic, srv.URL, srv.URL)
	if err := u.Ping(); err != nil {
		t.Errorf("expected ping to succeed, got: %v", err)
	}
}

func TestPingInvalidKey(t *testing.T) {
	srv := fakeServer(t, http.StatusUnauthorized, map[string]string{"error": "missing API key"})
	defer srv.Close()

	u := uploader.NewWithURLs("bad-key", uploader.VisibilityPublic, srv.URL, srv.URL)
	if err := u.Ping(); err == nil {
		t.Error("expected ping to fail with invalid key")
	}
}

func TestUploadSuccess(t *testing.T) {
	srv := fakeServer(t, http.StatusCreated, map[string]string{
		"id":       "abc-123",
		"location": "https://ballchasing.com/replay/abc-123",
	})
	defer srv.Close()

	u := uploader.NewWithURLs("test-key", uploader.VisibilityPublic, srv.URL, srv.URL)
	result := u.Upload(tempReplay(t))

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if result.Duplicate {
		t.Error("should not be marked duplicate")
	}
	if result.ReplayID != "abc-123" {
		t.Errorf("expected replay ID 'abc-123', got '%s'", result.ReplayID)
	}
}

func TestUploadDuplicate(t *testing.T) {
	srv := fakeServer(t, http.StatusConflict, map[string]string{
		"id":       "existing-456",
		"location": "https://ballchasing.com/replay/existing-456",
		"error":    "duplicate replay",
	})
	defer srv.Close()

	u := uploader.NewWithURLs("test-key", uploader.VisibilityPublic, srv.URL, srv.URL)
	result := u.Upload(tempReplay(t))

	if !result.Success {
		t.Errorf("expected success for duplicate, got: %s", result.Error)
	}
	if !result.Duplicate {
		t.Error("expected duplicate to be true")
	}
	if result.ReplayID != "existing-456" {
		t.Errorf("expected replay ID 'existing-456', got '%s'", result.ReplayID)
	}
}

func TestUploadServerError(t *testing.T) {
	srv := fakeServer(t, http.StatusInternalServerError, map[string]string{
		"error": "ballchasing.com fault",
	})
	defer srv.Close()

	u := uploader.NewWithURLs("test-key", uploader.VisibilityPublic, srv.URL, srv.URL)
	result := u.Upload(tempReplay(t))

	if result.Success {
		t.Error("expected failure on 500")
	}
	if result.Error == "" {
		t.Error("expected non-empty error message")
	}
}

func TestUploadMissingFile(t *testing.T) {
	u := uploader.New("test-key", uploader.VisibilityPublic)
	result := u.Upload("/does/not/exist.replay")

	if result.Success {
		t.Error("expected failure for missing file")
	}
	if result.Error == "" {
		t.Error("expected non-empty error message")
	}
}
