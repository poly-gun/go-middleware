package timeout_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/poly-gun/go-middleware/middleware/timeout"
)

func Test(t *testing.T) {
	tests := []struct {
		name       string
		middleware func(next http.Handler) http.Handler
		status     int
		handler    http.HandlerFunc
	}{
		{
			name:       "Successful-Configuration-Response",
			middleware: timeout.New().Settings(func(options *timeout.Options) { options.Timeout = time.Second * 5 }).Handler,
			status:     200,
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()

				select {
				case <-ctx.Done():
					return

				case <-time.After(1 * time.Second):
					datum := struct {
						Key   string `json:"key"`
						Value string `json:"value"`
					}{
						Key:   "Key",
						Value: "Value",
					}

					buffer, _ := json.MarshalIndent(datum, "", "    ")

					defer w.Write(buffer)

					w.Header().Set("Content-Type", "application/json")

					w.WriteHeader(http.StatusOK)

					return
				}
			}),
		},
		{
			name:       "Unsuccessful-Configuration-Response",
			middleware: timeout.New().Settings(func(options *timeout.Options) { options.Timeout = time.Second * 5 }).Handler,
			status:     504,
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()

				select {
				case <-ctx.Done():
					return

				case <-time.After(30 * time.Second):
					datum := struct {
						Key   string `json:"key"`
						Value string `json:"value"`
					}{
						Key:   "Key",
						Value: "Value",
					}

					buffer, _ := json.MarshalIndent(datum, "", "    ")

					defer w.Write(buffer)

					w.Header().Set("Content-Type", "application/json")

					w.WriteHeader(http.StatusOK)

					return
				}
			}),
		},
		{
			name:       "Successful-Defaults-Response",
			middleware: timeout.New().Handler,
			status:     200,
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()

				select {
				case <-ctx.Done():
					return

				case <-time.After(25 * time.Second):
					datum := struct {
						Key   string `json:"key"`
						Value string `json:"value"`
					}{
						Key:   "Key",
						Value: "Value",
					}

					buffer, _ := json.MarshalIndent(datum, "", "    ")

					defer w.Write(buffer)

					w.Header().Set("Content-Type", "application/json")

					w.WriteHeader(http.StatusOK)

					return
				}
			}),
		},
		{
			name:       "Unsuccessful-Defaults-Response",
			middleware: timeout.New().Handler,
			status:     504,
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()

				select {
				case <-ctx.Done():
					return

				case <-time.After(35 * time.Second):
					datum := struct {
						Key   string `json:"key"`
						Value string `json:"value"`
					}{
						Key:   "Key",
						Value: "Value",
					}

					buffer, _ := json.MarshalIndent(datum, "", "    ")

					defer w.Write(buffer)

					w.Header().Set("Content-Type", "application/json")

					w.WriteHeader(http.StatusOK)

					return
				}
			}),
		},
	}

	for _, matrix := range tests {
		t.Run(matrix.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(matrix.middleware(matrix.handler))

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

			status := response.StatusCode
			if status != matrix.status {
				t.Errorf("Status = %d\n    - Expectation = %d", status, matrix.status)
			}

			t.Logf("Successful\nStatus = %d\n    - Expectation = %d", status, matrix.status)
		})
	}

	t.Run("Context", func(t *testing.T) {
		t.Run("Default", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			value := timeout.Value(ctx)

			// By default, the value should always be > 0.
			if value <= 0 {
				t.Errorf("Timeout Context Value Default Should Be Greater Than 0. Received %d", value)
			}

			t.Logf("Successful Default Value Received = %d", value)
		})

		t.Run("Reset-Invalid-Duration", func(t *testing.T) {
			t.Parallel()

			ctx := context.WithValue(context.Background(), "x-testing-key", time.Duration(-(time.Second * 30)))

			value := timeout.Value(ctx)

			// By default, the value should always be > 0.
			if value <= 0 {
				t.Errorf("Timeout Context Value Default Should Be Greater Than 0. Received %d", value)
			}

			t.Logf("Successful Reset Value Received = %d", value)
		})

		t.Run("User-Specified-Value", func(t *testing.T) {
			t.Parallel()

			ctx := context.WithValue(context.Background(), "x-testing-key", time.Second*5)

			value := timeout.Value(ctx)

			// By default, the value should always be > 0.
			if value != time.Second*5 {
				t.Errorf("Invalid Context Value. Received %d, Expected %d", value, time.Second*5)
			}

			t.Logf("Successful User-Provided Value Received = %d", value)
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

			timeout.Value(ctx)

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

			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug,
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			ctx := context.WithValue(context.Background(), "x-testing-key", time.Second*5)

			timeout.Value(ctx)

			if buffer.String() != "" {
				t.Errorf("Unexpected Log Message: %s", buffer.String())
			}
		})

		t.Run("Context-Key-Value-Testing-Trace-Log-Message", func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug - 4, // the trace log level
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			ctx := context.WithValue(context.Background(), "x-testing-key", time.Second*5)

			timeout.Value(ctx)

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
