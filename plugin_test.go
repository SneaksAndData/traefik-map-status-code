package mapstatus_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	mapstatus "github.com/SneaksAndData/traefik-map-status-code"
)

func TestCreateConfigRemoveBody(t *testing.T) {
	for _, tc := range []struct {
		name string
		json string
		want bool
	}{
		{name: "default", json: `{}`, want: true},
		{name: "explicit false", json: `{"removeBody":false}`, want: false},
		{name: "explicit true", json: `{"removeBody":true}`, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := mapstatus.CreateConfig()
			if err := json.Unmarshal([]byte(tc.json), config); err != nil {
				t.Fatal(err)
			}
			if config.RemoveBody != tc.want {
				t.Errorf("RemoveBody = %t, want %t", config.RemoveBody, tc.want)
			}
		})
	}
}

func newPlugin(t *testing.T, configJSON string, next http.HandlerFunc) http.Handler {
	t.Helper()
	config := mapstatus.CreateConfig()
	if err := json.Unmarshal([]byte(configJSON), config); err != nil {
		t.Fatal(err)
	}
	handler, err := mapstatus.New(context.Background(), next, config, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestPluginResponseBody(t *testing.T) {
	for _, tc := range []struct {
		name       string
		config     string
		statuses   []int
		wantStatus int
		wantBody   string
	}{
		{
			name: "mapped body removed by default", config: `{"from":"404","to":"200"}`,
			statuses: []int{404}, wantStatus: 200,
		},
		{
			name: "disabled preserves mapped body", config: `{"from":"404","to":"200","removeBody":false}`,
			statuses: []int{404}, wantStatus: 200, wantBody: "payload",
		},
		{
			name: "unmapped body preserved", config: `{"from":"404","to":"200"}`,
			statuses: []int{500}, wantStatus: 500, wantBody: "payload",
		},
		{
			name: "implicit 200 mapped", config: `{"from":"200","to":"201"}`,
			wantStatus: 201,
		},
		{
			name: "implicit 200 mapped with removal disabled", config: `{"from":"200","to":"201","removeBody":false}`,
			wantStatus: 201, wantBody: "payload",
		},
		{
			name: "implicit 200 unmapped", config: `{"from":"404","to":"201"}`,
			wantStatus: 200, wantBody: "payload",
		},
		{
			name: "duplicate final header cannot restore body", config: `{"from":"404","to":"200"}`,
			statuses: []int{404, 500}, wantStatus: 200,
		},
		{
			name: "duplicate final header cannot remove body", config: `{"from":"404","to":"200"}`,
			statuses: []int{500, 404}, wantStatus: 500, wantBody: "payload",
		},
		{
			name: "identical mapped status still removes body", config: `{"from":"404","to":"404"}`,
			statuses: []int{404}, wantStatus: 404,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := newPlugin(t, tc.config, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", "7")
				w.Header().Set("Content-Type", "text/plain")
				w.Header().Set("X-Preserved", "yes")
				for _, status := range tc.statuses {
					w.WriteHeader(status)
				}
				for _, data := range []string{"", "pay", "load"} {
					n, err := w.Write([]byte(data))
					if n != len(data) || err != nil {
						t.Errorf("Write(%q) = (%d, %v), want (%d, nil)", data, n, err, len(data))
					}
					// A header after the first Write must not change the decision either.
					w.WriteHeader(http.StatusNotFound)
				}
			})
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
			response := recorder.Result()
			defer func() {
				if err := response.Body.Close(); err != nil {
					t.Errorf("closing response body: %v", err)
				}
			}()
			if response.StatusCode != tc.wantStatus {
				t.Errorf("status = %d, want %d", response.StatusCode, tc.wantStatus)
			}
			if got := recorder.Body.String(); got != tc.wantBody {
				t.Errorf("body = %q, want %q", got, tc.wantBody)
			}
			wantLength := "7"
			if tc.wantBody == "" {
				wantLength = ""
			}
			if got := response.Header.Get("Content-Length"); got != wantLength {
				t.Errorf("Content-Length = %q, want %q", got, wantLength)
			}
			if response.Header.Get("Content-Type") != "text/plain" || response.Header.Get("X-Preserved") != "yes" {
				t.Errorf("unrelated headers changed: %v", response.Header)
			}
		})
	}
}

func TestPluginBodyFraming(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config string
		drop   bool
	}{
		{name: "mapped", config: `{"from":"404","to":"200"}`, drop: true},
		{name: "disabled", config: `{"from":"404","to":"200","removeBody":false}`},
		{name: "unmapped", config: `{"from":"500","to":"200"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := newPlugin(t, tc.config, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", "7")
				w.Header().Set("Transfer-Encoding", "chunked")
				w.Header().Add("Trailer", "X-Checksum, X-Other")
				w.Header().Add("Trailer", "X-Extra")
				w.Header().Set("X-Checksum", "early")
				w.Header().Set("X-Other", "early")
				w.Header().Set("X-Extra", "early")
				w.Header().Set(http.TrailerPrefix+"X-Early", "early")
				w.WriteHeader(http.StatusNotFound)
				if n, err := w.Write([]byte("payload")); n != 7 || err != nil {
					t.Errorf("Write = (%d, %v), want (7, nil)", n, err)
				}
				w.Header().Set("X-Checksum", "late")
				w.Header().Set(http.TrailerPrefix+"X-Late", "late")
			})
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
			response := recorder.Result()
			defer func() {
				if err := response.Body.Close(); err != nil {
					t.Errorf("closing response body: %v", err)
				}
			}()
			if tc.drop {
				for _, name := range []string{"Content-Length", "Transfer-Encoding", "Trailer", "X-Checksum", "X-Other", "X-Extra", http.TrailerPrefix + "X-Early", http.TrailerPrefix + "X-Late"} {
					if values, ok := response.Header[name]; ok {
						t.Errorf("removed body retained header %s: %v", name, values)
					}
				}
				if recorder.Body.Len() != 0 || len(response.Trailer) != 0 {
					t.Errorf("body = %q, trailers = %v; want both empty", recorder.Body.String(), response.Trailer)
				}
			} else {
				if recorder.Body.String() != "payload" ||
					response.Header.Get("Content-Length") != "7" ||
					response.Header.Get("Transfer-Encoding") != "chunked" ||
					!reflect.DeepEqual(response.Header.Values("Trailer"), []string{"X-Checksum, X-Other", "X-Extra"}) {
					t.Errorf("body or framing changed: body = %q, headers = %v", recorder.Body.String(), response.Header)
				}
				for name, want := range map[string]string{"X-Checksum": "late", "X-Other": "early", "X-Extra": "early", "X-Early": "early", "X-Late": "late"} {
					if got := response.Trailer.Get(name); got != want {
						t.Errorf("trailer %s = %q, want %q", name, got, want)
					}
				}
			}
		})
	}
}

// ResponseRecorder treats 1xx as final, so record informational headers separately.
type informationalRecorder struct {
	*httptest.ResponseRecorder
	statuses []int
}

func (r *informationalRecorder) WriteHeader(status int) {
	r.statuses = append(r.statuses, status)
	if status >= 100 && status < 200 && status != http.StatusSwitchingProtocols {
		return
	}
	r.ResponseRecorder.WriteHeader(status)
}

func TestPluginInformationalHeaders(t *testing.T) {
	for _, tc := range []struct {
		name           string
		statuses       []int
		wantStatus     []int
		wantBody       string
		wantWriteError error
	}{
		{name: "informational then mapped", statuses: []int{103, 100, 404, 500}, wantStatus: []int{103, 100, 200}},
		{name: "informational then unmapped", statuses: []int{103, 500, 404}, wantStatus: []int{103, 500}, wantBody: "payload"},
		{name: "informational then implicit mapped", statuses: []int{103}, wantStatus: []int{103, 201}},
		{name: "101 is final", statuses: []int{101, 404}, wantStatus: []int{101}, wantBody: "payload", wantWriteError: http.ErrBodyNotAllowed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := `{"from":"404","to":"200"}`
			if tc.name == "informational then implicit mapped" {
				config = `{"from":"200","to":"201"}`
			}
			handler := newPlugin(t, config, func(w http.ResponseWriter, _ *http.Request) {
				for _, status := range tc.statuses {
					w.WriteHeader(status)
				}
				n, err := w.Write([]byte("payload"))
				wantN := len("payload")
				if tc.wantWriteError != nil {
					wantN = 0
				}
				if n != wantN || err != tc.wantWriteError {
					t.Errorf("Write = (%d, %v), want (%d, %v)", n, err, wantN, tc.wantWriteError)
				}
			})
			recorder := &informationalRecorder{ResponseRecorder: httptest.NewRecorder()}
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
			if !reflect.DeepEqual(recorder.statuses, tc.wantStatus) {
				t.Errorf("statuses = %v, want %v", recorder.statuses, tc.wantStatus)
			}
			if got := recorder.Body.String(); got != tc.wantBody {
				t.Errorf("body = %q, want %q", got, tc.wantBody)
			}
		})
	}
}

func TestPluginRemovedBodyOverHTTP(t *testing.T) {
	for _, framing := range []string{"content length", "chunked with trailers"} {
		t.Run(framing, func(t *testing.T) {
			handler := newPlugin(t, `{"from":"404","to":"200"}`, func(w http.ResponseWriter, _ *http.Request) {
				if framing == "content length" {
					w.Header().Set("Content-Length", "7")
				} else {
					w.Header().Set("Transfer-Encoding", "chunked")
					w.Header().Set("Trailer", "X-Checksum")
				}
				w.WriteHeader(http.StatusNotFound)
				if n, err := w.Write([]byte("payload")); n != 7 || err != nil {
					t.Errorf("Write = (%d, %v), want (7, nil)", n, err)
				}
				w.Header().Set("X-Checksum", "late")
				w.Header().Set(http.TrailerPrefix+"X-Late", "late")
			})
			server := httptest.NewServer(handler)
			defer server.Close()
			client := server.Client()
			client.Timeout = 5 * time.Second
			response, err := client.Get(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := response.Body.Close(); err != nil {
					t.Errorf("closing response body: %v", err)
				}
			}()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("reading removed body: %v", err)
			}
			if response.StatusCode != http.StatusOK || len(body) != 0 {
				t.Errorf("status = %d, body = %q; want 200 and empty body", response.StatusCode, body)
			}
			if response.ContentLength != 0 || len(response.TransferEncoding) != 0 || len(response.Trailer) != 0 {
				t.Errorf("unexpected framing: length = %d, encoding = %v, trailers = %v",
					response.ContentLength, response.TransferEncoding, response.Trailer)
			}
		})
	}
}
