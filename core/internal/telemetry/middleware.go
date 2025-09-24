package telemetry

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// HTTPMiddleware creates an OpenTelemetry HTTP middleware
func HTTPMiddleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a custom operation name based on the route
			operation := fmt.Sprintf("%s %s", r.Method, r.URL.Path)

			// Use otelhttp handler for automatic instrumentation
			handler := otelhttp.NewHandler(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Extract trace from context and add custom attributes
					span := trace.SpanFromContext(r.Context())
					if span.IsRecording() {
						span.SetAttributes(
							attribute.String("service.name", serviceName),
							attribute.String("http.user_agent", r.UserAgent()),
							attribute.String("http.remote_addr", r.RemoteAddr),
						)

						// Add request headers as attributes
						for k, v := range r.Header {
							if k == "Authorization" || k == "Cookie" {
								continue // Skip sensitive headers
							}
							if len(v) > 0 {
								span.SetAttributes(attribute.String(fmt.Sprintf("http.request.header.%s", k), v[0]))
							}
						}
					}

					next.ServeHTTP(w, r)
				}),
				operation,
				otelhttp.WithTracerProvider(otel.GetTracerProvider()),
				otelhttp.WithPropagators(otel.GetTextMapPropagator()),
			)

			handler.ServeHTTP(w, r)
		})
	}
}

// ExtractTraceContext extracts trace context from HTTP request
func ExtractTraceContext(r *http.Request) trace.SpanContext {
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	return trace.SpanContextFromContext(ctx)
}

// InjectTraceContext injects trace context into HTTP headers
func InjectTraceContext(r *http.Request, spanCtx trace.SpanContext) {
	ctx := trace.ContextWithSpanContext(r.Context(), spanCtx)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))
}

// ResponseWriter wraps http.ResponseWriter to capture status code
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// NewResponseWriter creates a new ResponseWriter
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

// WriteHeader captures the status code
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Status returns the status code
func (rw *ResponseWriter) Status() int {
	return rw.statusCode
}

// TracingMiddleware is a custom middleware that adds additional tracing capabilities
func TracingMiddleware(serviceName string) func(http.Handler) http.Handler {
	tracer := otel.Tracer(serviceName)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract parent span context from headers
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Start a new span
			ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path),
				trace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.url", r.URL.String()),
					attribute.String("http.target", r.URL.Path),
					attribute.String("http.host", r.Host),
					attribute.String("http.scheme", r.URL.Scheme),
					attribute.String("net.host.name", r.Host),
					attribute.String("http.user_agent", r.UserAgent()),
				),
				trace.WithSpanKind(trace.SpanKindServer),
			)
			defer span.End()

			// Wrap response writer to capture status code
			rw := NewResponseWriter(w)

			// Pass the request with the new context
			next.ServeHTTP(rw, r.WithContext(ctx))

			// Set status code attribute
			span.SetAttributes(attribute.Int("http.status_code", rw.Status()))

			// Set span status based on HTTP status code
			if rw.Status() >= 400 {
				span.SetStatus(codes.Error, http.StatusText(rw.Status()))
			}
		})
	}
}
