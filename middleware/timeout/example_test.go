package timeout_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/poly-gun/go-middleware"
	"github.com/poly-gun/go-middleware/middleware/timeout"
)

func Example() {
	middleware := middleware.New()

	middleware.Add(timeout.New().Settings(func(o *timeout.Options) { o.Timeout = time.Second * 30 }).Handler)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		datum := map[string]interface{}{
			"timeout": timeout.Value(ctx).String(),
		}

		process := time.Duration(time.Second)

		select {
		case <-ctx.Done():
			return

		case t := <-time.After(process):
			// The (<-time.After(process)) channel simulates a long-running action.

			datum["time"] = t.String()
			datum["duration"] = process.String()

			// Because datum["time"] is non-deterministic, which will fail the output in example testing, we delete the key.

			delete(datum, "time")

			defer json.NewEncoder(w).Encode(datum)
		}

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
	fmt.Printf("X-Timeout Header: %s", response.Header.Get("X-Timeout"))

	// Output:
	// {"duration":"1s","timeout":"30s"}
	// X-Timeout Header: 30s
}
