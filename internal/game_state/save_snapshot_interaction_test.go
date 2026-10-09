package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestSaveSnapshotPendingInteractionHydration(t *testing.T) {
	for _, test := range []struct {
		name   string
		sprite indexes.SpriteIndex
		want   map_units.PendingNPCInteraction
	}{
		{name: "rat", sprite: indexes.Rat_KeyIndex, want: map_units.PendingNPCInteractionHostileCombat},
		{name: "gargoyle", sprite: indexes.StoneGargoyle_KeyIndex, want: map_units.PendingNPCInteractionHostileCombat},
		{name: "daemon", sprite: indexes.Daemon_KeyIndex, want: map_units.PendingNPCInteractionHostileCombat},
		{name: "civilian", sprite: indexes.BardKeyIndex, want: map_units.PendingNPCInteractionHostileCombat},
		{name: "guard", sprite: indexes.Guard_KeyIndex, want: map_units.PendingNPCInteractionGuardArrest},
		{name: "guard frame1", sprite: indexes.Guard_KeyIndex + 1, want: map_units.PendingNPCInteractionGuardArrest},
		{name: "guard frame2", sprite: indexes.Guard_KeyIndex + 2, want: map_units.PendingNPCInteractionGuardArrest},
		{name: "guard frame3", sprite: indexes.Guard_KeyIndex + 3, want: map_units.PendingNPCInteractionGuardArrest},
		{name: "shadowlord frame0", sprite: indexes.ShadowLord_KeyIndex, want: map_units.PendingNPCInteractionHostileCombat},
		{name: "shadowlord frame1", sprite: indexes.ShadowLord_KeyIndex + 1, want: map_units.PendingNPCInteractionHostileCombat},
		{name: "shadowlord frame2", sprite: indexes.ShadowLord_KeyIndex + 2, want: map_units.PendingNPCInteractionHostileCombat},
		{name: "shadowlord frame3", sprite: indexes.ShadowLord_KeyIndex + 3, want: map_units.PendingNPCInteractionHostileCombat},
		{name: "noncombat shape", sprite: indexes.ItemAnkh, want: map_units.PendingNPCInteractionTalk},
	} {
		for _, action := range []references.AiType{references.ExtortOrAttackOrFollow, references.DrudgeWorthThing} {
			for _, intent := range []string{"authored", "override", "astaroth"} {
				t.Run(fmt.Sprintf("%s/action%d/%s", test.name, action, intent), func(t *testing.T) {
					reference := references.NPCReference{Location: references.Yew, Schedule: references.NPCSchedule{
						Ai:   [3]byte{byte(references.Talk3), byte(action), byte(references.Fixed)},
						Time: [4]byte{0, 8, 16, 20},
					}}
					reference.SetKeyIndex(test.sprite)
					source := map_units.NewNPCFriendly(reference, 15)
					want := test.want
					if intent != "authored" {
						provenance := references.AIOverrideProvenanceNone
						if intent == "astaroth" {
							provenance = references.AIOverrideProvenanceAutomaticAstaroth
							want = map_units.PendingNPCInteractionTalk
						}
						source.MapUnitDetails().SetAIOverride(action, provenance)
					}
					source.MapUnitDetails().PendingInteraction = want
					units := map_units.MapUnits{source}
					saved, err := captureControllerUnits(&units, 0, reference.Location)
					require.NoError(t, err)
					require.True(t, saved[0].LegacyWantsToAutoTalk)

					for _, hour := range []byte{7, 8} {
						state := &GameState{DateTime: datetime.UltimaDate{Hour: hour}}
						generated := map_units.MapUnits{map_units.NewNPCFriendly(reference, 15)}
						restored, hydrateErr := state.hydrateUnits(saved, &generated)
						require.NoError(t, hydrateErr)
						expected := want
						if intent == "authored" && hour == 7 {
							expected = map_units.PendingNPCInteractionTalk
						}
						assert.Equal(t, expected, restored[0].MapUnitDetails().PendingInteraction, "hour %d", hour)
					}

					saved[0].LegacyWantsToAutoTalk = false
					state := &GameState{DateTime: datetime.UltimaDate{Hour: 8}}
					generated := map_units.MapUnits{map_units.NewNPCFriendly(reference, 15)}
					restored, err := state.hydrateUnits(saved, &generated)
					require.NoError(t, err)
					assert.Equal(t, map_units.PendingNPCInteractionNone, restored[0].MapUnitDetails().PendingInteraction)
				})
			}
		}
	}
}

func TestSaveSnapshotPendingInteractionNonAttackOverride(t *testing.T) {
	for _, action := range []references.AiType{references.Fixed, references.Talk3} {
		t.Run(fmt.Sprintf("action%d", action), func(t *testing.T) {
			reference := references.NPCReference{Schedule: references.NPCSchedule{Ai: [3]byte{byte(references.DrudgeWorthThing)}}}
			reference.SetKeyIndex(indexes.Daemon_KeyIndex)
			generated := map_units.MapUnits{map_units.NewNPCFriendly(reference, 1)}
			state := &GameState{}
			restored, err := state.hydrateUnits([]save_snapshot.MapUnitSnapshot{{
				Kind: "npc", NPCNum: 1, LegacyWantsToAutoTalk: true,
				OverriddenAI: int(action), AIOverrideProvenance: string(references.AIOverrideProvenanceNone),
			}}, &generated)
			require.NoError(t, err)
			assert.Equal(t, map_units.PendingNPCInteractionTalk, restored[0].MapUnitDetails().PendingInteraction)
		})
	}
}

func TestSaveSnapshotRoundTripPendingAuthoredRatInteraction(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Yew).
		WithPlayerAt(15, 15).
		WithRandomSeed(484).
		Build()
	state.DateTime.Hour = 8
	state.initializeSnapshotMapsAndControllers()
	var source *map_units.NPCFriendly
	for _, unit := range *state.CurrentNPCAIController.GetNpcs() {
		if unit.MapUnitDetails().NPCNum == 15 {
			var ok bool
			source, ok = unit.(*map_units.NPCFriendly)
			require.True(t, ok)
			break
		}
	}
	require.NotNil(t, source)
	require.Equal(t, indexes.Rat_KeyIndex, source.NPCReference.GetSpriteIndex())
	require.Equal(t, references.Unset, source.MapUnitDetails().GetOverriddenAiType())
	require.Contains(t, []references.AiType{references.ExtortOrAttackOrFollow, references.DrudgeWorthThing}, source.GetIndividualBehaviourByUltimaData(state.DateTime).Ai)
	source.MapUnitDetails().PendingInteraction = map_units.PendingNPCInteractionHostileCombat

	before, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	restored, err := NewGameStateFromSaveSnapshot(before, state.GameConfig, state.GameReferences, xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
	require.NoError(t, err)
	assert.Equal(t, state.DateTime, restored.DateTime)
	var pending map_units.PendingNPCInteraction
	for _, unit := range *restored.CurrentNPCAIController.GetNpcs() {
		if unit.MapUnitDetails().NPCNum == source.MapUnitDetails().NPCNum {
			pending = unit.MapUnitDetails().PendingInteraction
			break
		}
	}
	assert.Equal(t, map_units.PendingNPCInteractionHostileCombat, pending)
	after, err := restored.CaptureSaveSnapshot()
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestSaveSnapshotRoundTripGeneratedHostileOverrides(t *testing.T) {
	for _, provenance := range []references.AIOverrideProvenance{
		references.AIOverrideProvenanceGuardHostility,
		references.AIOverrideProvenanceAutomaticAstaroth,
		references.AIOverrideProvenanceAutomaticNosfentor,
	} {
		t.Run(string(provenance), func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).
				WithLocation(references.Britain).
				WithPlayerAt(15, 15).
				WithRandomSeed(484).
				WithSystemCallbacks().
				Build()
			controller, ok := state.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
			require.True(t, ok)
			if provenance == references.AIOverrideProvenanceAutomaticAstaroth {
				controller.SetNPCsReactToShadowlord(references.Astaroth, func() bool { return true })
			} else {
				if provenance == references.AIOverrideProvenanceAutomaticNosfentor {
					controller.SetNPCsReactToShadowlord(references.Nosfentor, func() bool { return true })
				}
				_, err := state.ActivateGuards()
				require.NoError(t, err)
			}
			var source *map_units.NPCFriendly
			for _, unit := range *controller.GetNpcs() {
				friendly, isFriendly := unit.(*map_units.NPCFriendly)
				if !isFriendly || friendly.MapUnitDetails().AIOverrideProvenance() != provenance ||
					friendly.MapUnitDetails().GetOverriddenAiType() != references.DrudgeWorthThing {
					continue
				}
				if provenance == references.AIOverrideProvenanceAutomaticAstaroth || friendly.NPCReference.GetNPCType() == references.Guard {
					source = friendly
					break
				}
			}
			require.NotNil(t, source, "producer must generate action 7 with %s provenance", provenance)
			pending := map_units.PendingNPCInteractionGuardArrest
			if provenance == references.AIOverrideProvenanceAutomaticAstaroth {
				pending = map_units.PendingNPCInteractionTalk
			}
			source.MapUnitDetails().PendingInteraction = pending

			before, err := state.CaptureSaveSnapshot()
			require.NoError(t, err)
			require.Empty(t, save_snapshot.Validate(before))
			restored, err := NewGameStateFromSaveSnapshot(before, state.GameConfig, state.GameReferences, xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
			require.NoError(t, err)
			var restoredSource *map_units.MapUnitDetails
			for _, unit := range *restored.CurrentNPCAIController.GetNpcs() {
				if unit.MapUnitDetails().NPCNum == source.MapUnitDetails().NPCNum {
					restoredSource = unit.MapUnitDetails()
					break
				}
			}
			require.NotNil(t, restoredSource)
			assert.Equal(t, references.AiType(references.DrudgeWorthThing), restoredSource.GetOverriddenAiType())
			assert.Equal(t, provenance, restoredSource.AIOverrideProvenance())
			assert.Equal(t, pending, restoredSource.PendingInteraction)
			after, err := restored.CaptureSaveSnapshot()
			require.NoError(t, err)
			assert.Equal(t, before, after)
		})
	}
}
