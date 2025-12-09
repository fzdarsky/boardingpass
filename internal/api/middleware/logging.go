// Package middleware provides HTTP middleware for the BoardingPass API server.
package middleware

import (
	"net/http"
	"time"

	"github.com/fzdarsky/boardingpass/internal/logging"
)

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += n
	return n, err
}

// Logging returns middleware that logs HTTP requests and responses.
func Logging(logger *logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			rw := newResponseWriter(w)

			// Process request
			next.ServeHTTP(rw, r)

			// Log request details
			duration := time.Since(start)

			logger.Info("http request", map[string]any{
				"method":      r.Method,
				"path":        r.URL.Path,
				"remote_addr": r.RemoteAddr,
				"user_agent":  r.UserAgent(),
				"status":      rw.statusCode,
				"duration_ms": duration.Milliseconds(),
				"bytes":       rw.written,
			})
		})
	}
}
