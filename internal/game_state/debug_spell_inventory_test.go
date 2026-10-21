package game_state

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestDebugSpellInventorySetAndMaximumPreset(t *testing.T) {
	gameState := newDebugInventoryLockerTestState(t)
	gameState.PartyState.Inventory.Spells.Set(references.InLor, 0)
	gameState.PartyState.Inventory.Spells.Set(references.Nox, 7)

	changes, err := gameState.SetDebugSpellQuantity(references.InLor, 5)
	if err != nil || len(changes) != 1 || changes[0].Before != 0 || changes[0].After != 5 {
		t.Fatalf("SetDebugSpellQuantity() = (%+v, %v)", changes, err)
	}
	changes, err = gameState.SetDebugSpellQuantity(references.InLor, 5)
	if err != nil || len(changes) != 0 {
		t.Fatalf("no-op spell set = (%+v, %v)", changes, err)
	}
	if _, err := gameState.SetDebugSpellQuantity(references.Nox, 1); err == nil {
		t.Fatal("Nox spell set succeeded")
	}

	changes = gameState.SetAllDebugSpellQuantitiesToMaximum()
	if len(changes) != references.TotalSpellCount {
		t.Fatalf("maximum preset = %d changes", len(changes))
	}
	for index, change := range changes {
		if change.Path != fmt.Sprintf("spells.%d", index) || change.After != 99 {
			t.Fatalf("unexpected spell change: %+v", change)
		}
	}
	snapshot := gameState.GetDebugSpellInventorySnapshot()
	if len(snapshot) != references.TotalSpellCount {
		t.Fatalf("spell snapshot count = %d", len(snapshot))
	}
	for spell := references.InLor; spell <= references.AnTym; spell++ {
		if snapshot[spell] != 99 {
			t.Fatalf("spell %d = %d, want 99", spell, snapshot[spell])
		}
	}
	if gameState.PartyState.Inventory.Spells.Get(references.Nox) != 7 {
		t.Fatal("maximum preset changed Nox")
	}
	if changes := gameState.SetAllDebugSpellQuantitiesToMaximum(); len(changes) != 0 {
		t.Fatalf("idempotent maximum preset = %d changes", len(changes))
	}
}
