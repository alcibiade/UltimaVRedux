package game_state

import (
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// IsBlackthornPasswordAccepted matches original Talk dispatch's case-insensitive comparison
// of at most the first four entered characters.
// See ref:CRIME-0001.
func IsBlackthornPasswordAccepted(response string) bool {
	if len(response) > len(references.BlackthornPassword) {
		response = response[:len(references.BlackthornPassword)]
	}

	return strings.EqualFold(response, references.BlackthornPassword)
}

// GuardTributeKind identifies the location-specific procedural tribute policy.
type GuardTributeKind int

const (
	// GuardTributeGeneric charges 10 gold per active living party member.
	GuardTributeGeneric GuardTributeKind = iota
	// GuardTributeMinocCharity leaves the party with half its current gold.
	GuardTributeMinocCharity
)

// GuardTributePaymentResult reports whether an accepted demand could be paid.
type GuardTributePaymentResult int

const (
	// GuardTributePaid means the accepted tribute was collected.
	GuardTributePaid GuardTributePaymentResult = iota
	// GuardTributeInsufficientGold means a generic tribute could not be paid.
	GuardTributeInsufficientGold
)

// GuardTributeReview freezes the state used to present and commit one demand.
type GuardTributeReview struct {
	Kind       GuardTributeKind
	Location   references.Location
	GoldBefore uint16
	Amount     uint16
}

// CalculateGuardExtortion returns 10 gold for each active non-dead party member.
// Original Talk dispatch relies on the six-member party limit rather than clamping.
// See ref:CRIME-0001.
func CalculateGuardExtortion(partyState *party_state.PartyState) int {
	if partyState == nil {
		return 0
	}

	amount := 0
	partyState.ForEachCharacter(func(_ int, character *party_state.PlayerCharacter) {
		if character != nil && character.IsInParty() && !character.IsDead() {
			amount += references.TributePerLivingMember
		}
	})

	return amount
}

// ReviewGuardTribute captures the exact policy and gold state shown to the player.
func (gs *GameState) ReviewGuardTribute() GuardTributeReview {
	review := GuardTributeReview{
		Kind:       GuardTributeGeneric,
		Location:   gs.MapState.PlayerLocation.Location,
		GoldBefore: gs.PartyState.Inventory.Gold.Get(),
		Amount:     uint16(CalculateGuardExtortion(&gs.PartyState)),
	}
	if review.Location == references.Minoc {
		review.Kind = GuardTributeMinocCharity
		review.Amount = review.GoldBefore - review.GoldBefore/2
	}

	return review
}

// CommitGuardTribute applies an accepted demand without owning Talk timing or UI.
func (gs *GameState) CommitGuardTribute(review GuardTributeReview) GuardTributePaymentResult {
	if gs.MapState.PlayerLocation.Location != review.Location ||
		gs.PartyState.Inventory.Gold.Get() != review.GoldBefore {
		panic("guard tribute state changed after review")
	}

	switch review.Kind {
	case GuardTributeGeneric:
		if !gs.PartyState.Inventory.Gold.DecrementBy(review.Amount) {
			return GuardTributeInsufficientGold
		}
	case GuardTributeMinocCharity:
		gs.PartyState.Inventory.Gold.Set(review.GoldBefore / 2)
	default:
		panic("unsupported guard tribute policy")
	}

	logging.Debugf(logging.CategoryGameState,
		"Guard tribute paid: kind=%d amount=%d remaining=%d",
		review.Kind, review.Amount, gs.PartyState.Inventory.Gold.Get())

	return GuardTributePaid
}
