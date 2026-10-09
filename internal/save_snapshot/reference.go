package save_snapshot

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	// SourceProject identifies project-owned named captures.
	SourceProject Source = "project"
	// SourcePersonal identifies user-owned named captures.
	SourcePersonal Source = "personal"
	// SourceRollback identifies automatic rollback captures.
	SourceRollback Source = "rollback"
)

var (
	// ErrInvalidReference indicates an invalid source-qualified capture reference.
	ErrInvalidReference = errors.New("invalid save snapshot reference")
	referenceIDPattern  = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)
)

// Source identifies one configured repository root.
type Source string

// Reference uniquely identifies a capture and source.
type Reference struct {
	Source Source
	ID     string
}

// ParseReference parses project:<id>, personal:<id>, or rollback:<id>.
func ParseReference(value string) (Reference, error) {
	sourceText, id, found := strings.Cut(value, ":")
	reference := Reference{Source: Source(sourceText), ID: id}
	if !found || !reference.Source.Valid() || !referenceIDPattern.MatchString(reference.ID) {
		return Reference{}, fmt.Errorf("%w: %q", ErrInvalidReference, value)
	}

	return reference, nil
}

// Valid reports whether source is supported.
func (source Source) Valid() bool {
	return source == SourceProject || source == SourcePersonal || source == SourceRollback
}

// String returns the stable source-qualified representation.
func (reference Reference) String() string {
	return string(reference.Source) + ":" + reference.ID
}
