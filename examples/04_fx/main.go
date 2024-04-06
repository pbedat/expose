package main

import (
	"context"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pbedat/expose"
	exposefx "github.com/pbedat/expose/fx"
	"go.uber.org/fx"
)

var i = &atomic.Int32{}

func Inc(_ context.Context, delta int) (int, error) {
	return int(i.Add(int32(delta))), nil
}

func Get(context.Context, expose.Void) (int, error) {
	return int(i.Load()), nil
}

func main() {
	module := fx.Options(
		exposefx.ProvideFunc(
			expose.Func("/inc", Inc),
			expose.Func("/get", Get),
		),
		exposefx.ProvideHandler(
			expose.WithDefaultSpec(&openapi3.T{
				Servers: openapi3.Servers{
					&openapi3.Server{
						URL: "http://localhost:8000/rpc",
					},
				},
			}),
			expose.WithPathPrefix("/rpc"),
			expose.WithSwaggerUI("/swagger-ui")),
		fx.Invoke(func(h *expose.Handler) {
			http.Handle("/", h)

			log.Print("listening to :8000 - swagger-ui running at http://localhost:8000/rpc/swagger-ui")

			go http.ListenAndServe(":8000", nil)
		}),
	)

	app := fx.New(module)
	app.Run()
}
