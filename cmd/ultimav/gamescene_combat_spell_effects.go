package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

const (
	combatRevealDurationMs     = 600
	combatEarthquakeDurationMs = 1000
	combatEarthquakePixels     = 10.0
	combatRelTymPulseMs        = 180
	combatRelTymPulseGapMs     = 60
	combatRelTymPulsePeak      = 0.35
	combatLightBlueFlashRed    = 1.0 / 3.0
	combatLightBlueFlashGreen  = 1.0 / 3.0
	combatLightBlueFlashBlue   = 1.0
)

func (g *GameScene) thenScreenPulseWithTiming(timing combatPresentationTiming, red, green, blue, peak float64) event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			g.gameState.SystemCallbacks.Visual.SetScreenFlashColor(red, green, blue)
		}).Animate(timing.duration(combatRelTymPulseMs), func(progress float32) {
			intensity := math.Sin(float64(progress)*math.Pi) * peak
			g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(intensity)
		}).Then(func() {
			g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
		})
	}
}

func (g *GameScene) thenCastCombatSanctLor(casterIdx int) event_queue.Sequence {
	timing := g.captureCombatTiming()

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		pulse := g.thenScreenPulseWithTiming(timing,
			combatLightBlueFlashRed,
			combatLightBlueFlashGreen,
			combatLightBlueFlashBlue,
			combatRelTymPulsePeak,
		)

		return q.Then(g.combatTimingStep(timing, func() {
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
			if !g.gameState.CastCombatSanctLor(casterIdx) {
				panic("Sanct Lor caster is not active in combat")
			}
		})).ThenSequence(pulse).
			Wait(timing.durationMs(combatRelTymPulseGapMs)).
			ThenSequence(pulse)
	}
}

func (g *GameScene) thenCastCombatRelTym() event_queue.Sequence {
	timing := g.captureCombatTiming()

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		pulse := g.thenScreenPulseWithTiming(timing,
			combatLightBlueFlashRed,
			combatLightBlueFlashGreen,
			combatLightBlueFlashBlue,
			combatRelTymPulsePeak,
		)

		return q.Then(g.combatTimingStep(timing, func() {
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
			g.gameState.CastCombatRelTym()
		})).ThenSequence(pulse).
			Wait(timing.durationMs(combatRelTymPulseGapMs)).
			ThenSequence(pulse)
	}
}

func (g *GameScene) thenCastInSanct() event_queue.Sequence {
	timing := g.capturePresentationTiming(g.gameState.IsInCombat())

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(g.combatTimingStep(timing, func() {
			g.gameState.CastInSanct()
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
		})).ThenSequence(g.thenScreenPulseWithTiming(timing, 1, 1, 1, 0.4)).
			ThenSequence(g.thenScreenPulseWithTiming(timing, 0, 1, 1, 0.4))
	}
}

func (g *GameScene) thenCastCombatQuasAnWis() event_queue.Sequence {
	timing := g.captureCombatTiming()

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(g.combatTimingStep(timing, func() {
			g.gameState.CastCombatQuasAnWis()
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
		})).ThenSequence(g.thenScreenPulseWithTiming(timing, 1, 1, 0, 0.35))
	}
}

func (g *GameScene) thenCastCombatInAn() event_queue.Sequence {
	timing := g.captureCombatTiming()

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(g.combatTimingStep(timing, func() {
			g.gameState.CastCombatInAn()
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
		})).ThenSequence(g.thenScreenPulseWithTiming(timing, 1, 0, 1, 0.35))
	}
}

func (g *GameScene) thenCastAnTym() event_queue.Sequence {
	timing := g.capturePresentationTiming(g.gameState.IsInCombat())

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		succeeded := false

		return q.Then(g.combatTimingStep(timing, func() {
			if !g.gameState.CastAnTym() {
				g.gameState.SystemCallbacks.UI.ShowActionFeedback("Absorbed by Shadowlord!", true)

				return
			}
			succeeded = true
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
		})).Then(func() {
			if succeeded {
				g.gameState.SystemCallbacks.Visual.SetScreenFlashColor(1, 1, 1)
			}
		}).Animate(timing.duration(combatRelTymPulseMs), func(progress float32) {
			if succeeded {
				g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(math.Sin(float64(progress)*math.Pi) * 0.4)
			}
		}).Then(func() {
			g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
		})
	}
}

func (g *GameScene) thenCastCombatWisQuas() event_queue.Sequence {
	timing := g.captureCombatTiming()

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(g.combatTimingStep(timing, func() {
			positions := g.gameState.CastCombatWisQuas()
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundXRay)
			g.gameState.SystemCallbacks.Visual.SetScreenFlashColor(0.4, 0.9, 1.0)
			for _, position := range positions {
				g.gameState.SystemCallbacks.Visual.PlayKapowAtPosition(position)
			}
		})).Animate(timing.duration(combatRevealDurationMs), func(progress float32) {
			intensity := math.Sin(float64(progress)*math.Pi) * 0.65
			g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(intensity)
		}).Then(func() {
			g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
		})
	}
}

func (g *GameScene) thenCastCombatInVasPorYlem(casterIdx int) event_queue.Sequence {
	timing := g.captureCombatTiming()

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(g.combatTimingStep(timing, func() {
			g.presentCombatAreaSpellSummary(g.gameState.CastCombatEarthquakeResult(casterIdx))
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundEarthquake)
		})).Animate(timing.duration(combatEarthquakeDurationMs), func(progress float32) {
			phase := float64(progress) * math.Pi * 12
			envelope := math.Sin(float64(progress) * math.Pi)
			g.gameState.SystemCallbacks.Visual.SetScreenShakeOffset(
				math.Sin(phase)*combatEarthquakePixels*envelope,
				math.Cos(phase*1.3)*combatEarthquakePixels*envelope,
			)
		}).Then(func() {
			g.gameState.SystemCallbacks.Visual.SetScreenShakeOffset(0, 0)
		})
	}
}

func (g *GameScene) presentCombatAreaSpellSummary(result game_state.CombatAreaSpellResult) {
	if summary := combatAreaSpellSummary(result); summary != "" {
		// Resolution already committed. This is informative feedback, never a
		// casting validation failure or a second gameplay operation.
		g.gameState.SystemCallbacks.UI.ShowActionFeedback(summary, true)
	}
}

func combatAreaSpellSummary(result game_state.CombatAreaSpellResult) string {
	if result.Outcome() == game_state.CombatAreaSpellNoTargets {
		return "No targets reached."
	}
	concealed := false
	avoided, resisted, immune, zeroDamage := 0, 0, 0, 0
	for _, target := range result.Targets {
		if !target.Disclosed {
			concealed = true

			continue
		}
		switch target.Outcome {
		case game_state.CombatSpellDamaged, game_state.CombatSpellSlept, game_state.CombatSpellPoisoned:
			return "" // Per-target success already explains the cast, including status-only success.
		case game_state.CombatSpellAvoided:
			avoided++
		case game_state.CombatSpellResisted:
			resisted++
		case game_state.CombatSpellImmune:
			immune++
		case game_state.CombatSpellZeroDamage:
			zeroDamage++
		case game_state.CombatSpellUnspecified:
			panic("area spell summary requires resolved outcomes")
		}
	}
	if concealed {
		return "No visible effect."
	}
	parts := make([]string, 0, 4)
	if avoided > 0 {
		parts = append(parts, fmt.Sprintf("%d avoided", avoided))
	}
	if resisted > 0 {
		parts = append(parts, fmt.Sprintf("%d resisted", resisted))
	}
	if immune > 0 {
		parts = append(parts, fmt.Sprintf("%d immune", immune))
	}
	if zeroDamage > 0 {
		parts = append(parts, fmt.Sprintf("%d took no damage", zeroDamage))
	}

	return "No effect: " + strings.Join(parts, "; ") + "."
}
