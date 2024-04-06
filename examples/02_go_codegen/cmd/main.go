package main

import (
	"net/http"

	gocodegen "github.com/pbedat/expose/examples/02_go_codegen"
)

func main() {
	http.ListenAndServe(":8000", gocodegen.CreateHandler())
}
