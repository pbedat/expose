package expose

import (
	"fmt"
	"slices"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ysmood/got"
)

func TestWalkSchema(t *testing.T) {

	g := got.T(t)

	schemaJson := `{
	"$id": "root",
	"allOf": [
		{ "$id": "allOf1" },
		{ 
			"$id": "allOfNested",
			"anyOf": [
				{ "$id": "anyOf1" }
			]
		}
	],
	"oneOf": [
		{ "$id": "oneOf1"	},
		{ "$id": "oneOf2"	}
	],
	"properties": {
		"prop1": {
			"$id": "prop1",
			"type": "array",
			"items": { "$id": "item1" }
		},
		"prop2": {
			"$id": "prop2",
			"properties": {
				"prop3": { "$id": "nestedProp" }
			}
		}
	}
}`

	t.Run("traversal", func(t *testing.T) {
		var schema openapi3.Schema
		g.Must().Nil(schema.UnmarshalJSON([]byte(schemaJson)))

		var actual []string
		expected := []string{
			"allOf1", "anyOf1", "allOfNested", "oneOf1", "oneOf2",
			"nestedProp", "prop2", "item1", "prop1", "root",
		}

		g.Must().Nil(walkSchema(openapi3.NewSchemaRef("", &schema), func(s *openapi3.SchemaRef) (*openapi3.SchemaRef, error) {
			actual = append(actual, s.Value.Extensions["$id"].(string))
			return nil, nil
		}))

		slices.Sort(actual)
		slices.Sort(expected)

		g.Eq(actual, expected)
	})

	t.Run("replace", func(t *testing.T) {
		var schema openapi3.Schema
		g.Must().Nil(schema.UnmarshalJSON([]byte(schemaJson)))

		g.Must().Nil(walkSchema(openapi3.NewSchemaRef("", &schema), func(s *openapi3.SchemaRef) (*openapi3.SchemaRef, error) {
			id := s.Value.Extensions["$id"].(string)

			if id == "nestedProp" {
				return openapi3.NewSchemaRef("#/components/schemas/nestedProp", nil), nil
			}

			return nil, nil
		}))

		ref := schema.Properties["prop2"].Value.Properties["prop3"].Ref
		g.Must().Eq(ref, "#/components/schemas/nestedProp")

	})

	t.Run("nil", func(t *testing.T) {
		var schema openapi3.Schema
		g.Must().Nil(schema.UnmarshalJSON([]byte(schemaJson)))

		g.Must().Nil(walkSchema(openapi3.NewSchemaRef("", nil), func(s *openapi3.SchemaRef) (*openapi3.SchemaRef, error) {
			return nil, nil
		}))
	})

	t.Run("error", func(t *testing.T) {
		var schema openapi3.Schema
		g.Must().Nil(schema.UnmarshalJSON([]byte(schemaJson)))

		expected := []string{
			"allOf1", "anyOf1", "allOfNested", "oneOf1", "oneOf2",
			"nestedProp", "prop2", "item1", "prop1", "root",
		}

		for _, failId := range expected {
			t.Run(failId, func(t *testing.T) {
				failId := failId
				g := got.T(t)
				g.Must().NotNil(walkSchema(openapi3.NewSchemaRef("", &schema), func(s *openapi3.SchemaRef) (*openapi3.SchemaRef, error) {
					id := s.Value.Extensions["$id"].(string)

					if id == failId {
						return nil, fmt.Errorf("test")
					}

					return nil, nil
				}))
			})
		}
	})
}
