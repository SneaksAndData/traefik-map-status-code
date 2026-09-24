package mapstatus

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// CreateConfig creates a new plugin configuration. This an entry point expected by Traefik.
func CreateConfig() *Config {
	return &Config{RemoveBody: true}
}

// Config the plugin configuration.
type Config struct {
	// From is the HTTP status code to map from. Can be a single status code or range (e.g. "404,401" or "400-499").
	From string `json:"from"`

	// To is the HTTP status code to map to. Should be a single valid HTTP status code.
	To string `json:"to"`

	// Debug enables mapping and response status logs.
	Debug bool `json:"debug"`

	// RemoveBody removes the body only when the response status matches a mapping.
	RemoveBody bool `json:"removeBody"`
}

type Plugin struct {
	next       http.Handler
	inputCodes map[int]int
	name       string
	debug      bool
	removeBody bool
}

// New creates a new plugin instance. This an entry point expected by Traefik.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	target, err := strconv.Atoi(strings.TrimSpace(config.To))
	if err != nil {
		return nil, fmt.Errorf("target must be an integer, but was: %s", config.To)
	}

	inputCodes := buildStatusCodeMap(config.From, target)
	if config.Debug {
		fmt.Printf("[mapstatus] name=%s inputCodes=%v\n", name, inputCodes)
	}
	return &Plugin{next: next, inputCodes: inputCodes, name: name, debug: config.Debug, removeBody: config.RemoveBody}, nil
}

func (p *Plugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	nrw := newResponseWriterInterceptor(p.inputCodes, rw, p.debug, p.removeBody)
	p.next.ServeHTTP(nrw, req)
	if nrw.dropBody {
		// TrailerPrefix fields can be added after the final body write.
		nrw.removeBodyHeaders()
	}
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
