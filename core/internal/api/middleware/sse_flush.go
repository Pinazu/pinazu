package middleware

import (
	"bufio"
	"net"
	"net/http"
	"strings"
)

// SSEFlushWriter wraps http.ResponseWriter to automatically flush SSE data
type SSEFlushWriter struct {
	http.ResponseWriter
	flusher http.Flusher
	isSSE   bool
}

// NewSSEFlushWriter creates a new SSEFlushWriter
func NewSSEFlushWriter(w http.ResponseWriter) *SSEFlushWriter {
	flusher, _ := w.(http.Flusher)
	return &SSEFlushWriter{
		ResponseWriter: w,
		flusher:        flusher,
		isSSE:          false,
	}
}

// Header returns the header map that will be sent by WriteHeader
func (s *SSEFlushWriter) Header() http.Header {
	return s.ResponseWriter.Header()
}

// WriteHeader sends an HTTP response header with the provided status code
func (s *SSEFlushWriter) WriteHeader(statusCode int) {
	// Check if this is an SSE response by examining content-type
	contentType := s.Header().Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		s.isSSE = true
		// Ensure SSE headers are properly set for immediate streaming
		s.Header().Set("Cache-Control", "no-cache")
		s.Header().Set("Connection", "keep-alive")
		s.Header().Set("Access-Control-Allow-Origin", "*")
		s.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

		// Additional headers for better streaming support
		s.Header().Set("Transfer-Encoding", "chunked")
		s.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
		s.Header().Set("X-Content-Type-Options", "nosniff")

		// Keep-alive with longer timeout
		s.Header().Set("Keep-Alive", "timeout=300, max=1000")
	}
	s.ResponseWriter.WriteHeader(statusCode)
}

// Write writes the data to the connection as part of an HTTP reply
func (s *SSEFlushWriter) Write(data []byte) (int, error) {
	n, err := s.ResponseWriter.Write(data)

	// If this is an SSE response and we can flush, do it immediately
	if s.isSSE && s.flusher != nil && err == nil {
		s.flusher.Flush()
	}

	return n, err
}

// Hijack lets the caller take over the connection for WebSocket upgrades
func (s *SSEFlushWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := s.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Flush implements http.Flusher interface
func (s *SSEFlushWriter) Flush() {
	if s.flusher != nil {
		s.flusher.Flush()
	}
}

// SSEAutoFlushMiddleware automatically flushes Server-Sent Events responses
func SSEAutoFlushMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap the response writer with our SSE flush writer
			sseWriter := NewSSEFlushWriter(w)

			// Continue with the next handler using our wrapped writer
			next.ServeHTTP(sseWriter, r)
		})
	}
}
