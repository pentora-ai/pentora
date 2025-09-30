package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/pentora-ai/pentora/pkg/scanner"
)

func stubRunScan(t *testing.T, fn func(scanner.ScanJob) ([]scanner.Result, error)) {
	t.Helper()
	original := runScan
	runScan = fn
	t.Cleanup(func() {
		runScan = original
	})
}

func TestScanHandlerSuccess(t *testing.T) {
	expected := []scanner.Result{{IP: "192.168.1.10", Port: 22, Status: "open"}}
	stubRunScan(t, func(job scanner.ScanJob) ([]scanner.Result, error) {
		if len(job.Targets) != 1 || job.Targets[0] != "192.168.1.10" {
			t.Fatalf("unexpected targets: %+v", job.Targets)
		}
		if len(job.Ports) != 1 || job.Ports[0] != 22 {
			t.Fatalf("unexpected ports: %+v", job.Ports)
		}
		return expected, nil
	})

	payload := ScanRequest{Targets: []string{"192.168.1.10"}, Ports: []int{22}}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/scan", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ScanHandler(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}
	if want := "application/json"; res.Header.Get("Content-Type") != want {
		t.Fatalf("expected content-type %q, got %q", want, res.Header.Get("Content-Type"))
	}

	var actual []scanner.Result
	if err := json.NewDecoder(res.Body).Decode(&actual); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected response payload: %+v", actual)
	}
}

func TestScanHandlerInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/scan", bytes.NewBufferString("not-json"))
	rec := httptest.NewRecorder()

	ScanHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestScanHandlerRunError(t *testing.T) {
	stubRunScan(t, func(scanner.ScanJob) ([]scanner.Result, error) {
		return nil, errors.New("boom")
	})

	payload := ScanRequest{Targets: []string{"192.168.1.10"}, Ports: []int{22}}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/scan", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ScanHandler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestNewRouterRoutesScan(t *testing.T) {
	called := false
	stubRunScan(t, func(scanner.ScanJob) ([]scanner.Result, error) {
		called = true
		return nil, nil
	})

	router := NewRouter()

	payload := ScanRequest{Targets: []string{"10.0.0.1"}, Ports: []int{80}}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/scan", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !called {
		t.Fatal("expected scan runner to be invoked")
	}
}
