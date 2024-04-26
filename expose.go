package expose

import (
	"context"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Func creates an [Function] that can be registered with the [Handler]. The provided `fn`
// is then callable at the provided path.
// If you want to expose a function without an input or output parameter, you can parametrize with [Void], use
// [FuncVoid] or [FuncNullary] instead.
func Func[TReq any, TRes any](
	mountpoint string,
	fn func(ctx context.Context, req TReq) (TRes, error), opts ...FuncOpt) Function {
	n := mountpoint[strings.LastIndex(mountpoint, "/")+1:]

	return &functionDefinition[TReq, TRes]{
		name: n,
		path: mountpoint,
		fn: func(ctx context.Context, req any) (any, error) {
			return fn(ctx, req.(TReq))
		},
		settings: newSettings(opts...),
	}
}

func newSettings(opts ...FuncOpt) functionSettings {
	s := &functionSettings{}
	for _, opt := range opts {
		opt(s)
	}

	return *s
}

// Void is a placeholder for input or output parameters. When an input parameter is [Void].
// The function is treated as nullary. When the output paramtere is [Void], the function is treated as function without a return parameter.
type Void struct{}

func (v *Void) UnmarshalJSON(b []byte) error {
	return nil
}

// FuncVoid creates an [Function] for functions that do not return values. Shortcut for using [Func] with [Void] as request argument.
func FuncVoid[TReq any](mountpoint string, fn func(ctx context.Context, req TReq) error, opts ...FuncOpt) Function {
	n := mountpoint[strings.LastIndex(mountpoint, "/")+1:]

	return &functionDefinition[TReq, Void]{
		name: n,
		path: mountpoint,
		fn: func(ctx context.Context, req any) (any, error) {
			err := fn(ctx, req.(TReq))
			return Void{}, err
		},
		settings: newSettings(opts...),
	}
}

// FuncNullary creates an [Function] for functions without a request argument. See [Func].
func FuncNullary[TRes any](mountpoint string, fn func(ctx context.Context) (TRes, error), opts ...FuncOpt) Function {
	n := mountpoint[strings.LastIndex(mountpoint, "/")+1:]

	return &functionDefinition[Void, TRes]{
		name: n,
		path: mountpoint,
		fn: func(ctx context.Context, req any) (any, error) {
			res, err := fn(ctx)
			return res, err
		},
		settings: newSettings(opts...),
	}
}

// FuncNullaryVoid creates an [Function] for functions without a request argument and return no result. See [Func]
func FuncNullaryVoid(mountpoint string, fn func(ctx context.Context) error, opts ...FuncOpt) Function {
	n := mountpoint[strings.LastIndex(mountpoint, "/")+1:]

	return &functionDefinition[Void, Void]{
		name: n,
		path: mountpoint,
		fn: func(ctx context.Context, req any) (any, error) {
			err := fn(ctx)
			return Void{}, err
		},
		settings: newSettings(opts...),
	}
}

// Function defines a function, that should be registered as RPC endpoint in the [Handler].
// It carries all information, that is necessary to include it as an operation in the openapi spec of the [Handler],
// as well as the actual function wrapped in `Apply`
type Function interface {
	// Name is the name of the exposed function.
	// The name is part of the operationId in the spec.
	Name() string
	// Module is a qualifier, used in the operationId and as tag in the operation.
	Module() string
	// Path is the actual path, where the function is registered
	Path() string
	// Req returns an empty instance of the functions request argument.
	// Used for schema reflection.
	Req() any
	// Res returns an empty instance of the functions result value.
	// Used for schema reflection.
	Res() any
	// Apply calls the actual function by decoding the http request and passing it to the function
	Apply(ctx context.Context, dec Decoder, spec openapi3.T) (any, error)
}

type functionSettings struct {
	validate bool
}

// Validate enables the json schema validation for requests
func Validate(validate bool) FuncOpt {
	return func(s *functionSettings) {
		s.validate = validate
	}
}

type FuncOpt func(s *functionSettings)

// functionDefinition is an instance of [Function]
type functionDefinition[TReq any, TRes any] struct {
	name     string
	path     string
	fn       func(ctx context.Context, req any) (any, error)
	settings functionSettings
}

func (def *functionDefinition[TReq, TRes]) Name() string {
	return def.name
}

func (def *functionDefinition[TReq, TRes]) Module() string {
	i := strings.LastIndex(def.path, "/")
	return strings.TrimPrefix(strings.ReplaceAll(def.path[:i], "/", "."), ".")
}

func (def *functionDefinition[TReq, TRes]) Path() string {
	return def.path
}

func (def *functionDefinition[TReq, TRes]) Apply(ctx context.Context, dec Decoder, spec openapi3.T) (any, error) {
	var req TReq
	var res TRes

	if _, ok := def.Req().(Void); ok {
		return def.fn(ctx, req)
	}
	if err := dec.Decode(&req); err != nil {
		return res, err
	}

	if def.settings.validate {
		ref := spec.Paths.Find(def.Path()).Post.RequestBody.Value.Content.Get("application/json").Schema.Ref
		ref = strings.TrimPrefix(ref, "#/components/schemas/")
		if err := spec.Components.Schemas[ref].Value.VisitJSON(req, openapi3.EnableFormatValidation()); err != nil {
			return res, err
		}
	}

	return def.fn(ctx, req)
}

func (def *functionDefinition[TReq, TRes]) Req() any {
	var req TReq
	return req
}

func (def *functionDefinition[TReq, TRes]) Res() any {
	var res TRes
	return res
}
