package expose

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"reflect"

	"github.com/flowchartsman/swaggerui"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mitchellh/mapstructure"
)

// Handler handles RPC requests. See [NewHandler]
type Handler struct {
	http.Handler
}

type handlerSettings struct {
	*reflectSettings
	errorHandler  ErrorHandler
	defaultSpec   openapi3.T
	encoding      map[string]Encoding
	middlewares   []Middleware
	swaggerPath   string
	swaggerUIPath string
	basePath      string
}

// ErrorHandler is called, when a exposed function returns an error.
// Returning `handled == true` cancels any further error handling.
type ErrorHandler func(w http.ResponseWriter, enc Encoder, err error) (handled bool)

type HandlerOption func(settings *handlerSettings)

type Middleware func(next http.Handler) http.Handler

// NewHandler creates a http handler, that provides the exposed functions as HTTP POST endpoints.
// see [Handler]
// Requests and responses are encoded with JSON by default.
// The handler also provides the openapi spec at the path '/swagger.json'
//
// When an exposed function returns an error, the handler will respond with HTTP status 500 Internal Server Error by default.
// When the error is (see [errors.Is]) an [ErrApplication], the status 422 Unprocessable Entity will be returned instead.
// Errors can be marked with custom codes [SetErrCode], which will be included in the error response.
// To customize the error handling further, a [ErrorHandler] can be provided.
func NewHandler(fns []Function, options ...HandlerOption) (*Handler, error) {

	settings := &handlerSettings{
		reflectSettings: &reflectSettings{
			mapper:    func(t reflect.Type) *openapi3.Schema { return nil },
			typeNamer: DefaultSchemaIdentifier,
		},
		defaultSpec: openapi3.T{},
		encoding: map[string]Encoding{
			"*/*":              JsonEncoding,
			"application/json": JsonEncoding,
		},
		swaggerPath: "/swagger.json",
	}
	for _, applyOption := range options {
		applyOption(settings)
	}

	validationSpec, err := ReflectSpec(settings.defaultSpec, fns, withSettings(*settings.reflectSettings), SkipExtractSubSchemas())
	if err != nil {
		return nil, err
	}

	r := http.NewServeMux()

	for _, _fn := range fns {
		fn := _fn
		r.HandleFunc(fn.Path(), func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, fmt.Sprint("use method POST instead of ", r.Method), http.StatusBadRequest)
				return
			}

			contentType := r.Header.Get("content-type")
			if contentType == "" {
				for mimeType := range settings.encoding {
					contentType = mimeType
					break
				}
			}

			reqEncoding, hasReqEncoding := settings.encoding[contentType]
			if !hasReqEncoding {
				http.Error(w, fmt.Sprintf("content-type '%s' is not supported", contentType), http.StatusBadRequest)
				return
			}

			dec := reqEncoding.GetDecoder(r.Body)

			res, err := fn.Apply(r.Context(), dec, validationSpec)

			accept := r.Header.Get("accept")
			if accept == "" {
				accept = contentType
			}
			resEncoding, hasResEncoding := settings.encoding[accept]

			if err != nil {
				if hasResEncoding {
					encoder := resEncoding.GetEncoder(w)
					if settings.errorHandler != nil {
						if handled := settings.errorHandler(w, encoder, err); handled {
							return
						}
					}
					if errors.Is(err, ErrApplication) {
						w.WriteHeader(http.StatusUnprocessableEntity)
					} else {
						w.WriteHeader(500)
					}
					m := map[string]any{}
					if err := mapstructure.Decode(err, &m); err != nil {
						panic(err)
					}
					m["message"] = err.Error()

					if code, ok := GetErrCode(err); ok {
						m["code"] = code
					}

					encoder.Encode(m)
					return
				} else {
					if handled := settings.errorHandler(w, nil, err); handled {
						return
					}
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if _, ok := res.(Void); ok {
				return
			}

			if !hasResEncoding {
				http.Error(w, fmt.Sprintf("response format '%s' not suppported", accept), http.StatusBadRequest)
				return
			}

			w.Header().Set("content-type", resEncoding.MimeType)

			if err := resEncoding.GetEncoder(w).Encode(res); err != nil {
				panic(fmt.Errorf("failed to encode: %+v", res))
			}
		})
	}

	if settings.swaggerPath != "" {
		spec, err := ReflectSpec(settings.defaultSpec, fns, withSettings(*settings.reflectSettings))
		if err != nil {
			return nil, fmt.Errorf("failed to reflect spec: %w", err)
		}
		r.HandleFunc(settings.swaggerPath, func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewEncoder(w).Encode(spec); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		})
	}

	if settings.swaggerUIPath != "" {
		r.HandleFunc(settings.swaggerUIPath, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, path.Join(settings.basePath, settings.swaggerUIPath)+"/", http.StatusSeeOther)
		})
		r.Handle(
			settings.swaggerUIPath+"/",
			http.StripPrefix(settings.swaggerUIPath,
				NewSwaggerUIHandler(settings.defaultSpec, fns)))
	}

	r.HandleFunc("/", http.NotFound)

	var h http.Handler = r
	for _, mw := range settings.middlewares {
		h = mw(h)
	}

	return &Handler{h}, nil
}

var ErrApplication = errors.New("application error")

type SwaggerUIHandler struct {
	http.Handler
}

func NewSwaggerUIHandler(defaultSpec openapi3.T, fns []Function) *SwaggerUIHandler {

	spec, err := ReflectSpec(defaultSpec, fns)

	if err != nil {
		panic(err)
	}

	return &SwaggerUIHandler{

		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			spec := spec

			specJson, err := json.Marshal(spec)
			if err != nil {
				panic(err)
			}
			swaggerui.Handler(specJson).ServeHTTP(w, r)
		}),
	}
}
