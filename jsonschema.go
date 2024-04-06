package expose

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

// walkSchema traverses a schema depth first (the order of the visited schemas is not stable).
// It is just a utility to move all schema definitions of the openapi spec to components/schemas
// and does not resolve $ref's.
//
// When the visitor returns a SchemaRef, the currently visited ref will be replaced with it.
func walkSchema(ref *openapi3.SchemaRef, visitor visitorFn) error {

	if ref.Value == nil {
		return nil
	}

	s := ref.Value

	for _, ref := range s.AllOf {
		if ref.Value != nil {
			if err := walkSchema(ref, visitor); err != nil {
				return fmt.Errorf("allOf: %w", err)
			}
		}
	}

	for _, ref := range s.AnyOf {
		if ref.Value != nil {
			if err := walkSchema(ref, visitor); err != nil {
				return fmt.Errorf("allOf: %w", err)
			}
		}
	}

	for _, ref := range s.OneOf {
		if ref.Value != nil {
			if err := walkSchema(ref, visitor); err != nil {
				return fmt.Errorf("allOf: %w", err)
			}
		}
	}

	if s.Items != nil {
		if err := walkSchema(s.Items, visitor); err != nil {
			return fmt.Errorf("items: %w", err)
		}
	}

	for k, p := range s.Properties {
		if p.Value != nil {
			if err := walkSchema(p, visitor); err != nil {
				return fmt.Errorf("prop %s: %w", k, err)
			}
		}
	}

	replacedRef, err := visitor(ref)
	if err != nil {
		return err
	}

	if replacedRef != nil {
		*ref = *replacedRef
	}
	return nil
}

type visitorFn func(s *openapi3.SchemaRef) (*openapi3.SchemaRef, error)
