package rip_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/poly-gun/go-middleware/middleware/rip"
)

func Example() {
	// Define a mux to handle + define routes.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Usage of the middleware's context value.
		value := rip.Value(ctx)

		datum := map[string]interface{}{
			"rip": value,
		}

		defer json.NewEncoder(w).Encode(datum)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		return
	})

	// Wrap the mux instance with the user-agent middleware.
	server := httptest.NewServer(rip.New().Handler(mux))

	defer server.Close()

	client := server.Client()
	request, e := http.NewRequest(http.MethodGet, server.URL, nil)
	if e != nil {
		e = fmt.Errorf("unexpected error while generating request: %w", e)

		panic(e)
	}

	// Set a rip header as the request from a go httptest server instance doesn't include such header(s).
	request.Header.Set("X-Forwarded-For", "123.123.123.123")

	response, e := client.Do(request)
	if e != nil {
		e = fmt.Errorf("unexpected error while generating response: %w", e)

		panic(e)
	}

	defer response.Body.Close()

	body, e := io.ReadAll(response.Body)
	if e != nil {
		e = fmt.Errorf("unexpected error while reading response body: %w", e)

		panic(e)
	}

	fmt.Println(string(body))

	// Output: {"rip":"123.123.123.123"}
}
