package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestAddDebugSearchProvisionAppendsAndMergesWithoutOverwrite(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	position := references.Position{X: 10, Y: 10}

	if err := gameState.ItemStacksMap.Push(&position, &references.ItemStack{Items: references.ItemStacks{
		{Item: references.Gem, Quantity: 3},
		{Item: references.Gold, Quantity: 5},
	}}); err != nil {
		t.Fatalf("Push() error = %v", err)
	}
	if err := gameState.AddDebugSearchProvision(position, references.Gold, 7); err != nil {
		t.Fatalf("AddDebugSearchProvision(gold) error = %v", err)
	}
	if err := gameState.AddDebugSearchProvision(position, references.Food, 2); err != nil {
		t.Fatalf("AddDebugSearchProvision(food) error = %v", err)
	}

	stack, ok := gameState.ItemStacksMap.Snapshot(&position)
	if !ok {
		t.Fatal("stack missing after additions")
	}
	if len(stack.Items) != 3 {
		t.Fatalf("stack item count = %d, want 3", len(stack.Items))
	}
	if stack.Items[0].Item != references.Gem || stack.Items[0].Quantity != 3 {
		t.Fatalf("existing incompatible item was overwritten: %+v", stack.Items[0])
	}
	if stack.Items[1].Item != references.Gold || stack.Items[1].Quantity != 12 {
		t.Fatalf("gold was not merged: %+v", stack.Items[1])
	}
	if stack.Items[2].Item != references.Food || stack.Items[2].Quantity != 2 {
		t.Fatalf("food was not appended: %+v", stack.Items[2])
	}
	for index, provision := range []references.Provision{
		references.Gold,
		references.Torches,
		references.Key,
		references.Food,
	} {
		allowedPosition := references.Position{X: references.Coordinate(index), Y: 1}
		if err := gameState.AddDebugSearchProvision(allowedPosition, provision, 1); err != nil {
			t.Fatalf("supported provision %d error = %v", provision, err)
		}
	}

	if err := gameState.AddDebugSearchProvision(references.Position{X: -1, Y: 0}, references.Key, 1); err == nil {
		t.Fatal("out-of-bounds addition succeeded")
	}
	if err := gameState.AddDebugSearchProvision(position, references.Gem, 1); err == nil {
		t.Fatal("unsupported provision addition succeeded")
	}
}

func TestResetAuthenticSearchFoundFlagsHasExactScope(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	manager := gameState.SearchManager.authenticLocationManager
	manager.markAsFound(16)
	manager.markAsFound(18)
	manager.markAsFound(13)
	manager.markAsFound(14)
	manager.markAsFound(15)
	manager.skullKeysFoundDay = 9
	position := references.Position{X: 10, Y: 10}
	if err := gameState.ItemStacksMap.Push(&position, &references.ItemStack{Items: references.ItemStacks{
		{Item: references.Gold, Quantity: 4},
	}}); err != nil {
		t.Fatalf("Push() error = %v", err)
	}

	before, after := gameState.ResetAuthenticSearchFoundFlags()
	if before != 2 || after != 0 {
		t.Fatalf("ResetAuthenticSearchFoundFlags() = (%d, %d), want (2, 0)", before, after)
	}
	if manager.isAlreadyFound(16) || manager.isAlreadyFound(18) {
		t.Fatal("one-time search flags were not reset")
	}
	for _, specialIndex := range []int{13, 14, 15} {
		if !manager.isAlreadyFound(specialIndex) {
			t.Fatalf("special search flag %d was reset", specialIndex)
		}
	}
	if manager.skullKeysFoundDay != 9 {
		t.Fatalf("skull key day = %d, want 9", manager.skullKeysFoundDay)
	}
	if _, ok := gameState.ItemStacksMap.Snapshot(&position); !ok {
		t.Fatal("reset removed an existing item stack")
	}
}

func TestSetQuestItemPresentSupportsEveryQuestItem(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	refreshCount := 0
	gameState.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshCount++ }

	for item := references.Amulet; item < references.NumberOfQuestItems; item++ {
		gameState.PartyState.Inventory.QuestItems.Set(item, 0)
		before, after, err := gameState.SetQuestItemPresent(item, true)
		if err != nil || before != 0 || after != 1 {
			t.Fatalf("SetQuestItemPresent(%d, true) = (%d, %d, %v), want (0, 1, nil)", item, before, after, err)
		}
		gameState.PartyState.Inventory.QuestItems.Set(item, 3)
		before, after, err = gameState.SetQuestItemPresent(item, true)
		if err != nil || before != 3 || after != 3 {
			t.Fatalf("ensure existing quest item %d = (%d, %d, %v), want (3, 3, nil)", item, before, after, err)
		}
		before, after, err = gameState.SetQuestItemPresent(item, false)
		if err != nil || before != 3 || after != 0 {
			t.Fatalf("SetQuestItemPresent(%d, false) = (%d, %d, %v), want (3, 0, nil)", item, before, after, err)
		}
	}

	if refreshCount != references.NumberOfQuestItems*2 {
		t.Fatalf("inventory refresh count = %d, want %d", refreshCount, references.NumberOfQuestItems*2)
	}
	if _, _, err := gameState.SetQuestItemPresent(references.QuestItem(99), true); err == nil {
		t.Fatal("invalid quest item alteration succeeded")
	}
}
