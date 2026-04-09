package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type flushRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (r *flushRecorder) Flush() {
	r.flushed = true
	r.ResponseRecorder.Flush()
}

func (r *flushRecorder) ReadFrom(src io.Reader) (int64, error) {
	return io.Copy(r.ResponseRecorder, src)
}

func TestLoggingMiddlewarePreservesFlusher(t *testing.T) {
	handler := loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer does not implement http.Flusher")
		}
		w.WriteHeader(http.StatusNoContent)
		flusher.Flush()
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/ai/template/session/test/message?stream=1", nil)
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}

	handler.ServeHTTP(rec, req)

	if !rec.flushed {
		t.Fatal("expected wrapped response writer to flush underlying recorder")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: %d", rec.Code)
	}
}
