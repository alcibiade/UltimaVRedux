// Package semantic_input defines framework-independent input values used by core contracts.
package semantic_input

// Key identifies the meaning of a pressed key without exposing a windowing backend.
type Key uint8

// Semantic key values used by framework-independent input contracts.
const (
	// KeyOther represents a pressed key whose specific identity is irrelevant to core logic.
	KeyOther Key = iota
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeySpace
	KeyEnter
)
