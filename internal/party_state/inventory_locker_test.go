package party_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestInventoryLockerSnapshotIsCompleteDetachedAndDoesNotInsert(t *testing.T) {
	inventory := NewInventory()
	if len(inventory.Reagent.quantities) != 0 || len(inventory.Potions.quantities) != 0 || len(inventory.Scrolls.quantities) != 0 || len(inventory.Equipment.quantities) != 0 || len(inventory.Spells.quantities) != 0 {
		t.Fatal("new inventory quantity maps must be empty")
	}

	snapshot := inventory.LockerSnapshot()
	if len(snapshot.Provisions) != 6 || len(snapshot.Reagents) != 8 || len(snapshot.Potions) != 8 || len(snapshot.Scrolls) != 8 || len(snapshot.Equipment) != 48 {
		t.Fatalf("incomplete locker snapshot: %+v", snapshot)
	}
	if len(inventory.Reagent.quantities) != 0 || len(inventory.Potions.quantities) != 0 || len(inventory.Scrolls.quantities) != 0 || len(inventory.Equipment.quantities) != 0 || len(inventory.Spells.quantities) != 0 {
		t.Fatal("locker snapshot inserted missing inventory entries")
	}
	for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
		if snapshot.Equipment[equipment] != 0 {
			t.Fatalf("equipment %d = %d, want zero", equipment, snapshot.Equipment[equipment])
		}
	}
	for reagent := references.SulfurAsh; reagent <= references.MandrakeRoot; reagent++ {
		if snapshot.Reagents[reagent] != 0 {
			t.Fatalf("reagent %d = %d, want zero", reagent, snapshot.Reagents[reagent])
		}
	}
	for potion := references.Blue; potion <= references.White; potion++ {
		if snapshot.Potions[potion] != 0 {
			t.Fatalf("potion %d = %d, want zero", potion, snapshot.Potions[potion])
		}
	}
	for scroll := references.ScrollVasLor; scroll <= references.ScrollAnTym; scroll++ {
		if snapshot.Scrolls[scroll] != 0 {
			t.Fatalf("scroll %d = %d, want zero", scroll, snapshot.Scrolls[scroll])
		}
	}
	snapshot.Potions[references.Blue] = 9
	if inventory.Potions.Get(references.Blue) != 0 || inventory.Potions.HasSome(references.Blue) {
		t.Fatal("mutating snapshot changed inventory")
	}
	if len(inventory.Potions.quantities) != 0 {
		t.Fatal("inventory reads inserted a missing quantity")
	}
}

func TestSpellSnapshotIsCompleteDetachedAndDoesNotInsert(t *testing.T) {
	inventory := NewInventory()
	snapshot := inventory.SpellSnapshot()
	if len(snapshot) != references.TotalSpellCount || len(inventory.Spells.quantities) != 0 {
		t.Fatalf("spell snapshot size %d inserted %d entries", len(snapshot), len(inventory.Spells.quantities))
	}
	for spell := references.InLor; spell <= references.AnTym; spell++ {
		if snapshot[spell] != 0 {
			t.Fatalf("spell %d = %d, want zero", spell, snapshot[spell])
		}
	}
	if _, ok := snapshot[references.Nox]; ok {
		t.Fatal("spell snapshot included non-castable Nox")
	}
	snapshot[references.InLor] = 9
	if inventory.Spells.Get(references.InLor) != 0 || len(inventory.Spells.quantities) != 0 {
		t.Fatal("mutating spell snapshot changed inventory")
	}
}

func TestItemQuantityIncrementCapsWithoutOverflow(t *testing.T) {
	large := ItemQuantityLarge{}
	large.Set(9990)
	large.IncrementBy(^uint16(0))
	if got := large.Get(); got != ItemQuantityLargeMax {
		t.Fatalf("large quantity = %d, want %d", got, ItemQuantityLargeMax)
	}

	small := ItemQuantitySmall{}
	small.Set(90)
	small.IncrementBy(^uint16(0))
	if got := small.Get(); got != ItemQuantitySmallMax {
		t.Fatalf("small quantity = %d, want %d", got, ItemQuantitySmallMax)
	}
}
