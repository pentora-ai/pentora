package catalogsync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFileSource_Load(t *testing.T) {
	_, err := FileSource{Path: ""}.Load(context.Background())
	if err == nil {
		t.Fatalf("expected error for empty path")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(p, []byte("rules: []\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	b, err := FileSource{Path: p}.Load(context.Background())
	if err != nil || len(b) == 0 {
		t.Fatalf("expected bytes, err=%v", err)
	}
}

func TestFileStore_Save(t *testing.T) {
	err := FileStore{Path: ""}.Save(context.Background(), []byte("x"))
	if err == nil {
		t.Fatalf("expected error for empty path")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "out", "c.yaml")
	if err := (FileStore{Path: p}).Save(context.Background(), []byte("rules: []\n")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("expected file saved: %v", err)
	}
}

func TestHTTPSource_Load(t *testing.T) {
	if _, err := (HTTPSource{URL: ""}).Load(context.Background()); err == nil {
		t.Fatalf("expected error for empty url")
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("rules: []\n"))
	}))
	defer ts.Close()
	if b, err := (HTTPSource{URL: ts.URL}).Load(context.Background()); err != nil || len(b) == 0 {
		t.Fatalf("expected ok, err=%v", err)
	}
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts2.Close()
	if _, err := (HTTPSource{URL: ts2.URL}).Load(context.Background()); err == nil {
		t.Fatalf("expected status error")
	}
}
