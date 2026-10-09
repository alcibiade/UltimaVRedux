package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// NOTE: CheckBridgeTrollEncounter tests require full GameState with PartyVehicle
// initialized, which is complex to set up in unit tests. These are covered by
// integration tests that use real game data. Unit tests here focus on pure
// calculation functions that don't require vehicle state.

// Test toll calculation: toll = 99 - (3 * strength)
// See ref:ENV-0004.

func TestCalculateTrollToll_LowStrength(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)

	// Strength 10: toll = 99 - 30 = 69
	char := gs.PartyState.GetCharacter(0)
	char.PartyMap = party_state.PartyMapActive
	char.SetStatus(party_state.Good)
	char.Strength = 10

	toll := gs.CalculateTrollToll()

	expected := 69
	if toll != expected {
		t.Errorf("Expected toll %d for strength 10, got %d", expected, toll)
	}
}

func TestCalculateTrollToll_MediumStrength(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)

	// Strength 20: toll = 99 - 60 = 39
	char := gs.PartyState.GetCharacter(0)
	char.PartyMap = party_state.PartyMapActive
	char.SetStatus(party_state.Good)
	char.Strength = 20

	toll := gs.CalculateTrollToll()

	expected := 39
	if toll != expected {
		t.Errorf("Expected toll %d for strength 20, got %d", expected, toll)
	}
}

func TestCalculateTrollToll_HighStrength(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)

	// Strength 30: toll = 99 - 90 = 9
	char := gs.PartyState.GetCharacter(0)
	char.PartyMap = party_state.PartyMapActive
	char.SetStatus(party_state.Good)
	char.Strength = 30

	toll := gs.CalculateTrollToll()

	expected := 9
	if toll != expected {
		t.Errorf("Expected toll %d for strength 30, got %d", expected, toll)
	}
}

func TestCalculateTrollToll_VeryHighStrengthPreservesOriginalNegativeResult(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)

	// Original world-creature rules does not clamp edited stats above the normal range.
	// See ref:ENV-0004.
	char := gs.PartyState.GetCharacter(0)
	char.PartyMap = party_state.PartyMapActive
	char.SetStatus(party_state.Good)
	char.Strength = 35

	toll := gs.CalculateTrollToll()

	expected := -6
	if toll != expected {
		t.Errorf("Expected toll %d for strength 35, got %d", expected, toll)
	}
}

func TestCalculateTrollTollUsesFirstGoodOrPoisonedPartyMember(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)
	initializeAllCharactersNotInParty(gs)

	first := gs.PartyState.GetCharacter(0)
	first.PartyMap = party_state.PartyMapActive
	first.SetStatus(party_state.Poisoned)
	first.Strength = 10
	second := gs.PartyState.GetCharacter(1)
	second.PartyMap = party_state.PartyMapActive
	second.SetStatus(party_state.Good)
	second.Strength = 30

	assert.Equal(t, 69, gs.CalculateTrollToll())
}

// Test payment logic
// See ref:ENV-0004.

func TestPayTrollToll_Success(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)

	// Set initial gold
	gs.PartyState.Inventory.Gold.Set(100)

	// Pay toll of 50
	success := gs.PayTrollToll(50)

	if !success {
		t.Error("Expected payment to succeed with sufficient gold")
	}

	expectedGold := uint16(50)
	if gs.PartyState.Inventory.Gold.Get() != expectedGold {
		t.Errorf("Expected %d gold remaining, got %d", expectedGold, gs.PartyState.Inventory.Gold.Get())
	}
}

func TestPayTrollToll_InsufficientGold(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)

	// Set initial gold
	gs.PartyState.Inventory.Gold.Set(30)

	// Try to pay toll of 50
	success := gs.PayTrollToll(50)

	if success {
		t.Error("Expected payment to fail with insufficient gold")
	}

	// Gold should remain unchanged
	expectedGold := uint16(30)
	if gs.PartyState.Inventory.Gold.Get() != expectedGold {
		t.Errorf("Expected %d gold (unchanged), got %d", expectedGold, gs.PartyState.Inventory.Gold.Get())
	}
}

func TestPayTrollToll_ExactGold(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)

	// Set initial gold exactly equal to toll
	gs.PartyState.Inventory.Gold.Set(50)

	// Pay toll of 50
	success := gs.PayTrollToll(50)

	if !success {
		t.Error("Expected payment to succeed with exact gold amount")
	}

	expectedGold := uint16(0)
	if gs.PartyState.Inventory.Gold.Get() != expectedGold {
		t.Errorf("Expected %d gold remaining, got %d", expectedGold, gs.PartyState.Inventory.Gold.Get())
	}
}

func TestPayTrollToll_ZeroToll(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)

	gs.PartyState.Inventory.Gold.Set(100)

	// Zero toll should always succeed
	success := gs.PayTrollToll(0)

	if !success {
		t.Error("Expected zero toll payment to succeed")
	}

	// Gold should remain unchanged
	expectedGold := uint16(100)
	if gs.PartyState.Inventory.Gold.Get() != expectedGold {
		t.Errorf("Expected %d gold (unchanged), got %d", expectedGold, gs.PartyState.Inventory.Gold.Get())
	}
}

func TestPayTrollToll_NegativeOriginalTollAddsGold(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)
	gs.PartyState.Inventory.Gold.Set(100)

	assert.True(t, gs.PayTrollToll(-6))
	assert.Equal(t, uint16(106), gs.PartyState.Inventory.Gold.Get())
}

// Test the bridge sneak failure boundary with controlled random input. ref:ENV-0004

func TestPerformSneakCheck_HighDex_UsuallyPass(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)

	// Initialize RNG with seed for deterministic testing
	gs.SetRandomSeed(12345)

	char := gs.PartyState.GetCharacter(0)
	char.Dexterity = 30 // Max dex - should never get caught (roll 1-30 never > 30)

	// With max dex, should pass multiple times
	caughtCount := 0
	for i := 0; i < 100; i++ {
		if gs.PerformSneakCheck(char) {
			caughtCount++
		}
	}

	if caughtCount > 0 {
		t.Errorf("With dex 30, expected 0 catches but got %d", caughtCount)
	}
}

func TestPerformSneakCheck_LowDex_OftenCaught(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)

	// Initialize RNG with seed for deterministic testing
	gs.SetRandomSeed(12345)

	char := gs.PartyState.GetCharacter(0)
	char.Dexterity = 1 // Very low dex - almost always caught (roll 2-30 > 1)

	// With very low dex, should get caught most of the time
	caughtCount := 0
	for i := 0; i < 100; i++ {
		if gs.PerformSneakCheck(char) {
			caughtCount++
		}
	}

	// With dex 1, only roll of 1 passes, so ~97% should be caught
	if caughtCount < 90 {
		t.Errorf("With dex 1, expected high catch rate but only got %d/100", caughtCount)
	}
}

func TestPerformSneakCheck_MediumDex(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)

	// Initialize RNG with seed for deterministic testing
	gs.SetRandomSeed(12345)

	char := gs.PartyState.GetCharacter(0)
	char.Dexterity = 15 // Medium dex

	// With medium dex, should have roughly 50% catch rate
	caughtCount := 0
	for i := 0; i < 100; i++ {
		if gs.PerformSneakCheck(char) {
			caughtCount++
		}
	}

	// With dex 15, rolls 16-30 catch (15/30 = 50%)
	// Allow some variance
	if caughtCount < 35 || caughtCount > 65 {
		t.Errorf("With dex 15, expected ~50%% catch rate but got %d/100", caughtCount)
	}
}

// NOTE: CheckBridgeTrollEncounter tests require full GameState initialization
// with properly configured PartyVehicle interface. These tests are covered by
// integration tests that use real game data. See integration tests for:
// - TestCheckBridgeTrollEncounter_OnFoot_Triggers
// - TestCheckBridgeTrollEncounter_InVehicle_NoTrigger
// - TestCheckBridgeTrollEncounter_OnCarpet_NoTrigger

// Test the bridge sneak participant filter: active, not Dead, not Sleeping. ref:ENV-0004

// initializeAllCharactersNotInParty sets all 16 characters to AtTheInn (not in party)
// This is needed because LoadFromRaw may initialize characters with default InTheParty status
func initializeAllCharactersNotInParty(gs *GameState) {
	for i := 0; i < 16; i++ {
		char := gs.PartyState.GetCharacter(i)
		char.PartyMap = party_state.PartyMapAtInn(references.Britain)
		char.SetStatus(party_state.Good)
	}
}

func TestGetLivingAwakePartyMembers_AllAlive(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)

	// First, set all characters to not in party
	initializeAllCharactersNotInParty(gs)

	// Set up 3 party members as in party and good status
	for i := 0; i < 3; i++ {
		char := gs.PartyState.GetCharacter(i)
		char.PartyMap = party_state.PartyMapActive
		char.SetStatus(party_state.Good)
	}

	members := gs.GetLivingAwakePartyMembers()

	if len(members) != 3 {
		t.Errorf("Expected 3 living awake members, got %d", len(members))
	}
}

func TestGetLivingAwakePartyMembers_SomeDead(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)

	// First, set all characters to not in party
	initializeAllCharactersNotInParty(gs)

	// Set up 3 party members: 2 good, 1 dead
	gs.PartyState.GetCharacter(0).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Good)

	gs.PartyState.GetCharacter(1).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(1).SetStatus(party_state.Dead)

	gs.PartyState.GetCharacter(2).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(2).SetStatus(party_state.Good)

	members := gs.GetLivingAwakePartyMembers()

	if len(members) != 2 {
		t.Errorf("Expected 2 living awake members (1 dead excluded), got %d", len(members))
	}
}

func TestGetLivingAwakePartyMembers_SomeAsleep(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)

	// First, set all characters to not in party
	initializeAllCharactersNotInParty(gs)

	// Set up 3 party members: 2 good, 1 asleep
	gs.PartyState.GetCharacter(0).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Good)

	gs.PartyState.GetCharacter(1).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(1).SetStatus(party_state.Sleep)

	gs.PartyState.GetCharacter(2).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(2).SetStatus(party_state.Good)

	members := gs.GetLivingAwakePartyMembers()

	if len(members) != 2 {
		t.Errorf("Expected 2 living awake members (1 asleep excluded), got %d", len(members))
	}
}

func TestGetLivingAwakePartyMembers_NotInParty(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)

	// First, set all characters to not in party
	initializeAllCharactersNotInParty(gs)

	// Set up 3 characters: 2 in party, 1 at the inn
	gs.PartyState.GetCharacter(0).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Good)

	gs.PartyState.GetCharacter(1).PartyMap = party_state.PartyMapAtInn(references.Britain)
	gs.PartyState.GetCharacter(1).SetStatus(party_state.Good)

	gs.PartyState.GetCharacter(2).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(2).SetStatus(party_state.Good)

	members := gs.GetLivingAwakePartyMembers()

	if len(members) != 2 {
		t.Errorf("Expected 2 living awake members (1 at inn excluded), got %d", len(members))
	}
}

func TestGetLivingAwakePartyMembers_AllDead(t *testing.T) {
	gs := &GameState{}
	var rawData [4192]byte
	gs.PartyState = *party_state.LoadFromRaw(rawData)

	// First, set all characters to not in party
	initializeAllCharactersNotInParty(gs)

	// Set up 3 party members as in party but dead
	for i := 0; i < 3; i++ {
		char := gs.PartyState.GetCharacter(i)
		char.PartyMap = party_state.PartyMapActive
		char.SetStatus(party_state.Dead)
	}

	members := gs.GetLivingAwakePartyMembers()

	if len(members) != 0 {
		t.Errorf("Expected 0 living awake members, got %d", len(members))
	}
}

func TestQueueBridgeTrollCombatWaitsForWorkflowThenStartsSourceLessEncounter(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithSystemCallbacks().
		Build()
	gs.SetRandomSeed(95)
	before := gs.MapState.PlayerLocation
	gs.EventQueueManager.Workflow("bridge_dialog").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {}).
		Finally(func() {}).
		Start()
	gs.QueueBridgeTrollCombat()

	gs.FinishTurn()

	assert.Nil(t, gs.CombatScene)
	assert.True(t, gs.HasPendingOverworldContact())
	for gs.EventQueueManager.HasActiveWorkflows() {
		gs.ElapsedMs++
		gs.EventQueueManager.Update()
	}
	require.NoError(t, gs.ResolvePendingOverworldContact())
	require.NotNil(t, gs.CombatScene)
	assert.False(t, gs.HasPendingOverworldContact())
	assert.Equal(t, int(references.BigBridgeCombatMap), gs.CombatScene.CombatMapRef.MapIndex)
	assert.Nil(t, gs.CombatScene.OverworldOrigin)
	assert.GreaterOrEqual(t, gs.CombatScene.MonsterManager.Count(), 1)
	assert.LessOrEqual(t, gs.CombatScene.MonsterManager.Count(), 4)
	assert.Equal(t, references.MonsterTroll, gs.CombatScene.MonsterManager.GetMonster(0).MonsterType())

	gs.EndCombat(CombatVictory)
	assert.Equal(t, before, gs.MapState.PlayerLocation)
}

func TestBridgeTrollCombatReturnsToBridgeForEveryOutcome(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		reason CombatEndReason
	}{
		{name: "victory", reason: CombatVictory},
		{name: "defeat", reason: CombatDefeat},
		{name: "evacuated", reason: CombatEvacuated},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			gs, _ := NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld).
				WithPlayerAt(42, 73).
				WithSystemCallbacks().
				Build()
			before := gs.MapState.PlayerLocation
			gs.SetRandomSeed(95)
			gs.QueueBridgeTrollCombat()

			require.NoError(t, gs.ResolvePendingOverworldContact())
			require.NotNil(t, gs.CombatScene)
			gs.EndCombat(testCase.reason)

			assert.Nil(t, gs.CombatScene)
			assert.Equal(t, before, gs.MapState.PlayerLocation)
			assert.False(t, gs.HasPendingOverworldContact())
		})
	}
}
