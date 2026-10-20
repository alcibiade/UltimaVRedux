package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestHMSCapeUseCommitTogglesGlobalPreferenceWithoutRNG(t *testing.T) {
	state, _ := newUseCommitTestState(t)
	state.SetRandomSeed(462)
	state.PartyState.Inventory.QuestItems.Set(references.HMSCape, 1)
	rngCalls := state.GetRNG().Calls()

	enable, review := state.BeginHMSCapeUse()
	require.NotNil(t, enable)
	assert.Equal(t, HMSCapeUseReview{Result: HMSCapeUseReady, Quantity: 1, Enabled: false}, review)
	assert.True(t, enable.Enables())
	assert.False(t, state.IsHMSCapeActivated(), "review and confirmation preparation must not mutate")
	assert.Equal(t, HMSCapeUseEnabled, state.CommitHMSCapeUse(enable))
	assert.True(t, state.IsHMSCapeActivated())

	disable, review := state.BeginHMSCapeUse()
	require.NotNil(t, disable)
	assert.True(t, review.Enabled)
	assert.False(t, disable.Enables())
	assert.Equal(t, HMSCapeUseDisabled, state.CommitHMSCapeUse(disable))
	assert.False(t, state.IsHMSCapeActivated())
	assert.Equal(t, rngCalls, state.GetRNG().Calls())
}

func TestHMSCapeUseCommitRejectsMissingOrStalePlansWithoutMutation(t *testing.T) {
	state, _ := newUseCommitTestState(t)
	commit, review := state.BeginHMSCapeUse()
	assert.Nil(t, commit)
	assert.Equal(t, HMSCapeUseNoPlans, review.Result)

	state.PartyState.Inventory.QuestItems.Set(references.HMSCape, 1)
	commit, _ = state.BeginHMSCapeUse()
	require.NotNil(t, commit)
	state.SetHMSCapeActivated(true)

	assert.Equal(t, HMSCapeUseStale, state.CommitHMSCapeUse(commit))
	assert.True(t, state.IsHMSCapeActivated())
}

func TestPotionUseCommitOwnsFixedSeedRandomnessAndEffect(t *testing.T) {
	seed := findNormalYellowPotionSeed(t)

	first := commitTestYellowPotion(t, seed)
	second := commitTestYellowPotion(t, seed)

	assert.Equal(t, first, second)
	assert.Equal(t, references.Yellow, first.finalPotion)
	assert.Equal(t, PotionEffectHealed, first.effect)
	assert.Greater(t, first.hp, uint16(10))
	assert.LessOrEqual(t, first.hp, uint16(40))
	assert.Equal(t, []TransientIconKind{TransientIconHealing}, first.icons)
}

func TestYellowPotionUsesOriginalD30AfterMislabellingRolls(t *testing.T) {
	seed := findNormalYellowPotionSeed(t)
	wantRNG := beradrng.New(seed)
	switch wantRNG.Intn(potionRandomEffectRollMax) {
	case 0:
		t.Fatal("normal yellow potion seed selected orange")
	case 1:
		require.Equal(t, int(references.Yellow), wantRNG.Intn(potionRandomPotionMax))
	}
	wantHeal := originalD30FromSourceValue(wantRNG.Intn(originalD30SourceValueCount))

	observation := commitTestYellowPotion(t, seed)

	assert.Equal(t, uint16(10+wantHeal), observation.hp)
	assert.Equal(t, wantRNG.Calls(), observation.rngCalls)
}

func TestPotionUseCommitRejectsStaleInventoryWithoutConsumptionOrRNG(t *testing.T) {
	state, callbacks := newUseCommitTestState(t)
	state.SetRandomSeed(77)
	state.PartyState.Inventory.Potions.Set(references.Yellow, 1)
	commit, review := state.BeginPotionUse(references.Yellow, UsageField)
	require.NotNil(t, commit)
	require.True(t, review.Ready)
	state.PartyState.Inventory.Potions.Set(references.Yellow, 2)

	outcome, committed := state.CommitPotionUse(commit)

	assert.False(t, committed)
	assert.Nil(t, outcome)
	assert.NotContains(t, callbacks.SoundEffectsPlayed, SoundPotionDrink)
	assert.Equal(t, uint16(2), state.PartyState.Inventory.Potions.Get(references.Yellow))
	actualNextRoll := state.RandomIntInRange(0, potionRandomEffectRollMax-1)

	baseline, _ := newUseCommitTestState(t)
	baseline.SetRandomSeed(77)
	expectedNextRoll := baseline.RandomIntInRange(0, potionRandomEffectRollMax-1)
	assert.Equal(t, expectedNextRoll, actualNextRoll)
}

func TestPotionUseCancellationAndFullHealPreserveOriginalRNGOrder(t *testing.T) {
	state, callbacks := newUseCommitTestState(t)
	state.SetRandomSeed(findNormalYellowPotionSeed(t))
	state.PartyState.Inventory.Potions.Set(references.Yellow, 2)
	target := state.PartyState.GetCharacter(0)
	target.SetMaxHp(100)
	target.SetHp(100)
	target.SetStatus(party_state.Good)

	cancelCommit, _ := state.BeginPotionUse(references.Yellow, UsageField)
	rngCallsBeforeCancel := state.GetRNG().Calls()
	cancelledOutcome, committed := state.CommitPotionUse(cancelCommit)
	require.True(t, committed)
	require.NotNil(t, cancelledOutcome)
	assert.True(t, state.CancelPotionEffect(cancelledOutcome))
	assert.Equal(t, 1, countMatchingSounds(callbacks.SoundEffectsPlayed, SoundPotionDrink),
		"paid use has a single consumption request even if its effect is cancelled")
	assert.Equal(t, uint16(100), target.GetCurrentHp(), "not resolving models character-selection cancellation")
	assert.Equal(t, rngCallsBeforeCancel, state.GetRNG().Calls())
	assert.Panics(t, func() { state.CommitPotionEffect(cancelledOutcome, 0) })

	state.SetRandomSeed(findNormalYellowPotionSeed(t))
	failureCommit, _ := state.BeginPotionUse(references.Yellow, UsageField)
	failureOutcome, committed := state.CommitPotionUse(failureCommit)
	require.True(t, committed)
	assert.Equal(t, PotionEffectHealed, state.CommitPotionEffect(failureOutcome, 0))
	assert.Equal(t, uint16(100), target.GetCurrentHp())
	assert.Equal(t, uint16(0), state.PartyState.Inventory.Potions.Get(references.Yellow))
	assert.Equal(t, 2, countMatchingSounds(callbacks.SoundEffectsPlayed, SoundPotionDrink))
}

func TestCarpetBoardingCommitRejectsStaleStateWithoutPartialEffects(t *testing.T) {
	state, callbacks := newUseCommitTestState(t)
	state.SetRandomSeed(77)
	state.PartyState.Inventory.QuestItems.Set(references.Carpet, 1)
	commit, review := state.BeginCarpetBoarding()
	require.NotNil(t, commit)
	require.Equal(t, CarpetBoardingReady, review.Result)
	state.PartyState.Inventory.QuestItems.Set(references.Carpet, 2)
	rngCallsBefore := state.GetRNG().Calls()

	result := state.CommitCarpetBoarding(commit)

	assert.Equal(t, CarpetBoardingStale, result)
	assert.Equal(t, uint16(2), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
	assert.Equal(t, references.NoPartyVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
	assert.Empty(t, callbacks.TimeAdvanced())
	assert.Equal(t, rngCallsBefore, state.GetRNG().Calls())
}

func TestCarpetBoardingCommitConsumesPlacesBoardsAndAdvancesTime(t *testing.T) {
	state, callbacks := newUseCommitTestState(t)
	state.SetRandomSeed(77)
	expectedRoll := state.RandomIntInRange(0, 1)
	state.SetRandomSeed(77)
	state.PartyState.Inventory.QuestItems.Set(references.Carpet, 1)
	commit, _ := state.BeginCarpetBoarding()
	require.NotNil(t, commit)

	result := state.CommitCarpetBoarding(commit)

	assert.Equal(t, CarpetBoardingSuccess, result)
	assert.Zero(t, state.PartyState.Inventory.QuestItems.Get(references.Carpet))
	assert.Equal(t, references.CarpetVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
	assert.False(t, state.PartyVehicle.GetVehicleDetails().IsOwnedByNPC)
	expectedDirection := references.Right
	if expectedRoll == 1 {
		expectedDirection = references.Left
	}
	assert.Equal(t, expectedDirection, state.PartyVehicle.GetVehicleDetails().CurrentDirection())
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
	assert.Nil(t, state.CurrentNPCAIController.GetNpcs().GetVehicleAtPositionAndFloorOrNil(
		state.MapState.PlayerLocation.Position,
		state.MapState.PlayerLocation.Floor,
	))
	assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
}

func TestCarpetBoardingReviewRejectsOccupiedTileWithoutConsumption(t *testing.T) {
	state, callbacks := newUseCommitTestState(t)
	state.PartyState.Inventory.QuestItems.Set(references.Carpet, 1)
	location := state.MapState.PlayerLocation
	vehicleReference := *references.NewNPCReferenceForVehicle(
		references.HorseVehicle,
		location.Position,
		location.Floor,
	)
	vehicleReference.Schedule.OverrideAllPositions(byte(location.Position.X), byte(location.Position.Y))
	horse := *map_units.NewNPCFriendlyVehicle(references.HorseVehicle, vehicleReference)
	require.True(t, state.CurrentNPCAIController.GetNpcs().AddVehicle(horse))

	commit, review := state.BeginCarpetBoarding()

	assert.Nil(t, commit)
	assert.Equal(t, CarpetBoardingOccupied, review.Result)
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
	assert.Equal(t, references.NoPartyVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestCarpetBoardingUnderworldConsumesExactlyOneOnCurrentFloor(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(references.FloorNumber(references.UNDERWORLD)).
		WithPlayerAt(100, 100).
		WithRandomSeed(77).
		WithSystemCallbacks().
		Build()
	location := state.MapState.PlayerLocation
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapOverrideLayer, &location.Position, indexes.Grass)
	state.PartyState.Inventory.QuestItems.Set(references.Carpet, 2)
	state.SetRandomSeed(77)
	expectedDirection := references.Right
	if state.RandomIntInRange(0, 1) == 1 {
		expectedDirection = references.Left
	}
	state.SetRandomSeed(77)

	commit, review := state.BeginCarpetBoarding()

	require.NotNil(t, commit)
	require.Equal(t, CarpetBoardingReady, review.Result)
	assert.Equal(t, uint16(2), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
	assert.Zero(t, state.GetRNG().Calls(), "review is read-only")
	assert.Equal(t, CarpetBoardingSuccess, state.CommitCarpetBoardingForSurfaceUse(commit))
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
	assert.Equal(t, location, state.MapState.PlayerLocation)
	assert.Equal(t, references.CarpetVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
	assert.Equal(t, expectedDirection, state.PartyVehicle.GetVehicleDetails().CurrentDirection())
	assert.Equal(t, location.Floor, state.PartyVehicle.Floor())
	assert.Nil(t, state.CurrentNPCAIController.GetNpcs().GetVehicleAtPositionAndFloorOrNil(location.Position, location.Floor))
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
	assert.Empty(t, callbacks.TimeAdvanced(), "owning Use workflow completes the normal cycle")
}

func TestSkullKeyUseAndUnlockCommitsPreserveConsumptionOrder(t *testing.T) {
	state, callbacks := newUseCommitTestState(t)
	state.PartyState.Inventory.Provisions.SkullKeys.Set(1)
	target := state.MapState.PlayerLocation.Position.Up()
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapUnitLayer, &target, 0)
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapOverrideLayer, &target, indexes.MagicLockDoor)
	commit, review := state.BeginSkullKeyUse()
	require.NotNil(t, commit)
	require.Equal(t, SkullKeyUseReady, review.Result)

	assert.Equal(t, SkullKeyUseConsumed, state.CommitSkullKeyUse(commit))
	assert.Zero(t, state.PartyState.Inventory.Provisions.SkullKeys.Get())
	assert.Empty(t, callbacks.TimeAdvanced(), "consumption precedes the directional unlock")

	unlock := state.ReviewSkullKeyUnlock(commit, references.Up)
	require.Equal(t, SkullKeyUnlockReady, unlock.Result)
	assert.Equal(t, SkullKeyUnlockSuccess, state.CommitSkullKeyUnlock(unlock))
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor),
		state.GetLayeredMapByCurrentLocation().GetTopTile(&target).Index)
	assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
}

func TestSkullKeyCommitsRejectStaleAndNoEffectWithoutPartialMutation(t *testing.T) {
	state, callbacks := newUseCommitTestState(t)
	state.PartyState.Inventory.Provisions.SkullKeys.Set(1)
	commit, _ := state.BeginSkullKeyUse()
	require.NotNil(t, commit)
	state.PartyState.Inventory.Provisions.SkullKeys.Set(2)

	assert.Equal(t, SkullKeyUseStale, state.CommitSkullKeyUse(commit))
	assert.Equal(t, uint16(2), state.PartyState.Inventory.Provisions.SkullKeys.Get())
	assert.Empty(t, callbacks.TimeAdvanced())

	target := state.MapState.PlayerLocation.Position.Up()
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapUnitLayer, &target, 0)
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapOverrideLayer, &target, indexes.Grass)
	state.PartyState.Inventory.Provisions.SkullKeys.Set(1)
	commit, _ = state.BeginSkullKeyUse()
	require.Equal(t, SkullKeyUseConsumed, state.CommitSkullKeyUse(commit))
	unlock := state.ReviewSkullKeyUnlock(commit, references.Up)
	assert.Equal(t, SkullKeyUnlockNoEffect, unlock.Result)
	assert.Equal(t, SkullKeyUnlockNoEffect, state.CommitSkullKeyUnlock(unlock))
	assert.Equal(t, indexes.SpriteIndex(indexes.Grass),
		state.GetLayeredMapByCurrentLocation().GetTopTile(&target).Index)
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestSkullKeyUnlockCommitRejectsStaleDoorWithoutMutation(t *testing.T) {
	state, callbacks := newUseCommitTestState(t)
	state.PartyState.Inventory.Provisions.SkullKeys.Set(1)
	commit, _ := state.BeginSkullKeyUse()
	require.Equal(t, SkullKeyUseConsumed, state.CommitSkullKeyUse(commit))
	target := state.MapState.PlayerLocation.Position.Up()
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapUnitLayer, &target, 0)
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapOverrideLayer, &target, indexes.MagicLockDoor)
	review := state.ReviewSkullKeyUnlock(commit, references.Up)
	require.Equal(t, SkullKeyUnlockReady, review.Result)
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapOverrideLayer, &target, indexes.LockedDoorView)

	assert.Equal(t, SkullKeyUnlockStale, state.CommitSkullKeyUnlock(review))
	assert.Equal(t, indexes.SpriteIndex(indexes.LockedDoorView),
		state.GetLayeredMapByCurrentLocation().GetTopTile(&target).Index)
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestSkullKeyUnlockRequiresTheConsumedUseCommit(t *testing.T) {
	state, callbacks := newUseCommitTestState(t)
	target := state.MapState.PlayerLocation.Position.Up()
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapUnitLayer, &target, 0)
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapOverrideLayer, &target, indexes.MagicLockDoor)

	review := state.ReviewSkullKeyUnlock(nil, references.Up)

	assert.Equal(t, SkullKeyUnlockNoEffect, review.Result)
	assert.Equal(t, SkullKeyUnlockNoEffect, state.CommitSkullKeyUnlock(review))
	assert.Equal(t, indexes.SpriteIndex(indexes.MagicLockDoor),
		state.GetLayeredMapByCurrentLocation().GetTopTile(&target).Index)
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestSkullKeyDoesNotOpenOrdinaryLockedDoor(t *testing.T) {
	state, callbacks := newUseCommitTestState(t)
	state.PartyState.Inventory.Provisions.SkullKeys.Set(1)
	commit, _ := state.BeginSkullKeyUse()
	require.Equal(t, SkullKeyUseConsumed, state.CommitSkullKeyUse(commit))
	target := state.MapState.PlayerLocation.Position.Up()
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapUnitLayer, &target, 0)
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapOverrideLayer, &target, indexes.LockedDoor)

	review := state.ReviewSkullKeyUnlock(commit, references.Up)

	assert.Equal(t, SkullKeyUnlockNoEffect, review.Result)
	assert.Equal(t, SkullKeyUnlockNoEffect, state.CommitSkullKeyUnlock(review))
	assert.Equal(t, indexes.SpriteIndex(indexes.LockedDoor),
		state.GetLayeredMapByCurrentLocation().GetTopTile(&target).Index)
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapOverrideLayer, &target, indexes.MagicLockDoor)
	secondReview := state.ReviewSkullKeyUnlock(commit, references.Up)
	assert.Equal(t, SkullKeyUnlockNoEffect, secondReview.Result)
	assert.Equal(t, SkullKeyUnlockNoEffect, state.CommitSkullKeyUnlock(secondReview))
	assert.Equal(t, indexes.SpriteIndex(indexes.MagicLockDoor),
		state.GetLayeredMapByCurrentLocation().GetTopTile(&target).Index)
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestSkullKeyUseInvalidLocationPreservesInventory(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Deceit).
		WithSystemCallbacks().
		Build()
	state.PartyState.Inventory.Provisions.SkullKeys.Set(1)
	commit, review := state.BeginSkullKeyUse()
	require.NotNil(t, commit)
	require.Equal(t, SkullKeyUseNotHere, review.Result)

	assert.Equal(t, SkullKeyUseNotHere, state.CommitSkullKeyUse(commit))
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Provisions.SkullKeys.Get())
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestUseProjectionSharesCarpetAndSkullKeyAuthorityReviews(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Deceit).
		WithSystemCallbacks().
		Build()
	state.PartyState.Inventory.QuestItems.Set(references.Carpet, 1)
	state.PartyState.Inventory.Provisions.SkullKeys.Set(1)

	assert.Equal(t, CarpetBoardingNotHere, state.ReviewCarpetBoarding().Result)
	assert.Equal(t, SkullKeyUseNotHere, state.ReviewSkullKeyUse().Result)
	items := state.GetUsableItemsForContext(UsageField)
	assert.False(t, useCommitTestItem(t, items, references.Carpet).AvailableNow)
	assert.False(t, useCommitTestItem(t, items, references.SkullKeys).AvailableNow)
}

type potionCommitObservation struct {
	finalPotion references.Potion
	effect      PotionEffectResult
	hp          uint16
	rngCalls    uint64
	icons       []TransientIconKind
}

func commitTestYellowPotion(t *testing.T, seed uint64) potionCommitObservation {
	t.Helper()
	state, callbacks := newUseCommitTestState(t)
	state.SetRandomSeed(seed)
	state.PartyState.Inventory.Potions.Set(references.Yellow, 1)
	target := state.PartyState.GetCharacter(0)
	target.SetMaxHp(100)
	target.SetHp(10)
	commit, _ := state.BeginPotionUse(references.Yellow, UsageField)
	outcome, committed := state.CommitPotionUse(commit)
	require.True(t, committed)
	effect := state.CommitPotionEffect(outcome, 0)

	return potionCommitObservation{
		finalPotion: outcome.FinalPotion,
		effect:      effect,
		hp:          target.GetCurrentHp(),
		rngCalls:    state.GetRNG().Calls(),
		icons:       append([]TransientIconKind(nil), callbacks.TransientIconCalls...),
	}
}

func findNormalYellowPotionSeed(t *testing.T) uint64 {
	t.Helper()
	for seed := uint64(0); seed < 1000; seed++ {
		state, _ := newUseCommitTestState(t)
		state.SetRandomSeed(seed)
		state.PartyState.Inventory.Potions.Set(references.Yellow, 1)
		commit, _ := state.BeginPotionUse(references.Yellow, UsageField)
		outcome, committed := state.CommitPotionUse(commit)
		if committed {
			state.CommitPotionEffect(outcome, 0)
		}
		if committed && outcome.FinalPotion == references.Yellow {
			return seed
		}
	}
	t.Fatal("no deterministic normal Yellow potion seed found")

	return 0
}

func newUseCommitTestState(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(12, 12).
		WithSystemCallbacks().
		Build()
	position := state.MapState.PlayerLocation.Position
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapUnitLayer, &position, 0)
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.Grass)

	return state, callbacks
}

func useCommitTestItem(t *testing.T, items []UseableItemInfo, target references.Item) UseableItemInfo {
	t.Helper()
	for _, item := range items {
		if item.ItemType == target {
			return item
		}
	}
	t.Fatalf("Use item %v was not projected", target)

	return UseableItemInfo{}
}
