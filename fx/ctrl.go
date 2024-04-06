package exposefx

import (
	"github.com/pbedat/expose"
	"go.uber.org/fx"
)

// ProvideRouter provides the `ctor` as [Router]
func ProvideRouter(ctor any) fx.Option {
	return fx.Provide(fx.Annotate(ctor, fx.ResultTags(`group:"expose_routers"`), fx.As(new(Router))))
}

// Router can be implemented to colocate routes with their handlers
type Router interface {
	Expose() []expose.Function
}
