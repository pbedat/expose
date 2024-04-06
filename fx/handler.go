package exposefx

import (
	"github.com/pbedat/expose"
	"github.com/samber/lo"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In
	ExposedFunctions []expose.Function `group:"expose_functions"`
	ExposedRouters   []Router          `group:"expose_routers"`
}

func (p HandlerParams) Functions() []expose.Function {
	return append(p.ExposedFunctions,
		lo.FlatMap(p.ExposedRouters, func(c Router, _ int) []expose.Function {
			return c.Expose()
		})...,
	)
}

// ProvideHandler provides the expose handler
func ProvideHandler(opts ...expose.HandlerOption) fx.Option {
	return fx.Provide(func(p HandlerParams) (*expose.Handler, error) {
		return expose.NewHandler(p.Functions(), opts...)
	})
}
