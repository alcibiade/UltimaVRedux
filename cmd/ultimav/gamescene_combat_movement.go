package main

import (
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

const (
	combatPlayerMovementDurationMs uint64 = 333
	combatEnemyMovementDurationMs  uint64 = 650
	combatBatAnticipationMs        uint64 = 75
	combatBatMovementDurationMs    uint64 = 175
)

const (
	combatMovementSmoothstepScale = 3.0
	combatMovementSmoothstepBias  = 2.0
)

// CombatMovementAnimation visually interpolates an entity whose logical move is complete.
type CombatMovementAnimation struct {
	Active        bool
	IsPartyMember bool
	EntityIndex   int
	Slot          int
	From          references.Position
	To            references.Position
	Progress      float32
}

func (g *GameScene) startCombatMovement(
	isPartyMember bool,
	entityIndex, slot int,
	fromPos, toPos references.Position,
) {
	g.combatMovement = CombatMovementAnimation{
		Active:        true,
		IsPartyMember: isPartyMember,
		EntityIndex:   entityIndex,
		Slot:          slot,
		From:          fromPos,
		To:            toPos,
		Progress:      0,
	}
}

func (g *GameScene) updateCombatMovementProgress(progress float32) {
	if g.combatMovement.Active {
		g.combatMovement.Progress = progress
	}
}

func (g *GameScene) clearCombatMovement() {
	g.combatMovement = CombatMovementAnimation{
		Active:        false,
		IsPartyMember: false,
		EntityIndex:   0,
		Slot:          0,
		From:          references.Position{X: 0, Y: 0},
		To:            references.Position{X: 0, Y: 0},
		Progress:      0,
	}
}

func (g *GameScene) startPlayerCombatMovementWorkflow(
	identity game_state.CombatTurnIdentity,
	fromPos, toPos references.Position,
	token game_state.CombatPresentationToken,
) {
	g.startCombatMovement(true, identity.EntityIndex, identity.Slot, fromPos, toPos)
	g.gameState.EventQueueManager.Workflow("player_combat_movement").
		Priority(event_queue.PriorityHigh).
		Sequential().BlockInput().
		First(func() {}).
		Animate(g.combatMovementDuration(combatPlayerMovementDurationMs), g.updateCombatMovementProgress).Finally(

		func() {
			g.clearCombatMovement()
			g.completeScheduledCombatStep("player movement", token)
		}).
		Start()
}

func (g *GameScene) startControlledMonsterCombatMovementWorkflow(
	entityIndex int,
	fromPos, toPos references.Position,
	token game_state.CombatPresentationToken,
) {
	g.startCombatMovement(false, entityIndex, 0, fromPos, toPos)
	g.gameState.EventQueueManager.Workflow("controlled_monster_combat_movement").
		Priority(event_queue.PriorityHigh).
		Sequential().BlockInput().
		First(func() {}).
		Animate(g.combatMovementDuration(combatPlayerMovementDurationMs), g.updateCombatMovementProgress).
		Finally(func() {
			g.clearCombatMovement()
			g.completeScheduledCombatStep("controlled-monster movement", token)
		}).
		Start()
}

func (g *GameScene) combatMovementDuration(baseDurationMs uint64) uint64 {
	if g.gameConfig == nil {
		return baseDurationMs
	}

	return scaledCombatMovementDuration(baseDurationMs, g.gameConfig.DebugConfiguration)
}

func scaledCombatMovementDuration(
	baseDurationMs uint64,
	debugConfiguration *config.DebugConfiguration,
) uint64 {
	return scaleCombatMovementDuration(baseDurationMs, debugConfiguration.GetCombatMovementSpeedFactor())
}

func scaleCombatMovementDuration(baseDurationMs uint64, factor float64) uint64 {
	return max(uint64(math.Round(float64(baseDurationMs)/factor)), 1)
}

type combatMonsterPacing struct {
	anticipationMs uint64
	movementMs     uint64
}

// fastScheduledMonsterPacing captures both durations before any deferred queue
// stage runs. Only AI bats and mongbats use this profile; possessed PCs also
// enter the monster workflow, so reject party identities before monster lookup.
func (g *GameScene) fastScheduledMonsterPacing(actor game_state.CombatTurnIdentity) (combatMonsterPacing, bool) {
	if actor.IsPartyMember || g.gameState.CombatScene == nil {
		return combatMonsterPacing{}, false
	}
	monster := g.gameState.CombatScene.MonsterManager.GetMonster(actor.EntityIndex)
	if monster == nil || (monster.MonsterType() != references.MonsterBat &&
		monster.MonsterType() != references.MonsterMongbat) {
		return combatMonsterPacing{}, false
	}

	var debugConfiguration *config.DebugConfiguration
	if g.gameConfig != nil {
		debugConfiguration = g.gameConfig.DebugConfiguration
	}
	factor := debugConfiguration.GetCombatMovementSpeedFactor()

	return combatMonsterPacing{
		anticipationMs: scaleCombatMovementDuration(combatBatAnticipationMs, factor),
		movementMs:     scaleCombatMovementDuration(combatBatMovementDurationMs, factor),
	}, true
}

func (g *GameScene) combatMovementOffset(isPartyMember bool, entityIndex, slot int) (float64, float64) {
	movement := g.combatMovement
	if !movement.Active || movement.IsPartyMember != isPartyMember || movement.EntityIndex != entityIndex ||
		(isPartyMember && movement.Slot != slot) {
		return 0, 0
	}

	fromX, fromY := sprites.ConvertPosition16to18(int(movement.From.X), int(movement.From.Y))
	toX, toY := sprites.ConvertPosition16to18(int(movement.To.X), int(movement.To.Y))
	progress := float64(movement.Progress)
	eased := progress * progress *
		(combatMovementSmoothstepScale - combatMovementSmoothstepBias*progress)
	remaining := 1 - eased

	return math.Round((fromX - toX) * remaining), math.Round((fromY - toY) * remaining)
}
