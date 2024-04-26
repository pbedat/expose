Godoc: https://pkg.go.dev/github.com/pbedat/expose

# exposeRPC

exposeRPC allows you to create RPC interfaces, without the usual boilerplate.
Methods can be exposed directly in Go code, without any further generation or definition steps.
The resulting http interface provides an OpenAPI specification, that can be used to create type safe clients, to call the functions you exposed.

# Example

Expose the functions `Inc` and `Get` as RPC endpoints:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/pbedat/expose"
)

var i = &atomic.Int32{}


func Inc(_ context.Context, delta int) (int, error) {
	return int(i.Add(int32(delta))), nil
}

func Get(context.Context, expose.Void) (int, error) {
	return int(i.Load()), nil
}

func main() {
	h, err := expose.NewHandler(
		[]expose.Function{
			expose.Func("/counter/inc", Inc),
			expose.Func("/counter/get", Get),
		},
	)
	if err != nil {
		panic(err)
	}

	http.Handle("/", h)

	http.ListenAndServe(":8000", nil)
}
```

Perform the RPC calls:

```sh
curl -H "content-type: application/json" --data 1 localhost:8000/rpc/counter/inc
curl -X POST localhost:8000/rpc/counter/get
> 1
```

Get the OpenAPI Spec:

```sh
curl localhost:8000/rpc/swagger.json
```

```json
{
  "components": {
    "schemas": {
      "int": {
        "type": "integer"
      }
    }
  },
  "info": {
    "title": "Starter Example",
    "version": ""
  },
  "openapi": "3.0.2",
  "paths": {
    "/counter/get": {
      "post": {
        "operationId": "counter#get",
        "responses": {
          "200": {
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/int"
                }
              }
            }
          },
          "default": {
            "description": ""
          }
        },
        "tags": ["counter"]
      }
    },
    "/counter/inc": {
      "post": {
        "operationId": "counter#inc",
        "requestBody": {
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/int"
              }
            }
          }
        },
        "responses": {
          "200": {
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/int"
                }
              }
            }
          },
          "default": {
            "description": ""
          }
        },
        "tags": ["counter"]
      }
    }
  },
  "servers": [
    {
      "url": "http://localhost:8000/rpc"
    }
  ]
}
```
