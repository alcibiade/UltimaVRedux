package map_units

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestNPCFriendlyArrivedAndDesiredScheduleSlots(t *testing.T) {
	schedule := references.NPCSchedule{
		Ai:    [3]byte{byte(references.Fixed), byte(references.MerchantBuyingSelling), byte(references.Fixed)},
		X:     [3]byte{1, 5, 9},
		Y:     [3]byte{2, 6, 10},
		Floor: [3]byte{0, 0, 0},
		Time:  [4]byte{6, 12, 18, 0},
	}
	npc := NewNPCFriendly(references.NPCReference{Schedule: schedule, NpcType: references.NoStatedNpc}, 1)
	npc.PlaceAtScheduleSlot(0)

	workTime := datetime.UltimaDate{Hour: 13}
	if npc.ArrivedScheduleSlot().IsMerchantSlot() || !npc.NPCReference.Schedule.DesiredSlot(workTime).IsMerchantSlot() {
		t.Fatal("approaching merchant must have closed arrived slot and open desired slot")
	}
	if npc.RecordScheduleArrival(workTime) {
		t.Fatal("approaching merchant recorded arrival before reaching the destination")
	}

	npc.PlaceAtScheduleSlot(npc.NPCReference.Schedule.DesiredSlot(workTime))
	if !npc.ArrivedScheduleSlot().IsMerchantSlot() || !npc.NPCReference.Schedule.DesiredSlot(workTime).IsMerchantSlot() {
		t.Fatal("merchant at work must have open arrived and desired slots")
	}

	departingTime := datetime.UltimaDate{Hour: 19}
	if !npc.ArrivedScheduleSlot().IsMerchantSlot() || npc.NPCReference.Schedule.DesiredSlot(departingTime).IsMerchantSlot() {
		t.Fatal("departing merchant must retain arrived work slot while desired slot closes immediately")
	}
}

func TestNPCFriendlyResetTeleportAndActualArrivalUpdateArrivedSlot(t *testing.T) {
	schedule := references.NPCSchedule{
		X: [3]byte{1, 5, 9}, Y: [3]byte{2, 6, 10}, Floor: [3]byte{0, 1, 0}, Time: [4]byte{6, 12, 18, 0},
	}
	npc := NewNPCFriendly(references.NPCReference{Schedule: schedule, NpcType: references.NoStatedNpc}, 1)
	npc.PlaceAtScheduleSlot(0)
	npc.PlaceAtScheduleSlot(1)
	if npc.Pos() != (references.Position{X: 5, Y: 6}) || npc.Floor() != 1 || npc.ArrivedScheduleSlot() != 1 {
		t.Fatalf("schedule teleport did not place and arrive atomically: pos=%v floor=%d slot=%d", npc.Pos(), npc.Floor(), npc.ArrivedScheduleSlot())
	}

	npc.PlaceAtScheduleSlot(0)
	arrivalTime := datetime.UltimaDate{Hour: 19}
	npc.SetPos(references.Position{X: 9, Y: 10})
	if !npc.RecordScheduleArrival(arrivalTime) || npc.ArrivedScheduleSlot() != 2 {
		t.Fatalf("physical arrival did not update slot: %d", npc.ArrivedScheduleSlot())
	}
}

func TestNPCFriendlyApplyTalk3OneShotMutatesOnlyLoadedCurrentSlot(t *testing.T) {
	reference := references.NPCReference{
		NpcType:  references.NoStatedNpc,
		Schedule: references.NPCSchedule{Ai: [3]byte{byte(references.Talk3), byte(references.Talk3), byte(references.Talk3)}},
	}
	npc := NewNPCFriendly(reference, 1)
	npc.PlaceAtScheduleSlot(1)

	if !npc.ApplyTalk3OneShot() {
		t.Fatal("first TALK3 interaction did not mutate the arrived slot")
	}
	if npc.ApplyTalk3OneShot() {
		t.Fatal("second TALK3 interaction mutated the slot again")
	}
	if got := references.AiType(npc.NPCReference.Schedule.Ai[1]); got != references.Wander3 {
		t.Fatalf("loaded slot AI = %d, want WANDER3", got)
	}
	if npc.NPCReference.Schedule.Ai[0] != byte(references.Talk3) || npc.NPCReference.Schedule.Ai[2] != byte(references.Talk3) {
		t.Fatal("TALK3 mutation changed a non-current schedule slot")
	}
	if reference.Schedule.Ai[1] != byte(references.Talk3) {
		t.Fatal("TALK3 mutation escaped the loaded NPC schedule copy")
	}
	if slot, ok := npc.MapUnitDetails().ConsumedTalk3ScheduleSlot(); !ok || slot != 1 {
		t.Fatalf("consumed TALK3 slot = %d, %v, want 1, true", slot, ok)
	}

	// Schedule teleports update arrival but do not undo the one-shot mutation.
	npc.PlaceAtScheduleSlot(0)
	npc.PlaceAtScheduleSlot(1)
	if npc.ApplyTalk3OneShot() {
		t.Fatal("schedule teleport reset consumed TALK3 state")
	}
	if slot, ok := npc.MapUnitDetails().ConsumedTalk3ScheduleSlot(); !ok || slot != 1 {
		t.Fatalf("consumed TALK3 slot after teleports = %d, %v, want 1, true", slot, ok)
	}
}
