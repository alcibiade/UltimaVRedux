package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/camping"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
)

// DungeonCampingWatchEligibleCount returns active Good or Poisoned members.
func (g *GameState) DungeonCampingWatchEligibleCount() int {
	count := 0
	for index := range party_state.MaxActiveParty {
		character := g.PartyState.GetCharacter(index)
		if character.PartyMap == party_state.PartyMapActive &&
			(character.IsGood() || character.IsPoisoned()) {
			count++
		}
	}

	return count
}

// AddDungeonCampingSequenceToBuilder adds source-style dungeon rest and its
// retained combat handoff to the owning dungeon command workflow.
func (g *GameState) AddDungeonCampingSequenceToBuilder(
	queue *event_queue.SequentialEventBuilder,
	scene *camping.CampingScene,
	plan *DungeonCampingPlan,
	completionCleanup func(),
) {
	state := g.newCampingState(scene, false, false)
	queue.Then(func() {
		scene.StartCamping()
		if err := g.transitionToCampingMap(plan.combatMap); err != nil {
			panic("failed to transition to generated dungeon camping map")
		}
		g.SystemCallbacks.Screen.SetMessageDialogFadeDurations(
			WandererFadeAnimationMs,
			WandererFadeAnimationMs,
		)
		g.initializeCampingPartyRings(state)
		g.setupCampingScene(scene)
		g.SystemCallbacks.Audio.Play(SoundWind).Loop().Start()
	}).Wait(WandererFadeAnimationMs).
		Then(func() {
			g.SystemCallbacks.Message.AddRowStr("Setting camp...")
			g.setupBardGuard(scene, state)
			g.SystemCallbacks.Message.AddRowStr("")
			g.SystemCallbacks.Message.AddRowStr("Zzzzzzz...")
		}).If(func() bool { return state.BardMusicPlaying }, func(sequence *event_queue.SequentialEventBuilder) {
		// A leading event carries the conditional sub-chain's wait to the finish action.
		sequence.Then(func() {}).Wait(BardMusicDurationMs).Then(func() {
			g.finishBardGuardPerformance(state)
		})
	}).
		Then(scene.BeginSleeping).
		RepeatWithRuntimeDelay(scene.TotalCycles, func() int64 { return campingCycleDelay(scene) }, func(_ int) bool {
			if state.Interrupted || !scene.AdvanceExactCycle() {
				return false
			}
			state.CurrentCycle = scene.CurrentCycle

			return !g.processDungeonCampingCycle(scene, state, plan)
		}).
		Then(func() {
			scene.EndSleeping()
			if state.Interrupted {
				g.SystemCallbacks.Screen.CreateAndPushMessageDialog()

				return
			}
			g.SystemCallbacks.Screen.CreateAndPushCampingResultDialog(g.applyCampingHealing(scene))
		}).
		If(func() bool { return !state.Interrupted }, func(sequence *event_queue.SequentialEventBuilder) {
			sequence.Then(func() {
				g.SystemCallbacks.Screen.ShowPressAnyKeyPrompt()
			}).WaitForAnyKey().Then(func() {
				g.SystemCallbacks.Screen.ClearMessagePrompt()
				g.SystemCallbacks.Screen.HideMessageDialog()
			}).Wait(int64(g.SystemCallbacks.Screen.GetMessageDialogFadeOutDuration())).
				Then(func() { g.SystemCallbacks.Screen.PopMessageDialog() })
		}).Finally(func() {
		g.finalizeCampingSequence(scene, state, completionCleanup)
	})
}

func (g *GameState) processDungeonCampingCycle(
	scene *camping.CampingScene,
	state *CampingState,
	plan *DungeonCampingPlan,
) bool {
	g.ProcessRegenerationRingSweep()

	if g.DateTime.Hour != state.AmbushCheckHour {
		if g.CheckForAmbushDuringHour(false) {
			if err := g.QueueDungeonCampingAmbush(scene, plan, state, state.ActivePlayerIndex); err != nil {
				logging.Error(logging.CategoryGameState, "Unable to prepare dungeon camping combat: %v", err)
				g.SystemCallbacks.UI.ShowActionFeedback("Combat could not begin.", true)
			} else {
				state.AmbushedAt = scene.CurrentCycle
				g.SystemCallbacks.Audio.PlaySoundEffect(SoundAmbushAlert)
				g.SystemCallbacks.Message.AddRowStr("")
				g.SystemCallbacks.Message.AddRowStr("Ambushed!")
			}
			state.Interrupted = true
			scene.InterruptCamping()

			return true
		}
		state.AmbushCheckHour = g.DateTime.Hour
	}

	g.advanceDungeonCampingMinutes(camping.DungeonMinutesPerCycle)
	g.tryGuardWander(state)

	return false
}
