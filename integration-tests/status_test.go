//go:build integration

package integration_test

import (
	"io"
	"net/http"
	"testing"
	"time"
)

func TestStatusMapping(t *testing.T) {
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	defer client.CloseIdleConnections()

	for _, tc := range []struct {
		name      string
		url       string
		status    int
		emptyBody bool
	}{
		{
			name:   "backend returns 404",
			url:    "http://localhost:8081/missing-integration-test",
			status: http.StatusNotFound,
		},
		{
			name:      "traefik maps 404 to 200",
			url:       "http://localhost:8080/missing-integration-test",
			status:    http.StatusOK,
			emptyBody: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response, err := client.Get(tc.url)
			if err != nil {
				t.Fatalf("GET %s: %v; start the environment separately with just up", tc.url, err)
			}
			defer response.Body.Close()

			if response.StatusCode != tc.status {
				t.Errorf("GET %s: status = %d, want %d", tc.url, response.StatusCode, tc.status)
			}
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("GET %s: reading body: %v", tc.url, err)
			}
			if (len(body) == 0) != tc.emptyBody {
				t.Errorf("GET %s: body length = %d, want empty = %t", tc.url, len(body), tc.emptyBody)
			}
		})
	}
}
