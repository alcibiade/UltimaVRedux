package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
)

// ErrInvalidStarvationScenario indicates that starvation cannot be safely applied in the current state.
var ErrInvalidStarvationScenario = errors.New("invalid starvation scenario")

// StarvationCharacterResult reports one active character's deterministic damage outcome.
type StarvationCharacterResult struct {
	Slot         int
	Name         string
	HPBefore     uint16
	HPAfter      uint16
	StatusBefore party_state.CharacterStatus
	StatusAfter  party_state.CharacterStatus
}

// StarvationScenarioResult reports exact state surrounding one forced starvation check.
type StarvationScenarioResult struct {
	Seed         uint64
	ForcedHourly bool
	FoodBefore   uint16
	FoodAfter    uint16
	HourBefore   byte
	HourAfter    byte
	MinuteBefore byte
	MinuteAfter  byte
	Characters   []StarvationCharacterResult
	AllPartyDead bool
}

// RunStarvationScenario applies one deterministic direct or full-hourly starvation scenario.
func (g *GameState) RunStarvationScenario(seed uint64, forceHourly bool) (StarvationScenarioResult, error) {
	result := StarvationScenarioResult{}
	if g.IsInCombat() {
		return result, fmt.Errorf("%w: unavailable during combat", ErrInvalidStarvationScenario)
	}
	if g.PartyState.IsAllPartyDead() {
		return result, fmt.Errorf("%w: all active party members are already dead", ErrInvalidStarvationScenario)
	}

	result = g.starvationScenarioSnapshot(seed, forceHourly)
	g.SetRandomSeed(seed)
	if forceHourly {
		g.PartyState.Inventory.Provisions.Food.Set(0)
		g.ForceHourlyFoodCheck()
	} else {
		g.applyStarvationDamageWithFeedback()
	}
	g.completeStarvationScenarioSnapshot(&result)

	return result, nil
}

func (g *GameState) starvationScenarioSnapshot(seed uint64, forcedHourly bool) StarvationScenarioResult {
	characters := make([]StarvationCharacterResult, 0, party_state.MaxActiveParty)
	for slot := range party_state.MaxActiveParty {
		character := g.PartyState.GetCharacter(slot)
		if character == nil || !character.IsInParty() {
			continue
		}
		characters = append(characters, StarvationCharacterResult{
			Slot: slot, Name: character.GetNameAsString(), HPBefore: character.GetCurrentHp(), HPAfter: character.GetCurrentHp(),
			StatusBefore: character.GetStatus(), StatusAfter: character.GetStatus(),
		})
	}

	return StarvationScenarioResult{
		Seed: seed, ForcedHourly: forcedHourly,
		FoodBefore: g.PartyState.Inventory.Provisions.Food.Get(), FoodAfter: g.PartyState.Inventory.Provisions.Food.Get(),
		HourBefore: g.DateTime.Hour, HourAfter: g.DateTime.Hour, MinuteBefore: g.DateTime.Minute, MinuteAfter: g.DateTime.Minute,
		Characters: characters, AllPartyDead: g.PartyState.IsAllPartyDead(),
	}
}

func (g *GameState) completeStarvationScenarioSnapshot(result *StarvationScenarioResult) {
	result.FoodAfter = g.PartyState.Inventory.Provisions.Food.Get()
	result.HourAfter = g.DateTime.Hour
	result.MinuteAfter = g.DateTime.Minute
	for index := range result.Characters {
		character := g.PartyState.GetCharacter(result.Characters[index].Slot)
		result.Characters[index].HPAfter = character.GetCurrentHp()
		result.Characters[index].StatusAfter = character.GetStatus()
	}
	result.AllPartyDead = g.PartyState.IsAllPartyDead()
}
