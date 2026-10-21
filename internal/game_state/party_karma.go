package game_state

import (
	"errors"
	"fmt"
)

// ErrInvalidPartyKarma indicates that an exact karma value is outside the gameplay range.
var ErrInvalidPartyKarma = errors.New("invalid party karma")

// ErrInvalidScriptedGoldPayment identifies an impossible or unaffordable script payment.
var ErrInvalidScriptedGoldPayment = errors.New("invalid scripted gold payment")

// PartyKarmaAlteration reports the exact party karma before and after an update.
type PartyKarmaAlteration struct {
	Before int
	After  int
}

// ScriptedGoldPaymentOutcome reports one committed conversation payment.
type ScriptedGoldPaymentOutcome struct {
	GoldBefore  uint16
	GoldAfter   uint16
	KarmaBefore int
	KarmaAfter  int
	KarmaAward  int
}

// CurrentPartyKarma returns the exact party-wide karma value.
func (g *GameState) CurrentPartyKarma() int {
	return g.PartyState.Karma.Value
}

// AdjustPartyKarma applies one moral event and reports its actual clamped result.
func (g *GameState) AdjustPartyKarma(delta int) PartyKarmaAlteration {
	before := g.PartyState.Karma.Value
	g.PartyState.Karma.AddDiff(delta)
	result := PartyKarmaAlteration{Before: before, After: g.PartyState.Karma.Value}
	switch {
	case result.After > result.Before:
		g.SystemCallbacks.UI.ShowTransientIcon(TransientIconKarmaIncrease)
	case result.After < result.Before:
		g.SystemCallbacks.UI.ShowTransientIcon(TransientIconKarmaDecrease)
	}

	return result
}

// CommitScriptedGoldPayment applies a conversation-script gold deduction and
// the original beggar cooldown reward as one authoritative mutation.
func (g *GameState) CommitScriptedGoldPayment(amount int, isBeggar bool) (ScriptedGoldPaymentOutcome, error) {
	result := ScriptedGoldPaymentOutcome{
		GoldBefore:  g.PartyState.Inventory.Gold.Get(),
		KarmaBefore: g.PartyState.Karma.Value,
	}
	result.GoldAfter, result.KarmaAfter = result.GoldBefore, result.KarmaBefore
	if amount < 0 || amount > int(^uint16(0)) {
		return result, fmt.Errorf("%w: amount %d is outside the uint16 range", ErrInvalidScriptedGoldPayment, amount)
	}
	if !g.PartyState.Inventory.Gold.DecrementBy(uint16(amount)) {
		return result, fmt.Errorf("%w: insufficient gold", ErrInvalidScriptedGoldPayment)
	}

	result.GoldAfter = g.PartyState.Inventory.Gold.Get()
	if !isBeggar || g.beggarKarmaCounter < beggarKarmaRewardThreshold {
		return result, nil
	}

	g.beggarKarmaCounter = 0
	requestedAward := 1
	if result.GoldAfter == 0 {
		requestedAward += 2
	}
	change := g.AdjustPartyKarma(requestedAward)
	result.KarmaAfter = change.After
	result.KarmaAward = change.After - change.Before

	return result, nil
}

// SetPartyKarma sets party karma exactly without applying a moral event or advancing time.
func (g *GameState) SetPartyKarma(value int) (PartyKarmaAlteration, error) {
	before := g.PartyState.Karma.Value
	result := PartyKarmaAlteration{Before: before, After: before}
	if value < 0 || value > MaxKarmaValue {
		return result, fmt.Errorf("%w: value must be between 0 and %d", ErrInvalidPartyKarma, MaxKarmaValue)
	}
	if value == before {
		return result, nil
	}

	g.PartyState.Karma.Value = value
	result.After = value

	return result, nil
}
