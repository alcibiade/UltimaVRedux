package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func newDebugInventoryLockerTestState(t *testing.T) *GameState {
	t.Helper()
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	return gameState
}

func TestDebugInventoryLockerProvisionAndAdditiveCaps(t *testing.T) {
	gameState := newDebugInventoryLockerTestState(t)
	changes, err := gameState.SetDebugInventoryProvision(DebugInventoryProvisionGold, ^uint16(0))
	if err != nil || len(changes) != 1 || changes[0].After != party_state.ItemQuantityLargeMax {
		t.Fatalf("SetDebugInventoryProvision() = (%+v, %v)", changes, err)
	}
	if _, err := gameState.SetDebugInventoryProvision(DebugInventoryProvision("bad"), 1); err == nil {
		t.Fatal("invalid provision succeeded")
	}

	gameState.PartyState.Inventory.Potions.Set(references.Blue, 98)
	changes, err = gameState.AddDebugInventoryPotion(references.Blue, 10)
	if err != nil || len(changes) != 1 || changes[0].After != party_state.ItemQuantitySmallMax {
		t.Fatalf("AddDebugInventoryPotion() = (%+v, %v)", changes, err)
	}
	gameState.PartyState.Inventory.Scrolls.Set(references.ScrollVasLor, 9990)
	changes, err = gameState.AddDebugInventoryScroll(references.ScrollVasLor, ^uint16(0))
	if err != nil || len(changes) != 1 || changes[0].After != party_state.ItemQuantityLargeMax {
		t.Fatalf("AddDebugInventoryScroll() = (%+v, %v)", changes, err)
	}
	if _, err := gameState.AddDebugInventoryPotion(references.Blue, 0); err == nil {
		t.Fatal("zero potion addition succeeded")
	}
	if _, err := gameState.AddDebugInventoryScroll(references.Scroll(8), 1); err == nil {
		t.Fatal("invalid scroll addition succeeded")
	}
}

func TestDebugInventoryLockerExactReagentAndEquipmentSetters(t *testing.T) {
	gameState := newDebugInventoryLockerTestState(t)
	avatar := gameState.PartyState.GetAvatar()
	avatar.SetEquippedItem(references.SlotWeapon, references.LongSword)

	changes, err := gameState.SetDebugInventoryReagent(references.Garlic, ^uint16(0))
	if err != nil || len(changes) != 1 || changes[0].Before != 0 || changes[0].After != party_state.ItemQuantitySmallMax {
		t.Fatalf("SetDebugInventoryReagent() = (%+v, %v)", changes, err)
	}
	changes, err = gameState.SetDebugInventoryEquipment(references.LongSword, 7)
	if err != nil || len(changes) != 1 || changes[0].Before != 0 || changes[0].After != 7 {
		t.Fatalf("SetDebugInventoryEquipment() = (%+v, %v)", changes, err)
	}
	if avatar.GetEquippedItem(references.SlotWeapon) != references.LongSword {
		t.Fatal("equipment setter mutated the equipped loadout")
	}

	changes, err = gameState.SetDebugInventoryEquipment(references.LongSword, 7)
	if err != nil || len(changes) != 0 {
		t.Fatalf("no-op equipment set = (%+v, %v)", changes, err)
	}
	if _, err := gameState.SetDebugInventoryReagent(references.Reagent(8), 1); err == nil {
		t.Fatal("invalid reagent succeeded")
	}
	if _, err := gameState.SetDebugInventoryEquipment(references.NoEquipment, 1); err == nil {
		t.Fatal("invalid equipment succeeded")
	}
}

func TestClearDebugEquipmentStateClearsCarriedAndAllRosterLoadoutsOnce(t *testing.T) {
	gameState := newDebugInventoryLockerTestState(t)
	gameState.PartyState.Inventory.Equipment.Set(references.LongSword, 3)
	gameState.PartyState.Inventory.Equipment.Set(references.Arrows, 8)
	gameState.PartyState.ForEachCharacter(func(_ int, character *party_state.PlayerCharacter) {
		for _, slot := range debugEquipmentSlots() {
			character.SetEquippedItem(slot, references.NoEquipment)
		}
	})
	avatar := gameState.PartyState.GetAvatar()
	avatar.SetEquippedItem(references.SlotWeapon, references.LongSword)
	companion := gameState.PartyState.GetCharacter(7)
	companion.SetEquippedItem(references.SlotRing, references.RingProtection)

	changes := gameState.ClearDebugEquipmentState()
	if len(changes) != 4 {
		t.Fatalf("ClearDebugEquipmentState() returned %d changes, want 4: %+v", len(changes), changes)
	}
	if gameState.PartyState.Inventory.Equipment.Get(references.LongSword) != 0 ||
		gameState.PartyState.Inventory.Equipment.Get(references.Arrows) != 0 {
		t.Fatal("equipment clear retained carried equipment")
	}
	if avatar.GetEquippedItem(references.SlotWeapon) != references.NoEquipment ||
		companion.GetEquippedItem(references.SlotRing) != references.NoEquipment {
		t.Fatal("equipment clear retained a roster loadout item")
	}
	if changes := gameState.ClearDebugEquipmentState(); len(changes) != 0 {
		t.Fatalf("no-op clear = %+v", changes)
	}
}

func TestDebugInventoryLockerPresetsAreExact(t *testing.T) {
	gameState := newDebugInventoryLockerTestState(t)
	gameState.PartyState.Inventory.Equipment.Set(references.Arrows, 7)
	gameState.PartyState.Inventory.Equipment.Set(references.Quarrels, 8)
	changes, err := gameState.ApplyDebugInventoryPreset(DebugInventoryPresetEquipmentBowsNoAmmo5)
	if err != nil || len(changes) != 5 {
		t.Fatalf("bows preset = (%+v, %v)", changes, err)
	}
	snapshot := gameState.GetDebugInventorySnapshot()
	for _, equipment := range []references.Equipment{references.Bow, references.Crossbow, references.MagicBow} {
		if snapshot.Equipment[equipment] != 5 {
			t.Fatalf("bow equipment %d = %d, want 5", equipment, snapshot.Equipment[equipment])
		}
	}
	if snapshot.Equipment[references.Arrows] != 0 || snapshot.Equipment[references.Quarrels] != 0 {
		t.Fatal("bows preset retained ammunition")
	}
	changes, err = gameState.ApplyDebugInventoryPreset(DebugInventoryPresetEquipmentBowsNoAmmo5)
	if err != nil || len(changes) != 0 {
		t.Fatalf("idempotent bows preset = (%+v, %v)", changes, err)
	}

	if _, err := gameState.ApplyDebugInventoryPreset(DebugInventoryPresetReagentsAll20NoGarlic); err != nil {
		t.Fatalf("reagent preset error = %v", err)
	}
	snapshot = gameState.GetDebugInventorySnapshot()
	for reagent := references.SulfurAsh; reagent <= references.MandrakeRoot; reagent++ {
		want := uint16(20)
		if reagent == references.Garlic {
			want = 0
		}
		if snapshot.Reagents[reagent] != want {
			t.Fatalf("reagent %d = %d, want %d", reagent, snapshot.Reagents[reagent], want)
		}
	}
	if _, err := gameState.ApplyDebugInventoryPreset(DebugInventoryPresetScrollsAllMax); err != nil {
		t.Fatalf("scroll preset error = %v", err)
	}
	snapshot = gameState.GetDebugInventorySnapshot()
	for scroll := references.ScrollVasLor; scroll <= references.ScrollAnTym; scroll++ {
		if snapshot.Scrolls[scroll] != party_state.ItemQuantityLargeMax {
			t.Fatalf("scroll %d = %d, want max", scroll, snapshot.Scrolls[scroll])
		}
	}
	if _, err := gameState.ApplyDebugInventoryPreset(DebugInventoryPreset("bad")); err == nil {
		t.Fatal("invalid preset succeeded")
	}
}

func TestDebugInventoryLockerRemainingPresetScopes(t *testing.T) {
	gameState := newDebugInventoryLockerTestState(t)
	if _, err := gameState.ApplyDebugInventoryPreset(DebugInventoryPresetProvisionsMax); err != nil {
		t.Fatalf("provisions preset error = %v", err)
	}
	snapshot := gameState.GetDebugInventorySnapshot()
	if snapshot.Provisions[references.Gold] != party_state.ItemQuantityLargeMax || snapshot.Provisions[references.Food] != party_state.ItemQuantityLargeMax {
		t.Fatalf("large provisions not max: %+v", snapshot.Provisions)
	}
	for _, provision := range []references.Provision{references.Gem, references.Torches, references.Key, references.SkullKeys} {
		if snapshot.Provisions[provision] != party_state.ItemQuantitySmallMax {
			t.Fatalf("provision %d = %d, want 99", provision, snapshot.Provisions[provision])
		}
	}
	if _, err := gameState.ApplyDebugInventoryPreset(DebugInventoryPresetPotionsAll1); err != nil {
		t.Fatalf("potions all1 error = %v", err)
	}
	if _, err := gameState.ApplyDebugInventoryPreset(DebugInventoryPresetPotionsAll99); err != nil {
		t.Fatalf("potions all99 error = %v", err)
	}
	snapshot = gameState.GetDebugInventorySnapshot()
	for potion := references.Blue; potion <= references.White; potion++ {
		if snapshot.Potions[potion] != party_state.ItemQuantitySmallMax {
			t.Fatalf("potion %d = %d, want 99", potion, snapshot.Potions[potion])
		}
	}
	if _, err := gameState.ApplyDebugInventoryPreset(DebugInventoryPresetEquipmentAllEquippable5); err != nil {
		t.Fatalf("all equippable error = %v", err)
	}
	snapshot = gameState.GetDebugInventorySnapshot()
	for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
		want := uint16(0)
		if equipment.GetSlotType() != references.SlotNone {
			want = 5
		}
		if snapshot.Equipment[equipment] != want {
			t.Fatalf("equipment %d = %d, want %d", equipment, snapshot.Equipment[equipment], want)
		}
	}
	if _, err := gameState.ApplyDebugInventoryPreset(DebugInventoryPresetEquipmentWeapons5); err != nil {
		t.Fatalf("weapons error = %v", err)
	}
	if _, err := gameState.ApplyDebugInventoryPreset(DebugInventoryPresetEquipmentArmour5); err != nil {
		t.Fatalf("armour error = %v", err)
	}
}

func TestDebugInventoryEquipmentGrantPresetsAreAdditive(t *testing.T) {
	for _, test := range []struct {
		name    string
		preset  DebugInventoryEquipmentGrantPreset
		matches func(references.Equipment) bool
	}{
		{"all equippable", DebugInventoryEquipmentGrantPresetAllEquippable5, func(equipment references.Equipment) bool { return equipment.GetSlotType() != references.SlotNone }},
		{"weapons", DebugInventoryEquipmentGrantPresetWeapons5, func(equipment references.Equipment) bool { return equipment.IsWeapon() && !equipment.IsAmmo() }},
		{"armour", DebugInventoryEquipmentGrantPresetArmour5, func(equipment references.Equipment) bool { return equipment.IsArmour() }},
	} {
		t.Run(test.name, func(t *testing.T) {
			gameState := newDebugInventoryLockerTestState(t)
			changes, err := gameState.ApplyDebugInventoryEquipmentGrantPreset(test.preset)
			if err != nil {
				t.Fatalf("ApplyDebugInventoryEquipmentGrantPreset() error = %v", err)
			}
			snapshot := gameState.GetDebugInventorySnapshot()
			for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
				want := uint16(0)
				if test.matches(equipment) {
					want = 5
				}
				if snapshot.Equipment[equipment] != want {
					t.Fatalf("equipment %d = %d, want %d", equipment, snapshot.Equipment[equipment], want)
				}
			}
			if len(changes) == 0 {
				t.Fatal("grant reported no changes")
			}
		})
	}

	gameState := newDebugInventoryLockerTestState(t)
	gameState.PartyState.Inventory.Equipment.Set(references.Bow, 97)
	gameState.PartyState.Inventory.Equipment.Set(references.Arrows, 7)
	gameState.PartyState.Inventory.Equipment.Set(references.Quarrels, 8)
	changes, err := gameState.ApplyDebugInventoryEquipmentGrantPreset(DebugInventoryEquipmentGrantPresetBowsNoAmmo5)
	if err != nil || len(changes) != 5 {
		t.Fatalf("bows grant = (%+v, %v)", changes, err)
	}
	snapshot := gameState.GetDebugInventorySnapshot()
	if snapshot.Equipment[references.Bow] != party_state.ItemQuantitySmallMax || snapshot.Equipment[references.MagicBow] != 5 || snapshot.Equipment[references.Crossbow] != 5 {
		t.Fatalf("bow grant quantities = %+v", snapshot.Equipment)
	}
	if snapshot.Equipment[references.Arrows] != 0 || snapshot.Equipment[references.Quarrels] != 0 {
		t.Fatalf("bow grant ammo = arrows %d quarrels %d", snapshot.Equipment[references.Arrows], snapshot.Equipment[references.Quarrels])
	}
	gameState.PartyState.Inventory.Equipment.Set(references.MagicBow, party_state.ItemQuantitySmallMax)
	gameState.PartyState.Inventory.Equipment.Set(references.Crossbow, party_state.ItemQuantitySmallMax)
	changes, err = gameState.ApplyDebugInventoryEquipmentGrantPreset(DebugInventoryEquipmentGrantPresetBowsNoAmmo5)
	if err != nil || len(changes) != 0 {
		t.Fatalf("saturated bows grant = (%+v, %v)", changes, err)
	}
	if _, err := gameState.ApplyDebugInventoryEquipmentGrantPreset(DebugInventoryEquipmentGrantPreset("bad")); err == nil {
		t.Fatal("invalid equipment grant preset succeeded")
	}
}
