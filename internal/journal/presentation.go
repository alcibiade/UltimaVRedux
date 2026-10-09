// Package journal holds bounded, renderer-independent journal presentation data.
// It contains no gameplay knowledge or progression authority.
package journal

import (
	"errors"
	"fmt"
	"math"
	"slices"
)

// Curated tab identities are stable wire values; History has no newness metadata.
const (
	ChronicleTab = iota
	VirtuesTab
	ShadowlordsTab
)

const (
	// MaxChronicleEntries bounds the closed catalog of first-discovery events.
	MaxChronicleEntries = 51
	// MaxChanges includes Chronicle entries, eight virtues, and three Shadowlords.
	MaxChanges = MaxChronicleEntries + 8 + 3
)

// ErrInvalidPresentation identifies rejected checkpoint presentation data.
var ErrInvalidPresentation = errors.New("invalid journal presentation")

// Presentation is the scene-owned, renderer-independent checkpoint projection.
type Presentation struct {
	History string `json:"history"`
	Visits  Visits `json:"visits"`
}

// Visits stores finite change stamps and independent curated-tab baselines.
type Visits struct {
	Revision uint64    `json:"revision"`
	Changes  []Change  `json:"changes"`
	Seen     [3]uint64 `json:"seen"`
}

// Change identifies an entry, not its gameplay values or rendered text.
type Change struct {
	Tab      int    `json:"tab"`
	Entry    int    `json:"entry"`
	Revision uint64 `json:"revision"`
}

// EmptyPresentation is the canonical, quiet default for imports and new games.
func EmptyPresentation() Presentation {
	return Presentation{History: "", Visits: Visits{Revision: 0, Changes: []Change{}, Seen: [3]uint64{}}}
}

func validKey(tab, entry int) bool {
	switch tab {
	case ChronicleTab:
		return entry >= 1 && entry <= MaxChronicleEntries
	case VirtuesTab:
		return entry >= 0 && entry < 8
	case ShadowlordsTab:
		return entry >= 0 && entry < 3
	default:
		return false
	}
}

// Record stamps an actual presentation change, replacing that key's old stamp.
func (v *Visits) Record(tab, entry int) {
	if !validKey(tab, entry) || v.Revision == math.MaxUint64 {
		panic("invalid journal change key or exhausted revision")
	}
	v.Revision++
	change := Change{Tab: tab, Entry: entry, Revision: v.Revision}
	index := 0
	for index < len(v.Changes) {
		previous := v.Changes[index]
		if previous.Tab == tab && previous.Entry == entry {
			v.Changes[index] = change

			return
		}
		if previous.Tab > tab || (previous.Tab == tab && previous.Entry > entry) {
			break
		}
		index++
	}
	v.Changes = slices.Insert(v.Changes, index, change)
}

// IsNew reports whether the key changed after its tab's last acknowledged visit.
func (v Visits) IsNew(tab, entry int) bool {
	if !validKey(tab, entry) {
		return false
	}
	for _, change := range v.Changes {
		if change.Tab == tab && change.Entry == entry {
			return change.Revision > v.Seen[tab]
		}
	}

	return false
}

// Acknowledge never consumes changes made after the detached opening snapshot.
func (v *Visits) Acknowledge(openingRevision uint64, visited [3]bool) {
	for tab, visit := range visited {
		if visit {
			v.Seen[tab] = max(v.Seen[tab], min(openingRevision, v.Revision))
		}
	}
}

// Clone detaches the change slice and canonicalizes a zero-value empty slice.
func (v Visits) Clone() Visits {
	v.Changes = append([]Change{}, v.Changes...)

	return v
}

// ValidatePresentation enforces the bounded canonical checkpoint contract.
func ValidatePresentation(p Presentation) error {
	if err := ValidateHistory(p.History); err != nil {
		return fmt.Errorf("journal presentation history: %w", err)
	}
	visits := p.Visits
	if visits.Changes == nil || len(visits.Changes) > MaxChanges {
		return fmt.Errorf("%w: changes must be a non-null array of at most %d keys", ErrInvalidPresentation, MaxChanges)
	}
	for _, seen := range visits.Seen {
		if seen > visits.Revision {
			return fmt.Errorf("%w: seen revision exceeds current revision", ErrInvalidPresentation)
		}
	}
	for index, change := range visits.Changes {
		if !validKey(change.Tab, change.Entry) || change.Revision == 0 || change.Revision > visits.Revision {
			return fmt.Errorf("%w: invalid change at index %d", ErrInvalidPresentation, index)
		}
		if index > 0 {
			previous := visits.Changes[index-1]
			if previous.Tab > change.Tab || (previous.Tab == change.Tab && previous.Entry >= change.Entry) {
				return fmt.Errorf("%w: changes must have sorted unique keys", ErrInvalidPresentation)
			}
		}
	}

	return nil
}
