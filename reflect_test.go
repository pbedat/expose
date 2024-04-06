package expose

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/ysmood/got"
)

func TestReflectSpec(t *testing.T) {
	type req struct{ Foo string }
	type res struct{ Bar int }

	g := got.T(t)
	spec := openapi3.T{
		Info: &openapi3.Info{
			Title: "test",
		},
	}
	actual, err := ReflectSpec(spec, []Function{
		Func("/foo/bar/baz", func(ctx context.Context, req req) (res, error) {
			return struct{ Bar int }{}, nil
		}),
	})

	g.Must().Nil(err)

	g.Snapshot("golden spec", actual)
}

func TestReflection(t *testing.T) {
	var mapper SchemaMapper = func(t reflect.Type) *openapi3.Schema {
		return nil
	}
	g := got.T(t)

	t.Run("default schema identifier resolves pointers", func(t *testing.T) {
		g.Must().Eq(DefaultSchemaIdentifier(reflect.TypeOf(dup{})), DefaultSchemaIdentifier(reflect.TypeOf(&dup{})))
	})

	t.Run("schema identifier", func(t *testing.T) {
		schemas := openapi3.Schemas{}
		s, err := reflectSchema(dedup1{}, schemas, reflectSettings{mapper: mapper, typeNamer: ShortSchemaIdentifier})
		g.Must().Nil(err)

		g.NotZero(schemas["expose.dedup1"])
		g.Eq(s.Ref, "#/components/schemas/expose.dedup1")
	})

	t.Run("list", func(t *testing.T) {
		schemas := openapi3.Schemas{}
		s, err := reflectSchema([]string{}, schemas, reflectSettings{mapper: mapper, typeNamer: DefaultSchemaIdentifier})
		g.Must().Nil(err)

		g.NotZero(schemas["stringList"])
		g.Eq(s.Ref, "#/components/schemas/stringList")
	})

	t.Run("dedup", func(t *testing.T) {
		schemas := openapi3.Schemas{}
		settings := reflectSettings{mapper: mapper, typeNamer: DefaultSchemaIdentifier}
		s1ref, err := reflectSchema(dedup1{}, schemas, settings)
		g.Must().Nil(err)

		s2ref, err := reflectSchema(&dedup2{}, schemas, settings)
		g.Must().Nil(err)

		s1 := schemas[strings.TrimPrefix(s1ref.Ref, "#/components/schemas/")]
		s2 := schemas[strings.TrimPrefix(s2ref.Ref, "#/components/schemas/")]

		g.Must().Eq(s1.Value.Properties["Dup1"].Ref, "#/components/schemas/github.com.pbedat.expose.dup")
		g.Must().Eq(s2.Value.Properties["Dup2"].Ref, "#/components/schemas/github.com.pbedat.expose.dup")
	})

}

func TestCustomSchema(t *testing.T) {
	g := got.T(t)

	var gen openapi3gen.Generator
	schemas := openapi3.Schemas{}
	gen = *openapi3gen.NewGenerator(openapi3gen.SchemaCustomizer(
		newCustomizerFlow(useCutomType(&gen, schemas)),
	))

	customType := reflect.TypeOf(custom{})
	ref, err := gen.GenerateSchemaRef(customType)
	g.Must().Nil(err)

	actual := ref.Value
	expected := openapi3.NewStringSchema()

	g.Eq(actual, expected)
}

type custom struct {
}

func (custom) JSONSchema(gen *openapi3gen.Generator, schemas openapi3.Schemas) (*openapi3.SchemaRef, error) {
	if gen == nil {
		panic("gen must not be nil")
	}
	return openapi3.NewSchemaRef("", openapi3.NewStringSchema()), nil
}

type dedup1 struct {
	Dup1 dup
}
type dup struct {
	Foo string
}
type dedup2 struct {
	Dup2 dup
}

func TestRequired(t *testing.T) {

	t.Run("with tags", func(t *testing.T) {
		g := got.T(t)
		actual := getRequiredProps(reflect.TypeOf(struct {
			Foo string `json:"foo"`
			Bar string `json:"bar,omitempty"`
		}{}))
		expected := []string{"foo"}

		g.Eq(actual, expected)
	})

	t.Run("without tags", func(t *testing.T) {
		g := got.T(t)
		actual := getRequiredProps(reflect.TypeOf(struct {
			Foo string
			Bar string
		}{}))
		expected := []string{"Foo", "Bar"}

		slices.Sort(actual)
		slices.Sort(expected)

		g.Eq(actual, expected)
	})

	t.Run("embedded type", func(t *testing.T) {
		g := got.T(t)
		type bar struct {
			Bar   string `json:"bar"`
			Nope1 string `json:"nope1,omitempty"`
		}
		type baz struct {
			Baz   string `json:"baz"`
			Nope2 string `json:"nope2,omitempty"`
		}
		actual := getRequiredProps(reflect.TypeOf(struct {
			Foo string
			bar
			*baz
		}{}))
		expected := []string{"Foo", "bar", "baz"}

		slices.Sort(actual)
		slices.Sort(expected)

		g.Eq(actual, expected)
	})

	t.Run("ignored field", func(t *testing.T) {
		g := got.T(t)
		actual := getRequiredProps(reflect.TypeOf(struct {
			Foo string `json:"-"`
			Bar string
		}{}))
		expected := []string{"Bar"}

		slices.Sort(actual)
		slices.Sort(expected)

		g.Eq(actual, expected)
	})

	t.Run("unnamed omitempty", func(t *testing.T) {
		g := got.T(t)
		actual := getRequiredProps(reflect.TypeOf(struct {
			Foo string `json:",omitempty"`
			Bar string
		}{}))
		expected := []string{"Bar"}

		slices.Sort(actual)
		slices.Sort(expected)

		g.Eq(actual, expected)
	})
}
