package expose

import (
	"context"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/iancoleman/strcase"
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

// Struct traverses the provided struct recursively and registers all public methods
// that match the function signatures supported by Func, FuncVoid, FuncNullary, or FuncNullaryVoid.
// The basePath is used as the prefix for all registered functions.
func Struct(basePath string, v any, opts ...FuncOpt) []Function {
	var functions []Function
	traverseStruct(basePath, reflect.ValueOf(v), &functions, opts)
	return functions
}

func traverseStruct(path string, v reflect.Value, functions *[]Function, opts []FuncOpt) {
	// Handle pointer types
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	// Handle interface types by checking their underlying value
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		v = v.Elem()
		// Recursively call traverseStruct with the concrete value
		traverseStruct(path, v, functions, opts)
		return
	}

	// Only process structs
	if v.Kind() != reflect.Struct {
		return
	}

	t := v.Type()

	// Get all exposable methods on the current type
	methods := getExposableMethods(v)

	if len(methods) == 1 {
		// Single method: register at the struct path directly
		if fn := createFunction(path, methods[0].name, methods[0].method, opts); fn != nil {
			*functions = append(*functions, fn)
		}
	} else if len(methods) > 1 {
		// Multiple methods: register each at path/methodname
		for _, methodInfo := range methods {
			methodPath := path + "/" + strcase.ToKebab(methodInfo.name)
			if fn := createFunction(methodPath, methodInfo.name, methodInfo.method, opts); fn != nil {
				*functions = append(*functions, fn)
			}
		}
	}

	// Traverse struct fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Build path for nested field
		fieldPath := path + "/" + strcase.ToKebab(field.Name)

		// Recursively traverse nested structs
		traverseStruct(fieldPath, fieldValue, functions, opts)
	}
}

type methodInfo struct {
	name   string
	method reflect.Value
}

func getExposableMethods(v reflect.Value) []methodInfo {
	var methods []methodInfo

	// Get the type to inspect methods
	t := v.Type()

	// Check methods on the value
	for i := 0; i < v.NumMethod(); i++ {
		method := v.Method(i)
		methodType := t.Method(i)
		if methodType.IsExported() && isExposableMethod(method) {
			methods = append(methods, methodInfo{
				name:   methodType.Name,
				method: method,
			})
		}
	}

	// If v is not a pointer, also check pointer methods
	if v.Kind() != reflect.Ptr && v.CanAddr() {
		ptrValue := v.Addr()
		ptrType := ptrValue.Type()
		for i := 0; i < ptrValue.NumMethod(); i++ {
			method := ptrValue.Method(i)
			methodType := ptrType.Method(i)
			// Skip if we already have this method from the value
			found := false
			for _, m := range methods {
				if m.name == methodType.Name {
					found = true
					break
				}
			}
			if !found && methodType.IsExported() && isExposableMethod(method) {
				methods = append(methods, methodInfo{
					name:   methodType.Name,
					method: method,
				})
			}
		}
	}

	return methods
}

func isExposableMethod(method reflect.Value) bool {
	methodType := method.Type()

	// Check if method has the right number of parameters and returns
	if methodType.NumIn() < 1 || methodType.NumOut() < 1 || methodType.NumOut() > 2 {
		return false
	}

	// First parameter should be context.Context
	if !methodType.In(0).Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		return false
	}

	// Last return value should be error (if there are 2 returns)
	if methodType.NumOut() == 2 && !methodType.Out(methodType.NumOut()-1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return false
	}

	// If only one return value, it must be error
	if methodType.NumOut() == 1 && !methodType.Out(0).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return false
	}

	return true
}

func createFunction(path string, methodName string, method reflect.Value, opts []FuncOpt) Function {
	methodType := method.Type()

	// Check if method has the right number of parameters and returns
	if methodType.NumIn() < 1 || methodType.NumOut() < 1 || methodType.NumOut() > 2 {
		return nil
	}

	// First parameter should be context.Context
	if !methodType.In(0).Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		return nil
	}

	// Last return value should be error (if there are 2 returns)
	if methodType.NumOut() == 2 && !methodType.Out(methodType.NumOut()-1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return nil
	}

	// Determine the function type based on signature
	switch {
	case methodType.NumIn() == 2 && methodType.NumOut() == 2:
		// func(ctx, req) (res, error)
		reqType := methodType.In(1)
		resType := methodType.Out(0)
		return createTypedFunc(path, methodName, method, reqType, resType, false, false, opts)

	case methodType.NumIn() == 2 && methodType.NumOut() == 1:
		// func(ctx, req) error
		reqType := methodType.In(1)
		return createTypedFunc(path, methodName, method, reqType, nil, false, true, opts)

	case methodType.NumIn() == 1 && methodType.NumOut() == 2:
		// func(ctx) (res, error)
		resType := methodType.Out(0)
		return createTypedFunc(path, methodName, method, nil, resType, true, false, opts)

	case methodType.NumIn() == 1 && methodType.NumOut() == 1:
		// func(ctx) error
		return createTypedFunc(path, methodName, method, nil, nil, true, true, opts)
	}

	return nil
}

func createTypedFunc(path string, methodName string, method reflect.Value, reqType, resType reflect.Type, isNullary, isVoid bool, opts []FuncOpt) Function {
	// Create a generic struct function definition that can handle dynamic types
	return &structFunctionDefinition{
		name:      methodName,
		path:      path,
		method:    method,
		reqType:   reqType,
		resType:   resType,
		isNullary: isNullary,
		isVoid:    isVoid,
		settings:  newSettings(opts...),
	}
}

// structFunctionDefinition is a Function implementation for methods found via struct traversal
type structFunctionDefinition struct {
	name      string
	path      string
	method    reflect.Value
	reqType   reflect.Type
	resType   reflect.Type
	isNullary bool
	isVoid    bool
	settings  functionSettings
}

func (def *structFunctionDefinition) Name() string {
	return def.name
}

func (def *structFunctionDefinition) Module() string {
	i := strings.LastIndex(def.path, "/")
	return strings.TrimPrefix(strings.ReplaceAll(def.path[:i], "/", "."), ".")
}

func (def *structFunctionDefinition) Path() string {
	return def.path
}

func (def *structFunctionDefinition) Req() any {
	if def.isNullary || def.reqType == nil {
		return Void{}
	}
	return reflect.New(def.reqType).Elem().Interface()
}

func (def *structFunctionDefinition) Res() any {
	if def.isVoid || def.resType == nil {
		return Void{}
	}
	return reflect.New(def.resType).Elem().Interface()
}

func (def *structFunctionDefinition) Apply(ctx context.Context, dec Decoder, spec openapi3.T) (any, error) {
	// Handle nullary functions
	if def.isNullary {
		args := []reflect.Value{reflect.ValueOf(ctx)}
		results := def.method.Call(args)

		if def.isVoid {
			if results[0].IsNil() {
				return Void{}, nil
			}
			return Void{}, results[0].Interface().(error)
		}

		if results[1].IsNil() {
			return results[0].Interface(), nil
		}
		return results[0].Interface(), results[1].Interface().(error)
	}

	// Handle functions with request parameter
	req := reflect.New(def.reqType).Interface()
	if err := dec.Decode(req); err != nil {
		return nil, err
	}

	if def.settings.validate {
		ref := spec.Paths.Find(def.Path()).Post.RequestBody.Value.Content.Get("application/json").Schema.Ref
		ref = strings.TrimPrefix(ref, "#/components/schemas/")
		if err := spec.Components.Schemas[ref].Value.VisitJSON(req, openapi3.EnableFormatValidation()); err != nil {
			return nil, err
		}
	}

	// Dereference the pointer to get the actual value
	reqValue := reflect.ValueOf(req).Elem()
	args := []reflect.Value{reflect.ValueOf(ctx), reqValue}
	results := def.method.Call(args)

	if def.isVoid {
		if results[0].IsNil() {
			return Void{}, nil
		}
		return Void{}, results[0].Interface().(error)
	}

	if results[1].IsNil() {
		return results[0].Interface(), nil
	}
	return results[0].Interface(), results[1].Interface().(error)
}
