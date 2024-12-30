package versioning_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/poly-gun/go-middleware/middleware/versioning"
)

func Test(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		datum := map[string]interface{}{
			"Key-1": "Value-1",
			"Key-2": "Value-2",
			"Key-3": "Value-3",
		}

		defer json.NewEncoder(w).Encode(datum)

		w.Header().Set("Content-Type", "application/json")

		w.WriteHeader(http.StatusOK)

		return
	})

	t.Run("Middleware", func(t *testing.T) {
		t.Run("Successful-Service-Headers", func(t *testing.T) {
			const header = "X-Service-Version"

			const version = "1.0.0"
			server := httptest.NewServer(versioning.New().Settings(func(o *versioning.Options) {
				o.Service = version
			}).Handler(handler))

			defer server.Close()

			client := server.Client()
			request, e := http.NewRequest(http.MethodGet, server.URL, nil)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Request: %v", e)
			}

			response, e := client.Do(request)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Response: %v", e)
			}

			defer response.Body.Close()

			if v := response.Header.Get(header); v != version {
				t.Errorf("Incorrect Value for Header %s, Expected: %v, Received: %s", header, version, v)
			} else {
				t.Logf("Correctly Verified %s Value: %s", header, version)
			}
		})

		t.Run("Successful-API-Headers", func(t *testing.T) {
			const header = "X-API-Version"

			const version = "1.0.0"
			server := httptest.NewServer(versioning.New().Settings(func(o *versioning.Options) {
				o.API = version
			}).Handler(handler))

			defer server.Close()

			client := server.Client()
			request, e := http.NewRequest(http.MethodGet, server.URL, nil)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Request: %v", e)
			}

			response, e := client.Do(request)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Response: %v", e)
			}

			defer response.Body.Close()

			if v := response.Header.Get(header); v != version {
				t.Errorf("Incorrect Value for Header %s, Expected: %v, Received: %s", header, version, v)
			} else {
				t.Logf("Correctly Verified %s Value: %s", header, version)
			}
		})

		t.Run("Successful-API-Headers-From-Istio-Mutations", func(t *testing.T) {
			const header = "X-API-Version"

			const version = "1.0.0"

			server := httptest.NewServer(versioning.New().Handler(handler))

			defer server.Close()

			client := server.Client()
			request, e := http.NewRequest(http.MethodGet, server.URL, nil)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Request: %v", e)
			}

			request.Header.Set(header, version)

			response, e := client.Do(request)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Response: %v", e)
			}

			defer response.Body.Close()

			if v := response.Header.Get(header); v != version {
				t.Errorf("Incorrect Value for Header %s, Expected: %v, Received: %s", header, version, v)
			} else {
				t.Logf("Correctly Verified %s Value: %s", header, version)
			}
		})

		t.Run("No-Emitted-Warning", func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug,
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			server := httptest.NewServer(versioning.New().Settings(func(o *versioning.Options) {
				o.Warnings = false
			}).Handler(handler))

			defer server.Close()

			client := server.Client()
			request, e := http.NewRequest(http.MethodGet, server.URL, nil)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Request: %v", e)
			}

			response, e := client.Do(request)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Response: %v", e)
			}

			defer response.Body.Close()

			if buffer.String() != "" {
				t.Errorf("Unexpected Log-Level Level Achieved: %s", buffer.String())
			} else {
				t.Logf("No Warnings Received")
			}
		})
	})

	t.Run("Context", func(t *testing.T) {
		t.Run("Default", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			value := versioning.Value(ctx)

			if value != nil {
				t.Errorf("Unexpected Non-Default Value: %v", value)
			} else {
				t.Logf("Successful Default Value Received = %v", value)
			}
		})

		t.Run("User-Specified-Value", func(t *testing.T) {
			t.Parallel()

			v := &versioning.Versions{
				API:     "1.0.0",
				Service: "0.0.0",
			}

			ctx := context.WithValue(context.Background(), "x-testing-key", v)

			value := versioning.Value(ctx)

			t.Run("API", func(t *testing.T) {
				if value.API != "1.0.0" {
					t.Errorf("Unexpected Context Value Received: %v", value)
				}
			})

			t.Run("Service", func(t *testing.T) {
				if value.Service != "0.0.0" {
					t.Errorf("Unexpected Context Value Received: %v", value)
				}
			})

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

			versioning.Value(ctx)

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

			var v = &versioning.Versions{
				API:     "1.0.0",
				Service: "0.0.0",
			}

			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug,
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			ctx := context.WithValue(context.Background(), "x-testing-key", v)

			versioning.Value(ctx)

			if buffer.String() != "" {
				t.Errorf("Unexpected Log Message: %s", buffer.String())
			}
		})

		t.Run("Context-Key-Value-Testing-Trace-Log-Message", func(t *testing.T) {
			t.Parallel()

			var v = &versioning.Versions{
				API:     "1.0.0",
				Service: "0.0.0",
			}

			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug - 4, // the trace log level
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			ctx := context.WithValue(context.Background(), "x-testing-key", v)

			versioning.Value(ctx)

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
