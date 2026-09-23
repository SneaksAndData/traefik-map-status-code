package mapstatus

import (
	"fmt"
	"net/http"
)

var _ http.ResponseWriter = (*responseWriterInterceptor)(nil)

// responseWriterInterceptor is an http.ResponseWriter that maps status codes according to a provided mapping.
type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCodeMapping map[int]int
	debug             bool
}

// newResponseWriterInterceptor creates a new responseWriterInterceptor that maps status codes according to the provided mapping.
func newResponseWriterInterceptor(statusCodeMapping map[int]int, underlying http.ResponseWriter, debug bool) http.ResponseWriter {
	return &responseWriterInterceptor{
		statusCodeMapping: statusCodeMapping,
		ResponseWriter:    underlying,
		debug:             debug,
	}
}

// WriteHeader intercepts the WriteHeader call and maps the status code if it matches the mapping.
func (ri *responseWriterInterceptor) WriteHeader(statusCode int) {
	oldStatus := statusCode
	if ri.statusCodeMapping[statusCode] > 0 {
		statusCode = ri.statusCodeMapping[statusCode]
	}

	if ri.debug {
		fmt.Printf("[mapstatus] oldStatus=%d statusCode=%d\n", oldStatus, statusCode)
	}
	ri.ResponseWriter.WriteHeader(statusCode)
}
