package mapstatus

import "net/http"

var _ http.ResponseWriter = (*responseWriterInterceptor)(nil)

// responseWriterInterceptor is an http.ResponseWriter that maps status codes according to a provided mapping.
type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCodeMapping map[int]int
}

// NewResponseWriterInterceptor creates a new responseWriterInterceptor that maps status codes according to the provided mapping.
func NewResponseWriterInterceptor(statusCodeMapping map[int]int, underlying http.ResponseWriter) http.ResponseWriter {
	return &responseWriterInterceptor{
		statusCodeMapping: statusCodeMapping,
		ResponseWriter:    underlying,
	}
}

// WriteHeader intercepts the WriteHeader call and maps the status code if it matches the mapping.
func (ri *responseWriterInterceptor) WriteHeader(statusCode int) {
	if ri.statusCodeMapping[statusCode] > 0 {
		statusCode = ri.statusCodeMapping[statusCode]
	}

	ri.ResponseWriter.WriteHeader(statusCode)
}
