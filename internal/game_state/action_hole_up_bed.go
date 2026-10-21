package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// Bed rest timing constants
const (
	// Event queue delays (milliseconds)
	BedRestInitialDelayMs       = 500  // Delay before fade to black
	BedRestFadeToBlackMs        = 1500 // Duration of fade to black
	BedRestPreHealingDelayMs    = 200  // Pause before healing starts
	BedRestHealingCycleDelayMs  = 83   // Delay between healing cycles (~83ms per 10-minute game cycle)
	BedRestPreWakeDelayMs       = 200  // Pause before fade back from black
	BedRestFadeFromBlackMs      = 1500 // Normal fade from black duration
	BedRestQuickFadeFromBlackMs = 500  // Quick fade from black after death
	BedRestPreSleepNPCPasses    = 16
)

// ActionHoleUpSmallMap starts the town bed-rest selection workflow.
func (g *GameState) ActionHoleUpSmallMap() bool {
	playerPos := &g.MapState.PlayerLocation.Position
	currentTile := g.GetLayeredMapByCurrentLocation().GetTileByLayer(map_state.MapLayer, playerPos)

	// Rest dispatch accepts only the underlying 0xAB terrain byte.
	// See ref:CAMP-0006.
	if currentTile.Index != indexes.LeftBed {
		g.SystemCallbacks.Message.AddRowStr("Find a bed first!")

		return false
	}

	// NPC conflict checking implemented (see isNPCScheduledAtPosition)
	// See ref:CAMP-0006.

	// Show duration selection dialog
	g.SystemCallbacks.Flow.Rest.RequestSleepDuration()

	return true
}

// BedRestState tracks the state of bed rest across Event Queue callbacks
type BedRestState struct {
	TargetHour        int
	CurrentCycle      int
	Interrupted       bool
	AttackInterrupted bool
	DeathOccurred     bool
	PartyWipePending  bool
	TimeStopped       bool
	OldActivePlayer   int
	Started           bool
	sleepingChanged   [party_state.MaxActiveParty]bool
}

// BedRestProvisionResult reports one explicitly processed rest-hour consequence.
type BedRestProvisionResult struct {
	Starving    bool
	DeadIndexes []int
}

func (g *GameState) processBedRestProvisionHour(hour, diners int) BedRestProvisionResult {
	g.lastProcessedHour = hour
	if g.PartyState.Inventory.Provisions.Food.Get() == 0 {
		return BedRestProvisionResult{Starving: true, DeadIndexes: g.applyStarvationDamage()}
	}
	g.consumeMealWithFeedback(hour, diners, true)

	return BedRestProvisionResult{}
}

// NewBedRestState captures the source hour-bucket target before any NPC passes.
func (g *GameState) NewBedRestState(hours int) *BedRestState {
	// REDUX DEVIATION: bed rest subtracts 23 once when the target exceeds
	// 23. Redux uses safe modulo-24 arithmetic while retaining its hour target.
	// See ref:CAMP-0006.
	return &BedRestState{
		TargetHour: (int(g.DateTime.Hour) + hours) % 24,
		// REDUX DEVIATION: original bed rest retains occupancy as an exit even
		// when Time Stop cannot expire. Redux exits immediately for duration
		// 0 or 255, avoiding reliance on a future NPC interruption for safety.
		// See ref:CAMP-0006.
		TimeStopped: g.HasActiveDurationEffect(DurationEffectTimeStop) &&
			(g.ActiveDurationEffect.TurnsRemaining == 0 ||
				g.ActiveDurationEffect.TurnsRemaining == ToggleItemEffectDuration),
		OldActivePlayer: g.GetActivePlayerIndex(),
	}
}

// ProcessBedRestPreSleepNPCPass runs one unchanged-hour npcdriver pass and
// reports only attack intent. Ordinary Talk does not prevent sleep.
func (g *GameState) ProcessBedRestPreSleepNPCPass() bool {
	g.smallMapProcessNPCs()
	g.SystemCallbacks.Screen.UpdateDisplay()
	if g.pendingSmallMapEncounter != nil {
		return true
	}

	var finalInteraction map_units.PendingNPCInteraction
	finalNPCNum := -1
	for _, unit := range *g.CurrentNPCAIController.GetNpcs() {
		if unit.IsEmptyMapUnit() || !unit.IsVisible() {
			continue
		}
		details := unit.MapUnitDetails()
		if details.PendingInteraction == map_units.PendingNPCInteractionNone || details.NPCNum < finalNPCNum {
			continue
		}
		finalInteraction = details.PendingInteraction
		finalNPCNum = details.NPCNum
		if details.PendingInteraction == map_units.PendingNPCInteractionTalk {
			if friendly, ok := unit.(*map_units.NPCFriendly); ok &&
				friendly.MapUnitDetails().AIOverrideProvenance() == references.AIOverrideProvenanceAutomaticAstaroth {
				finalInteraction = map_units.PendingNPCInteractionHostileCombat
			}
		}
	}
	// Only the final pending NPC command from the preliminary pass matters.
	// Ordinary town completion runs a new NPC phase, replacing that pending
	// command. ref:CAMP-0006
	for _, unit := range *g.CurrentNPCAIController.GetNpcs() {
		if !unit.IsEmptyMapUnit() {
			unit.MapUnitDetails().PendingInteraction = map_units.PendingNPCInteractionNone
		}
	}

	switch finalInteraction {
	case map_units.PendingNPCInteractionGuardArrest,
		map_units.PendingNPCInteractionHostileCombat:
		return true
	case map_units.PendingNPCInteractionTalk,
		map_units.PendingNPCInteractionNone:
	default:
		panic("unsupported pending NPC interaction during bed rest")
	}

	return false
}

// BeginBedRest changes only active Good characters to Sleep and remembers the
// exact records changed by this rest operation.
func (g *GameState) BeginBedRest(state *BedRestState) {
	state.Started = true
	for index := range party_state.MaxActiveParty {
		character := g.PartyState.GetCharacter(index)
		if character.PartyMap != party_state.PartyMapActive || !character.IsGood() {
			continue
		}
		state.sleepingChanged[index] = true
		character.SetStatus(party_state.Sleep)
	}
	g.ClearActivePlayer()
}

// BedRestCycleResult reports authoritative effects from one complete ten-minute attempt.
type BedRestCycleResult struct {
	PoisonDeaths    []int
	Provision       BedRestProvisionResult
	NPCAtBed        bool
	ReachedTarget   bool
	TotalPartyDeath bool
}

// ProcessBedRestCycle applies addtime and updateplayers ordering, then rebuilds
// live NPC placement before checking the bed's actual map-unit occupancy.
func (g *GameState) ProcessBedRestCycle(state *BedRestState) BedRestCycleResult {
	hourBefore := int(g.DateTime.Hour)
	g.AdvanceBedRestMinutes(10)
	statusCache := g.cachePlayerUpdateStatuses()
	result := BedRestCycleResult{
		PoisonDeaths: g.applyCachedPlayerUpdatePoison(statusCache, false),
	}
	if int(g.DateTime.Hour) != hourBefore {
		result.Provision = g.processBedRestProvisionHour(int(g.DateTime.Hour), statusCache.Diners)
	}
	g.advanceBeggarKarmaCounter()
	g.processDurationEffects()
	g.processToggleItems()
	g.ProcessRegenerationRingSweep()
	state.CurrentCycle++
	state.DeathOccurred = state.DeathOccurred || len(result.PoisonDeaths) > 0 || len(result.Provision.DeadIndexes) > 0

	if g.PartyState.IsAllPartyDead() && !state.PartyWipePending {
		state.PartyWipePending = true
		result.TotalPartyDeath = true
	}

	g.CurrentNPCAIController.HardResetNPCs(true)
	g.CurrentNPCAIController.FreshenExistingNPCsOnMap()
	position := g.MapState.PlayerLocation.Position
	result.NPCAtBed = g.GetLayeredMapByCurrentLocation().
		SpriteIndexByLayer(map_state.MapUnitLayer, position) > 0
	result.ReachedTarget = int(g.DateTime.Hour) == state.TargetHour

	return result
}

// FinishBedRest restores only statuses changed by BeginBedRest.
func (g *GameState) FinishBedRest(state *BedRestState) {
	if state == nil || !state.Started {
		return
	}
	for index, changed := range state.sleepingChanged {
		character := g.PartyState.GetCharacter(index)
		if changed && character.IsAsleep() {
			character.SetStatus(party_state.Good)
		}
	}
	if state.OldActivePlayer >= 0 {
		g.SetActivePlayer(state.OldActivePlayer)
	} else {
		g.SystemCallbacks.Screen.UpdateStatsDisplay()
	}
	state.Started = false
	if state.PartyWipePending {
		g.checkForPartyWipe()
	}
}

// GetFadeDuration returns appropriate fade duration based on death status
func (brs *BedRestState) GetFadeDuration() uint64 {
	if brs.DeathOccurred {
		return BedRestQuickFadeFromBlackMs // Quick fade for death
	}

	return BedRestFadeFromBlackMs // Normal fade
}

// MoveAvatarOutOfBed moves the avatar one tile east after bed rest, matching original U5 behavior.
// This prevents the player from remaining on the bed and potentially blocking NPCs or interactions.
func (g *GameState) MoveAvatarOutOfBed() {
	// Only move avatar in small maps (towns) - original behavior
	if g.MapState.PlayerLocation.Location.GetMapType() != references.SmallMapType {
		return
	}

	currentPos := g.MapState.PlayerLocation.Position
	eastPos := references.Right.GetNewPositionInDirection(&currentPos)

	// Safety check: only move if the destination is passable and unoccupied
	if !g.IsOutOfBounds(*eastPos) &&
		g.IsPassable(eastPos) {
		g.MapState.PlayerLocation.Position = *eastPos
		logging.Debug(logging.CategoryGameState, "Avatar moved out of bed: (%d,%d) -> (%d,%d)",
			currentPos.X, currentPos.Y, eastPos.X, eastPos.Y)

		// Trigger display update to show the movement
		g.SystemCallbacks.Screen.UpdateDisplay()
	} else {
		logging.Debug(logging.CategoryGameState, "Could not move avatar out of bed - destination blocked or invalid")
	}
}
