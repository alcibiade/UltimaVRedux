package semantic

import (
	"fmt"
	"strings"
)

// Diagnostic describes one structural or semantic problem in a document.
type Diagnostic struct {
	Path    string
	Line    int
	Column  int
	Message string
}

func (d Diagnostic) String() string {
	if d.Line > 0 {
		return fmt.Sprintf("%s at line %d, column %d: %s", d.Path, d.Line, d.Column, d.Message)
	}

	return fmt.Sprintf("%s: %s", d.Path, d.Message)
}

// ValidationError contains all semantic problems found in one document.
type ValidationError struct {
	Diagnostics []Diagnostic
}

func (e *ValidationError) Error() string {
	messages := make([]string, 0, len(e.Diagnostics))
	for _, diagnostic := range e.Diagnostics {
		messages = append(messages, diagnostic.String())
	}

	return "semantic validation failed: " + strings.Join(messages, "; ")
}

type documentError struct {
	diagnostic Diagnostic
}

func (e *documentError) Error() string { return e.diagnostic.String() }
