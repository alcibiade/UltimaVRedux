package game_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// DoomEntranceEncounterOrigin retains the Underworld state behind a temporary generic Shadowlord battle.
type DoomEntranceEncounterOrigin struct {
	ReturnLocation    references.PlayerLocation
	ReturnVehicle     map_units.VehicleDetailsSnapshot
	ActivePlayerIndex int
	SceptreReclaimed  bool
}

func (g *GameState) startDoomEntranceCombatEncounter() (*CombatEncounterResult, error) {
	location := g.MapState.PlayerLocation
	if location.Location != references.BritanniaUnderworld ||
		location.Floor != references.FloorNumber(references.UNDERWORLD) ||
		location.Position != g.GetDungeonEntranceCoordinates()[references.Doom] ||
		g.PartyVehicle.GetVehicleDetails().VehicleType != references.NoPartyVehicle ||
		g.ShadowlordState == nil || g.ShadowlordState.AllDefeated() {
		return nil, fmt.Errorf("%w: Doom entrance battle has no valid Underworld origin", ErrInvalidCombatEncounter)
	}
	origin := &DoomEntranceEncounterOrigin{
		ReturnLocation:    location,
		ReturnVehicle:     g.PartyVehicle.GetVehicleDetails().Snapshot(),
		ActivePlayerIndex: g.GetActivePlayerIndex(),
	}
	request := CombatEncounterRequest{
		Map: CombatEncounterMap{
			Territory:    references.BritanniaCombatTerritory,
			BritanniaMap: references.ShadowlordCombatMap,
		},
		EntryDirection: CombatEntryNorth,
		MonsterType:    references.MonsterShadowlord,
		CountMode:      CombatEncounterCountExact,
		ExactCount:     1,
		FriendsEnabled: false,
	}

	result, err := g.startCombatEncounter(request, func(combat *CombatState) error {
		combat.DoomEntranceOrigin = origin
		origin.SceptreReclaimed = g.reclaimSceptreForShadowlordCombat()

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("starting Doom entrance combat: %w", err)
	}

	return result, nil
}

func (g *GameState) resolveDoomEntranceEncounterOrigin(origin *DoomEntranceEncounterOrigin) {
	if origin == nil {
		return
	}
	if g.MapState.PlayerLocation != origin.ReturnLocation {
		panic("Doom entrance combat returned to an unexpected origin")
	}

	g.PartyVehicle.GetVehicleDetails().Restore(origin.ReturnVehicle)
	g.restorePreCombatActivePlayer(origin.ActivePlayerIndex)
	g.checkForPartyWipe()
}

func (g *GameState) reclaimSceptreForShadowlordCombat() bool {
	if !g.PartyState.Inventory.QuestItems.HasSome(references.Sceptre) {
		return false
	}

	g.SystemCallbacks.Message.AddRowStr("The Sceptre is reclaimed!")
	g.SystemCallbacks.Audio.PlaySoundEffect(SoundShardPulse)
	g.PartyState.Inventory.QuestItems.Set(references.Sceptre, 0)
	g.SystemCallbacks.Screen.RefreshInventoryDisplay()

	return true
}
