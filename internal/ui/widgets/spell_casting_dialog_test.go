package widgets

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func newCastBrowserTest(t *testing.T) (*game_state.GameState, *SpellCastingDialog, func(...ebiten.Key)) {
	t.Helper()
	setResponsiveDialogTestResolution(t, 1280, 720)
	state, _ := game_state.NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithSystemCallbacks().Build()
	state.PartyState.GetAvatar().Class = party_state.Wizard
	state.PartyState.GetAvatar().Level = 8
	state.PartyState.GetAvatar().SetMp(30)
	held := map[ebiten.Key]bool{}
	keyboard := input.NewKeyboardWithKeyStateSource(165, func(key ebiten.Key) bool { return held[key] })
	dialog := NewSpellCastingDialog(keyboard, &state.PartyState, references.ContextTown,
		state.CurrentSpellCatalog(), state.ValidateSpellTargetingPrerequisites)
	press := func(keys ...ebiten.Key) {
		for _, key := range keys {
			held[key] = true
		}
		dialog.Update()
		clear(held)
		keyboard.ClearFrameConsumption()
		dialog.Update()
	}
	return state, dialog, press
}

func TestSpellCastingDialogUsesAuthoritativeReadiness(t *testing.T) {
	state, dialog, _ := newCastBrowserTest(t)
	dialog.browser.selectID(references.InLor.ID())
	state.PartyState.Inventory.Spells.Set(references.InLor, 1)
	caster := state.PartyState.GetAvatar()
	caster.Class = party_state.Fighter
	caster.SetMp(0)
	valid, message := dialog.getSpellValidationStatus()
	require.False(t, valid)
	require.Equal(t, game_state.SpellNoMagicAbilityMessage, message)
	caster.Class = party_state.Wizard
	valid, message = dialog.getSpellValidationStatus()
	require.False(t, valid)
	require.Equal(t, game_state.SpellNoMPMessage, message)
	dialog.attemptCast()
	require.False(t, dialog.IsDismissed())
	require.Equal(t, game_state.SpellNoMPMessage, dialog.notice)
	require.False(t, dialog.HasPendingCastRequest())
	caster.SetMp(10)
	dialog.attemptCast()
	require.True(t, dialog.HasPendingCastRequest())
	require.Equal(t, references.InLor, dialog.ConsumePendingCastRequest().Spell)
	require.Equal(t, uint16(1), state.PartyState.Inventory.Spells.Get(references.InLor))
	require.Equal(t, byte(10), caster.GetCurrentMp())
}

func TestSpellCastingDialogLockedCasterAndNonCasterBrowsing(t *testing.T) {
	state, dialog, press := newCastBrowserTest(t)
	dialog.LockCasterSelection(2)
	press(ebiten.KeyTab)
	dialog.selectCaster(0)
	require.Equal(t, 2, dialog.selectedCasterIdx)
	state.PartyState.GetCharacter(2).Class = party_state.Fighter
	dialog.ensureLayout()
	dialog.prepareStatus(dialog.view.detailRect.Inset(dialog.view.padding))
	require.NotEmpty(t, dialog.view.body.GetOutputStr(false), "non-casters may still browse and prepare")
	require.Contains(t, dialog.view.status.GetOutputStr(false), "No magical ability")
}

func TestSpellCastingDialogMixIntentPreservesSelectionAndRefreshesStock(t *testing.T) {
	state, dialog, press := newCastBrowserTest(t)
	press(ebiten.KeyA)
	press(ebiten.KeyN)
	require.Equal(t, references.AnNox, dialog.GetSelectedSpell())
	press(ebiten.KeyControl, ebiten.KeyM)
	require.True(t, dialog.HasPendingMixRequest())
	require.Equal(t, "AN", dialog.browser.syllables, "Ctrl+M cannot type Mani")
	require.Equal(t, references.AnNox, dialog.ConsumePendingMixSpell())
	state.PartyState.Inventory.Spells.Set(references.AnNox, 5)
	dialog.RefreshCatalog(state.CurrentSpellCatalog())
	require.Equal(t, references.AnNox, dialog.GetSelectedSpell())
	require.Equal(t, "AN", dialog.browser.syllables)
	require.Equal(t, uint16(5), dialog.browser.current().Prepared)
	dialog.currentContext = references.ContextCombat
	press(ebiten.KeyControl, ebiten.KeyM)
	require.False(t, dialog.HasPendingMixRequest())
	require.Equal(t, "Cannot mix during combat.", dialog.notice)
}

func TestSpellCastingDialogDebugOverridesUseCoreValidation(t *testing.T) {
	state, dialog, _ := newCastBrowserTest(t)
	state.PartyState.GetAvatar().SetMp(0)
	state.PartyState.GetAvatar().Level = 1
	state.PartyState.Inventory.Spells.Set(references.AnTym, 0)
	dialog.browser.selectID(references.AnTym.ID())
	state.DebugOptions.UnlimitedCastingResources = true
	state.DebugOptions.BypassSpellLevelRestrictions = true
	valid, _ := dialog.getSpellValidationStatus()
	require.True(t, valid)
	state.DebugOptions.BypassSpellLevelRestrictions = false
	valid, message := dialog.getSpellValidationStatus()
	require.False(t, valid)
	require.Equal(t, "Level too low!", message)
}
