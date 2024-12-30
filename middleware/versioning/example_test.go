package versioning_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/poly-gun/go-middleware"
	"github.com/poly-gun/go-middleware/middleware/versioning"
)

func Example() {
	middleware := middleware.New()

	middleware.Add(versioning.New().Settings(func(o *versioning.Options) { o.Service, o.Warnings = "1.0.0", false }).Handler)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		datum := map[string]interface{}{
			"service-version": versioning.Value(ctx).Service,
		}

		defer json.NewEncoder(w).Encode(datum)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		return
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

	version := response.Header.Get("X-Service-Version")

	fmt.Println(version)

	// Output: 1.0.0
}
