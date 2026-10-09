package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestToggleItemType_FriendlyName(t *testing.T) {
	tests := []struct {
		itemType ToggleItemType
		expected string
	}{
		{ToggleItemAmulet, "Amulet"},
		{ToggleItemCrown, "Crown"},
		{ToggleItemBadge, "Badge"},
		{ToggleItemType(0), "Item"}, // Unknown type
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.itemType.FriendlyName())
		})
	}
}

func TestToggleItem_Activation(t *testing.T) {
	gs := createMinimalGameStateForToggleTests(t)

	// Initially inactive
	assert.False(t, gs.IsAmuletActive())
	assert.False(t, gs.IsCrownActive())
	assert.False(t, gs.IsBadgeActive())

	// Activate Amulet
	result := gs.ToggleAmulet()
	assert.True(t, result, "ToggleAmulet should return true")
	assert.True(t, gs.IsAmuletActive())
	assert.Equal(t, ToggleItemDuration, gs.AmuletState.TurnsRemaining)

	// Other items still inactive
	assert.False(t, gs.IsCrownActive())
	assert.False(t, gs.IsBadgeActive())
}

func TestToggleItem_Deactivation(t *testing.T) {
	gs := createMinimalGameStateForToggleTests(t)

	// Activate then deactivate
	gs.ToggleAmulet()
	assert.True(t, gs.IsAmuletActive())

	result := gs.ToggleAmulet()
	assert.True(t, result, "ToggleAmulet (deactivate) should return true")
	assert.False(t, gs.IsAmuletActive())
	assert.Equal(t, 0, gs.AmuletState.TurnsRemaining)
}

func TestToggleItem_MultipleConcurrent(t *testing.T) {
	gs := createMinimalGameStateForToggleTests(t)

	// Activate all three
	gs.ToggleAmulet()
	gs.ToggleCrown()
	gs.ToggleBadge()

	// All should be active
	assert.True(t, gs.IsAmuletActive())
	assert.True(t, gs.IsCrownActive())
	assert.True(t, gs.IsBadgeActive())

	// Deactivate one - others remain active
	gs.ToggleCrown()
	assert.True(t, gs.IsAmuletActive())
	assert.False(t, gs.IsCrownActive())
	assert.True(t, gs.IsBadgeActive())
}

func TestToggleBadgePreservesIndependentEffects(t *testing.T) {
	gs := createMinimalGameStateForToggleTests(t)
	gs.AmuletState = ToggleItemState{IsActive: true, TurnsRemaining: 77}
	gs.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 9}

	require.True(t, gs.ToggleBadge())
	assert.Equal(t, ToggleItemState{IsActive: true, TurnsRemaining: 77}, gs.AmuletState)
	assert.Equal(t, ToggleItemState{}, gs.CrownState)
	assert.Equal(t, DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 9}, gs.ActiveDurationEffect)
	assert.Equal(t, ToggleItemState{IsActive: true, TurnsRemaining: ToggleItemDuration}, gs.BadgeState)

	require.True(t, gs.ToggleBadge())
	assert.Equal(t, ToggleItemState{IsActive: true, TurnsRemaining: 77}, gs.AmuletState)
	assert.Equal(t, ToggleItemState{}, gs.CrownState)
	assert.Equal(t, DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 9}, gs.ActiveDurationEffect)
	assert.Equal(t, ToggleItemState{}, gs.BadgeState)
}

func TestCrownCompetesWithEveryDurationSpell(t *testing.T) {
	for _, test := range []struct {
		name     string
		effect   DurationEffectType
		activate func(*GameState, int) bool
	}{
		{"Protection", DurationEffectProtection, (*GameState).ActivateProtectionEffect},
		{"Quickness", DurationEffectQuickness, (*GameState).ActivateQuicknessEffect},
		{"Confusion", DurationEffectConfusion, (*GameState).ActivateConfusionEffect},
		{"NegateMagic", DurationEffectNegateMagic, (*GameState).ActivateNegateMagicEffect},
		{"TimeStop", DurationEffectTimeStop, (*GameState).ActivateTimeStopEffect},
	} {
		t.Run(test.name, func(t *testing.T) {
			gs := newPersistentEffectsTestState(t)
			gs.MagicLightMinutesRemaining = 100
			gs.MapState.Lighting.SetMagicLight(true)
			gs.ToggleBadge()
			gs.ToggleAmulet()
			badge, amulet := gs.BadgeState, gs.AmuletState
			gs.ToggleCrown()
			assert.Equal(t, PersistentEffectCrownSuppression, gs.GetPersistentEffectStatuses()[1].ID)
			assert.True(t, gs.GetWornArtifactStatuses()[0].Worn)

			require.True(t, test.activate(gs, 9))
			assert.Equal(t, ToggleItemState{}, gs.CrownState, "spell removes Crown and its timer")
			assert.Equal(t, DurationEffect{EffectType: test.effect, TurnsRemaining: 9}, gs.ActiveDurationEffect)
			assert.Equal(t, badge, gs.BadgeState)
			assert.Equal(t, amulet, gs.AmuletState)
			assert.Equal(t, persistentDurationEffectID(test.effect), gs.GetPersistentEffectStatuses()[1].ID)
			assert.False(t, gs.GetWornArtifactStatuses()[0].Worn)

			gs.processDurationEffects()
			require.True(t, test.activate(gs, 3))
			assert.Equal(t, DurationEffect{EffectType: test.effect, TurnsRemaining: 3}, gs.ActiveDurationEffect,
				"same spell refreshes rather than toggling off")
			for range 3 {
				gs.processDurationEffects()
			}
			assert.Equal(t, DurationEffect{}, gs.ActiveDurationEffect)
			assert.Equal(t, ToggleItemState{}, gs.CrownState, "expired spell never resumes Crown")
			assert.Equal(t, []PersistentEffectStatus{{ID: PersistentEffectMagicLight, Atmosphere: EffectAtmosphereSteady}},
				gs.GetPersistentEffectStatuses())

			test.activate(gs, 9)
			gs.ToggleCrown()
			assert.Equal(t, DurationEffect{}, gs.ActiveDurationEffect, "Crown replaces the spell")
			assert.Equal(t, ToggleItemState{IsActive: true, TurnsRemaining: ToggleItemDuration}, gs.CrownState)
			gs.ToggleCrown()
			assert.Equal(t, DurationEffect{}, gs.ActiveDurationEffect, "removing Crown never resumes the spell")
			assert.Equal(t, ToggleItemState{}, gs.CrownState)
			assert.Equal(t, badge, gs.BadgeState)
			assert.Equal(t, amulet, gs.AmuletState)
			assert.Equal(t, 100, gs.MagicLightMinutesRemaining)
			assert.True(t, gs.MapState.Lighting.HasMagicLight())
		})
	}
}

func TestIndependentArtifactTogglesPreserveSpell(t *testing.T) {
	gs := createMinimalGameStateForToggleTests(t)
	gs.ActivateProtectionEffect(9)
	want := gs.ActiveDurationEffect
	for range 2 {
		gs.ToggleAmulet()
		gs.ToggleBadge()
		assert.Equal(t, want, gs.ActiveDurationEffect)
		assert.False(t, gs.IsCrownActive())
	}
	gs.processToggleItems()
	assert.Equal(t, want, gs.ActiveDurationEffect, "inactive Crown processing must not clear a spell")
}

func TestDebugDurationActivationReplacesCrownButLightDoesNot(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	gs.ToggleCrown()
	_, err := gs.ActivateDebugDurationEffect(DurationEffectLight, "", 0, 0)
	require.NoError(t, err)
	assert.True(t, gs.IsCrownActive())
	_, err = gs.ActivateDebugDurationEffect(DurationEffectTimeStop, "", 0, gs.MagicLightMinutesRemaining)
	require.NoError(t, err)
	assert.Equal(t, ToggleItemState{}, gs.CrownState)
	assert.True(t, gs.HasActiveDurationEffect(DurationEffectTimeStop))
}

func TestToggleItem_PermanentNeverDecrements(t *testing.T) {
	gs := createMinimalGameStateForToggleTests(t)

	gs.ToggleAmulet()
	// Verify starts at permanent duration (255)
	assert.Equal(t, ToggleItemDuration, gs.AmuletState.TurnsRemaining)

	// Process multiple turns - should stay at 255 (permanent)
	for i := 0; i < 10; i++ {
		gs.processToggleItems()
	}

	assert.Equal(t, ToggleItemDuration, gs.AmuletState.TurnsRemaining)
	assert.True(t, gs.IsAmuletActive())
}

func TestToggleItem_NonPermanentDecrements(t *testing.T) {
	gs := createMinimalGameStateForToggleTests(t)

	gs.ToggleAmulet()
	// Set to a non-permanent value to test decrement
	gs.AmuletState.TurnsRemaining = 100

	// Process one turn
	gs.processToggleItems()

	assert.Equal(t, 99, gs.AmuletState.TurnsRemaining)
	assert.True(t, gs.IsAmuletActive())
}

func TestToggleItem_Expiration(t *testing.T) {
	gs := createMinimalGameStateForToggleTests(t)

	gs.ToggleAmulet()
	// Set to 1 turn remaining
	gs.AmuletState.TurnsRemaining = 1

	// Process - should expire
	gs.processToggleItems()

	assert.False(t, gs.IsAmuletActive())
	assert.Equal(t, 0, gs.AmuletState.TurnsRemaining)
}

func TestWornArtifactProjectionPreservesEveryOwnershipAndToggleCombination(t *testing.T) {
	items := []references.QuestItem{references.Crown, references.Amulet, references.BlackBadge}
	identities := []ToggleItemType{ToggleItemCrown, ToggleItemAmulet, ToggleItemBadge}
	for ownedMask := range 8 {
		for wornMask := range 8 {
			gs := createMinimalGameStateForToggleTests(t)
			states := []*ToggleItemState{&gs.CrownState, &gs.AmuletState, &gs.BadgeState}
			for index, item := range items {
				if ownedMask&(1<<index) != 0 {
					gs.PartyState.Inventory.QuestItems.Set(item, 1)
				}
				*states[index] = ToggleItemState{IsActive: wornMask&(1<<index) != 0, TurnsRemaining: ToggleItemDuration}
			}
			projection := gs.GetWornArtifactStatuses()
			for index, id := range identities {
				assert.Equal(t, WornArtifactStatus{ID: id, Owned: ownedMask&(1<<index) != 0, Worn: wornMask&(1<<index) != 0}, projection[index])
				assert.Equal(t, wornMask&(1<<index) != 0, states[index].IsActive, "renderer never repairs ownership")
				assert.Equal(t, ToggleItemDuration, states[index].TurnsRemaining)
			}
			projection[0] = WornArtifactStatus{}
			assert.Equal(t, ToggleItemCrown, gs.GetWornArtifactStatuses()[0].ID, "detached fixed array")
		}
	}
}

func TestWornArtifactProjectionKeepsSlotsAcrossRemovalAndInventoryLoss(t *testing.T) {
	gs := createMinimalGameStateForToggleTests(t)
	gs.ToggleBadge()
	gs.ToggleAmulet()
	gs.ToggleCrown()
	gs.PartyState.Inventory.QuestItems.Set(references.Crown, 1)
	gs.PartyState.Inventory.QuestItems.Set(references.Amulet, 1)
	gs.PartyState.Inventory.QuestItems.Set(references.BlackBadge, 1)
	want := gs.GetWornArtifactStatuses()
	gs.ToggleCrown()
	gs.PartyState.Inventory.QuestItems.Set(references.Amulet, 0)
	got := gs.GetWornArtifactStatuses()
	assert.Equal(t, WornArtifactStatus{ID: ToggleItemCrown, Owned: true}, got[0])
	assert.Equal(t, WornArtifactStatus{ID: ToggleItemAmulet, Worn: true}, got[1])
	assert.Equal(t, want[2], got[2])
}

// createMinimalGameStateForToggleTests creates a minimal GameState for testing toggle items.
func createMinimalGameStateForToggleTests(t *testing.T) *GameState {
	t.Helper()

	gs := &GameState{}
	gs.SystemCallbacks = createMockSystemCallbacksForToggleTests()

	return gs
}

// createMockSystemCallbacksForToggleTests creates mock callbacks for toggle item tests.
func createMockSystemCallbacksForToggleTests() *SystemCallbacks {
	return &SystemCallbacks{
		UI: UICallbacks{
			ShowActionFeedback: func(message string, important bool) {
				// No-op for tests
			},
			ShowSpellExpiredPopup: func(effectName string) {
				// No-op for tests
			},
		},
		Flow: FlowCallbacks{
			Items: FlowItemCallbacks{
				TriggerToggleItemFlash: func(itemType ToggleItemType) {
					// No-op for tests - flash effect is UI-only
				},
			},
		},
	}
}
