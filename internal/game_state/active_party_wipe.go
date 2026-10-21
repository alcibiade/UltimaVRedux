package game_state

import (
	"errors"
	"fmt"
	"hash"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
)

var (
	// ErrActivePartyWipeStale means active-party membership or health changed after review.
	ErrActivePartyWipeStale = errors.New("active party changed after wipe review")
	// ErrActivePartyWipeInvalid means the party cannot be safely wiped in the current context.
	ErrActivePartyWipeInvalid = errors.New("invalid active party wipe")
)

// ActivePartyWipeMemberSnapshot captures the health state relevant to an active-party wipe.
type ActivePartyWipeMemberSnapshot struct {
	Slot   int
	Name   string
	HP     uint16
	Status party_state.CharacterStatus
}

// ActivePartyWipeReview is a detached snapshot used to review a destructive party wipe.
type ActivePartyWipeReview struct {
	Fingerprint string
	Members     []ActivePartyWipeMemberSnapshot
}

// ActivePartyWipeMemberChange reports one active member whose HP or status was normalized to dead.
type ActivePartyWipeMemberChange struct {
	Before ActivePartyWipeMemberSnapshot
	After  ActivePartyWipeMemberSnapshot
}

// ActivePartyWipeResult reports the exact committed active-party changes.
type ActivePartyWipeResult struct {
	BeforeFingerprint string
	AfterFingerprint  string
	Changes           []ActivePartyWipeMemberChange
}

// ReviewActivePartyWipe returns the active membership and health fingerprint for destructive review.
func (g *GameState) ReviewActivePartyWipe() ActivePartyWipeReview {
	members := g.activePartyWipeMembers()

	return ActivePartyWipeReview{
		Fingerprint: activePartyWipeFingerprint(members),
		Members:     members,
	}
}

// WipeActiveParty atomically sets every active member to zero HP and Dead after stale-state validation.
func (g *GameState) WipeActiveParty(expectedFingerprint string) (ActivePartyWipeResult, error) {
	if err := g.validateReviewedMutationContext("active party wipe"); err != nil {
		return ActivePartyWipeResult{}, fmt.Errorf("%w: %w", ErrActivePartyWipeInvalid, err)
	}
	review := g.ReviewActivePartyWipe()
	if expectedFingerprint == "" || expectedFingerprint != review.Fingerprint {
		return ActivePartyWipeResult{}, ErrActivePartyWipeStale
	}
	if len(review.Members) == 0 {
		return ActivePartyWipeResult{}, fmt.Errorf("%w: active party is empty", ErrActivePartyWipeInvalid)
	}

	changes := make([]ActivePartyWipeMemberChange, 0, len(review.Members))
	for _, before := range review.Members {
		if before.HP == 0 && before.Status == party_state.Dead {
			continue
		}
		character := g.PartyState.GetCharacter(before.Slot)
		// SetHp does not infer death, and can revive an already-zero character. Status must be assigned last.
		character.SetHp(0)
		character.SetStatus(party_state.Dead)
		changes = append(changes, ActivePartyWipeMemberChange{
			Before: before,
			After:  activePartyWipeMemberSnapshot(before.Slot, character),
		})
	}
	after := g.ReviewActivePartyWipe()

	return ActivePartyWipeResult{
		BeforeFingerprint: review.Fingerprint,
		AfterFingerprint:  after.Fingerprint,
		Changes:           changes,
	}, nil
}

func (g *GameState) activePartyWipeMembers() []ActivePartyWipeMemberSnapshot {
	members := make([]ActivePartyWipeMemberSnapshot, 0, party_state.MaxActiveParty)
	for slot := range party_state.MaxActiveParty {
		character := g.PartyState.GetCharacter(slot)
		if !character.IsInParty() {
			continue
		}
		members = append(members, activePartyWipeMemberSnapshot(slot, character))
	}

	return members
}

func activePartyWipeMemberSnapshot(slot int, character *party_state.PlayerCharacter) ActivePartyWipeMemberSnapshot {
	return ActivePartyWipeMemberSnapshot{
		Slot: slot, Name: character.GetNameAsString(), HP: character.GetCurrentHp(), Status: character.GetStatus(),
	}
}

func activePartyWipeFingerprint(members []ActivePartyWipeMemberSnapshot) string {
	return reviewedMutationFingerprint(func(digest hash.Hash) {
		for _, member := range members {
			writeFingerprintf(digest, "%d:%q:%d:%d;", member.Slot, member.Name, member.HP, member.Status)
		}
	})
}
