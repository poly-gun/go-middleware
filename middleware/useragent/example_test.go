package useragent_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/poly-gun/go-middleware/middleware/useragent"
)

func Example() {
	// Define a mux to handle + define routes.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Usage of the middleware's context value.
		value := useragent.Value(ctx)

		datum := map[string]interface{}{
			"user-agent": value,
		}

		defer json.NewEncoder(w).Encode(datum)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		return
	})

	// Wrap the mux instance with the user-agent middleware.
	server := httptest.NewServer(useragent.New().Handler(mux))

	defer server.Close()

	client := server.Client()
	request, e := http.NewRequest(http.MethodGet, server.URL, nil)
	if e != nil {
		e = fmt.Errorf("unexpected error while generating request: %w", e)

		panic(e)
	}

	// Set a user-agent as the request from a go httptest server instance includes a potentially non-deterministic value (e.g. "Go-http-client/1.1").
	request.Header.Set("User-Agent", "Go-HTTP-Testing-Client")

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

	// Output: {"user-agent":"Go-HTTP-Testing-Client"}
}
