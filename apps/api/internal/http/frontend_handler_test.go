package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRootServesFrontendIndex(t *testing.T) {
	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "<title>Agent Control Plane</title>") {
		t.Fatalf("expected frontend index at root")
	}
}

func TestFrontendAssetPathServedByAPI(t *testing.T) {
	r := NewRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/app.js", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "const state =") {
		t.Fatalf("expected frontend asset content")
	}
}
