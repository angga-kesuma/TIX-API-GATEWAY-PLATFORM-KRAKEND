package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterStaticResponse_NilConfigReturnsNext(t *testing.T) {
	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("next"))
	})

	h, err := RegisterStaticResponse(next, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if h == nil {
		t.Fatalf("expected non-nil handler")
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/whatever", nil)

	h.ServeHTTP(rr, req)

	if !nextCalled {
		t.Fatalf("expected next handler to be called")
	}

	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected status %d, got %d", http.StatusTeapot, rr.Code)
	}

	if rr.Body.String() != "next" {
		t.Fatalf("expected body %q, got %q", "next", rr.Body.String())
	}
}

func TestRegisterStaticResponse_PathMatch_ReturnsStaticResponse(t *testing.T) {
	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("should-not-be-called"))
	})

	cfg := map[string]any{
		"/static": map[string]any{
			"status":  200.0,
			"body":    "hello static",
			"headers": map[string]any{"X-Static": "1"},
		},
	}

	h, err := RegisterStaticResponse(next, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/static", nil)

	h.ServeHTTP(rr, req)

	if nextCalled {
		t.Fatalf("expected static handler, next should not be called")
	}

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if got := rr.Body.String(); got != "hello static" {
		t.Fatalf("expected body %q, got %q", "hello static", got)
	}

	if got := rr.Header().Get("X-Static"); got != "1" {
		t.Fatalf("expected header X-Static=1, got %q", got)
	}
}

func TestRegisterStaticResponse_PathNotMatch_CallsNext(t *testing.T) {
	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("from-next"))
	})

	cfg := map[string]any{
		"/static": map[string]any{
			"status":  200.0,
			"body":    "hello static",
			"headers": map[string]any{"X-Static": "1"},
		},
	}

	h, err := RegisterStaticResponse(next, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/other", nil)

	h.ServeHTTP(rr, req)

	if !nextCalled {
		t.Fatalf("expected next handler to be called")
	}

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	if got := rr.Body.String(); got != "from-next" {
		t.Fatalf("expected body %q, got %q", "from-next", got)
	}
}

func TestNewStaticResponseConfig_Success(t *testing.T) {
	cfg := map[string]any{
		"/static": map[string]any{
			"status":  204.0,
			"body":    "no content",
			"headers": map[string]any{"X-Test": "abc"},
		},
	}

	mapping, err := newStaticResponseConfig(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mapping) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(mapping))
	}

	resp, ok := mapping["/static"]
	if !ok {
		t.Fatalf("expected key /static to exist")
	}

	if resp.Status != 204 {
		t.Fatalf("expected status=204, got %d", resp.Status)
	}

	if resp.Body != "no content" {
		t.Fatalf("expected body %q, got %q", "no content", resp.Body)
	}

	if v := resp.Headers["X-Test"]; v != "abc" {
		t.Fatalf("expected header X-Test=abc, got %q", v)
	}
}

func TestNewStaticResponseConfig_InvalidRawType(t *testing.T) {
	cfg := map[string]any{
		"/bad": "this-is-wrong-type",
	}

	_, err := newStaticResponseConfig(cfg)
	if err == nil {
		t.Fatalf("expected error for invalid cfg, got nil")
	}
}
