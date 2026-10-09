// Package player_save persists the fixed set of player-facing native save slots.
package player_save

import (
	"errors"
	"fmt"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

const slotCount = 8

const (
	// Slot01 is the first player save slot.
	Slot01 SlotID = "slot-01"
	// Slot02 is the second player save slot.
	Slot02 SlotID = "slot-02"
	// Slot03 is the third player save slot.
	Slot03 SlotID = "slot-03"
	// Slot04 is the fourth player save slot.
	Slot04 SlotID = "slot-04"
	// Slot05 is the fifth player save slot.
	Slot05 SlotID = "slot-05"
	// Slot06 is the sixth player save slot.
	Slot06 SlotID = "slot-06"
	// Slot07 is the seventh player save slot.
	Slot07 SlotID = "slot-07"
	// Slot08 is the eighth player save slot.
	Slot08 SlotID = "slot-08"
)

var (
	// ErrInvalidSlotID indicates a value outside the fixed player slot set.
	ErrInvalidSlotID = errors.New("invalid player save slot ID")
	// ErrSlotNotFound indicates that a requested player slot is empty.
	ErrSlotNotFound = errors.New("player save slot not found")
	// ErrStaleRevision indicates that a slot changed after it was inspected.
	ErrStaleRevision = errors.New("player save slot revision is stale")
	// ErrIncompatible indicates that a decoded record lacks player identity or runtime compatibility.
	ErrIncompatible = errors.New("player save slot is incompatible")
	// ErrInvalidRoot indicates that the repository root is empty.
	ErrInvalidRoot = errors.New("player save repository requires a root")
)

// SlotID identifies one of the eight stable player save slots.
type SlotID string

// ParseSlotID validates a stable player save slot ID.
func ParseSlotID(value string) (SlotID, error) {
	slot := SlotID(value)
	if !slot.Valid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidSlotID, value)
	}

	return slot, nil
}

// AllSlotIDs returns the eight stable player save slots in slot-number order.
func AllSlotIDs() []SlotID {
	slots := fixedSlotIDs()

	return slots[:]
}

// Valid reports whether the ID belongs to the fixed player save slot set.
func (slot SlotID) Valid() bool {
	switch slot {
	case Slot01, Slot02, Slot03, Slot04, Slot05, Slot06, Slot07, Slot08:
		return true
	default:
		return false
	}
}

// SlotStatus classifies one fixed slot for player-facing browsing.
type SlotStatus string

const (
	// SlotEmpty means that no file exists for the slot.
	SlotEmpty SlotStatus = "empty"
	// SlotReady means that the slot is valid and compatible.
	SlotReady SlotStatus = "ready"
	// SlotCorrupt means that strict snapshot decoding failed.
	SlotCorrupt SlotStatus = "corrupt"
	// SlotIncompatible means that player identity or runtime compatibility validation failed.
	SlotIncompatible SlotStatus = "incompatible"
	// SlotUnavailable means that the slot could not be safely inspected.
	SlotUnavailable SlotStatus = "unavailable"
)

// MetadataSummary contains bounded player-facing data derived from a snapshot.
type MetadataSummary struct {
	CreatedAt        time.Time
	PlaythroughID    string
	UserDescription  string
	SnapshotVersion  int
	AvatarName       string
	Location         int
	Floor            int
	Calendar         save_snapshot.CalendarSnapshot
	ActivePartyCount int
}

// SlotSummary describes one entry in the fixed eight-slot catalog.
type SlotSummary struct {
	Slot     SlotID
	Status   SlotStatus
	Metadata MetadataSummary
	Revision string
	Problem  string
}

// Record contains an inspected slot and its stable SHA-256 revision. Supported
// snapshots use canonical bytes; unsupported or corrupt records use exact file
// bytes. Corrupt records have no decoded envelope, but retain private rollback bytes.
type Record struct {
	Slot        SlotID
	Envelope    save_snapshot.Envelope
	Revision    string
	rawData     []byte
	rawRevision string
}

// CompatibilityValidator validates a structurally valid decoded snapshot against
// runtime data. A nil validator treats every decoded snapshot as compatible.
type CompatibilityValidator func(save_snapshot.SaveSnapshot) error

// NotFoundError identifies the empty slot involved in a failed operation.
type NotFoundError struct {
	Slot SlotID
}

// Error implements error.
func (notFound *NotFoundError) Error() string {
	if notFound.Slot.Valid() {
		return fmt.Sprintf("%v: %s", ErrSlotNotFound, notFound.Slot)
	}

	return ErrSlotNotFound.Error()
}

// Unwrap supports errors.Is(err, ErrSlotNotFound).
func (notFound *NotFoundError) Unwrap() error {
	return ErrSlotNotFound
}

// StaleRevisionError reports the expected and current revisions.
type StaleRevisionError struct {
	Slot     SlotID
	Expected string
	Actual   string
}

// Error implements error.
func (stale *StaleRevisionError) Error() string {
	return fmt.Sprintf("%v: %s", ErrStaleRevision, stale.Slot)
}

// Unwrap supports errors.Is(err, ErrStaleRevision).
func (stale *StaleRevisionError) Unwrap() error {
	return ErrStaleRevision
}

// IncompatibleError identifies a player eligibility failure for a decoded slot.
type IncompatibleError struct {
	Slot  SlotID
	Cause error
}

// Error implements error.
func (incompatible *IncompatibleError) Error() string {
	return fmt.Sprintf("%v: %s: %v", ErrIncompatible, incompatible.Slot, incompatible.Cause)
}

// Is supports errors.Is(err, ErrIncompatible).
func (incompatible *IncompatibleError) Is(target error) bool {
	return target == ErrIncompatible
}

// Unwrap exposes the validator's original error.
func (incompatible *IncompatibleError) Unwrap() error {
	return incompatible.Cause
}

func fixedSlotIDs() [slotCount]SlotID {
	return [...]SlotID{Slot01, Slot02, Slot03, Slot04, Slot05, Slot06, Slot07, Slot08}
}
