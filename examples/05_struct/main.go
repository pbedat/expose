package main

import (
	"context"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pbedat/expose"
)

var i = &atomic.Int32{}

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	Inc IncHandler
}

type IncHandler struct {
}

func (h *IncHandler) Handle(ctx context.Context, foo string) error {
	i.Add(1)
	return nil
}

type Queries struct {
	Count CountQueryHandler
}

type CountQueryHandler struct {
}

func (h *CountQueryHandler) Handle(ctx context.Context) (int, error) {
	return int(i.Load()), nil
}

func main() {

	app := &Application{
		Commands: Commands{
			Inc: IncHandler{},
		},
		Queries: Queries{
			Count: CountQueryHandler{},
		},
	}

	h, err := expose.NewHandler(
		expose.Struct("/app", app),
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
