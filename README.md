# `middlewares` - HTTP Middleware Handler(s)

Provides an HTTP middleware for `go` HTTP servers.

## Documentation

Official `godoc` documentation (with examples) can be found at the [Package Registry](https://pkg.go.dev/github.com/poly-gun/go-middleware).

## Usage

###### Add Package Dependency

```bash
go get -u github.com/poly-gun/go-middleware
```

###### Import and Implement

`main.go`

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "net"
    "net/http"
    "time"

    "github.com/poly-gun/go-middleware"
)

func main() {
    ctx := context.Background()
    mux := http.NewServeMux()
    server := &http.Server{
        Addr:           fmt.Sprintf("0.0.0.0:%s", "8080"),
        Handler:        mux,
        ReadTimeout:    15 * time.Second,
        WriteTimeout:   60 * time.Second,
        IdleTimeout:    30 * time.Second,
        MaxHeaderBytes: http.DefaultMaxHeaderBytes,
        BaseContext: func(net.Listener) context.Context {
            return ctx
        },
    }

    defer server.Shutdown(ctx)

    handle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var response = map[string]interface{}{
            "key-1": "value-1",
            "key-2": "value-2",
            "key-3": "value-3",
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)

        return
    })

    middleware := new(middlewares.Middleware)

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

    mux.Handle("/", middleware.Handler(handle))

    if e := server.ListenAndServe(); e != nil {
        slog.ErrorContext(ctx, "Server Error", slog.Any("error", e))
    }

    return
}
```

- Please refer to the [code examples](./example_test.go) for additional usage and implementation details.
- See https://pkg.go.dev/github.com/poly-gun/go-middleware for additional documentation.

## Contributions

See the [**Contributing Guide**](./CONTRIBUTING.md) for additional details on getting started.
