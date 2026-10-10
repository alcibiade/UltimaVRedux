package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// ShadowlordTownMinLocation is the minimum valid town location for Shadowlords.
// Towns 1-8 in original: Moonglow(1), Britain(2), Jhelom(3), Yew(4),
// Minoc(5), Trinsic(6), SkaraBrae(7), NewMagincia(8).
const ShadowlordTownMinLocation = 1

// ShadowlordTownMaxLocation is the maximum valid town location for Shadowlords.
const ShadowlordTownMaxLocation = 8

// ShadowlordTownCount is the number of valid towns (1-8).
const ShadowlordTownCount = 8

// GetShadowlordForLocation returns the living Shadowlord persistently assigned
// to the location, or NoShadowlord when the town is unassigned.
func (g *GameState) GetShadowlordForLocation(location references.Location) references.Shadowlord {
	if !IsShadowlordTown(location) {
		return references.NoShadowlord
	}
	for i := 0; i < references.ShadowlordCount; i++ {
		lord := references.Shadowlord(i)
		if g.ShadowlordState.IsAlive(lord) && g.ShadowlordState.TownAssignments[i] == location {
			return lord
		}
	}

	return references.NoShadowlord
}

// moveShadowlordsAtMidnight assigns every living Shadowlord a new town in
// identity order. Candidates are rejected against the currently loaded map and
// the assignment array as it is mutated, matching calendar advancement.
// See ref:SHADOW-0001.
func (g *GameState) moveShadowlordsAtMidnight() {
	for index := range references.ShadowlordCount {
		lord := references.Shadowlord(index)
		if !g.ShadowlordState.IsAlive(lord) {
			continue
		}

		for {
			candidate := references.Location(g.RandomIntInRange(ShadowlordTownMinLocation, ShadowlordTownMaxLocation))
			if candidate == g.MapState.PlayerLocation.Location || shadowlordTownIsAssigned(g.ShadowlordState.TownAssignments, candidate) {
				continue
			}
			g.ShadowlordState.TownAssignments[index] = candidate

			break
		}
	}
}

func shadowlordTownIsAssigned(assignments [references.ShadowlordCount]references.Location, candidate references.Location) bool {
	for _, assignment := range assignments {
		if assignment == candidate {
			return true
		}
	}

	return false
}

// IsShadowlordTown returns true if the location is a valid Shadowlord town (1-8).
// Based on original game's valid town range for lords[] array.
func IsShadowlordTown(loc references.Location) bool {
	return loc >= ShadowlordTownMinLocation && loc <= ShadowlordTownMaxLocation
}

// GetShadowlordSpawnPosition returns the reference town spawn point.
// All eight towns use X=15 and their distinct reference Y coordinates. ref:SHADOW-0002
func GetShadowlordSpawnPosition(townLocation references.Location) references.Position {
	// Original shadow_ypos indexed by town-1 (0-7)
	shadowYPos := [ShadowlordTownCount]int{4, 9, 15, 8, 17, 10, 11, 10}

	if !IsShadowlordTown(townLocation) {
		return references.Position{X: 15, Y: 10} // Default fallback
	}

	townIndex := int(townLocation) - ShadowlordTownMinLocation

	return references.Position{
		X: 15,
		Y: references.Coordinate(shadowYPos[townIndex]),
	}
}

// SpawnAutomaticShadowlordNPC uses the town’s fixed reference position
// for automatic presence, unlike Yell summons near the party.
// Return whether insertion succeeded. ref:SHADOW-0002
func (g *GameState) SpawnAutomaticShadowlordNPC(lord references.Shadowlord, location references.Location) bool {
	if !IsShadowlordTown(location) || g.MapState.PlayerLocation.Location != location {
		logging.Warn(logging.CategoryGameState,
			"SpawnAutomaticShadowlordNPC: %s is not the loaded town", location.String())

		return false
	}
	spawnPos := GetShadowlordSpawnPosition(location)

	logging.Debug(logging.CategoryGameState,
		"SpawnAutomaticShadowlordNPC: spawning %s at (%d, %d) in %s",
		lord.String(), spawnPos.X, spawnPos.Y, location.String())

	if g.CurrentNPCAIController == nil {
		logging.Warn(logging.CategoryGameState,
			"SpawnAutomaticShadowlordNPC: CurrentNPCAIController is nil, cannot spawn")

		return false
	}

	return g.insertShadowlordNPCAtFloorPosition(lord, spawnPos, references.MainFloor, true) != nil
}
