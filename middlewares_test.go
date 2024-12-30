package middleware_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/poly-gun/go-middleware"
)

func Test(t *testing.T) {
	t.Run("Middlewares", func(t *testing.T) {
		mux := http.NewServeMux()
		server := httptest.NewServer(mux)
		defer server.Close()

		handle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNoContent)
			return
		})

		middleware := middleware.New()

		middleware.Add(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()

				slog.InfoContext(ctx, "Middleware-1")

				ctx = context.WithValue(r.Context(), "Context-Key-1", "Context-Key-Value-1")

				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})

		middleware.Add(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()

				slog.InfoContext(ctx, "Middleware-2")

				if v, ok := ctx.Value("Context-Key-1").(string); ok {
					w.Header().Set("X-Context-Key-1", v)
				} else {
					t.Fatal("Invalid Previous Middleware Context-Key")
				}

				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})

		mux.Handle("/", middleware.Handler(handle))

		t.Run("Request", func(t *testing.T) {
			reader := new(bytes.Buffer)
			request, e := http.NewRequest(http.MethodGet, server.URL, reader)
			if e != nil {
				t.Fatalf("Unexpected Fatal Error While Generating Request: %v", e)
			}

			response, e := server.Client().Do(request)
			if e != nil {
				t.Fatalf("Unexpected Fatal Error While Generating Response: %v", e)
			}

			t.Logf("Status: %s %d", response.Status, response.StatusCode)
			if response.StatusCode != http.StatusNoContent {
				t.Fatalf("Unexpected Status Code: %d", response.StatusCode)
			}

			validations := map[string]bool{
				"header": false,
				"value":  false,
			}

			for header := range response.Header {
				t.Logf("Header: %s => %s", header, response.Header[header])
				if header == "X-Context-Key-1" {
					validations["header"] = true
				}

				for index := range response.Header[header] {
					value := response.Header[header][index]
					if value == "Context-Key-Value-1" {
						validations["value"] = true
					}
				}
			}

			if !(validations["header"] && validations["value"]) {
				t.Error("Context, Header, Value Validation Failure")
			}
		})
	})
}
