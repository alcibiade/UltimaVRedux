package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

func TestRecruitNPCPersistsSourceAbsenceAcrossReloadAtLocationBoundaries(t *testing.T) {
	for _, location := range []references.Location{references.Moonglow, references.SerpentsHold} {
		t.Run(location.String(), func(t *testing.T) {
			state, source, companionName := newRecruitmentTestState(t, location)
			sourceSlot := source.MapUnitDetails().NPCNum

			require.NoError(t, state.RecruitNPC(source, companionName))
			assert.True(t, state.PartyState.IsNPCDead(location, sourceSlot))
			assert.Equal(t, 1, countActiveRosterName(state, companionName))
			assert.Zero(t, countFriendlyNPCSlot(state, sourceSlot))

			companion := activeRosterCharacter(t, state, companionName)
			companion.PartyMap = party_state.PartyMapAtInn(location)
			assert.True(t, state.PartyState.IsNPCDead(location, sourceSlot), "leaving the companion must not restore the source NPC")
			require.NoError(t, state.PartyState.AddCharacterToActiveParty(1))
			assert.True(t, state.PartyState.IsNPCDead(location, sourceSlot), "retrieving the companion must not restore the source NPC")

			state.UpdateSmallMap(state.GameReferences.TileReferences, state.GameReferences.LocationReferences)
			assert.Zero(t, countFriendlyNPCSlot(state, sourceSlot), "town reconstruction recreated the recruited NPC")
		})
	}
}

func TestRecruitNPCFailuresDoNotMutateRosterOrAbsence(t *testing.T) {
	for _, test := range []struct {
		name    string
		prepare func(*GameState, *map_units.NPCFriendly, string) (*map_units.NPCFriendly, string)
	}{
		{
			name: "roster name missing",
			prepare: func(_ *GameState, source *map_units.NPCFriendly, _ string) (*map_units.NPCFriendly, string) {
				return source, "Missing"
			},
		},
		{
			name: "party full",
			prepare: func(state *GameState, source *map_units.NPCFriendly, companionName string) (*map_units.NPCFriendly, string) {
				for index := range party_state.MaxActiveParty {
					state.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapActive
				}
				return source, companionName
			},
		},
		{
			name: "detached source",
			prepare: func(_ *GameState, source *map_units.NPCFriendly, companionName string) (*map_units.NPCFriendly, string) {
				detached := map_units.NewNPCFriendly(source.NPCReference, source.MapUnitDetails().NPCNum)
				detached.SetPos(source.Pos())
				detached.SetFloor(source.Floor())
				return detached, companionName
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, source, companionName := newRecruitmentTestState(t, references.Britain)
			sourceSlot := source.MapUnitDetails().NPCNum
			requestedSource, requestedName := test.prepare(state, source, companionName)
			beforeRoster := state.CurrentPartyRosterSnapshot()
			beforeMet, beforeDead := state.PartyState.NPCFlagsSnapshot()
			beforeUnits := append([]map_units.MapUnit(nil), (*state.CurrentNPCAIController.GetNpcs())...)

			err := state.RecruitNPC(requestedSource, requestedName)

			require.Error(t, err)
			if test.name == "party full" {
				require.ErrorIs(t, err, ErrPartyCapacity)
			}
			assert.Equal(t, beforeRoster, state.CurrentPartyRosterSnapshot())
			afterMet, afterDead := state.PartyState.NPCFlagsSnapshot()
			assert.Equal(t, beforeMet, afterMet)
			assert.Equal(t, beforeDead, afterDead)
			assert.Equal(t, beforeUnits, append([]map_units.MapUnit(nil), (*state.CurrentNPCAIController.GetNpcs())...))
			assert.False(t, state.PartyState.IsNPCDead(references.Britain, sourceSlot))
			assert.Equal(t, 1, countFriendlyNPCSlot(state, sourceSlot))
		})
	}
}

func TestRecruitNPCNativeSnapshotAndRollbackPreserveOneRepresentation(t *testing.T) {
	state, source, companionName := newRecruitmentTestState(t, references.Britain)
	sourceSlot := source.MapUnitDetails().NPCNum
	before, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)

	require.NoError(t, state.RecruitNPC(source, companionName))
	recruited, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	assert.Contains(t, recruited.Party.DeadNPCs, save_snapshot.NPCFlagSnapshot{
		Location: int(references.Britain - references.Moonglow),
		NPC:      sourceSlot,
	})
	assert.False(t, snapshotHasNPCSlot(recruited, references.Britain, sourceSlot))

	hydrated, err := NewGameStateFromSaveSnapshot(
		recruited,
		state.GameConfig,
		state.GameReferences,
		xTilesVisibleOnGameScreen,
		yTilesVisibleOnGameScreen,
	)
	require.NoError(t, err)
	assert.True(t, hydrated.PartyState.IsNPCDead(references.Britain, sourceSlot))
	assert.Equal(t, 1, countActiveRosterName(hydrated, companionName))
	assert.Zero(t, countFriendlyNPCSlot(hydrated, sourceSlot))

	rolledBack, err := NewGameStateFromSaveSnapshot(
		before,
		state.GameConfig,
		state.GameReferences,
		xTilesVisibleOnGameScreen,
		yTilesVisibleOnGameScreen,
	)
	require.NoError(t, err)
	assert.False(t, rolledBack.PartyState.IsNPCDead(references.Britain, sourceSlot))
	assert.Zero(t, countActiveRosterName(rolledBack, companionName))
	assert.Equal(t, 1, countFriendlyNPCSlot(rolledBack, sourceSlot))
}

func TestRecruitNPCSearchesInactiveRosterFromEndAndNeverMatchesAvatar(t *testing.T) {
	state, source, companionName := newRecruitmentTestState(t, references.Britain)
	companion := state.PartyState.GetCharacter(8)
	avatar := state.PartyState.GetAvatar()
	copy(avatar.Name[:], companionName)

	require.NoError(t, state.RecruitNPC(source, companionName))

	assert.True(t, avatar.IsInParty())
	assert.True(t, companion.IsInParty(), "the matching inactive companion must join")
	assert.True(t, state.PartyState.IsNPCDead(references.Britain, source.MapUnitDetails().NPCNum))
}

func newRecruitmentTestState(
	t *testing.T,
	location references.Location,
) (*GameState, *map_units.NPCFriendly, string) {
	t.Helper()
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(location).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	for index := range party_state.NRosterSlots {
		state.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapUnrecruited
	}
	state.PartyState.GetAvatar().PartyMap = party_state.PartyMapActive
	companionName := state.PartyState.GetCharacter(8).GetNameAsString()
	require.GreaterOrEqual(t, len(companionName), 3)

	for _, unit := range *state.CurrentNPCAIController.GetNpcs() {
		friendly, ok := unit.(*map_units.NPCFriendly)
		if ok && friendly.NPCReference.GetNPCType() != references.Vehicle {
			require.Equal(t, location, friendly.NPCReference.Location)
			return state, friendly, companionName
		}
	}

	t.Fatalf("location %d has no recruitable test source", location)
	return nil, nil, ""
}

func activeRosterCharacter(t *testing.T, state *GameState, name string) *party_state.PlayerCharacter {
	t.Helper()
	for index := range party_state.NRosterSlots {
		character := state.PartyState.GetCharacter(index)
		if character.IsInParty() && character.GetNameAsString() == name {
			return character
		}
	}
	t.Fatalf("active companion %q not found", name)
	return nil
}

func countActiveRosterName(state *GameState, name string) int {
	count := 0
	state.PartyState.ForEachCharacter(func(_ int, character *party_state.PlayerCharacter) {
		if character.IsInParty() && character.GetNameAsString() == name {
			count++
		}
	})
	return count
}

func countFriendlyNPCSlot(state *GameState, slot int) int {
	count := 0
	for _, unit := range *state.CurrentNPCAIController.GetNpcs() {
		if _, ok := unit.(*map_units.NPCFriendly); ok && unit.MapUnitDetails().NPCNum == slot {
			count++
		}
	}
	return count
}

func snapshotHasNPCSlot(snapshot save_snapshot.SaveSnapshot, location references.Location, slot int) bool {
	for _, unit := range snapshot.MapUnits {
		if unit.Kind == "npc" && unit.Position.Location == int(location) && unit.NPCNum == slot {
			return true
		}
	}
	return false
}
