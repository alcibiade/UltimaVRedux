package game_state

import (
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
)

// Visual effects constants for PartyStateBuilder flash/shake helpers
const (
	// Flash effect durations (milliseconds)
	flashStandardDurationMs = 200 // Used by FlashRed, FlashWater, FlashGray
	flashPoisonDurationMs   = 400 // Used by FlashGreen (longer for subtle effect)

	// Flash effect intensities
	flashStandardIntensity = 0.7 // Standard damage flash intensity
	flashPoisonIntensity   = 0.3 // Subtle poison mist intensity

	// Flash RGB values (floats 0.0-1.0)
	flashRedR = 1.0 // Fire/lava red
	flashRedG = 0.0
	flashRedB = 0.0

	flashWaterR = 0.0 // Water blue-cyan
	flashWaterG = 0.3
	flashWaterB = 1.0

	flashGrayR = 0.5 // Impact gray
	flashGrayG = 0.5
	flashGrayB = 0.5

	flashGreenR = 0.2 // Poison green
	flashGreenG = 0.6
	flashGreenB = 0.2

	flashOrangeR = 1.0 // Explosion/acid orange
	flashOrangeG = 0.5
	flashOrangeB = 0.0

	// Potion effect flash colors
	flashPurpleR = 0.6 // Sleep purple
	flashPurpleG = 0.0
	flashPurpleB = 0.8

	flashWhiteR = 1.0 // Awaken white
	flashWhiteG = 1.0
	flashWhiteB = 1.0

	// Potion effect durations and intensities
	flashInvertedDurationMs = 500 // Inverted flash for healing/cure (half second - more noticeable)
	flashSleepDurationMs    = 300 // Longer drowsy effect
	flashAwakenDurationMs   = 150 // Quick wake-up flash

	flashSleepIntensity  = 0.4 // Subtle drowsy effect
	flashAwakenIntensity = 0.6 // Bright wake-up flash

	// Shake effect parameters
	shakeModerateDurationMs = 200 // Standard impact shake
	shakeModeratePixels     = 6.0

	shakeStrongDurationMs = 300 // Severe impact shake
	shakeStrongPixels     = 10.0

	screenShakeCycles = 15.0
)

// PartyStateBuilder provides fluent API for applying damage/healing with visual effects
type PartyStateBuilder struct {
	gs *GameState

	// Damage configuration
	damageMin         int
	damageMax         int
	originalD30Damage bool

	// Status effects
	applyPoison bool // Whether to apply poison status to targets

	// Targeting
	targetIndices []int

	// Visual effects
	flashLayer    *FlashLayer
	shakeLayer    *ShakeLayer
	invertedLayer *InvertedLayer // Inverted flash effect (healing/cure)
	soundEffect   SoundEffect
	hasSound      bool
}

// InvertedLayer defines inverted flash effect parameters (for healing/cure effects)
type InvertedLayer struct {
	Duration uint64 // Duration in milliseconds
}

// FlashLayer defines flash effect parameters
type FlashLayer struct {
	Duration  uint64
	R, G, B   float64
	Intensity float64
}

// ShakeLayer defines shake effect parameters
type ShakeLayer struct {
	Duration uint64
	Pixels   float64
}

// NewPartyStateBuilder creates a new party state builder
func NewPartyStateBuilder(gs *GameState) *PartyStateBuilder {
	return &PartyStateBuilder{
		gs: gs,
	}
}

// WithRandomDamage sets random damage range
func (b *PartyStateBuilder) WithRandomDamage(minimum, maximum int) *PartyStateBuilder {
	b.damageMin = minimum
	b.damageMax = maximum

	return b
}

// WithOriginalD30Damage uses transformed d30 for source-confirmed damage paths.
// See ref:RNG-0001.
func (b *PartyStateBuilder) WithOriginalD30Damage() *PartyStateBuilder {
	b.originalD30Damage = true

	return b
}

// WithFixedDamage sets fixed damage amount
func (b *PartyStateBuilder) WithFixedDamage(amount int) *PartyStateBuilder {
	b.damageMin = amount
	b.damageMax = amount

	return b
}

// ToAllParty targets all living party members
func (b *PartyStateBuilder) ToAllParty() *PartyStateBuilder {
	b.targetIndices = b.getAllLivingIndices()

	return b
}

// ToCharacter targets specific character by index
func (b *PartyStateBuilder) ToCharacter(index int) *PartyStateBuilder {
	b.targetIndices = []int{index}

	return b
}

// ToCharacters targets multiple specific characters by indices
// Used for skill-based hazards (climbing falls, dexterity checks, etc.)
func (b *PartyStateBuilder) ToCharacters(indices []int) *PartyStateBuilder {
	b.targetIndices = indices

	return b
}

// WithFlash adds flash effect with specified parameters
func (b *PartyStateBuilder) WithFlash(durationMs int, r, g, blue, intensity float64) *PartyStateBuilder {
	b.flashLayer = &FlashLayer{
		Duration:  uint64(durationMs),
		R:         r,
		G:         g,
		B:         blue,
		Intensity: intensity,
	}

	return b
}

// WithShake adds shake effect with specified parameters
func (b *PartyStateBuilder) WithShake(durationMs int, pixels float64) *PartyStateBuilder {
	b.shakeLayer = &ShakeLayer{
		Duration: uint64(durationMs),
		Pixels:   pixels,
	}

	return b
}

// WithSound adds sound effect to play with the damage
func (b *PartyStateBuilder) WithSound(effect SoundEffect) *PartyStateBuilder {
	b.soundEffect = effect
	b.hasSound = true

	return b
}

// WithPoison marks targets to be poisoned (instead of/in addition to damage)
// Used for GAS/POISON traps and swamp poison effects.
func (b *PartyStateBuilder) WithPoison() *PartyStateBuilder {
	b.applyPoison = true

	return b
}

// Flash helper functions for common colors and patterns

// FlashRed adds red flash effect (fire/lava damage pattern)
// Standard: 200ms duration, 0.7 intensity
func (b *PartyStateBuilder) FlashRed() *PartyStateBuilder {
	return b.WithFlash(flashStandardDurationMs, flashRedR, flashRedG, flashRedB, flashStandardIntensity)
}

// FlashWater adds blue-cyan flash effect (water splash pattern)
// Standard: 200ms duration, 0.7 intensity
func (b *PartyStateBuilder) FlashWater() *PartyStateBuilder {
	return b.WithFlash(flashStandardDurationMs, flashWaterR, flashWaterG, flashWaterB, flashStandardIntensity)
}

// FlashGray adds gray flash effect (fall/impact damage pattern)
// Standard: 200ms duration, 0.7 intensity
func (b *PartyStateBuilder) FlashGray() *PartyStateBuilder {
	return b.WithFlash(flashStandardDurationMs, flashGrayR, flashGrayG, flashGrayB, flashStandardIntensity)
}

// FlashLongGray adds an extended gray flash effect (fall/impact damage pattern).
// Standard: 1000ms duration, 0.7 intensity.
func (b *PartyStateBuilder) FlashLongGray() *PartyStateBuilder {
	return b.WithFlash(flashStandardDurationMs*5, flashGrayR, flashGrayG, flashGrayB, flashStandardIntensity)
}

// FlashGreen adds green flash effect (poison/swamp pattern)
// Standard: 400ms duration, 0.3 intensity (subtle sickly green mist)
func (b *PartyStateBuilder) FlashGreen() *PartyStateBuilder {
	return b.WithFlash(flashPoisonDurationMs, flashGreenR, flashGreenG, flashGreenB, flashPoisonIntensity)
}

// FlashOrange adds orange flash effect (explosion/acid damage pattern)
// Standard: 200ms duration, 0.7 intensity
func (b *PartyStateBuilder) FlashOrange() *PartyStateBuilder {
	return b.WithFlash(flashStandardDurationMs, flashOrangeR, flashOrangeG, flashOrangeB, flashStandardIntensity)
}

// FlashPurple adds purple flash effect (sleep/drowsy pattern)
// 300ms duration, 0.4 intensity (subtle drowsy effect)
func (b *PartyStateBuilder) FlashPurple() *PartyStateBuilder {
	return b.WithFlash(flashSleepDurationMs, flashPurpleR, flashPurpleG, flashPurpleB, flashSleepIntensity)
}

// FlashWhite adds white flash effect (awaken/cleanse pattern)
// 150ms duration, 0.6 intensity (quick bright wake-up flash)
func (b *PartyStateBuilder) FlashWhite() *PartyStateBuilder {
	return b.WithFlash(flashAwakenDurationMs, flashWhiteR, flashWhiteG, flashWhiteB, flashAwakenIntensity)
}

// FlashInverted adds inverted screen flash effect (healing/cure pattern)
// 500ms duration, inverts screen colors using triangle wave (0→1→0)
func (b *PartyStateBuilder) FlashInverted() *PartyStateBuilder {
	b.invertedLayer = &InvertedLayer{
		Duration: uint64(flashInvertedDurationMs),
	}

	return b
}

// Shake helper functions for common amplitudes

// ShakeModerate adds moderate shake effect (standard impact)
// 200ms duration, 6.0 pixel range
func (b *PartyStateBuilder) ShakeModerate() *PartyStateBuilder {
	return b.WithShake(shakeModerateDurationMs, shakeModeratePixels)
}

// ShakeStrong adds strong shake effect (severe impact)
// 300ms duration, 10.0 pixel range
func (b *PartyStateBuilder) ShakeStrong() *PartyStateBuilder {
	return b.WithShake(shakeStrongDurationMs, shakeStrongPixels)
}

// Build compiles builder into event queue modifier function
func (b *PartyStateBuilder) Build() func(*event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		// Apply damage and setup effects immediately
		q = q.Then(func() {
			b.applyDamage()
			b.setupEffects()
		})

		// Animate effects if any exist
		if b.hasVisualEffects() {
			maxDuration := b.calculateMaxDuration()
			q = q.Animate(maxDuration, func(progress float32) {
				b.animateEffects(progress, maxDuration)
			}).Then(func() {
				b.clearEffects()
			})
		}

		return q
	}
}

// applyDamage applies damage and/or poison to targeted characters
// Matches original Ultima V pattern: only affects party members with InTheParty status
func (b *PartyStateBuilder) applyDamage() {
	changes := transientStatusChanges{}
	for _, idx := range b.targetIndices {
		char := b.gs.PartyState.GetCharacter(idx)

		// Skip if not in active party
		if char.PartyMap != party_state.PartyMapActive {
			continue
		}

		// Skip dead party members
		if char.IsDead() {
			continue
		}

		// Apply damage if configured
		if b.originalD30Damage || b.damageMin > 0 || b.damageMax > 0 {
			var damage int
			if b.originalD30Damage {
				damage = b.gs.RollOriginalD30()
			} else {
				damage = b.gs.RandomIntInRange(b.damageMin, b.damageMax)
			}
			char.SubtractHp(uint16(damage))
		}

		// Apply poison if configured (skip if already poisoned)
		if b.applyPoison && !char.IsPoisoned() {
			statusBefore := char.GetStatus()
			char.SetStatus(party_state.Poisoned)
			changes.observePoison(statusBefore, char)
		}
	}
	b.gs.publishTransientStatusChanges(changes)
}

// getAllLivingIndices returns indices of all characters (filtering happens in applyDamage)
func (b *PartyStateBuilder) getAllLivingIndices() []int {
	indices := make([]int, b.gs.PartyState.GetCharacterCount())
	for i := range indices {
		indices[i] = i
	}

	return indices
}

// hasVisualEffects checks if any visual effects are configured
func (b *PartyStateBuilder) hasVisualEffects() bool {
	return b.flashLayer != nil || b.shakeLayer != nil || b.invertedLayer != nil
}

// calculateMaxDuration returns the longest effect duration
func (b *PartyStateBuilder) calculateMaxDuration() uint64 {
	maxDur := uint64(0)
	if b.flashLayer != nil && b.flashLayer.Duration > maxDur {
		maxDur = b.flashLayer.Duration
	}
	if b.shakeLayer != nil && b.shakeLayer.Duration > maxDur {
		maxDur = b.shakeLayer.Duration
	}
	if b.invertedLayer != nil && b.invertedLayer.Duration > maxDur {
		maxDur = b.invertedLayer.Duration
	}

	return maxDur
}

// setupEffects initializes visual effects (flash color, sound)
func (b *PartyStateBuilder) setupEffects() {
	if b.flashLayer != nil {
		b.gs.SystemCallbacks.Visual.SetScreenFlashColor(
			b.flashLayer.R, b.flashLayer.G, b.flashLayer.B)
	}
	if b.hasSound {
		b.gs.SystemCallbacks.Audio.PlaySoundEffect(b.soundEffect)
	}
}

// animateEffects updates visual effects based on animation progress
func (b *PartyStateBuilder) animateEffects(progress float32, maxDuration uint64) {
	currentMs := uint64(float32(maxDuration) * progress)

	// Flash animation (triangle wave: 0 → max → 0)
	if b.flashLayer != nil {
		if currentMs <= b.flashLayer.Duration {
			flashProgress := float32(currentMs) / float32(b.flashLayer.Duration)
			intensity := b.calculateFlashIntensity(flashProgress, b.flashLayer.Intensity)
			b.gs.SystemCallbacks.Visual.SetScreenFlashIntensity(intensity)
		} else {
			b.gs.SystemCallbacks.Visual.SetScreenFlashIntensity(0.0)
		}
	}

	// Shake animation (deterministic oscillation with dampening)
	if b.shakeLayer != nil {
		if currentMs <= b.shakeLayer.Duration {
			shakeProgress := float32(currentMs) / float32(b.shakeLayer.Duration)
			b.applyShake(shakeProgress)
		} else {
			b.gs.SystemCallbacks.Visual.SetScreenShakeOffset(0.0, 0.0)
		}
	}

	// Inverted flash animation (triangle wave: 0 → 1 → 0)
	if b.invertedLayer != nil {
		if currentMs <= b.invertedLayer.Duration {
			invertProgress := float32(currentMs) / float32(b.invertedLayer.Duration)
			intensity := b.calculateFlashIntensity(invertProgress, 1.0) // Max 1.0 for full inversion
			b.gs.SystemCallbacks.Visual.SetScreenInvert(intensity, 0)   // durationMs=0, we control timing
		} else {
			b.gs.SystemCallbacks.Visual.SetScreenInvert(0.0, 0)
		}
	}
}

// calculateFlashIntensity computes triangle wave intensity (0 → max → 0)
func (b *PartyStateBuilder) calculateFlashIntensity(progress float32, maxIntensity float64) float64 {
	if progress < 0.5 {
		return maxIntensity * float64(progress*2.0)
	}

	return maxIntensity * float64(2.0-progress*2.0)
}

// applyShake applies screen shake with dampening.
func (b *PartyStateBuilder) applyShake(progress float32) {
	dampen := 1.0 - float64(progress)
	offsetX, offsetY := calculateScreenShakeOffset(progress, b.shakeLayer.Pixels*dampen)
	b.gs.SystemCallbacks.Visual.SetScreenShakeOffset(offsetX, offsetY)
}

func calculateScreenShakeOffset(progress float32, amplitude float64) (float64, float64) {
	phase := float64(progress) * screenShakeCycles * 2 * math.Pi

	return math.Sin(phase) * amplitude, math.Cos(phase) * amplitude
}

// clearEffects resets all visual effects to clean state
func (b *PartyStateBuilder) clearEffects() {
	b.gs.SystemCallbacks.Visual.SetScreenFlashIntensity(0.0)
	b.gs.SystemCallbacks.Visual.SetScreenShakeOffset(0.0, 0.0)
	b.gs.SystemCallbacks.Visual.SetScreenInvert(0.0, 0)
}
