package mapstatus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Config the plugin configuration.
type Config struct {
	// From is the HTTP status code to map from. Can be a single status code or range (e.g. "404,401" or "400-499").
	From string `json:"from"`

	// To is the HTTP status code to map to. Should be a single valid HTTP status code.
	To string `json:"to"`
}

type MapStatusCode struct {
	next       http.Handler
	inputCodes map[int]int
	name       string
}

func NewMapStatusCode(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	target, err := strconv.Atoi(strings.TrimSpace(config.To))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("target must be an integer, but was: %s", config.To))
	}

	inputCodes := buildStatusCodeMap(config.From, target)
	return &MapStatusCode{next: next, inputCodes: inputCodes, name: name}, nil
}

func (p *MapStatusCode) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	nrw := NewResponseWriterInterceptor(p.inputCodes, rw)
	p.next.ServeHTTP(nrw, req)
}

func buildStatusCodeMap(from string, target int) map[int]int {
	inputCodes := make(map[int]int)

	for _, part := range strings.Split(from, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			start, _ := strconv.Atoi(strings.TrimSpace(bounds[0]))
			end, _ := strconv.Atoi(strings.TrimSpace(bounds[1]))

			for code := start; code <= end; code++ {
				inputCodes[code] = target
			}
			continue
		}

		code, err := strconv.Atoi(part)
		if err == nil {
			inputCodes[code] = target
		}
	}

	return inputCodes
}
