package telemetrics_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/poly-gun/go-middleware/middleware/telemetrics"
)

// id creates a random 128-bit ID (16 bytes), and returns it as a lowercase hex-encoded string.
func id(t *testing.T) string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		t.Fatalf("Failed to Generate a Request ID: %v", e)
	}

	return hex.EncodeToString(b)
}

func Test(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		datum := map[string]interface{}{
			"telemetry-context": telemetrics.Value(ctx),
		}

		defer json.NewEncoder(w).Encode(datum)

		w.Header().Set("Content-Type", "application/json")

		w.WriteHeader(http.StatusOK)

		return
	})

	t.Run("Middleware", func(t *testing.T) {
		t.Run("Default-Server-Headers", func(t *testing.T) {
			server := httptest.NewServer(telemetrics.New().Settings(func(o *telemetrics.Options) { o.Debug = true }).Handler(handler))

			defer server.Close()

			client := server.Client()
			request, e := http.NewRequest(http.MethodGet, server.URL, nil)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Request: %v", e)
			}

			request.Header.Set("X-Request-ID", id(t))                            // typical test case
			request.Header.Set("X-B3-Sampled", "")                               // empty value test case
			request.Header.Set("x-amzn-trace-id", fmt.Sprintf("root:%s", id(t))) // non-canonical test case

			response, e := client.Do(request)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Response: %v", e)
			}

			defer response.Body.Close()

			var body map[string]interface{}
			if e := json.NewDecoder(response.Body).Decode(&body); e != nil {
				t.Fatalf("Unexpected Error While Decoding Response Body: %v", e)
			}

			t.Logf("Response Body: %v", body)

			valuer, valid := body["telemetry-context"].(map[string]interface{})
			if !valid {
				t.Fatalf("Missing Expected 'telemetry-context' Field from Response")
			}

			t.Logf("Valuer: %v", valuer)

			headers, ok := valuer["headers"].(map[string]interface{})
			if !ok {
				t.Fatalf("Missing Expected 'headers' Field from Response")
			}

			t.Logf("Headers: %v", headers)

			if _, ok := headers["X-Request-Id"]; !(ok) {
				t.Errorf("Missing X-Request-ID Header")
			}

			if _, ok := headers["X-B3-Sampled"]; !ok {
				t.Errorf("Missing X-B3-Sampled Header")
			}

			if _, ok := headers["X-Amzn-Trace-Id"]; !ok {
				t.Errorf("Missing X-Amzn-Trace-ID Header")
			}
		})

		t.Run("Additional-Server-Headers", func(t *testing.T) {
			server := httptest.NewServer(telemetrics.New().Settings(func(o *telemetrics.Options) {
				o.Additions = []string{
					"x-test-header",
				}
				o.Debug = true
			}).Handler(handler))

			defer server.Close()

			client := server.Client()
			request, e := http.NewRequest(http.MethodGet, server.URL, nil)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Request: %v", e)
			}

			request.Header.Set("X-Test-Header", "Test-Value") // custom addition

			response, e := client.Do(request)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Response: %v", e)
			}

			defer response.Body.Close()

			var body map[string]interface{}
			if e := json.NewDecoder(response.Body).Decode(&body); e != nil {
				t.Fatalf("Unexpected Error While Decoding Response Body: %v", e)
			}

			t.Logf("Response Body: %v", body)

			valuer, valid := body["telemetry-context"].(map[string]interface{})
			if !valid {
				t.Fatalf("Missing Expected 'telemetry-context' Field from Response")
			}

			t.Logf("Valuer: %v", valuer)

			headers, ok := valuer["headers"].(map[string]interface{})
			if !ok {
				t.Fatalf("Missing Expected 'headers' Field from Response")
			}

			t.Logf("Headers: %v", headers)

			if _, ok := headers["X-Test-Header"]; !ok {
				t.Errorf("Missing X-Test-Header Header")
			}
		})

		t.Run("Excluded-Server-Headers", func(t *testing.T) {
			server := httptest.NewServer(telemetrics.New().Settings(func(o *telemetrics.Options) {
				o.Debug = true
				o.Exclusions = []string{
					"x-amzn-trace-id",
				}
			}).Handler(handler))

			defer server.Close()

			client := server.Client()
			request, e := http.NewRequest(http.MethodGet, server.URL, nil)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Request: %v", e)
			}

			request.Header.Set("X-Request-ID", id(t))                            // typical test case
			request.Header.Set("X-B3-Sampled", "")                               // empty value test case
			request.Header.Set("x-amzn-trace-id", fmt.Sprintf("root:%s", id(t))) // non-canonical test case

			response, e := client.Do(request)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Response: %v", e)
			}

			defer response.Body.Close()

			var body map[string]interface{}
			if e := json.NewDecoder(response.Body).Decode(&body); e != nil {
				t.Fatalf("Unexpected Error While Decoding Response Body: %v", e)
			}

			t.Logf("Response Body: %v", body)

			valuer, valid := body["telemetry-context"].(map[string]interface{})
			if !valid {
				t.Fatalf("Missing Expected 'telemetry-context' Field from Response")
			}

			t.Logf("Valuer: %v", valuer)

			headers, ok := valuer["headers"].(map[string]interface{})
			if !ok {
				t.Fatalf("Missing Expected 'headers' Field from Response")
			}

			t.Logf("Headers: %v", headers)

			if _, ok := headers["X-Request-Id"]; !(ok) {
				t.Errorf("Missing X-Request-ID Header")
			}

			if _, ok := headers["X-B3-Sampled"]; !ok {
				t.Errorf("Missing X-B3-Sampled Header")
			}

			if _, ok := headers["X-Amzn-Trace-Id"]; ok {
				t.Errorf("Unexpected X-Amzn-Trace-ID Header")
			}
		})
	})

	t.Run("Context", func(t *testing.T) {
		t.Run("Default", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			value := telemetrics.Value(ctx)

			if value != nil {
				t.Errorf("Unexpected Non-Default Context Value Received: %v", value)
			}

			t.Logf("Successful Default Value Received = %v", value)
		})

		t.Run("User-Specified-Value", func(t *testing.T) {
			t.Parallel()

			v := telemetrics.Valuer{
				Path: "/testing",
				Headers: http.Header{
					"X-Request-ID": []string{id(t)},
				},
			}

			ctx := context.WithValue(context.Background(), "x-testing-key", &v)

			value := telemetrics.Value(ctx)

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

			telemetrics.Value(ctx)

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

			v := telemetrics.Valuer{
				Path: "/testing",
				Headers: http.Header{
					"X-Request-ID": []string{id(t)},
				},
			}

			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug,
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			ctx := context.WithValue(context.Background(), "x-testing-key", &v)

			telemetrics.Value(ctx)

			if buffer.String() != "" {
				t.Errorf("Unexpected Log Message: %s", buffer.String())
			}
		})

		t.Run("Context-Key-Value-Testing-Trace-Log-Message", func(t *testing.T) {
			t.Parallel()

			v := telemetrics.Valuer{
				Path: "/testing",
				Headers: http.Header{
					"X-Request-ID": []string{id(t)},
				},
			}

			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug - 4, // the trace log level
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			ctx := context.WithValue(context.Background(), "x-testing-key", &v)

			telemetrics.Value(ctx)

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
