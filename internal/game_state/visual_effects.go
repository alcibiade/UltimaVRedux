package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
)

// Visual flash effect constants for rest damage feedback
// These constants define the color tints and timing for various damage types
// during rest sequences (bed rest, camping, etc.)
const (
	poisonFeedbackQueueName     = "poison_feedback"
	deathFeedbackQueueName      = "death_feedback"
	starvationFeedbackQueueName = "starvation_feedback"

	// Death flash (dark gray)
	deathFlashRed        = 0.3
	deathFlashGreen      = 0.3
	deathFlashBlue       = 0.3
	deathFlashIntensity  = 0.6
	deathFlashDurationMs = 300

	// Poison flash (green tint)
	poisonModifier        = 3
	poisonFlashRed        = 0.2 / poisonModifier
	poisonFlashGreen      = 0.6 / poisonModifier
	poisonFlashBlue       = 0.2 / poisonModifier
	poisonFlashIntensity  = 0.4 / poisonModifier
	poisonFlashDurationMs = 200

	// Starvation flash (brown/yellow tint)
	starvationFlashRed        = 0.6
	starvationFlashGreen      = 0.4
	starvationFlashBlue       = 0.1
	starvationFlashIntensity  = 0.5
	starvationFlashDurationMs = 250
)

// ShowStarvationFeedback provides consistent starvation visual/audio feedback
// Used during bed rest and camping when party has no food
func (g *GameState) ShowStarvationFeedback() {
	g.EventQueueManager.TriggerFeedback(
		starvationFeedbackQueueName,
		event_queue.PriorityNormal,
		func(sequence *event_queue.SequentialEventBuilder) {
			sequence.ThenSequence(
				NewPartyStateBuilder(g).
					ToAllParty().
					WithFlash(
						starvationFlashDurationMs,
						starvationFlashRed,
						starvationFlashGreen,
						starvationFlashBlue,
						starvationFlashIntensity,
					).
					WithSound(SoundStarvation).
					Build())
		},
	)
}

// ShowDeathFeedback provides consistent death visual/audio feedback
// Used when characters die during rest sequences
func (g *GameState) ShowDeathFeedback() {
	g.EventQueueManager.TriggerFeedback(
		deathFeedbackQueueName,
		event_queue.PriorityHigh,
		func(sequence *event_queue.SequentialEventBuilder) {
			sequence.ThenSequence(
				NewPartyStateBuilder(g).
					ToAllParty().
					WithFlash(
						deathFlashDurationMs,
						deathFlashRed,
						deathFlashGreen,
						deathFlashBlue,
						deathFlashIntensity,
					).
					WithSound(SoundDeath).
					Build())
		},
	)
}

// ShowPoisonFeedback provides consistent poison visual/audio feedback
// Used when poison damage is applied during bed rest
func (g *GameState) ShowPoisonFeedback() {
	g.EventQueueManager.TriggerFeedback(
		poisonFeedbackQueueName,
		event_queue.PriorityNormal,
		func(sequence *event_queue.SequentialEventBuilder) {
			sequence.First(func() {}).
				Then(func() {
					g.SystemCallbacks.Message.AddRowStr("Oof!")
				}).ThenSequence(

				NewPartyStateBuilder(g).
					ToAllParty().
					WithFlash(
						poisonFlashDurationMs,
						poisonFlashRed,
						poisonFlashGreen,
						poisonFlashBlue,
						poisonFlashIntensity,
					).
					WithSound(SoundWhiteNoiseDamage).
					Build())
		},
	)
}
