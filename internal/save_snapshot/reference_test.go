package save_snapshot

import (
	"errors"
	"testing"
)

func TestParseReferenceSupportsAllQualifiedSources(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"project:release-1", "personal:manual", "rollback:000042"} {
		reference, err := ParseReference(value)
		if err != nil {
			t.Fatalf("ParseReference(%q) error = %v", value, err)
		}
		if reference.String() != value {
			t.Fatalf("Reference.String() = %q, want %q", reference, value)
		}
	}
	for _, value := range []string{"project:../escape", "personal:/tmp", "rollback:", "other:name", "project:bad_id", "project:-bad"} {
		if _, err := ParseReference(value); !errors.Is(err, ErrInvalidReference) {
			t.Fatalf("ParseReference(%q) error = %v, want ErrInvalidReference", value, err)
		}
	}
}
