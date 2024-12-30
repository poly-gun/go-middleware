package envoy_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/poly-gun/go-middleware"
	"github.com/poly-gun/go-middleware/middleware/envoy"
)

func Example() {
	middleware := middleware.New()

	middleware.Add(envoy.New().Settings(func(o *envoy.Options) { o.Debug = false }).Handler)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		datum := map[string]interface{}{
			"envoy-headers": envoy.Value(ctx),
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

	request.Header.Set("X-Envoy-Internal", "true")
	request.Header.Set("X-Envoy-Request-Count", "1")
	request.Header.Set("X-Envoy-Original-Path", "/v1/test")

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

	// Output: {"envoy-headers":{"X-Envoy-Internal":["true"],"X-Envoy-Original-Path":["/v1/test"],"X-Envoy-Request-Count":["1"]}}
}
