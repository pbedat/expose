package exposefx

import (
	"github.com/pbedat/expose"
	"go.uber.org/fx"
)

type ExposeResult struct {
	fx.Out
	Function expose.Function `group:"expose_functions"`
}

// Provide registers the exposed functions of serivces in fx
func Provide[TService any](fns ...func(s TService) expose.Function) fx.Option {
	var providers []any
	for _, fn := range fns {
		_fn := fn
		providers = append(providers, func(s TService) ExposeResult {
			return ExposeResult{Function: _fn(s)}
		})
	}
	return fx.Provide(providers...)
}

// ProvideFunc registers the exposed functions in fx
func ProvideFunc(fns ...expose.Function) fx.Option {
	var providers []any
	for _, fn := range fns {
		_fn := fn
		providers = append(providers, func() ExposeResult {
			return ExposeResult{Function: _fn}
		})
	}
	return fx.Provide(providers...)
}
