package webserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"dotsynx/internal/config"
)

func TestWebserverNullSafety(t *testing.T) {
	cfg := config.NewDefault()
	srv, err := NewServer(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}

	// 1. /api/status
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	srv.handleStatus(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var statusResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &statusResp); err != nil {
		t.Fatal(err)
	}
	conflicts, ok := statusResp["conflicts"].([]any)
	if !ok || conflicts == nil {
		t.Errorf("expected conflicts to be non-nil array, got %v", statusResp["conflicts"])
	}

	// 2. /api/files
	wFiles := httptest.NewRecorder()
	srv.handleFiles(wFiles, httptest.NewRequest(http.MethodGet, "/api/files", nil))
	var filesResp []any
	if err := json.Unmarshal(wFiles.Body.Bytes(), &filesResp); err != nil {
		t.Fatal(err)
	}
	if filesResp == nil {
		t.Errorf("expected files to be non-nil array")
	}

	// 3. /api/conflicts
	wConf := httptest.NewRecorder()
	srv.handleConflicts(wConf, httptest.NewRequest(http.MethodGet, "/api/conflicts", nil))
	var confResp []any
	if err := json.Unmarshal(wConf.Body.Bytes(), &confResp); err != nil {
		t.Fatal(err)
	}
	if confResp == nil {
		t.Errorf("expected conflicts to be non-nil array")
	}
}
