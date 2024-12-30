package cors_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/poly-gun/go-middleware"
	"github.com/poly-gun/go-middleware/middleware/cors"
)

func Example() {
	middleware := middleware.New()

	middleware.Add(cors.New().Handler)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		datum := map[string]interface{}{
			"cors-enabled": cors.Value(ctx),
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

	body, e := io.ReadAll(response.Body)
	if e != nil {
		e = fmt.Errorf("unexpected error while reading response body: %w", e)

		panic(e)
	}

	fmt.Println(strings.TrimSpace(string(body)))
	fmt.Printf("Access-Control-Allow-Headers: %s", response.Header.Get("Access-Control-Allow-Headers"))

	// Output:
	// {"cors-enabled":true}
	// Access-Control-Allow-Headers: *
}
