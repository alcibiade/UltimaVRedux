package game_state

import "errors"

// CombatTurnScheduling selects whether non-player queue boundaries execute
// immediately or wait for presentation to complete an exact scheduled step.
type CombatTurnScheduling int

const (
	// CombatTurnsImmediate keeps combat fully authoritative and synchronous.
	CombatTurnsImmediate CombatTurnScheduling = iota
	// CombatTurnsPresented exposes monster and auto-pass steps to the scene.
	CombatTurnsPresented
)

// CombatPresentationStepKind describes the presentation required before the
// authoritative DEX queue may continue.
type CombatPresentationStepKind int

// Combat presentation step kinds identify the authoritative action awaiting presentation.
const (
	CombatPresentationMonsterTurn CombatPresentationStepKind = iota
	CombatPresentationAutoPass
	CombatPresentationTurnCompletion
)

// CombatPresentationToken is an opaque identity for one scheduled transition.
// Its owner, encounter, and sequence fields deliberately remain private so
// callers can only return a token to the GameState that issued it.
type CombatPresentationToken struct {
	owner       *GameState
	encounterID CombatEncounterID
	sequence    uint64
}

// CombatPresentationStep is an immutable scene-facing review of scheduled work.
type CombatPresentationStep struct {
	Token        CombatPresentationToken
	Kind         CombatPresentationStepKind
	Actor        CombatTurnIdentity
	MonsterIndex int
	Target       CombatTurnIdentity
	HasTarget    bool
}

type combatPresentationAction int

const (
	combatPresentationAdvance combatPresentationAction = iota
	combatPresentationIdleMonster
	combatPresentationMonsterAI
	combatPresentationPossessedPC
)

type pendingCombatPresentationStep struct {
	step                        CombatPresentationStep
	action                      combatPresentationAction
	entry                       *TurnQueueEntry
	completeUndefinedPlayerTail bool
	reviewed                    bool
	resolved                    bool
}

var (
	// ErrCombatPresentationInactive means the encounter owning a token has ended.
	ErrCombatPresentationInactive = errors.New("combat presentation is no longer active")
	// ErrCombatPresentationStale means a token does not identify the current step.
	ErrCombatPresentationStale = errors.New("combat presentation token is stale")
	// ErrCombatPresentationDuplicate means the exact step was already resolved or completed.
	ErrCombatPresentationDuplicate = errors.New("combat presentation step already completed")
	// ErrCombatPresentationUnresolved means completion preceded authoritative monster resolution.
	ErrCombatPresentationUnresolved = errors.New("combat presentation step is unresolved")
	// ErrCombatPresentationNotExecutable means the step has no deferred monster action.
	ErrCombatPresentationNotExecutable = errors.New("combat presentation step is not executable")
)

// The combat transition handshake has one authoritative owner:
//
//	headless: AdvanceCombatTurn -> execute actor -> continue DEX scan
//	visual:   BeginCombatTurns -> schedule(token) -> ReviewScheduledCombatStep
//	          -> scene delay -> ResolveScheduledCombatStep(token)
//	          -> scene movement/projectile -> CompleteScheduledCombatStep(token)
//	          -> AdvanceCombatTurn
//
// GameScene owns only presentation between review and exact completion. It
// cannot install gameplay closures or write turn-transition flags.

// BeginCombatTurns starts the DEX queue once using the requested scheduling mode.
func (g *GameState) BeginCombatTurns(scheduling CombatTurnScheduling) {
	if g.CombatScene == nil {
		return
	}
	if scheduling != CombatTurnsImmediate && scheduling != CombatTurnsPresented {
		panic("BeginCombatTurns called with invalid scheduling mode")
	}
	if g.CombatScene.turnsStarted {
		return
	}
	g.CombatScene.turnsStarted = true
	g.CombatScene.presentTurns = scheduling == CombatTurnsPresented
	g.AdvanceCombatTurn()
}

// CombatTurnsStarted reports whether the first DEX actor has been selected.
func (combatState *CombatState) CombatTurnsStarted() bool {
	return combatState != nil && combatState.turnsStarted
}

// HasScheduledCombatStep reports whether authoritative progression is waiting
// for presentation completion.
func (combatState *CombatState) HasScheduledCombatStep() bool {
	return combatState != nil && combatState.pendingPresentationStep != nil
}

// ReviewScheduledCombatStep claims the current scene-scheduled step exactly once.
func (g *GameState) ReviewScheduledCombatStep() (CombatPresentationStep, bool) {
	if g.CombatScene == nil || g.CombatScene.pendingPresentationStep == nil ||
		g.CombatScene.pendingPresentationStep.reviewed ||
		g.CombatScene.pendingPresentationStep.step.Kind == CombatPresentationTurnCompletion {
		return CombatPresentationStep{}, false
	}
	pending := g.CombatScene.pendingPresentationStep
	pending.reviewed = true

	return pending.step, true
}

// ResolveScheduledCombatStep executes the authoritative action for an exact
// monster step after the scene's anticipation delay.
func (g *GameState) ResolveScheduledCombatStep(
	token CombatPresentationToken,
) (MonsterTurnExecutionResult, error) {
	pending, err := g.validateCombatPresentationToken(token)
	if err != nil {
		return MonsterTurnExecutionResult{}, err
	}
	if pending.resolved {
		return MonsterTurnExecutionResult{}, ErrCombatPresentationDuplicate
	}

	var result MonsterTurnExecutionResult
	switch pending.action {
	case combatPresentationIdleMonster:
		g.resolveCombatActorStandingEffects(pending.step.Actor)
	case combatPresentationMonsterAI:
		result = g.executeMonsterTurn(pending.entry)
	case combatPresentationPossessedPC:
		from := g.CombatScene.GetCombatantPosition(pending.step.Actor)
		g.executePossessedPCTurn(pending.step.Actor)
		to := g.CombatScene.GetCombatantPosition(pending.step.Actor)
		g.resolveCombatPCStandingEffects(pending.step.Actor)
		if pending.completeUndefinedPlayerTail {
			g.processDurationEffects()
		}
		result.Moved = to != from
	case combatPresentationAdvance:
		return MonsterTurnExecutionResult{}, ErrCombatPresentationNotExecutable
	default:
		panic("unknown combat presentation action")
	}
	pending.resolved = true

	return result, nil
}

// CompleteScheduledCombatStep completes only the step identified by token and
// immediately continues the authoritative DEX transition. Duplicate, stale,
// and post-combat callbacks are rejected without mutating a replacement turn.
func (g *GameState) CompleteScheduledCombatStep(token CombatPresentationToken) error {
	pending, err := g.validateCombatPresentationToken(token)
	if err != nil {
		return err
	}
	if pending.step.Kind == CombatPresentationMonsterTurn && !pending.resolved {
		return ErrCombatPresentationUnresolved
	}

	combat := g.CombatScene
	combat.pendingPresentationStep = nil
	combat.lastCompletedPresentation = token
	combat.ActiveMonsterIndex = NoActiveMonster
	g.clearActiveCombatTarget()
	g.AdvanceCombatTurn()

	return nil
}

func (g *GameState) scheduleCombatPresentationStep(
	kind CombatPresentationStepKind,
	action combatPresentationAction,
	actor CombatTurnIdentity,
	monsterIndex int,
	target CombatTurnIdentity,
	hasTarget bool,
	entry *TurnQueueEntry,
) CombatPresentationToken {
	combat := g.CombatScene
	if combat == nil {
		panic("scheduleCombatPresentationStep called outside combat")
	}
	if combat.pendingPresentationStep != nil {
		panic("combat presentation step already scheduled")
	}
	combat.nextPresentationSequence++
	token := CombatPresentationToken{
		owner:       g,
		encounterID: combat.EncounterID,
		sequence:    combat.nextPresentationSequence,
	}
	combat.pendingPresentationStep = &pendingCombatPresentationStep{
		step: CombatPresentationStep{
			Token: token, Kind: kind, Actor: actor, MonsterIndex: monsterIndex,
			Target: target, HasTarget: hasTarget,
		},
		action: action,
		entry:  entry,
	}

	return token
}

func (g *GameState) validateCombatPresentationToken(
	token CombatPresentationToken,
) (*pendingCombatPresentationStep, error) {
	if g.CombatScene == nil {
		return nil, ErrCombatPresentationInactive
	}
	combat := g.CombatScene
	if token.owner != g || token.encounterID != combat.EncounterID || token.sequence == 0 {
		return nil, ErrCombatPresentationStale
	}
	if token == combat.lastCompletedPresentation {
		return nil, ErrCombatPresentationDuplicate
	}
	if combat.pendingPresentationStep == nil || token != combat.pendingPresentationStep.step.Token {
		return nil, ErrCombatPresentationStale
	}

	return combat.pendingPresentationStep, nil
}
