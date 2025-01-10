package cors_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/poly-gun/go-middleware/middleware/cors"
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

	_ = handler

	t.Run("Middleware", func(t *testing.T) {
		t.Run("Include-CORS-Headers", func(t *testing.T) {
			server := httptest.NewServer(cors.New().Settings(func(o *cors.Options) { o.Debug = true }).Handler(handler))

			defer server.Close()

			client := server.Client()
			request, e := http.NewRequest(http.MethodGet, server.URL, nil)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Request: %v", e)
			}

			request.Header.Set("Origin", server.URL)

			response, e := client.Do(request)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Response: %v", e)
			}

			defer response.Body.Close()

			// Check status code.
			if response.StatusCode != http.StatusOK {
				t.Errorf("Expected Status 200 OK, Received: %d", response.StatusCode)
			}

			// Check the body to ensure the response passed through the middleware.
			body, e := io.ReadAll(response.Body)
			if e != nil {
				t.Fatalf("Unexpected Error While Reading Response Body: %v", e)
			}

			if len(body) == 0 {
				t.Errorf("Empty Response Body Received")
			}

			t.Run("Headers", func(t *testing.T) {
				t.Run("Access-Control-Allow-Origin", func(t *testing.T) {
					if got, want := response.Header.Get("Access-Control-Allow-Origin"), server.URL; got != want {
						t.Errorf("Expected Access-Control-Allow-Origin = %q, got %q", want, got)
					}
				})

				// t.Run("Access-Control-Allow-Methods", func(t *testing.T) {
				// 	if got, want := response.Header.Get("Access-Control-Allow-Methods"), fmt.Sprintf("%s, %s, %s, %s, %s, %s", http.MethodHead, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete); got != want {
				// 		t.Errorf("Expected Access-Control-Allow-Methods = %q, got %q", want, got)
				// 	}
				// })

				// t.Run("Access-Control-Allow-Headers", func(t *testing.T) {
				// 	if got, want := response.Header.Get("Access-Control-Allow-Headers"), "*"; got != want {
				// 		t.Errorf("Expected Access-Control-Allow-Headers = %q, got %q", want, got)
				// 	}
				// })

				t.Run("Access-Control-Expose-Headers", func(t *testing.T) {
					if got, want := response.Header.Get("Access-Control-Expose-Headers"), "*"; got != want {
						t.Errorf("Expected Access-Control-Expose-Headers = %q, got %q", want, got)
					}
				})

				t.Run("Access-Control-Allow-Credentials", func(t *testing.T) {
					if got, want := response.Header.Get("Access-Control-Allow-Credentials"), "true"; got != want {
						t.Errorf("Expected Access-Control-Allow-Credentials = %q, got %q", want, got)
					}
				})
			})
		})

		// t.Run("Preflight-Include-CORS-Headers", func(t *testing.T) {
		// 	server := httptest.NewServer(cors.New().Settings(func(o *cors.Options) { o.Debug = true }).Handler(handler))
		//
		// 	defer server.Close()
		//
		// 	client := server.Client()
		// 	request, e := http.NewRequest(http.MethodOptions, server.URL, nil)
		// 	if e != nil {
		// 		t.Fatalf("Unexpected Error While Generating Request: %v", e)
		// 	}
		//
		// 	request.Header.Set("Origin", server.URL)
		//
		// 	response, e := client.Do(request)
		// 	if e != nil {
		// 		t.Fatalf("Unexpected Error While Generating Response: %v", e)
		// 	}
		//
		// 	defer response.Body.Close()
		//
		// 	// Check status code.
		// 	// if response.StatusCode != http.StatusNoContent {
		// 	// 	t.Errorf("Expected Status 204 No-Content, Received: %d", response.StatusCode)
		// 	// }
		//
		// 	// Check the body to ensure the response passed through the middleware.
		// 	body, e := io.ReadAll(response.Body)
		// 	if e != nil {
		// 		t.Fatalf("Unexpected Error While Reading Response Body: %v", e)
		// 	}
		//
		// 	if len(body) != 0 {
		// 		t.Errorf("Non-Empty Response Body Received")
		// 	}
		//
		// 	t.Run("Headers", func(t *testing.T) {
		// 		t.Run("Access-Control-Allow-Origin", func(t *testing.T) {
		// 			if got, want := response.Header.Get("Access-Control-Allow-Origin"), server.URL; got != want {
		// 				t.Errorf("Expected Access-Control-Allow-Origin = %q, got %q", want, got)
		// 			}
		// 		})
		//
		// 		// t.Run("Access-Control-Allow-Methods", func(t *testing.T) {
		// 		// 	if got, want := response.Header.Get("Access-Control-Allow-Methods"), fmt.Sprintf("%s, %s, %s, %s, %s, %s", http.MethodHead, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete); got != want {
		// 		// 		t.Errorf("Expected Access-Control-Allow-Methods = %q, got %q", want, got)
		// 		// 	}
		// 		// })
		//
		// 		// t.Run("Access-Control-Allow-Headers", func(t *testing.T) {
		// 		// 	if got, want := response.Header.Get("Access-Control-Allow-Headers"), "*"; got != want {
		// 		// 		t.Errorf("Expected Access-Control-Allow-Headers = %q, got %q", want, got)
		// 		// 	}
		// 		// })
		//
		// 		t.Run("Access-Control-Expose-Headers", func(t *testing.T) {
		// 			if got, want := response.Header.Get("Access-Control-Expose-Headers"), "*"; got != want {
		// 				t.Errorf("Expected Access-Control-Expose-Headers = %q, got %q", want, got)
		// 			}
		// 		})
		//
		// 		t.Run("Access-Control-Allow-Credentials", func(t *testing.T) {
		// 			if got, want := response.Header.Get("Access-Control-Allow-Credentials"), "true"; got != want {
		// 				t.Errorf("Expected Access-Control-Allow-Credentials = %q, got %q", want, got)
		// 			}
		// 		})
		// 	})
		// })
	})

	t.Run("Context", func(t *testing.T) {
		t.Run("Default", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			value := cors.Value(ctx)

			if value != false {
				t.Errorf("Unexpected Non-Default Context Value Received: %v", value)
			}

			t.Logf("Successful Default Value Received = %v", value)
		})

		t.Run("User-Specified-Value", func(t *testing.T) {
			t.Parallel()

			ctx := context.WithValue(context.Background(), "x-testing-key", true)

			value := cors.Value(ctx)

			if value != true {
				t.Errorf("Unexpected Context Value Received: %v, Expected: %v", value, true)
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

			cors.Value(ctx)

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

			ctx := context.WithValue(context.Background(), "x-testing-key", true)

			cors.Value(ctx)

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

			ctx := context.WithValue(context.Background(), "x-testing-key", true)

			cors.Value(ctx)

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
