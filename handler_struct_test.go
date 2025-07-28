package expose_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pbedat/expose"
)

var i = &atomic.Int32{}

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	Inc IncHandler
}

type IncHandler struct {
}

func (h *IncHandler) Handle(ctx context.Context, foo string) error {
	i.Add(1)
	return nil
}

type Queries struct {
	Count CountQueryHandler
}

type CountQueryHandler struct {
}

func (h *CountQueryHandler) Handle(ctx context.Context) (int, error) {
	return int(i.Load()), nil
}

func TestStruct(t *testing.T) {

	app := &Application{
		Commands: Commands{
			Inc: IncHandler{},
		},
		Queries: Queries{
			Count: CountQueryHandler{},
		},
	}

	h, err := expose.NewHandler(
		expose.Struct("/app", app),
		expose.WithPathPrefix("/rpc"),
		expose.WithDefaultSpec(&openapi3.T{
			Info: &openapi3.Info{
				Title: "Starter Example",
			},
			Servers: openapi3.Servers{
				&openapi3.Server{
					URL: "http://localhost:8000/rpc",
				},
			},
		}),
		expose.WithSwaggerUI("/swagger-ui"),
	)
	if err != nil {
		panic(err)
	}

	incReq := httptest.NewRequest("POST", "/rpc/app/commands/inc", strings.NewReader(`"test"`))
	incReq.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, incReq)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	queryReq := httptest.NewRequest("POST", "/rpc/app/queries/count", nil)
	queryReq.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, queryReq)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if strings.TrimSpace(rec.Body.String()) != "1" {
		t.Fatalf("expected body '1', got '%s'", rec.Body.String())
	}
}

type multimethod struct {
	i int
}

func (m *multimethod) Inc(ctx context.Context) error {
	m.i++
	return nil
}

func (m *multimethod) Count(ctx context.Context) (int, error) {
	return m.i, nil
}

func TestStructMultipleMethods(t *testing.T) {

	app := &multimethod{}

	h, err := expose.NewHandler(
		expose.Struct("/app", app),
		expose.WithPathPrefix("/rpc"),
		expose.WithDefaultSpec(&openapi3.T{
			Info: &openapi3.Info{
				Title: "Starter Example",
			},
			Servers: openapi3.Servers{
				&openapi3.Server{
					URL: "http://localhost:8000/rpc",
				},
			},
		}),
		expose.WithSwaggerUI("/swagger-ui"),
	)
	if err != nil {
		panic(err)
	}

	incReq := httptest.NewRequest("POST", "/rpc/app/inc", strings.NewReader(`"test"`))
	incReq.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, incReq)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	queryReq := httptest.NewRequest("POST", "/rpc/app/count", nil)
	queryReq.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, queryReq)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if strings.TrimSpace(rec.Body.String()) != "1" {
		t.Fatalf("expected body '1', got '%s'", rec.Body.String())
	}
}
