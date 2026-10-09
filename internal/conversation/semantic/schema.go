package semantic

import _ "embed"

//go:embed conversation-v1.schema.json
var schemaJSON []byte

// JSONSchema returns an independent copy of the embedded v1 JSON Schema.
func JSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}
