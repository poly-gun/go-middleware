package envoy_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/poly-gun/go-middleware/middleware/envoy"
)

func Test(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		datum := map[string][]string{
			"Key-1": {"Value-A", "Value-B"},
		}

		if v := envoy.Value(ctx); v != nil {
			maps.Copy(datum, *v)
		}

		defer json.NewEncoder(w).Encode(datum)

		w.Header().Set("Content-Type", "application/json")

		w.WriteHeader(http.StatusOK)

		return
	})

	t.Run("Middleware", func(t *testing.T) {
		t.Run("Envoy-Header-Response", func(t *testing.T) {
			server := httptest.NewServer(envoy.New().Settings(func(options *envoy.Options) { options.Debug = true }).Handler(handler))

			defer server.Close()

			client := server.Client()
			request, e := http.NewRequest(http.MethodGet, server.URL, nil)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Request: %v", e)
			}

			request.Header.Set("X-Envoy-Internal", "true")
			request.Header.Set("X-Envoy-Request-Count", "1")
			request.Header.Set("X-Envoy-Original-Path", "/v1/test")

			response, e := client.Do(request)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Response: %v", e)
			}

			defer response.Body.Close()

			var datum map[string][]string
			if e := json.NewDecoder(response.Body).Decode(&datum); e != nil {
				t.Fatalf("Unexpected Error While Parsing Response: %v", e)
			}

			t.Run("Header-X-Envoy-Internal", func(t *testing.T) {
				const key = "X-Envoy-Internal"

				values, ok := datum[key]
				if !(ok) {
					t.Errorf("Expected Response To Include Key (%s)", key)
				} else {
					t.Logf("Header (%s) Value(s): %v", key, values)
				}
			})

			t.Run("Header-X-Envoy-Request-Count", func(t *testing.T) {
				const key = "X-Envoy-Request-Count"

				values, ok := datum[key]
				if !(ok) {
					t.Errorf("Expected Response To Include Key (%s)", key)
				} else {
					t.Logf("Header (%s) Value(s): %v", key, values)
				}
			})

			t.Run("Header-X-Envoy-Original-Path", func(t *testing.T) {
				const key = "X-Envoy-Original-Path"

				values, ok := datum[key]
				if !(ok) {
					t.Errorf("Expected Response To Include Key (%s)", key)
				} else {
					t.Logf("Header (%s) Value(s): %v", key, values)
				}
			})
		})

		t.Run("Envoy-Debug-Messages", func(t *testing.T) {
			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug,
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			server := httptest.NewServer(envoy.New().Settings(func(options *envoy.Options) { options.Debug = true }).Handler(handler))

			defer server.Close()

			client := server.Client()
			request, e := http.NewRequest(http.MethodGet, server.URL, nil)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Request: %v", e)
			}

			request.Header.Set("X-Envoy-Internal", "true")
			request.Header.Set("X-Envoy-Request-Count", "1")
			request.Header.Set("X-Envoy-Original-Path", "/v1/test")

			response, e := client.Do(request)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Response: %v", e)
			}

			defer response.Body.Close()

			var datum map[string]interface{}
			if e := json.Unmarshal(buffer.Bytes(), &datum); e != nil {
				t.Fatalf("Unexpected Error While Unmarshalling Buffer: %v", e)
			}

			t.Logf("Envoy-Debug-Messages (Buffer):\n%s", buffer.String())

			headers, valid := datum["headers"].(map[string]interface{})
			if !(valid) {
				t.Errorf("Unable to Typecast Headers to Expected Type: %v", headers)
			}

			t.Run("Log-Message-X-Envoy-Internal", func(t *testing.T) {
				const key = "X-Envoy-Internal"

				values, ok := headers[key]
				if !(ok) {
					t.Errorf("Expected Response To Include Key (%s)", key)
				} else {
					t.Logf("Header (%s) Value(s): %v", key, values)
				}
			})

			t.Run("Log-Message-X-Envoy-Request-Count", func(t *testing.T) {
				const key = "X-Envoy-Request-Count"

				values, ok := headers[key]
				if !(ok) {
					t.Errorf("Expected Response To Include Key (%s)", key)
				} else {
					t.Logf("Header (%s) Value(s): %v", key, values)
				}
			})

			t.Run("Log-Message-X-Envoy-Original-Path", func(t *testing.T) {
				const key = "X-Envoy-Original-Path"

				values, ok := headers[key]
				if !(ok) {
					t.Errorf("Expected Response To Include Key (%s)", key)
				} else {
					t.Logf("Header (%s) Value(s): %v", key, values)
				}
			})
		})

		t.Run("Envoy-No-Debug-Messages", func(t *testing.T) {
			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug,
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			server := httptest.NewServer(envoy.New().Settings(func(options *envoy.Options) { options.Debug = false }).Handler(handler))

			defer server.Close()

			client := server.Client()
			request, e := http.NewRequest(http.MethodGet, server.URL, nil)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Request: %v", e)
			}

			request.Header.Set("X-Envoy-Internal", "true")
			request.Header.Set("X-Envoy-Request-Count", "1")
			request.Header.Set("X-Envoy-Original-Path", "/v1/test")

			response, e := client.Do(request)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Response: %v", e)
			}

			defer response.Body.Close()

			if buffer.Len() > 0 {
				t.Errorf("Unexpected Logging from Envoy Middleware:\n%s", buffer.String())
			}
		})
	})

	t.Run("Context", func(t *testing.T) {
		t.Run("Default", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			value := envoy.Value(ctx)

			if value != nil {
				t.Errorf("Unexpected Non-Default Context Value Received: %v", value)
			}

			t.Logf("Successful Default Value Received = %v", value)
		})

		t.Run("User-Specified-Value", func(t *testing.T) {
			t.Parallel()

			v := http.Header{"X-Envoy-Test-Header": []string{"Value-1", "Value-2"}}
			ctx := context.WithValue(context.Background(), "x-testing-key", &v)
			value := envoy.Value(ctx)

			if value != &v {
				t.Errorf("Unexpected Context Value Received: %v, Expected: %s", value, v)
			}

			t.Logf("Successful User-Provided Value Received = %v", value)
		})
	})

	t.Run("Logging", func(t *testing.T) {
		t.Run("Context-Key-Value-Warning-Log-Level", func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug,
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			ctx := context.Background()

			envoy.Value(ctx)

			var message map[string]interface{}
			if e := json.Unmarshal(buffer.Bytes(), &message); e != nil {
				t.Fatalf("Fatal, Unexpected Error While Unmarshalling Log Message: %v", e)
			}

			if v, ok := message["level"]; ok {
				if typecast, valid := v.(string); valid {
					if typecast == slog.LevelWarn.String() {
						t.Logf("Successful, Expected Log-Level Level Achieved")
					} else {
						t.Errorf("Unexpected Log-Level Level: %s", typecast)
					}
				} else {
					t.Errorf("Unable to Typecast Level to String Type: %v", v)
				}
			} else {
				t.Errorf("No Valid Level Key Found: %v", message)
			}
		})

		t.Run("Context-Key-Value-No-Log-Message", func(t *testing.T) {
			t.Parallel()

			v := http.Header{"X-Envoy-Test-Header": []string{"Value-1", "Value-2"}}

			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug,
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			ctx := context.WithValue(context.Background(), "x-testing-key", &v)

			envoy.Value(ctx)

			if buffer.String() != "" {
				t.Errorf("Unexpected Log Message: %s", buffer.String())
			}
		})

		t.Run("Context-Key-Value-Testing-Trace-Log-Message", func(t *testing.T) {
			t.Parallel()

			v := http.Header{"X-Envoy-Test-Header": []string{"Value-1", "Value-2"}}

			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug - 4, // the trace log level
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			ctx := context.WithValue(context.Background(), "x-testing-key", &v)

			envoy.Value(ctx)

			if buffer.String() == "" {
				t.Errorf("Expected a Trace Testing Log Message")
			} else {
				t.Logf("Successfully Received a Trace Tesing Log Message:\n%s", buffer.String())
			}

			var message map[string]interface{}
			if e := json.Unmarshal(buffer.Bytes(), &message); e != nil {
				t.Fatalf("Fatal, Unexpected Error While Unmarshalling Log Message: %v", e)
			}

			if v, ok := message["level"]; ok {
				if typecast, valid := v.(string); valid {
					if typecast == (slog.LevelDebug - 4).String() {
						t.Logf("Successful, Expected Log-Level Level Achieved")
					} else {
						t.Errorf("Unexpected Log-Level Level: %s", typecast)
					}
				} else {
					t.Errorf("Unable to Typecast Level to String Type: %v", v)
				}
			} else {
				t.Errorf("No Valid Level Key Found: %v", message)
			}
		})
	})
}
