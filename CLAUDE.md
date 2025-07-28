# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

### Building and Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test
go test -run TestName ./...

# Build the module
go build ./...

# Check for compilation errors
go vet ./...

# Format code
go fmt ./...
```

### Working with Examples
```bash
# Run the starter example
cd examples/01_starter && go run main.go

# Generate Go client code
cd examples/02_go_codegen && ./generate.sh

# Generate TypeScript client code  
cd examples/03_ts_codegen && ./generate.sh
```

## Architecture Overview

exposeRPC is a Go library for creating RPC interfaces with minimal boilerplate. The architecture follows these key principles:

### Core Components

1. **Function Registration** (`expose.go`): The entry point for exposing functions as RPC endpoints
   - `Func[TReq, TRes]`: Generic function for request/response patterns
   - `FuncVoid`, `FuncNullary`, `FuncNullaryVoid`: Specialized variants for different function signatures
   - Uses generics to maintain type safety while providing a uniform interface

2. **HTTP Handler** (`handler.go`): Manages HTTP routing and request/response processing
   - Automatically generates OpenAPI 3.0 specifications
   - Supports multiple content encodings (JSON by default)
   - Provides error handling with customizable error handlers
   - Serves Swagger UI for API exploration

3. **Schema Reflection** (`reflect.go`, `jsonschema.go`): Automatically generates OpenAPI schemas from Go types
   - Uses reflection to analyze function signatures
   - Converts Go types to JSON Schema representations
   - Supports custom type mapping and naming strategies

4. **Encoding System** (`encoding.go`): Pluggable encoding/decoding for different content types
   - JSON encoding provided by default
   - Extensible to support other formats

5. **Fx Integration** (`fx/` directory): Optional integration with Uber's Fx dependency injection framework
   - Provides patterns for registering exposed functions in Fx applications
   - Enables modular service composition

### Request Flow

1. HTTP POST request arrives at registered path
2. Handler validates content-type and method
3. Request body is decoded using appropriate encoder
4. Optional JSON schema validation (if enabled)
5. Function is invoked with decoded request
6. Response is encoded and returned
7. Errors are handled according to error handler configuration

### Key Design Patterns

- **Generic Functions**: Heavy use of Go generics to maintain type safety
- **Functional Options**: Configuration through option functions (e.g., `FuncOpt`, `HandlerOption`)
- **Interface-based Extension**: `Function`, `Decoder`, `Encoder` interfaces allow custom implementations
- **Error Wrapping**: Custom error types with metadata support (`errors.go`)

### OpenAPI Generation

The library automatically generates OpenAPI 3.0 specifications by:
- Reflecting on registered function signatures
- Converting Go types to JSON Schema
- Building operation definitions for each exposed function
- Organizing functions by module (path-based grouping)

This allows automatic client generation in any language that supports OpenAPI.