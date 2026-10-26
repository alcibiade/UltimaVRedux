package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestDungeonAnSanctTransformsEveryClosedChestVariant(t *testing.T) {
	for raw := 0x40; raw <= 0x4f; raw++ {
		t.Run(fmt.Sprintf("%02x", raw), func(t *testing.T) {
			state, _ := newDungeonInteractionTestState(t, references.DungeonCell(raw), uint64(raw))
			state.activeDungeon.SetChestAssessment(0, state.MapState.PlayerLocation.Position, references.ChestUncertain)
			beforeRevision := state.activeDungeon.revision
			beforeRolls := state.GetRNG().Calls()

			result := state.CastDungeonPackedSpell(references.AnSanct)

			require.True(t, result.Applied)
			assert.Equal(t, state.MapState.PlayerLocation.Position, result.Change.Position)
			assert.Equal(t, references.DungeonCell(0x70|(raw&0x08)), currentDungeonInteractionCell(t, state))
			assert.Equal(t, raw&0x01 != 0, result.Disarmed)
			assert.Equal(t, beforeRevision+1, result.Change.Revision)
			assert.Equal(t, references.ChestUninspected, state.activeDungeon.ChestAssessment(0, state.MapState.PlayerLocation.Position))
			assert.Equal(t, beforeRolls, state.GetRNG().Calls())
		})
	}
}

func TestDungeonAnSanctUsesCurrentBeforeWrappedFacingCell(t *testing.T) {
	state, _ := newDungeonInteractionTestState(t, 0x41, 42)
	state.SetPlayerPosition(references.Position{X: 7, Y: 1})
	state.activeDungeon.setFacingAndLastDirection(references.DungeonFacingEast)
	current := state.MapState.PlayerLocation.Position
	facing := references.Position{X: 0, Y: 1}
	setDungeonInteractionCell(t, state, current, 0x41)
	setDungeonInteractionCell(t, state, facing, 0x4e)

	first := state.CastDungeonPackedSpell(references.AnSanct)
	second := state.CastDungeonPackedSpell(references.AnSanct)

	require.True(t, first.Applied)
	assert.Equal(t, current, first.Change.Position)
	assert.Equal(t, references.DungeonCell(0x70), dungeonInteractionCellAt(t, state, current))
	require.True(t, second.Applied)
	assert.Equal(t, facing, second.Change.Position)
	assert.Equal(t, references.DungeonCell(0x78), dungeonInteractionCellAt(t, state, facing))
}

func TestDungeonFieldSpellsEncodeFacingPassageAndOpening(t *testing.T) {
	tests := []struct {
		spell references.Spell
		field byte
	}{
		{spell: references.InFlamGrav, field: 0x82},
		{spell: references.InNoxGrav, field: 0x81},
		{spell: references.InZuGrav, field: 0x80},
		{spell: references.InSanctGrav, field: 0x83},
	}
	for _, test := range tests {
		for _, raw := range []byte{0x00, 0x08} {
			t.Run(fmt.Sprintf("%s/%02x", references.GetSpellName(test.spell), raw), func(t *testing.T) {
				state, _ := newDungeonInteractionTestState(t, 0x40, uint64(raw))
				state.SetPlayerPosition(references.Position{X: 7, Y: 1})
				state.activeDungeon.setFacingAndLastDirection(references.DungeonFacingEast)
				setDungeonInteractionCell(t, state, state.MapState.PlayerLocation.Position, 0x40)
				facing := references.Position{X: 0, Y: 1}
				setDungeonInteractionCell(t, state, facing, references.DungeonCell(raw))

				result := state.CastDungeonPackedSpell(test.spell)

				require.True(t, result.Applied)
				assert.Equal(t, facing, result.Change.Position)
				assert.Equal(t, references.DungeonCell(test.field|(raw&0x08)), dungeonInteractionCellAt(t, state, facing))
				assert.Equal(t, references.DungeonCell(0x40), currentDungeonInteractionCell(t, state),
					"field creation must never target the current cell")
			})
		}
	}
}

func TestDungeonFieldSpellsRejectEveryOtherPackedFacingCell(t *testing.T) {
	state, _ := newDungeonInteractionTestState(t, 0x00, 42)
	state.activeDungeon.setFacingAndLastDirection(references.DungeonFacingEast)
	facing := references.Position{X: 2, Y: 1}
	spells := []references.Spell{
		references.InFlamGrav,
		references.InNoxGrav,
		references.InZuGrav,
		references.InSanctGrav,
	}
	for _, spell := range spells {
		for raw := 0; raw <= 0xff; raw++ {
			setDungeonInteractionCell(t, state, facing, references.DungeonCell(raw))
			valid := raw == 0x00 || raw == 0x08
			assert.Equal(t, valid, state.CanCastDungeonPackedSpell(spell), "%s/%02x", references.GetSpellName(spell), raw)
		}
	}
}

func TestDungeonAnGravTransformsEveryFieldVariant(t *testing.T) {
	for raw := 0x80; raw <= 0x8f; raw++ {
		t.Run(fmt.Sprintf("%02x", raw), func(t *testing.T) {
			state, _ := newDungeonInteractionTestState(t, references.DungeonCell(raw), uint64(raw))

			result := state.CastDungeonPackedSpell(references.AnGrav)

			require.True(t, result.Applied)
			assert.Equal(t, references.DungeonCell(raw&0x08), currentDungeonInteractionCell(t, state))
		})
	}
}

func TestDungeonPackedSpellValidationCommitsOnlyValidTargets(t *testing.T) {
	spells := []references.Spell{
		references.AnSanct,
		references.InFlamGrav,
		references.InNoxGrav,
		references.InZuGrav,
		references.InSanctGrav,
		references.AnGrav,
	}
	for _, spell := range spells {
		t.Run(references.GetSpellName(spell), func(t *testing.T) {
			state, _ := newDungeonInteractionTestState(t, 0x00, 42)
			caster := state.PartyState.GetCharacter(0)
			caster.Class = party_state.Wizard
			caster.Level = 8
			caster.SetMp(8)
			state.PartyState.Inventory.Spells.Set(spell, 1)
			setValidDungeonPackedSpellTarget(t, state, spell)
			request := CastRequest{Spell: spell, CasterIdx: 0}

			valid := state.ValidateAndConsumeSpellResources(request)

			require.True(t, valid.Valid)
			assert.Equal(t, byte(8-references.GetSpellCircle(spell)), caster.GetCurrentMp())
			assert.Zero(t, state.PartyState.Inventory.Spells.Get(spell))

			caster.SetMp(8)
			state.PartyState.Inventory.Spells.Set(spell, 1)
			setInvalidDungeonPackedSpellTarget(t, state)
			beforeRevision := state.activeDungeon.revision

			invalid := state.ValidateAndConsumeSpellResources(request)

			require.False(t, invalid.Valid)
			assert.Equal(t, "Failed!", invalid.ErrorMsg)
			assert.Equal(t, byte(8), caster.GetCurrentMp())
			assert.Equal(t, uint16(1), state.PartyState.Inventory.Spells.Get(spell))
			assert.Equal(t, beforeRevision, state.activeDungeon.revision)
		})
	}
}

func TestDungeonPackedSpellMutationSurvivesSaveSnapshotRoundTrip(t *testing.T) {
	state, _ := newDungeonInteractionTestState(t, 0x00, 42)
	state.activeDungeon.setFacingAndLastDirection(references.DungeonFacingEast)
	facing := references.Position{X: 2, Y: 1}
	setDungeonInteractionCell(t, state, facing, 0x08)
	require.True(t, state.CastDungeonPackedSpell(references.InSanctGrav).Applied)

	hydrated := roundTripSaveSnapshot(t, state)

	assert.Equal(t, references.DungeonCell(0x8b), dungeonInteractionCellAt(t, hydrated, facing))
}

func TestDungeonFloorSpellValidationAcceptsOnlyPassageDestination(t *testing.T) {
	for raw := 0; raw <= 0xff; raw++ {
		state, _ := newDungeonInteractionTestState(t, 0x00, uint64(raw))
		state.MapState.PlayerLocation.Floor = 1
		position := state.MapState.PlayerLocation.Position
		_, err := state.TransformDungeonCell(0, position, func(references.DungeonCell) references.DungeonCell {
			return references.DungeonCell(raw)
		})
		require.NoError(t, err)

		assert.Equal(t, raw&0xf0 == 0, state.CanCastDungeonFloorSpell(references.UusPor), "%02x", raw)
	}
}

func TestDungeonFloorSpellInternalTravelPreservesPositionAndGeneratesWanderer(t *testing.T) {
	state, _ := newDungeonInteractionTestState(t, 0x00, 42)
	state.MapState.PlayerLocation.Floor = 1
	position := state.MapState.PlayerLocation.Position
	_, err := state.TransformDungeonCell(0, position, func(references.DungeonCell) references.DungeonCell { return 0x08 })
	require.NoError(t, err)
	token, _ := state.BeginDungeonCommandCycle()

	result := state.CastDungeonFloorSpell(token, references.UusPor)
	state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)

	assert.Equal(t, DungeonFloorSpellChangedFloor, result.Outcome)
	assert.False(t, result.CommandCompleted)
	assert.Equal(t, references.FloorNumber(0), state.MapState.PlayerLocation.Floor)
	assert.Equal(t, position, state.MapState.PlayerLocation.Position)
	snapshot, active := state.DungeonStateSnapshot()
	require.True(t, active)
	assert.Equal(t, references.DungeonArrivedFromBelow, snapshot.LastDirection)
	assert.NotNil(t, snapshot.Wanderer)
	assert.True(t, state.IsAtDungeonCommandBoundary())
}

func TestDungeonFloorSpellBoundaryExitCompletesCommandBeforeTransition(t *testing.T) {
	tests := []struct {
		name      string
		floor     references.FloorNumber
		spell     references.Spell
		wantFloor references.FloorNumber
	}{
		{name: "Uus Por to Britannia", floor: 0, spell: references.UusPor, wantFloor: 0},
		{name: "Des Por to Underworld", floor: 7, spell: references.DesPor, wantFloor: -1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, _ := newDungeonInteractionTestState(t, 0x00, 42)
			state.MapState.PlayerLocation.Floor = test.floor
			dungeon := state.MapState.PlayerLocation.Location
			state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 5}
			token, _ := state.BeginDungeonCommandCycle()

			result := state.CastDungeonFloorSpell(token, test.spell)

			assert.Equal(t, DungeonFloorSpellExited, result.Outcome)
			assert.True(t, result.CommandCompleted)
			assert.Equal(t, references.BritanniaUnderworld, state.MapState.PlayerLocation.Location)
			assert.Equal(t, test.wantFloor, state.MapState.PlayerLocation.Floor)
			assert.Equal(t, state.GetDungeonEntranceCoordinates()[dungeon], state.MapState.PlayerLocation.Position)
			assert.Equal(t, 5, state.ActiveDurationEffect.TurnsRemaining)
			assert.True(t, state.IsAtDungeonCommandBoundary())
		})
	}
}

func TestDungeonFloorSpellsRejectDoomAndBlockedDestinationBeforeCommitment(t *testing.T) {
	tests := []struct {
		name     string
		location references.Location
		cell     references.DungeonCell
	}{
		{name: "Doom", location: references.Doom, cell: 0x00},
		{name: "blocked", location: references.Deceit, cell: 0xb0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).
				WithLocation(test.location).
				WithFloor(1).
				WithPlayerAt(1, 1).
				WithSystemCallbacks().
				Build()
			caster := state.PartyState.GetCharacter(0)
			caster.Class = party_state.Wizard
			caster.Level = 8
			caster.SetMp(8)
			state.PartyState.Inventory.Spells.Set(references.UusPor, 1)
			_, err := state.TransformDungeonCell(0, state.MapState.PlayerLocation.Position,
				func(references.DungeonCell) references.DungeonCell { return test.cell })
			require.NoError(t, err)

			result := state.ValidateAndConsumeSpellResources(CastRequest{Spell: references.UusPor, CasterIdx: 0})

			require.False(t, result.Valid)
			assert.Equal(t, "Failed!", result.ErrorMsg)
			assert.Equal(t, byte(8), caster.GetCurrentMp())
			assert.Equal(t, uint16(1), state.PartyState.Inventory.Spells.Get(references.UusPor))
		})
	}
}

func TestDungeonTimeSpellsUsePreCastActorPhaseAndProgressNewDuration(t *testing.T) {
	state, _ := newDungeonInteractionTestState(t, 0x00, 42)
	position := references.Position{X: 4, Y: 4}
	state.activeDungeon.replaceWanderer(&DungeonWanderer{
		Monster: references.MonsterRat, Floor: 0, Position: position, PreviousPosition: position,
	})
	for _, target := range []references.Position{{X: 4, Y: 3}, {X: 5, Y: 4}, {X: 4, Y: 5}, {X: 3, Y: 4}} {
		_, err := state.TransformDungeonCell(0, target,
			func(references.DungeonCell) references.DungeonCell { return 0x00 })
		require.NoError(t, err)
	}

	token, start := state.BeginDungeonCommandCycle()
	require.True(t, state.CastAnTym())
	result := state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)

	assert.True(t, start.DungeonActorsMove)
	assert.Equal(t, references.MonsterRat, result.WandererMovement.Monster,
		"the actor phase selected before An Tym must still run")
	assert.Equal(t, anTymDurationTurns-1, state.ActiveDurationEffect.TurnsRemaining)

	token, stopped := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
	assert.True(t, stopped.TimeStopped)
	assert.False(t, stopped.DungeonActorsMove)
	assert.Zero(t, stopped.ElapsedMinutes)
	assert.Equal(t, anTymDurationTurns-2, state.ActiveDurationEffect.TurnsRemaining)
}

func TestDungeonQuicknessCastPreservesAlternatingCadenceAcrossSnapshot(t *testing.T) {
	state, _ := newDungeonInteractionTestState(t, 0x00, 42)
	token, _ := state.BeginDungeonCommandCycle()
	require.True(t, state.CastCombatRelTym())
	state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
	assert.Equal(t, relTymDurationTurns-1, state.ActiveDurationEffect.TurnsRemaining)

	token, first := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
	token, second := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
	assert.True(t, first.DungeonActorsMove)
	assert.False(t, second.DungeonActorsMove)
	assert.Equal(t, 1, first.ElapsedMinutes)
	assert.Zero(t, second.ElapsedMinutes)

	hydrated := roundTripSaveSnapshot(t, state)
	assert.Equal(t, state.ActiveDurationEffect, hydrated.ActiveDurationEffect)

	token, expected := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
	token, actual := hydrated.BeginDungeonCommandCycle()
	hydrated.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
	assert.Equal(t, expected, actual)
}

func setValidDungeonPackedSpellTarget(t *testing.T, state *GameState, spell references.Spell) {
	t.Helper()
	state.activeDungeon.setFacingAndLastDirection(references.DungeonFacingEast)
	setDungeonInteractionCell(t, state, state.MapState.PlayerLocation.Position, 0x00)
	setDungeonInteractionCell(t, state, references.Position{X: 2, Y: 1}, 0x00)
	switch spell {
	case references.AnSanct:
		setDungeonInteractionCell(t, state, state.MapState.PlayerLocation.Position, 0x40)
	case references.AnGrav:
		setDungeonInteractionCell(t, state, state.MapState.PlayerLocation.Position, 0x80)
	}
}

func setInvalidDungeonPackedSpellTarget(t *testing.T, state *GameState) {
	t.Helper()
	state.activeDungeon.setFacingAndLastDirection(references.DungeonFacingEast)
	setDungeonInteractionCell(t, state, state.MapState.PlayerLocation.Position, 0x00)
	setDungeonInteractionCell(t, state, references.Position{X: 2, Y: 1}, 0xb0)
}
