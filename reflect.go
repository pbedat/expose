package expose

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

type reflectSettings struct {
	mapper                SchemaMapper
	typeNamer             SchemaIdentifier
	skipExtractSubSchemas bool
	customizers           []SchemaCustomizer
}

type reflectSpecOpt func(s *reflectSettings)

func WithSchemaMapper(mapper SchemaMapper) reflectSpecOpt {
	return func(s *reflectSettings) {
		s.mapper = mapper
	}
}

// WithSchemaCustomizers appends custom schema customizers to the reflection pipeline.
// Customizers run after setID but before the built-in mapper, custom type, and required-properties pipes.
// Multiple calls to WithSchemaCustomizers are cumulative.
func WithSchemaCustomizers(customizers ...SchemaCustomizer) reflectSpecOpt {
	return func(s *reflectSettings) {
		s.customizers = append(s.customizers, customizers...)
	}
}

func withSettings(settings reflectSettings) reflectSpecOpt {
	return func(s *reflectSettings) {
		*s = settings
	}
}

// ReflectSpec reflects all provided exposed functions `fns` and generates
// an openapi3 specification.
// The provided spec is the template for the resulting specification. Use it e.g. to define
// the spec info or additional schemas and operations
func ReflectSpec(root openapi3.T, fns []Function, opts ...reflectSpecOpt) (openapi3.T, error) {
	fail := func(err error) (openapi3.T, error) {
		return openapi3.T{}, fmt.Errorf("failed to reflect openapi spec: %w", err)
	}

	settings := reflectSettings{
		mapper: func(t reflect.Type) *openapi3.Schema {
			return nil
		},
		typeNamer: DefaultSchemaIdentifier,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&settings)
	}

	root.OpenAPI = "3.0.2"

	components := openapi3.NewComponents()
	if root.Components == nil {
		root.Components = &components
	}
	if components.Schemas == nil {
		components.Schemas = openapi3.Schemas{}
	}

	for _, fn := range fns {
		op := openapi3.NewOperation()
		op.OperationID = fmt.Sprint(fn.Module(), "#", fn.Name())

		if _, ok := fn.Req().(Void); !ok {
			body := openapi3.NewRequestBody()
			reqSchemaRef, err := reflectSchema(fn.Req(), components.Schemas, settings)
			if err != nil {
				return fail(err)
			}

			body.WithSchemaRef(
				reqSchemaRef,
				[]string{"application/json"})

			op.RequestBody = &openapi3.RequestBodyRef{}
			op.RequestBody.Value = body
		}

		response := openapi3.NewResponse()

		resSchema, err := reflectSchema(fn.Res(), components.Schemas, settings)
		if err != nil {
			return fail(err)
		}

		response.WithJSONSchemaRef(resSchema)
		op.AddResponse(200, response)

		op.Tags = append(op.Tags, fn.Module())

		root.AddOperation(fn.Path(), "POST", op)
	}

	return root, nil
}

type SchemaMapper func(t reflect.Type) *openapi3.Schema

// reflectSchema reflects the type of `val` and returns a `openapi3.SchemaRef`
// that matches this type
// Provided values must be structs or struct pointers.
//
// Warning: this is no general purpose reflection method. It is tailored to
// reflect schemas for an openapi spec.
//
// The `schemas` argument will become the components/schema section of the reflected spec
// the schema of `val` and all sub schemas will be stored in `schemas` and their occurrences will
// be replaced with $ref pointers.
//
// The provided mapper can be used to provide custom reflection for the type of `val` or any
// sub type.
//
// Returns a ref to the reflected schema of `val`. All reflected schemas will receive an $id (see [idSlug])
func reflectSchema(val any, schemas openapi3.Schemas, settings reflectSettings) (*openapi3.SchemaRef, error) {
	fail := func(err error) (*openapi3.SchemaRef, error) {
		return nil, fmt.Errorf("failed to reflect schema %T; %w", val, err)
	}

	t := reflect.TypeOf(val)

	id := settings.typeNamer(t)
	if _, ok := schemas[id]; ok {
		return openapi3.NewSchemaRef("#/components/schemas/"+id, nil), nil
	}

	var gen openapi3gen.Generator

	pipes := []SchemaCustomizer{
		setID(t, settings.typeNamer),
	}
	pipes = append(pipes, settings.customizers...)
	pipes = append(pipes,
		tryMap(settings.mapper),
		useCutomType(&gen, schemas),
		markPropertiesRequired(),
	)

	gen = *openapi3gen.NewGenerator(
		openapi3gen.UseAllExportedFields(),
		openapi3gen.SchemaCustomizer(
			newCustomizerFlow(pipes...)))
	ref, err := gen.NewSchemaRefForValue(val, schemas)
	if err != nil {
		return fail(err)
	}
	schemas[id] = ref

	if !settings.skipExtractSubSchemas && ref.Value != nil {
		if err := walkSchema(ref, extractSubSchemas(schemas)); err != nil {
			return fail(err)
		}
	}

	return openapi3.NewSchemaRef("#/components/schemas/"+id, nil), nil

}

// extractSubSchemas creates a visitor, that moves the schema in `ref` to the provided `schemas`
// the provided schemas will become the components/schemas in the openapi spec
func extractSubSchemas(schemas openapi3.Schemas) visitorFn {
	return func(ref *openapi3.SchemaRef) (*openapi3.SchemaRef, error) {
		if ref.Value == nil {
			return nil, nil
		}

		s := ref.Value

		idAny, ok := s.Extensions["$id"]
		if !ok {
			return nil, nil
		}

		id := idAny.(string)
		id = strings.TrimPrefix(id, "#")

		if _, ok := schemas[id]; ok {
			return openapi3.NewSchemaRef("#/components/schemas/"+id, nil), nil
		} else {
			ref := *ref
			schemas[id] = &ref
			return openapi3.NewSchemaRef("#/components/schemas/"+id, nil), nil
		}
	}
}

// DefaultSchemaIdentifier creates a schema identifier for the provided type `t`
// in the form of '<path>.<to>.<my>.<package>.<name>
func DefaultSchemaIdentifier(t reflect.Type) string {

	if t.Kind() == reflect.Slice {
		return DefaultSchemaIdentifier(t.Elem()) + "List"
	}

	if t.Kind() == reflect.Pointer {
		return DefaultSchemaIdentifier(t.Elem())
	}

	var sb strings.Builder

	if t.PkgPath() != "" {
		sb.WriteString(strings.ReplaceAll(t.PkgPath(), "/", "."))
		sb.WriteString(".")
	}
	sb.WriteString(t.Name())

	return sb.String()
}

// ShortSchemaIdentifier creates a schema identifier for the provided type `t`
// in the form of '<package.<name>'
func ShortSchemaIdentifier(t reflect.Type) string {
	if t.Kind() == reflect.Slice {
		return ShortSchemaIdentifier(t.Elem()) + "List"
	}

	if t.Kind() == reflect.Pointer {
		return ShortSchemaIdentifier(t.Elem())
	}

	return t.String()
}

// getRequiredProps iterates over all struct fields of `t`
// It returns all fields, that are not flagged with `omitempty`
// Fields without a `json` struct tag are returned as is.
// Fields with the `json` return their alias instead
func getRequiredProps(t reflect.Type) []string {

	if t.Kind() == reflect.Pointer {
		return getRequiredProps(t.Elem())
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	var props []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		if f.Anonymous {
			props = append(props, getRequiredProps(f.Type)...)
			continue
		}

		jsonTag := f.Tag.Get("json")
		if jsonTag == "" {
			props = append(props, f.Name)
			continue
		}

		alias, option, found := strings.Cut(jsonTag, ",")

		if option == "omitempty" {
			continue
		}

		name := jsonTag
		if found {
			name = alias
		}
		if name == "" {
			name = f.Name
		}

		if name == "-" {
			continue
		}

		props = append(props, name)
	}
	return props
}

// SchemaProvider overrides the schema reflection with the provided custom type
type SchemaProvider interface {
	JSONSchema(gen *openapi3gen.Generator, schemas openapi3.Schemas) (*openapi3.SchemaRef, error)
}

// setID sets the $id of the schema. See [idSlug].
func setID(mainType reflect.Type, namer SchemaIdentifier) SchemaCustomizer {
	mainStructType := mainType
	if mainStructType.Kind() == reflect.Pointer {
		mainStructType = mainStructType.Elem()
	}

	return func(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) (bool, error) {
		if t != mainStructType && t.Kind() == reflect.Struct {
			id := namer(t)
			if schema.Extensions == nil {
				schema.Extensions = make(map[string]interface{})
			}
			schema.Extensions["$id"] = fmt.Sprint("#", id)
		}

		return false, nil
	}
}

// tryMap uses the user defined mappings to acquire the schema of a type. When a schema is found, no further customizations will be applied.
func tryMap(mapper SchemaMapper) SchemaCustomizer {
	return func(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) (stop bool, err error) {
		if mapper == nil {
			return false, nil
		}
		if s := mapper(t); s != nil {
			*schema = *s
			return true, nil
		}

		return
	}
}

// useCustomType checks the provided type whether it implements [SchemaProvider].
// When it does, the provided schema will be used an no further customizations will be applied.
func useCutomType(gen *openapi3gen.Generator, schemas openapi3.Schemas) SchemaCustomizer {
	return func(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) (stop bool, err error) {
		if t.Implements(reflect.TypeOf((*SchemaProvider)(nil)).Elem()) {
			p := reflect.New(t).Interface().(SchemaProvider)
			customSchema, err := p.JSONSchema(gen, schemas)
			if err != nil {
				return true, fmt.Errorf("failed to generate custom schema for %s: %w", t.Name(), err)
			}
			*schema = *customSchema.Value
			return true, nil
		}
		return
	}
}

// markPropertiesRequired flags a schema property as required unless the json struct tag defines `omitempty`
func markPropertiesRequired() SchemaCustomizer {
	return func(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) (stop bool, err error) {
		schema.Required = append(schema.Required, getRequiredProps(t)...)
		return
	}
}

// SchemaCustomizer is a function that customizes an OpenAPI schema during reflection.
// It receives the field name, the Go reflect.Type, the struct tag, and the schema being built.
// Returning stop=true halts the customizer pipeline for this schema; no further customizers will run.
type SchemaCustomizer func(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) (stop bool, err error)

// newCustomizerFlow create an [openapi3gen.SchemaCustomizerFn], that iterates over all provided pipes
// until an error is returned, a pipe returns with `stop = true` or all pipes have run.
func newCustomizerFlow(pipes ...SchemaCustomizer) openapi3gen.SchemaCustomizerFn {
	return func(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
		for _, p := range pipes {
			stop, err := p(name, t, tag, schema)
			if err != nil {
				return err
			}
			if stop {
				return nil
			}
		}

		return nil
	}
}

// SkipExtractSubSchemas prevents the extraction sub schemas into compeonents/schemas while reflecting a spec
func SkipExtractSubSchemas(skip ...bool) reflectSpecOpt {
	return func(s *reflectSettings) {
		if len(skip) > 0 {
			s.skipExtractSubSchemas = skip[0]
			return
		}
		s.skipExtractSubSchemas = true
	}
}
