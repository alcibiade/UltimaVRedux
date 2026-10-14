package audio

import (
	"io"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

const ambientRetryMaximum = 30 * time.Second

type ambientAttempt struct {
	definition AmbientSoundDefinition
	failures   int
	next       time.Time
	err        error
	completed  bool
}

// AmbientFailure is detached diagnostic information for a currently desired
// source. Departure/suppression removes it; this is not an unbounded error log.
type AmbientFailure struct {
	Attempts int
	RetryAt  time.Time
	Message  string
}

// Failures reports source failures currently waiting for a bounded retry.
func (asm *AmbientSoundManager) Failures() map[AmbientSoundType]AmbientFailure {
	asm.mu.RLock()
	defer asm.mu.RUnlock()
	result := make(map[AmbientSoundType]AmbientFailure)
	for key, attempt := range asm.attempts {
		if attempt.err != nil {
			result[key] = AmbientFailure{Attempts: attempt.failures, RetryAt: attempt.next, Message: attempt.err.Error()}
		}
	}

	return result
}

// Update services owned voices without interpreting an absent source snapshot
// as departure. Fades, decoder failures, EOF, and retries use presentation time.
func (asm *AmbientSoundManager) Update() {
	asm.mu.Lock()
	defer asm.mu.Unlock()
	if asm.enabled {
		asm.serviceLocked()
	}
}

func (asm *AmbientSoundManager) serviceLocked() {
	admitted := asm.rankSourcesLocked()
	asm.reconcileAdmittedLocked(admitted)
	for key, instance := range asm.activeSounds {
		if err := instance.Player.Failure(); err != nil {
			asm.failAmbientLocked(key, instance.Definition, err)
			asm.retireAmbientLocked(key)

			continue
		}
		if !instance.Player.IsPlaying() {
			if instance.Definition.LoopSound {
				asm.failAmbientLocked(key, instance.Definition, io.ErrUnexpectedEOF)
				asm.retireAmbientLocked(key)

				continue
			}
			if desired, exists := asm.desired[key]; exists && sameAmbientAsset(desired.definition, instance.Definition) {
				asm.attempts[key] = ambientAttempt{definition: instance.Definition, completed: true,
					failures: 0, next: time.Time{}, err: nil}
			}
			asm.retireAmbientLocked(key)

			continue
		}
		asm.updateFade(instance)
		if instance.FadeState == AmbientFadeOut && instance.FadeGain == 0 {
			asm.retireAmbientLocked(key)

			continue
		}
		instance.Player.SetVolume(asm.effectiveVolume(instance.Volume * instance.FadeGain))
	}
	asm.startDesiredSourcesLocked(admitted)
}

func (asm *AmbientSoundManager) failAmbientLocked(
	key AmbientSoundType, definition AmbientSoundDefinition, failure error,
) {
	if desired, exists := asm.desired[key]; !exists || !sameAmbientAsset(desired.definition, definition) {
		return // A departing/replaced source cannot retain retry bookkeeping.
	}
	attempt := asm.attempts[key]
	attempt.definition, attempt.err = definition, failure
	// Saturate the count after reaching the longest delay; no arithmetic wrap.
	const maximumAttempts = 6
	attempt.failures = min(maximumAttempts, attempt.failures+1)
	delay := min(ambientRetryMaximum, time.Second<<(attempt.failures-1))
	attempt.next = asm.now().Add(delay)
	asm.attempts[key] = attempt
	logging.Warn(logging.CategoryAudio, "Ambient source %s failed; retry in %s: %v", key, delay, failure)
}

func (asm *AmbientSoundManager) retireAmbientLocked(key AmbientSoundType) {
	if err := asm.activeSounds[key].Player.Close(); err != nil {
		logging.Warn(logging.CategoryAudio, "Retire ambient source %s: %v", key, err)
	}
	delete(asm.activeSounds, key)
}
