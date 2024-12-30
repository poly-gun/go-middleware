package telemetrics_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/poly-gun/go-middleware"
	"github.com/poly-gun/go-middleware/middleware/telemetrics"
)

func Example() {
	middleware := middleware.New()

	middleware.Add(telemetrics.New().Handler)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		valuer := telemetrics.Value(ctx)
		if valuer == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		datum := map[string]interface{}{
			"headers": valuer.Headers,
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

	request.Header.Set("X-Request-ID", "1")

	response, e := client.Do(request)
	if e != nil {
		e = fmt.Errorf("unexpected error while generating response: %w", e)

		panic(e)
	}

	defer response.Body.Close()

	var body map[string]interface{}
	if e := json.NewDecoder(response.Body).Decode(&body); e != nil {
		e = fmt.Errorf("unexpected error while decoding response body: %w", e)
		panic(e)
	}

	headers, ok := body["headers"].(map[string]interface{})
	if !ok {
		e = fmt.Errorf("unexpected error while converting request-body headers")
		panic(e)
	}

	for k := range headers {
		if strings.ToLower(k) != "x-request-id" {
			delete(headers, k)
		}
	}

	output, e := json.Marshal(headers)
	if e != nil {
		e = fmt.Errorf("unexpected error while marshalling request-body headers: %w", e)
		panic(e)
	}

	fmt.Println(string(output))

	// Output: {"X-Request-Id":["1"]}
}
