package audio

import (
	"fmt"
	"strings"
)

func validateSearchTerms(terms []string) error {
	seen := make(map[string]bool, len(terms))
	for _, term := range terms {
		key := strings.ToLower(term)
		if term == "" || term != strings.TrimSpace(term) || seen[key] || strings.ContainsAny(term, "\r\n\t,") {
			return fmt.Errorf("%w: keywords must be nonempty, trimmed, unique phrases without commas or newlines",
				ErrAudioLibrary)
		}
		seen[key] = true
	}

	return nil
}
