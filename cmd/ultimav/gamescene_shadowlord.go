package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/internal/visual"
	rng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

// Shadowlord presence effect constants
const (
	// Flash effect timing (slow, ominous)
	shadowlordFlashDurationMs   = 2000 // 2 second slow flash
	shadowlordFlashMaxIntensity = 0.5  // Not too overwhelming

	// Dark purple/red color for ominous effect (RGB normalized 0-1)
	shadowlordFlashR = 0.4
	shadowlordFlashG = 0.0
	shadowlordFlashB = 0.3

	shadowlordSummonFizzDurationMs = 1000
	shadowlordSummonFizzSeed       = 0x534841444f574649
)

type shadowlordSummonPresentation struct {
	lord       references.Shadowlord
	location   references.Location
	floor      references.FloorNumber
	position   references.Position
	enemy      *map_units.NPCEnemy
	dissolveID int
	started    bool
}

// TriggerShadowlordSummoned handles the UI effects when a Shadowlord is summoned via YELL.
// Called after GameState has already spawned the Shadowlord NPC (via action_yell.go).
// ref:SHADOW-0004: appearance message, finite pulse, then
// one dissolve reveal at the newly spawned Shadowlord's position.
//
// IMPORTANT: Per original Ultima V behavior documented in docs/gamefacts/SHADOWLORDS.md,
// yell summoning does NOT apply NPC behavior effects; it only spawns the
// Shadowlord NPC for combat (ref:SHADOW-0004).
// NPC effects (hostile/flee) only occur with automatic town-assignment presence.
//
// Note: The Shadowlord NPC is spawned by GameState before this callback is invoked.
func (g *GameScene) TriggerShadowlordSummoned(lord references.Shadowlord) {
	enemy := g.authoritativeSummonedShadowlord(lord)
	if enemy == nil {
		return
	}
	presentation := &shadowlordSummonPresentation{
		lord: lord, location: g.gameState.MapState.PlayerLocation.Location,
		floor: enemy.Floor(), position: enemy.Pos(), enemy: enemy,
	}

	g.gameState.EventQueueManager.TriggerFeedback(
		"shadowlord_summon",
		event_queue.PriorityNormal,
		func(q *event_queue.SequentialEventBuilder) {
			q.First(func() {
				if !g.isShadowlordSummonCurrent(presentation) {
					return
				}

				g.gameState.SystemCallbacks.Message.AddRowStr("A shadowlord appears!")
				g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundShadowlordSummon)
				presentation.dissolveID = g.startTileDissolveWithOptions(TileDissolveOptions{
					Position: presentation.position, SpriteIndex: enemy.GetKeySpriteIndex(),
					Mode: visual.DissolveModeReveal, OutlineColor: sprites.WhiteOutline,
					PositionMode: PositionModeWorld, RenderLayer: RenderLayerDefault,
				}, rng.New(shadowlordSummonFizzSeed))
				presentation.started = true
				g.shadowlordSummonPresentation = presentation
			}).Animate(shadowlordSummonFizzDurationMs, func(progress float32) {
				if !g.isShadowlordSummonCurrent(presentation) {
					g.clearShadowlordSummonPresentation(presentation)

					return
				}
				g.UpdateTileDissolveProgress(presentation.dissolveID, float64(progress))
			}).Finally(func() {
				g.clearShadowlordSummonPresentation(presentation)
			})
		},
	)
}

func (g *GameScene) authoritativeSummonedShadowlord(lord references.Shadowlord) *map_units.NPCEnemy {
	state := g.gameState.ShadowlordState
	if state == nil || !state.Summoned || state.LastCalled != lord || g.gameState.CurrentNPCAIController == nil {
		return nil
	}
	floor := g.gameState.MapState.PlayerLocation.Floor

	return g.gameState.CurrentNPCAIController.GetNpcs().GetFirstEnemyMatching(func(enemy *map_units.NPCEnemy) bool {
		identity, ok := enemy.ShadowlordIdentity()

		return ok && identity == lord && enemy.IsVisible() && enemy.Floor() == floor
	})
}

func (g *GameScene) isShadowlordSummonCurrent(presentation *shadowlordSummonPresentation) bool {
	if presentation == nil || g.disposed || g.gameState == nil || g.gameState.CurrentNPCAIController == nil ||
		g.gameState.MapState.PlayerLocation.Location != presentation.location ||
		g.gameState.MapState.PlayerLocation.Floor != presentation.floor ||
		!g.gameState.ShadowlordState.Summoned || g.gameState.ShadowlordState.LastCalled != presentation.lord ||
		!presentation.enemy.IsVisible() || presentation.enemy.Pos() != presentation.position ||
		presentation.enemy.Floor() != presentation.floor {
		return false
	}
	identity, ok := presentation.enemy.ShadowlordIdentity()
	if !ok || identity != presentation.lord {
		return false
	}

	return g.gameState.CurrentNPCAIController.GetNpcs().
		GetVisibleEnemyAtPositionAndFloorOrNil(presentation.position, presentation.floor) == presentation.enemy
}

func (g *GameScene) isShadowlordSummonFizzTarget(unit map_units.MapUnit) bool {
	presentation := g.shadowlordSummonPresentation
	enemy, ok := unit.(*map_units.NPCEnemy)

	return ok && presentation != nil && presentation.enemy == enemy && g.isShadowlordSummonCurrent(presentation)
}

func (g *GameScene) clearShadowlordSummonPresentation(presentation *shadowlordSummonPresentation) {
	if presentation == nil {
		return
	}
	if presentation.dissolveID != 0 {
		g.StopTileDissolve(presentation.dissolveID)
		presentation.dissolveID = 0
	}
	if presentation.started {
		g.gameState.SystemCallbacks.Audio.Stop(game_state.SoundShadowlordSummon).Now()
		presentation.started = false
	}
	if g.shadowlordSummonPresentation == presentation {
		g.shadowlordSummonPresentation = nil
	}
}

// ThenShadowlordPresenceEffect returns a builder modifier for Shadowlord presence UI effects.
// Handles both single Shadowlord (towns) and multiple (Stonegate).
// Returns a function that chains effects onto an existing event queue builder.
// Town presence presentation follows ref:SHADOW-0002.
//
// Sequence:
//  1. Play ominous sound effect
//  2. Animate slow dark purple/red screen flash (via PartyStateBuilder.WithFlash)
//  3. Show stacked message dialog AFTER flash completes
//  4. Wait for any key to dismiss dialog
//  5. Cleanup
func (g *GameScene) ThenShadowlordPresenceEffect(lords []references.Shadowlord) func(
	*event_queue.SequentialEventBuilder,
) *event_queue.SequentialEventBuilder {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		if len(lords) == 0 {
			return q
		}

		var presenceDialog *widgets.MessageDialog

		return q.
			Then(func() {
				g.shadowlordPresenceActive = true
				g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundShadowlordPresence)
			}).ThenSequence(

			game_state.NewPartyStateBuilder(g.gameState).
				WithFlash(
					shadowlordFlashDurationMs,
					shadowlordFlashR,
					shadowlordFlashG,
					shadowlordFlashB,
					shadowlordFlashMaxIntensity,
				).
				Build()).
			Then(func() {
				// Calculate dynamic height based on Shadowlord count:
				// - Each Shadowlord message: 1 AppendLine call
				// - Blank line between each: (count - 1) lines
				// - Final blank + separate prompt: 2 lines
				// Formula: count + (count-1) + 2 = (count * 2) + 1
				// 3 lords = 7, 2 lords = 5, 1 lord = 3
				linesNeeded := (len(lords) * 2) + 1

				// 3. Create dialog AFTER animation (prevents early dismissal)
				presenceDialog = widgets.NewMessageDialog("").
					WithTextLines(linesNeeded + 1).
					WithCharsPerLine(40).
					WithSeparatePromptArea().
					WithEffects(true).
					WithVerticalPosition(defaultMessageVertical).
					WithHorizontalPosition(widgets.DialogHorizontalMapOnly).
					WithContentTextAlignment(widgets.ContentAlignCenter).
					Build()

				// Add stacked presence messages (one per Shadowlord)
				for i, lord := range lords {
					if i > 0 {
						presenceDialog.AppendLine("") // Blank line between messages
					}

					presenceDialog.AppendLine(lord.GetPresenceMessage())
				}

				presenceDialog.AppendLine("")
				presenceDialog.SetAcknowledgementPrompt(widgets.DialogAnyKeyHint)
				presenceDialog.ShowWithoutAnimation()
				g.dialogStack.PushModalDialog(presenceDialog)
				g.shadowlordPresenceDialog = presenceDialog
			}).
			// 4. Wait for any key to dismiss dialog
			WaitForAnyKey().
			Then(func() {
				g.clearShadowlordPresencePresentation()
			})
	}
}

func (g *GameScene) clearShadowlordPresencePresentation() {
	if dialog := g.shadowlordPresenceDialog; dialog != nil {
		dialog.Hide()
		g.dialogStack.RemoveWidgetInstance(dialog)
		g.shadowlordPresenceDialog = nil
	}
	if !g.shadowlordPresenceActive {
		return
	}
	g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
	g.gameState.SystemCallbacks.Audio.Stop(game_state.SoundShadowlordPresence).Now()
	g.shadowlordPresenceActive = false
}

func (g *GameScene) clearShadowlordPresentationForSceneTransition() {
	if g.shadowlordSummonPresentation != nil {
		g.clearShadowlordSummonPresentation(g.shadowlordSummonPresentation)
	}
	g.clearShadowlordPresencePresentation()
	if g.shadowlordShardActive {
		g.clearShadowlordShardPresentation()
		g.gameState.SystemCallbacks.Audio.Stop(game_state.SoundShardPulse).Now()
		g.gameState.SystemCallbacks.Audio.Stop(game_state.SoundShadowlordVictory).Now()
	}
}
