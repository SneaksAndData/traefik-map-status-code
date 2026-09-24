package mapstatus

import (
	"fmt"
	"net/http"
	"strings"
)

var _ http.ResponseWriter = (*responseWriterInterceptor)(nil)

// responseWriterInterceptor is an http.ResponseWriter that maps status codes according to a provided mapping.
type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCodeMapping map[int]int
	debug             bool
	removeBody        bool
	wroteHeader       bool
	dropBody          bool
}

// newResponseWriterInterceptor creates a new responseWriterInterceptor that maps status codes according to the provided mapping.
func newResponseWriterInterceptor(statusCodeMapping map[int]int, underlying http.ResponseWriter, debug, removeBody bool) *responseWriterInterceptor {
	return &responseWriterInterceptor{
		statusCodeMapping: statusCodeMapping,
		ResponseWriter:    underlying,
		debug:             debug,
		removeBody:        removeBody,
	}
}

// WriteHeader intercepts the WriteHeader call and maps the status code if it matches the mapping.
func (ri *responseWriterInterceptor) WriteHeader(statusCode int) {
	if ri.wroteHeader {
		return
	}

	oldStatus := statusCode
	mappedStatus := ri.statusCodeMapping[statusCode]
	if mappedStatus > 0 {
		statusCode = mappedStatus
	}

	// Informational responses do not commit the final response, except for upgrades.
	if statusCode < 100 || statusCode >= 200 || statusCode == http.StatusSwitchingProtocols {
		ri.wroteHeader = true
		ri.dropBody = ri.removeBody && mappedStatus > 0
		if ri.dropBody {
			ri.removeBodyHeaders()
		}
	}

	if ri.debug {
		fmt.Printf("[mapstatus] oldStatus=%d statusCode=%d\n", oldStatus, statusCode)
	}
	ri.ResponseWriter.WriteHeader(statusCode)
}

func (ri *responseWriterInterceptor) Write(p []byte) (int, error) {
	if !ri.wroteHeader {
		ri.WriteHeader(http.StatusOK)
	}
	if ri.dropBody {
		return len(p), nil
	}
	return ri.ResponseWriter.Write(p)
}

func (ri *responseWriterInterceptor) removeBodyHeaders() {
	header := ri.Header()
	for _, trailers := range header.Values("Trailer") {
		for _, trailer := range strings.Split(trailers, ",") {
			header.Del(strings.TrimSpace(trailer))
		}
	}
	header.Del("Content-Length")
	header.Del("Transfer-Encoding")
	header.Del("Trailer")
	for name := range header {
		if strings.HasPrefix(name, http.TrailerPrefix) {
			delete(header, name)
		}
	}
}
