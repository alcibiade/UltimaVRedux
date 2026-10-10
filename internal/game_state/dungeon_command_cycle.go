package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

// DungeonCommandCycle is an opaque exact-once capability. Its identity ties
// asynchronous command completion to its precharged or deferred cycle.
type DungeonCommandCycle struct {
	id uint64
}

// DungeonCommandCycleStart reports the precharge and actor phase selected at begin.
type DungeonCommandCycleStart struct {
	ElapsedMinutes     int
	DungeonActorsMove  bool
	TimeStopped        bool
	QuicknessCycleSkip bool
}

// DungeonCommandCycleResult reports synchronous actor processing. Wanderer
// contact is returned for the encounter orchestration owned by the encounter workflow.
type DungeonCommandCycleResult struct {
	WandererMovement DungeonWandererMovementResult
}

type dungeonCommandCycleState struct {
	token              DungeonCommandCycle
	dungeonActorsMove  bool
	timeStopped        bool
	quicknessCycleSkip bool
	actionResolved     bool
	completionStarted  bool
	expectedCompletion DungeonCommandCompletion
	deferredCharge     bool
}

// BeginDungeonCommandCycle precharges one accepted command and returns its exact-once token.
func (g *GameState) BeginDungeonCommandCycle() (DungeonCommandCycle, DungeonCommandCycleStart) {
	return g.beginDungeonCommandCycle(false)
}

// BeginDeferredDungeonCommandCycle reserves Cast/Use ownership without advancing
// the calendar, quickness phase, or RNG until final valid confirmation.
// REDUX DEVIATION: original dungeon rules precharges input; original prepared-spell rules and
// original item-use rules charge before targeting. Cast/Use cancellation is free.
// See ref:TIME-0002.
func (g *GameState) BeginDeferredDungeonCommandCycle() DungeonCommandCycle {
	token, _ := g.beginDungeonCommandCycle(true)

	return token
}

func (g *GameState) beginDungeonCommandCycle(deferred bool) (DungeonCommandCycle, DungeonCommandCycleStart) {
	// REDUX DEVIATION: Original dungeon rules precharges before waiting for any key.
	// Redux begins on an accepted command key so idle input never owns an
	// in-flight token. Only Cast/Use defer charging until final confirmation;
	// rejection and cancellation of other accepted commands still cost a cycle.
	// See ref:TIME-0002.
	dungeon := g.requireActiveDungeon()
	if g.activeDungeonCommand != nil {
		panic("dungeon command cycle is already active")
	}
	if dungeon.hasPendingRoomContinuation() {
		panic("cannot begin a dungeon command while a room continuation is pending")
	}

	g.dungeonCommandSequence++
	token := DungeonCommandCycle{id: g.dungeonCommandSequence}
	g.activeDungeonCommand = &dungeonCommandCycleState{
		token:              token,
		actionResolved:     false,
		completionStarted:  false,
		expectedCompletion: DungeonCommandOnly,
		deferredCharge:     true,
	}
	if deferred {
		return token, DungeonCommandCycleStart{}
	}

	return token, g.chargeDungeonCommandCycle(g.activeDungeonCommand)
}

func (g *GameState) chargeDungeonCommandCycle(cycle *dungeonCommandCycleState) DungeonCommandCycleStart {
	cycle.deferredCharge = false
	timeStopped := g.HasActiveDurationEffect(DurationEffectTimeStop)
	dungeonActorsMove, quicknessCycleSkip := g.requireActiveDungeon().beginActorPhase(
		timeStopped, g.HasActiveDurationEffect(DurationEffectQuickness))
	cycle.dungeonActorsMove = dungeonActorsMove
	cycle.timeStopped = timeStopped
	cycle.quicknessCycleSkip = quicknessCycleSkip
	elapsed := 0
	if dungeonActorsMove {
		elapsed = g.advanceGameplayMinutes(1, false)
	}

	return DungeonCommandCycleStart{
		ElapsedMinutes:     elapsed,
		DungeonActorsMove:  dungeonActorsMove,
		TimeStopped:        timeStopped,
		QuicknessCycleSkip: quicknessCycleSkip,
	}
}

func (d *DungeonState) beginActorPhase(timeStopped, quicknessActive bool) (bool, bool) {
	if timeStopped {
		if !quicknessActive {
			d.quicknessSkipNext = false
		}

		return false, false
	}
	if !quicknessActive {
		d.quicknessSkipNext = false

		return true, false
	}

	actorsMove := !d.quicknessSkipNext
	d.quicknessSkipNext = !d.quicknessSkipNext

	return actorsMove, !actorsMove
}

// CompleteDungeonCommandCycle consumes token and optionally runs full dungeon processing.
func (g *GameState) CompleteDungeonCommandCycle(
	token DungeonCommandCycle,
	completion DungeonCommandCompletion,
) DungeonCommandCycleResult {
	cycle := g.beginDungeonCommandCompletion(token, completion)
	result := DungeonCommandCycleResult{
		WandererMovement: DungeonWandererMovementResult{Monster: references.MonsterNone},
	}
	if completion == DungeonFullEndPlayer && !cycle.deferredCharge {
		var deferred bool
		result, deferred = g.processDungeonEndPlayer(token, cycle.dungeonActorsMove)
		if result.WandererMovement.Contact {
			if g.startContactDungeonWandererCombat(token, result.WandererMovement) {
				return result
			}
			deferred = g.processDungeonEndPlayerAfterWanderer(token)
		}
		if deferred {
			return result
		}
	}
	g.finishDungeonCommandCycle(token)

	return result
}

func (g *GameState) beginDungeonCommandCompletion(
	token DungeonCommandCycle,
	completion DungeonCommandCompletion,
) *dungeonCommandCycleState {
	cycle := g.activeDungeonCommand
	if cycle == nil {
		panic("no dungeon command cycle is active")
	}
	if token.id == 0 || token.id != cycle.token.id {
		panic("stale or foreign dungeon command cycle")
	}
	if completion != DungeonCommandOnly && completion != DungeonFullEndPlayer {
		panic("invalid dungeon command completion")
	}
	if cycle.actionResolved && completion != cycle.expectedCompletion {
		panic("dungeon command completion does not match resolved action")
	}
	if cycle.completionStarted {
		panic("dungeon command cycle completion already started")
	}
	cycle.completionStarted = true

	return cycle
}

func (g *GameState) finishDungeonCommandCycle(token DungeonCommandCycle) {
	cycle := g.activeDungeonCommand
	if cycle == nil || token.id == 0 || token.id != cycle.token.id {
		panic("stale or foreign dungeon command cycle")
	}
	if !cycle.completionStarted {
		panic("dungeon command cycle completion has not started")
	}
	g.activeDungeonCommand = nil
}

// IsAtDungeonCommandBoundary reports whether stable snapshot capture can proceed.
func (g *GameState) IsAtDungeonCommandBoundary() bool {
	return g.activeDungeonCommand == nil && !g.activeDungeon.hasPendingRoomContinuation()
}

func (g *GameState) claimDungeonCommandAction(completion DungeonCommandCompletion) {
	cycle := g.activeDungeonCommand
	if cycle == nil {
		panic("dungeon action requires an active command cycle")
	}
	if cycle.actionResolved {
		panic("dungeon command cycle already resolved an action")
	}
	if completion != DungeonCommandOnly && completion != DungeonFullEndPlayer {
		panic("invalid dungeon action completion")
	}
	cycle.actionResolved = true
	cycle.expectedCompletion = completion
}

// ClaimDungeonViewCommand binds an asynchronous View-capable workflow to its accepted dungeon command.
func (g *GameState) ClaimDungeonViewCommand(token DungeonCommandCycle) {
	g.ClaimDungeonFullCommand(token)
}

// ClaimDungeonFullCommand binds an asynchronous full-completion workflow to its accepted command.
func (g *GameState) ClaimDungeonFullCommand(token DungeonCommandCycle) {
	cycle := g.activeDungeonCommand
	if cycle == nil || token.id == 0 || token.id != cycle.token.id {
		panic("dungeon workflow received a stale or foreign command cycle")
	}
	g.requireActiveDungeon()
	g.claimDungeonCommandAction(DungeonFullEndPlayer)
}

func (g *GameState) processDungeonEndPlayer(
	token DungeonCommandCycle,
	dungeonActorsMove bool,
) (DungeonCommandCycleResult, bool) {
	result := DungeonCommandCycleResult{
		WandererMovement: g.processDungeonEndPlayerBeforeCell(dungeonActorsMove),
	}
	if result.WandererMovement.Contact {
		return result, false
	}
	deferred := g.processDungeonEndPlayerAfterWanderer(token)

	return result, deferred
}

func (g *GameState) processDungeonEndPlayerAfterWanderer(token DungeonCommandCycle) bool {
	if g.queueCurrentDungeonRoomEncounter(true, true, false, dungeonRoomClearWanderer) {
		return false
	}
	if g.startCurrentDungeonHiddenPitFall(token) {
		return true
	}
	g.processDungeonEndPlayerAfterCell()

	return false
}

func (g *GameState) processDungeonEndPlayerBeforeCell(dungeonActorsMove bool) DungeonWandererMovementResult {
	g.processAutoWake()
	if dungeonActorsMove {
		return g.moveDungeonWanderer()
	}

	return DungeonWandererMovementResult{Monster: references.MonsterNone}
}

func (g *GameState) processDungeonEndPlayerAfterCell() {
	g.processDungeonCurrentCellBomb()
	g.processDungeonCurrentCellField()
	g.processDungeonEndPlayerFinalEffects()
}

func (g *GameState) processDungeonEndPlayerFinalEffects() {
	statusCache := g.cachePlayerUpdateStatuses()
	g.processDungeonPoisonDamage(statusCache)
	g.checkHourlyFoodConsumptionForDiners(statusCache.Diners)
	g.advanceBeggarKarmaCounter()
	g.processDurationEffects()
	g.processToggleItems()
	g.ProcessRegenerationRingSweep()
	g.checkForPartyWipe()
}
