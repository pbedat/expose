package expose

import (
	"net/http"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
)

// WithSwaggerUI, adds a SwaggerUI handler at the provided `path`
func WithSwaggerUI(path string) HandlerOption {
	return func(settings *handlerSettings) {
		settings.swaggerUIPath = path
	}
}

// WithErrorHandler registers a custom [ErrorHandler]
func WithErrorHandler(h ErrorHandler) HandlerOption {
	return func(settings *handlerSettings) {
		settings.errorHandler = h
	}
}

// WithSwaggerJSONPath overrides the default path (/swagger.json), where the spec is served
func WithSwaggerJSONPath(path string) HandlerOption {
	return func(settings *handlerSettings) {
		settings.swaggerPath = path
	}
}

// WithEncodings registers additional encodings.
// Encodings are selected based on the provided "Content-Type" and "Accept" headers
func WithEncodings(encodings ...Encoding) HandlerOption {
	return func(settings *handlerSettings) {
		for _, enc := range encodings {
			settings.encoding[enc.MimeType] = enc
		}
	}
}

// WithDefaultSpec allows you to define a base spec.
// The handler fills this base spec with the operations and schemas
// reflected from the exposed functions.
func WithDefaultSpec(spec *openapi3.T) HandlerOption {
	return func(settings *handlerSettings) {
		settings.defaultSpec = *spec
	}
}

// WithPathPrefix defines the path prefix of the handler.
// When using it with WithSwaggerUI, make sure that your `Servers` section in
// the default spec [WithDefaultSpec] adds this prefix as well
func WithPathPrefix(prefixPath string) HandlerOption {
	return func(settings *handlerSettings) {
		settings.basePath = prefixPath
		settings.middlewares = append([]Middleware{
			func(next http.Handler) http.Handler {
				return http.StripPrefix(prefixPath, next)
			},
		}, settings.middlewares...)
	}
}

// WithReflection sets options for the schema reflection
func WithReflection(opts ...reflectSpecOpt) HandlerOption {
	return func(settings *handlerSettings) {
		for _, opt := range opts {
			if opt == nil {
				continue
			}
			opt(settings.reflectSettings)
		}
	}
}

// WithSchemaIdentifier sets an alternative [SchemaIdentifier]. Default: [DefaultSchemaIdentifier]
func WithSchemaIdentifier(namer SchemaIdentifier) reflectSpecOpt {
	return func(s *reflectSettings) {
		s.typeNamer = namer
	}
}

// WithMiddleware adds middleware to the handler chain
func WithMiddleware(middlewares ...Middleware) HandlerOption {
	return func(settings *handlerSettings) {
		settings.middlewares = append(settings.middlewares, middlewares...)
	}
}

// TypeNamers are used to generate a schema identifier for a go type
type SchemaIdentifier func(t reflect.Type) string
