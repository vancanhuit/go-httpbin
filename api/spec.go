package apispec

import _ "embed"

// OpenAPIYAML is the source OpenAPI document served by the application.
//
//go:embed openapi.yaml
var OpenAPIYAML []byte
