package gocodegen_test

import (
	"context"
	"net/http/httptest"
	"testing"

	gocodegen "github.com/pbedat/expose/examples/02_go_codegen"
	"github.com/pbedat/expose/examples/02_go_codegen/client"
)

//go:generate ./generate.sh

func TestClient(t *testing.T) {
	h := gocodegen.CreateHandler()
	srv := httptest.NewServer(h)

	defer srv.Close()

	conf := client.NewConfiguration()
	conf.Servers = []client.ServerConfiguration{
		{URL: srv.URL + "/rpc"},
	}
	rpc := client.NewAPIClient(conf)

	_, _, err := rpc.CounterAPI.CounterInc(context.Background()).Body(1).Execute()
	if err != nil {
		t.Fatal(err)
	}

	count, _, err := rpc.CounterAPI.CounterGet(context.Background()).Execute()
	if err != nil {
		t.Fatal()
	}

	if count != 1 {
		t.Fatal("count must be 1, was ", count)
	}
}
