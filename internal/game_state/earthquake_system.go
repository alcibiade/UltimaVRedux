package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
)

// EarthquakeDamage reports one committed party mutation in active slot order.
type EarthquakeDamage struct {
	CharacterIndex int
	CharacterName  string
	Damage         int
	HPBefore       uint16
	HPAfter        uint16
	StatusBefore   party_state.CharacterStatus
	StatusAfter    party_state.CharacterStatus
}

// EarthquakeResult is the authoritative result presented after all damage commits.
type EarthquakeResult struct {
	Damage []EarthquakeDamage
}

// EarthquakeCallbacks presents an already committed random earthquake.
type EarthquakeCallbacks interface {
	PresentEarthquake(result EarthquakeResult)
}

// EarthquakeSystem owns deterministic random-earthquake rolls and damage.
type EarthquakeSystem struct {
	rng       interface{ Intn(n int) int }
	callbacks EarthquakeCallbacks
}

func NewEarthquakeSystem(rng interface{ Intn(n int) int }, callbacks EarthquakeCallbacks) *EarthquakeSystem {
	return &EarthquakeSystem{rng: rng, callbacks: callbacks}
}

func (e *EarthquakeSystem) SetCallbacks(callbacks EarthquakeCallbacks) {
	e.callbacks = callbacks
}

// CheckForRandomEarthquake performs exactly one 1/256 check when eligible.
func (e *EarthquakeSystem) CheckForRandomEarthquake(
	isUnderworld bool,
	partyState *party_state.PartyState,
) (EarthquakeResult, bool) {
	if !isUnderworld || e.rng.Intn(256) != 0x69 {
		return EarthquakeResult{}, false
	}

	logging.Debug(logging.CategoryGameState, "Underworld earthquake triggered!")

	return e.triggerEarthquake(partyState), true
}

// TriggerEarthquakeManually commits earthquake damage without the random eligibility roll.
func (e *EarthquakeSystem) TriggerEarthquakeManually(partyState *party_state.PartyState) EarthquakeResult {
	return e.triggerEarthquake(partyState)
}

func (e *EarthquakeSystem) triggerEarthquake(partyState *party_state.PartyState) EarthquakeResult {
	result := EarthquakeResult{Damage: make([]EarthquakeDamage, 0, party_state.MaxActiveParty)}
	for index := range party_state.MaxActiveParty {
		if !partyState.HasCharacter(index) {
			continue
		}
		character := partyState.GetCharacter(index)
		if !character.IsInParty() || character.IsDead() {
			continue
		}

		damage := e.rng.Intn(8) + 1
		committed := EarthquakeDamage{
			CharacterIndex: index,
			CharacterName:  character.GetNameAsString(),
			Damage:         damage,
			HPBefore:       character.GetCurrentHp(),
			StatusBefore:   character.GetStatus(),
		}
		character.SubtractHp(uint16(damage))
		committed.HPAfter = character.GetCurrentHp()
		committed.StatusAfter = character.GetStatus()
		result.Damage = append(result.Damage, committed)
	}

	e.callbacks.PresentEarthquake(result)

	return result
}
