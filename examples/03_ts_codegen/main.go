package main

import (
	"context"
	"embed"
	_ "embed"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pbedat/expose"
)

//go:generate ./generate.sh

//go:embed static
var static embed.FS

var i = &atomic.Int32{}

func Inc(_ context.Context, delta int) (int, error) {
	return int(i.Add(int32(delta))), nil
}

func Get(context.Context, expose.Void) (int, error) {
	return int(i.Load()), nil
}

func main() {
	h, err := expose.NewHandler(
		[]expose.Function{
			expose.Func("/counter/inc", Inc),
			expose.Func("/counter/get", Get),
		},
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
	)
	if err != nil {
		panic(err)
	}

	http.Handle("/static/", http.FileServerFS(static))

	http.Handle("/", h)

	log.Print("listening to :8000 - swagger-ui running at http://localhost:8000/rpc/swagger-ui")

	http.ListenAndServe(":8000", nil)
}
