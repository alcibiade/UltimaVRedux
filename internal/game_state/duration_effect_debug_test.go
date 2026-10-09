package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func newDurationEffectTestState(t *testing.T) *GameState {
	t.Helper()
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	state.SystemCallbacks.UI.ShowActionFeedback = func(string, bool) {}

	return state
}

func TestDurationEffectStatusIsDetachedAndReadOnly(t *testing.T) {
	state := newDurationEffectTestState(t)
	status := state.GetDurationEffectStatus()
	if status.Active || status.EffectCode != "" || status.TurnsRemaining != 0 || status.MagicLightActive {
		t.Fatalf("inactive status = %+v", status)
	}

	state.ActivateProtectionEffect(20)
	status = state.GetDurationEffectStatus()
	if !status.Active || status.EffectCode != "P" || status.Name != "Protection" ||
		status.TurnsRemaining != 20 {
		t.Fatalf("active status = %+v", status)
	}
	status.TurnsRemaining = 1
	if state.ActiveDurationEffect.TurnsRemaining != 20 {
		t.Fatal("mutating detached status changed authoritative state")
	}
}

func TestDurationEffectExpirationUsesOwnedDisplayNames(t *testing.T) {
	state := newDurationEffectTestState(t)
	for effect, name := range map[DurationEffectType]string{
		DurationEffectProtection: "Protection", DurationEffectNegateMagic: "Negate Magic",
		DurationEffectTimeStop: "Time Stop", DurationEffectQuickness: "Quickness",
		DurationEffectConfusion: "Confusion",
	} {
		t.Run(name, func(t *testing.T) {
			var message string
			state.SystemCallbacks.UI.ShowSpellExpiredPopup = func(value string) { message = value }
			state.setDurationEffect(effect, 1)
			state.processDurationEffects()
			if message != name || state.HasAnyDurationEffect() {
				t.Fatalf("expiration: message=%q active=%v", message, state.HasAnyDurationEffect())
			}
		})
	}
}

func TestLightRemainsIndependentOfActiveDurationEffect(t *testing.T) {
	state := newDurationEffectTestState(t)
	state.ActivateLightEffect(240)
	if !state.MapState.Lighting.HasMagicLight() {
		t.Fatal("light activation did not enable magic light")
	}

	state.ActivateTimeStopEffect(20)
	if !state.HasActiveDurationEffect(DurationEffectTimeStop) || !state.MapState.Lighting.HasMagicLight() {
		t.Fatal("activating Time Stop replaced independent magic light")
	}

	state.ActivateLightEffect(1)
	state.ClearDurationEffect()
	state.AdvanceGameplayMinutes(1)
	status := state.GetDurationEffectStatus()
	if status.Active || status.MagicLightActive {
		t.Fatalf("expired light status = %+v", status)
	}
}
