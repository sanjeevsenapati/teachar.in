package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"teachar.in/middleware"
)

func TestRateLimiter(t *testing.T) {
	limiter := middleware.NewRateLimiter(3, 1*time.Minute)

	handler := limiter.LimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/api/status", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// First 3 requests should pass
	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected request %d to pass with 200 OK, got %d", i+1, rr.Code)
		}
	}

	// 4th request should be blocked with 429 Too Many Requests
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 4th request to be blocked with 429, got %d", rr.Code)
	}
}
