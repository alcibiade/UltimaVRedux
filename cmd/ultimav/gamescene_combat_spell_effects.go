package main

import (
	"math"

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

func (g *GameScene) thenCombatScreenPulse(red, green, blue, peak float64) event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			g.gameState.SystemCallbacks.Visual.SetScreenFlashColor(red, green, blue)
		}).Animate(combatRelTymPulseMs, func(progress float32) {
			intensity := math.Sin(float64(progress)*math.Pi) * peak
			g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(intensity)
		}).Then(func() {
			g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
		})
	}
}

func (g *GameScene) thenCastCombatSanctLor(casterIdx int) event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		pulse := g.thenCombatScreenPulse(
			combatLightBlueFlashRed,
			combatLightBlueFlashGreen,
			combatLightBlueFlashBlue,
			combatRelTymPulsePeak,
		)

		return q.Then(func() {
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
			if !g.gameState.CastCombatSanctLor(casterIdx) {
				panic("Sanct Lor caster is not active in combat")
			}
		}).ThenSequence(pulse).
			Wait(combatRelTymPulseGapMs).
			ThenSequence(pulse)
	}
}

func (g *GameScene) thenCastCombatRelTym() event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		pulse := g.thenCombatScreenPulse(
			combatLightBlueFlashRed,
			combatLightBlueFlashGreen,
			combatLightBlueFlashBlue,
			combatRelTymPulsePeak,
		)

		return q.Then(func() {
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
			g.gameState.CastCombatRelTym()
		}).ThenSequence(pulse).
			Wait(combatRelTymPulseGapMs).
			ThenSequence(pulse)
	}
}

func (g *GameScene) thenCastInSanct() event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			g.gameState.CastInSanct()
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
		}).ThenSequence(g.thenCombatScreenPulse(1, 1, 1, 0.4)).
			ThenSequence(g.thenCombatScreenPulse(0, 1, 1, 0.4))
	}
}

func (g *GameScene) thenCastCombatQuasAnWis() event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			g.gameState.CastCombatQuasAnWis()
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
		}).ThenSequence(g.thenCombatScreenPulse(1, 1, 0, 0.35))
	}
}

func (g *GameScene) thenCastCombatInAn() event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			g.gameState.CastCombatInAn()
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
		}).ThenSequence(g.thenCombatScreenPulse(1, 0, 1, 0.35))
	}
}

func (g *GameScene) thenCastAnTym() event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		succeeded := false

		return q.Then(func() {
			if !g.gameState.CastAnTym() {
				g.gameState.SystemCallbacks.UI.ShowActionFeedback("Absorbed by Shadowlord!", true)

				return
			}
			succeeded = true
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
		}).Then(func() {
			if succeeded {
				g.gameState.SystemCallbacks.Visual.SetScreenFlashColor(1, 1, 1)
			}
		}).Animate(combatRelTymPulseMs, func(progress float32) {
			if succeeded {
				g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(math.Sin(float64(progress)*math.Pi) * 0.4)
			}
		}).Then(func() {
			g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
		})
	}
}

func (g *GameScene) thenCastCombatWisQuas() event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			positions := g.gameState.CastCombatWisQuas()
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundXRay)
			g.gameState.SystemCallbacks.Visual.SetScreenFlashColor(0.4, 0.9, 1.0)
			for _, position := range positions {
				g.gameState.SystemCallbacks.Visual.PlayKapowAtPosition(position)
			}
		}).Animate(combatRevealDurationMs, func(progress float32) {
			intensity := math.Sin(float64(progress)*math.Pi) * 0.65
			g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(intensity)
		}).Then(func() {
			g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
		})
	}
}

func (g *GameScene) thenCastCombatInVasPorYlem(casterIdx int) event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			g.gameState.CastCombatInVasPorYlem(casterIdx)
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundEarthquake)
		}).Animate(combatEarthquakeDurationMs, func(progress float32) {
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
