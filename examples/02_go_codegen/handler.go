package gocodegen

import (
	"context"
	"sync/atomic"

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

func CreateHandler() *expose.Handler {
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
		}),
		expose.WithSwaggerUI("/swagger-ui"),
	)
	if err != nil {
		panic(err)
	}
	return h
}
