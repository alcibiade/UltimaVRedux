package game_state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestSmallMapHardResetPlacesNPCAtDesiredSlotAndRecordsArrival(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		Build()
	controller, ok := state.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
	if !ok {
		t.Fatalf("current NPC controller = %T, want small-map controller", state.CurrentNPCAIController)
	}

	var npc *map_units.NPCFriendly
	for _, unit := range *controller.GetNpcs() {
		if friendly, friendlyOK := unit.(*map_units.NPCFriendly); friendlyOK {
			npc = friendly
			break
		}
	}
	if npc == nil {
		t.Fatal("Britain has no friendly NPC for hard-reset test")
	}
	desired := npc.NPCReference.Schedule.DesiredSlot(state.DateTime)
	npc.NPCReference.Schedule.Ai[desired] = byte(references.Talk3)
	npc.PlaceAtScheduleSlot(desired)
	if !npc.ApplyTalk3OneShot() {
		t.Fatal("test NPC did not consume TALK3 before reset")
	}
	npc.MapUnitDetails().RestoreArrivedScheduleSlot((desired + 1) % 3)
	npc.SetPos(references.Position{X: 63, Y: 63})

	controller.HardResetNPCs(false)
	want := npc.NPCReference.Schedule.GetIndividualNPCBehaviourBySlot(desired)
	if npc.Pos() != want.Position || npc.Floor() != want.Floor || npc.ArrivedScheduleSlot() != desired {
		t.Fatalf("hard reset NPC = pos %v floor %d slot %d, want pos %v floor %d slot %d",
			npc.Pos(), npc.Floor(), npc.ArrivedScheduleSlot(), want.Position, want.Floor, desired)
	}
	if references.AiType(npc.NPCReference.Schedule.Ai[desired]) != references.Wander3 {
		t.Fatal("hard reset undid consumed TALK3 schedule mutation")
	}
	if slot, consumed := npc.MapUnitDetails().ConsumedTalk3ScheduleSlot(); !consumed || slot != desired {
		t.Fatalf("hard reset consumed TALK3 slot = %d, %v, want %d, true", slot, consumed, desired)
	}
}

func TestHydrateUnitsInfersHistoricalMerchantArrivalFromExactDestination(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
	saved, err := captureControllerUnits(
		&map_units.MapUnits{npc},
		0,
		state.MapState.PlayerLocation.Location,
	)
	require.NoError(t, err)
	saved[0].ArrivedScheduleSlot = nil

	generatedNPC := map_units.NewNPCFriendly(npc.NPCReference, npc.MapUnitDetails().NPCNum)
	generated := map_units.MapUnits{generatedNPC}
	restored, err := state.hydrateUnits(saved, &generated)
	require.NoError(t, err)
	restoredNPC, ok := restored[0].(*map_units.NPCFriendly)
	require.True(t, ok)
	require.Equal(t, references.NPCScheduleSlot(1), restoredNPC.ArrivedScheduleSlot())
	state.CurrentNPCAIController.GetNpcs().ReplaceWith(restored)
	_, err = state.BeginMerchantSession(restoredNPC)
	require.NoError(t, err, "historical save at the work destination must restore merchant availability")
}

func TestHydrateUnitsDefaultsHistoricalUnmatchedArrivalToClosedSlot(t *testing.T) {
	reference := references.NPCReference{
		NpcType: references.NoStatedNpc,
		Schedule: references.NPCSchedule{
			X: [3]byte{1, 5, 9}, Y: [3]byte{2, 6, 10}, Floor: [3]byte{0, 1, 0},
		},
	}
	generatedNPC := map_units.NewNPCFriendly(reference, 1)
	generated := map_units.MapUnits{generatedNPC}
	saved := []save_snapshot.MapUnitSnapshot{{
		Kind: "npc", NPCNum: 1,
		Position:             save_snapshot.PositionSnapshot{Location: int(references.Britain), X: 20, Y: 21, Floor: 0},
		AIOverrideProvenance: string(references.AIOverrideProvenanceNone),
	}}

	restored, err := (&GameState{}).hydrateUnits(saved, &generated)
	require.NoError(t, err)
	restoredNPC := restored[0].(*map_units.NPCFriendly)
	require.Equal(t, references.NPCScheduleSlot(0), restoredNPC.ArrivedScheduleSlot())
}

func TestHydrateUnitsRestoresConsumedTalk3Mutation(t *testing.T) {
	reference := references.NPCReference{
		NpcType: references.NoStatedNpc,
		Schedule: references.NPCSchedule{
			Ai: [3]byte{byte(references.Talk3), byte(references.Talk3), byte(references.Talk3)},
			X:  [3]byte{1, 5, 9}, Y: [3]byte{2, 6, 10},
		},
	}
	source := map_units.NewNPCFriendly(reference, 1)
	source.PlaceAtScheduleSlot(1)
	require.True(t, source.ApplyTalk3OneShot())
	saved, err := captureControllerUnits(&map_units.MapUnits{source}, 0, references.Britain)
	require.NoError(t, err)
	require.NotNil(t, saved[0].ConsumedTalk3ScheduleSlot)
	require.Equal(t, 1, *saved[0].ConsumedTalk3ScheduleSlot)

	generatedNPC := map_units.NewNPCFriendly(reference, 1)
	generated := map_units.MapUnits{generatedNPC}
	restored, err := (&GameState{}).hydrateUnits(saved, &generated)
	require.NoError(t, err)
	restoredNPC := restored[0].(*map_units.NPCFriendly)
	require.Equal(t, references.AiType(references.Wander3), references.AiType(restoredNPC.NPCReference.Schedule.Ai[1]))
	require.Equal(t, references.AiType(references.Talk3), references.AiType(restoredNPC.NPCReference.Schedule.Ai[0]))
	slot, consumed := restoredNPC.MapUnitDetails().ConsumedTalk3ScheduleSlot()
	require.True(t, consumed)
	require.Equal(t, references.NPCScheduleSlot(1), slot)
	require.False(t, restoredNPC.ApplyTalk3OneShot())
}

func TestHydrateUnitsRetainsIncomingWalkAndLastArrival(t *testing.T) {
	reference := references.NPCReference{
		Location: references.Trinsic,
		NpcType:  references.NoStatedNpc,
		Schedule: references.NPCSchedule{
			X: [3]byte{3, 13, 20}, Y: [3]byte{3, 10, 20}, Floor: [3]byte{1, 0, 0},
		},
	}
	source := map_units.NewNPCFriendly(reference, 7)
	source.PlaceAtScheduleSlot(0)
	// A receiving-floor entry keeps the previous arrival and its walking route.
	entry := references.Position{X: 10, Y: 10}
	path := []references.Position{{X: 11, Y: 10}, {X: 12, Y: 10}, {X: 13, Y: 10}}
	source.SetFloor(0)
	source.SetPos(entry)
	source.MapUnitDetails().SetCurrentPath(path)
	saved, err := captureControllerUnits(&map_units.MapUnits{source}, 0, references.Trinsic)
	require.NoError(t, err)
	generated := map_units.MapUnits{map_units.NewNPCFriendly(reference, 7)}
	restored, err := (&GameState{}).hydrateUnits(saved, &generated)
	require.NoError(t, err)
	npc, ok := restored[0].(*map_units.NPCFriendly)
	require.True(t, ok)
	require.Equal(t, entry, npc.Pos())
	require.Equal(t, references.FloorNumber(0), npc.Floor())
	require.Equal(t, references.NPCScheduleSlot(0), npc.ArrivedScheduleSlot())
	require.Equal(t, path, npc.MapUnitDetails().CurrentPath)
	// Hidden relocation/reset retires that saved route rather than reviving it.
	npc.PlaceAtScheduleSlot(2)
	require.Equal(t, references.NPCScheduleSlot(2), npc.ArrivedScheduleSlot())
	require.Empty(t, npc.MapUnitDetails().CurrentPath)
}

func TestHydrateUnitsReconstructsTypedInteractionFromLegacyAutoTalk(t *testing.T) {
	for _, test := range []struct {
		name       string
		location   references.Location
		npcSlot    int
		sprite     indexes.SpriteIndex
		override   references.AiType
		provenance references.AIOverrideProvenance
		want       map_units.PendingNPCInteraction
	}{
		{
			name:       "ordinary NPC Talk",
			location:   references.Yew,
			npcSlot:    1,
			sprite:     indexes.GuardKeyIndex,
			want:       map_units.PendingNPCInteractionTalk,
			provenance: references.AIOverrideProvenanceNone,
		},
		{
			name:       "hostile Guard arrest",
			location:   references.Yew,
			npcSlot:    1,
			sprite:     indexes.GuardKeyIndex,
			override:   references.ExtortOrAttackOrFollow,
			provenance: references.AIOverrideProvenanceGuardHostility,
			want:       map_units.PendingNPCInteractionGuardArrest,
		},
		{
			name:       "hostile Windemere devil combat",
			location:   references.Windemere,
			npcSlot:    10,
			sprite:     indexes.DaemonKeyIndex,
			override:   references.ExtortOrAttackOrFollow,
			provenance: references.AIOverrideProvenanceGuardHostility,
			want:       map_units.PendingNPCInteractionHostileCombat,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			reference := references.NPCReference{
				Location:     test.location,
				DialogNumber: byte(references.DialogGuard),
			}
			reference.SetKeyIndex(test.sprite)
			generatedNPC := map_units.NewNPCFriendly(reference, test.npcSlot)
			generated := map_units.MapUnits{generatedNPC}
			saved := []save_snapshot.MapUnitSnapshot{{
				Kind: "npc", NPCNum: test.npcSlot,
				Position:              save_snapshot.PositionSnapshot{Location: int(test.location)},
				LegacyWantsToAutoTalk: true,
				OverriddenAI:          int(test.override),
				AIOverrideProvenance:  string(test.provenance),
			}}

			restored, err := (&GameState{}).hydrateUnits(saved, &generated)
			require.NoError(t, err)
			require.Equal(t, test.want, restored[0].MapUnitDetails().PendingInteraction)
		})
	}
}
