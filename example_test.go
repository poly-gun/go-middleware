package middleware_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/poly-gun/go-middleware"
)

func Example() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		datum := map[string]interface{}{
			"key": "value",
		}

		defer json.NewEncoder(w).Encode(datum)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
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
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	server := httptest.NewServer(middleware.Handler(mux))

	defer server.Close()

	client := server.Client()
	request, e := http.NewRequest(http.MethodGet, server.URL, nil)
	if e != nil {
		e = fmt.Errorf("unexpected error while generating request: %w", e)

		panic(e)
	}

	response, e := client.Do(request)
	if e != nil {
		e = fmt.Errorf("unexpected error while generating response: %w", e)

		panic(e)
	}

	defer response.Body.Close()

	header := response.Header.Get("X-Context-Key-1")

	fmt.Printf("X-Context-Key-1 Middleware Header: %s", header)

	// Output:
	// X-Context-Key-1 Middleware Header: Context-Key-Value-1
}
