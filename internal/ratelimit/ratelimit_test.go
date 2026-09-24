package ratelimit_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vkemu/internal/ratelimit"
)

func TestRateLimitBlocksBurst(t *testing.T) {
	now := time.Unix(0, 0)
	limiter := ratelimit.New(ratelimit.Options{
		Rate:  1,
		Burst: 2,
		Now:   func() time.Time { return now },
	})
	defer limiter.Close()

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	status := func() int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:5555"
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := status(); code != http.StatusOK {
		t.Fatalf("first request: %d", code)
	}
	if code := status(); code != http.StatusOK {
		t.Fatalf("second request: %d", code)
	}
	if code := status(); code != http.StatusTooManyRequests {
		t.Fatalf("third request should be limited: %d", code)
	}

	now = now.Add(2 * time.Second)
	if code := status(); code != http.StatusOK {
		t.Fatalf("after refill: %d", code)
	}
}

func TestRateLimitSeparatesClients(t *testing.T) {
	limiter := ratelimit.New(ratelimit.Options{Rate: 1, Burst: 1})
	defer limiter.Close()
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func(addr string) int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = addr
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := call("10.0.0.1:1"); code != http.StatusOK {
		t.Fatalf("client A first: %d", code)
	}
	if code := call("10.0.0.1:2"); code != http.StatusTooManyRequests {
		t.Fatalf("client A second: %d", code)
	}
	if code := call("10.0.0.2:1"); code != http.StatusOK {
		t.Fatalf("client B first: %d", code)
	}
}

func TestMaxConnsRejectsOverflow(t *testing.T) {
	limiter := ratelimit.New(ratelimit.Options{MaxConns: 1})
	defer limiter.Close()

	release := make(chan struct{})
	entered := make(chan struct{})
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		<-release
		w.WriteHeader(http.StatusOK)
	}))

	go func() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	}()
	<-entered

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	close(release)
}

func TestSkipBypassesLimit(t *testing.T) {
	limiter := ratelimit.New(ratelimit.Options{
		Rate:  1,
		Burst: 1,
		Skip:  func(r *http.Request) bool { return r.URL.Path == "/longpoll" },
	})
	defer limiter.Close()
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func(path string) int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = "1.1.1.1:1"
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	for i := 0; i < 5; i++ {
		if code := call("/longpoll"); code != http.StatusOK {
			t.Fatalf("longpoll request %d: %d", i, code)
		}
	}
}
