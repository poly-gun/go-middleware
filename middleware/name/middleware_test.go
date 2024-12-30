package name_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/poly-gun/go-middleware/middleware/name"
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
		t.Run("Successful-Default-Server-Headers", func(t *testing.T) {
			const k = "X-Server-Name"
			const v = "Test-Server-1"

			server := httptest.NewServer(name.New().Settings(func(options *name.Options) {
				options.Name = v
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

			failure := true
			for header := range response.Header {
				t.Logf("Header: %s - %v", header, response.Header[header])

				if header == k {
					for _, value := range response.Header[header] {
						if value == v {
							t.Logf("Successfully Verified Header %s: %s", k, value)

							failure = false
						}
					}
				}
			}

			if failure {
				t.Errorf("Incorrect Value for Header %s, Expected %v", k, v)
			}
		})

		t.Run("Successful-Custom-Server-Headers", func(t *testing.T) {
			const k = "X-Test-Server-Name"
			const v = "Test-Server-2"

			server := httptest.NewServer(name.New().Settings(func(options *name.Options) {
				options.Name = v
				options.Header = k
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

			failure := true
			for header := range response.Header {
				t.Logf("Header: %s - %v", header, response.Header[header])

				if header == k {
					for _, value := range response.Header[header] {
						if value == v {
							t.Logf("Successfully Verified Header %s: %s", "X-Test-Header", value)

							failure = false
						}
					}
				}
			}

			if failure {
				t.Errorf("Incorrect Value for Header %s, Expected %v", k, v)
			}
		})

		t.Run("Emitted-Default-Warning", func(t *testing.T) {
			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug,
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			server := httptest.NewServer(name.New().Handler(handler))

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

			var message map[string]interface{}
			if e := json.Unmarshal(buffer.Bytes(), &message); e != nil {
				t.Fatalf("Fatal, Unexpected Error While Unmarshalling Log Message: %v", e)
			}

			if v, ok := message["level"]; ok {
				if typecast, valid := v.(string); valid {
					if typecast == (slog.LevelWarn).String() {
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

		t.Run("No-Emitted-Warning", func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug,
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			server := httptest.NewServer(name.New().Settings(func(o *name.Options) {
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

			value := name.Value(ctx)

			if value != "" {
				t.Errorf("Unexpected Non-Default Context Value Received: %v", value)
			}

			t.Logf("Successful Default Value Received = %v", value)
		})

		t.Run("User-Specified-Value", func(t *testing.T) {
			t.Parallel()

			const v = "Test-Server"

			ctx := context.WithValue(context.Background(), "x-testing-key", v)

			value := name.Value(ctx)

			if value != v {
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

			name.Value(ctx)

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

			const v = "Test-Server"

			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug,
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			ctx := context.WithValue(context.Background(), "x-testing-key", v)

			name.Value(ctx)

			if buffer.String() != "" {
				t.Errorf("Unexpected Log Message: %s", buffer.String())
			}
		})

		t.Run("Context-Key-Value-Testing-Trace-Log-Message", func(t *testing.T) {
			t.Parallel()

			const v = "Test-Server"

			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug - 4, // the trace log level
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			ctx := context.WithValue(context.Background(), "x-testing-key", v)

			name.Value(ctx)

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
