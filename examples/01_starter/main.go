package main

import (
	"context"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pbedat/expose"
)

var i = &atomic.Int32{}

func Inc(_ context.Context, delta int) (int, error) {
	return int(i.Add(int32(delta))), nil
}

func Get(context.Context, expose.Void) (int, error) {
	return int(i.Load()), nil
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)

		next.ServeHTTP(w, r)

		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	h, err := expose.NewHandler(
		[]expose.Function{
			expose.Func("/counter/inc", Inc),
			expose.Func("/counter/get", Get),
		},
		expose.WithMiddleware(loggingMiddleware, corsMiddleware),
		expose.WithPathPrefix("/rpc"),
		expose.WithDefaultSpec(&openapi3.T{
			Info: &openapi3.Info{
				Title: "Starter Example",
			},
			Servers: openapi3.Servers{
				&openapi3.Server{
					URL: "http://localhost:8000/rpc",
				},
			},
		}),
		expose.WithSwaggerUI("/swagger-ui"),
	)
	if err != nil {
		panic(err)
	}

	http.Handle("/", h)

	log.Print("listening to :8000 - swagger-ui running at http://localhost:8000/rpc/swagger-ui")

	http.ListenAndServe(":8000", nil)
}
