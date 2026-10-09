package widgets

import (
	"errors"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestSaveDescriptionEnterUpdateSubmitsCommittedNativeText(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyEnter, ebiten.KeyNumpadEnter} {
		t.Run(key.String(), func(t *testing.T) {
			held := map[ebiten.Key]bool{}
			modal, field := newNativeEditorTest(t, true, "", held)
			var results []LegacyPathEntryResult
			modal.onResult = func(value string, submitted bool) {
				results = append(results, LegacyPathEntryResult{Text: value, Submitted: submitted})
			}
			field.nextInput = func() { field.commit("b_skiff") }
			modal.Update()
			require.Empty(t, results)
			held[key] = true
			modal.Update()
			modal.Update()
			require.Equal(t, []LegacyPathEntryResult{{Text: "b_skiff", Submitted: true}}, results)
			require.False(t, field.IsFocused())
		})
	}
}

func TestSaveDescriptionEnterInputErrorDoesNotDismissOrSubmit(t *testing.T) {
	field := &fakeLegacyPathTextField{handleErr: errors.New("native input failed")}
	var results []LegacyPathEntryResult
	modal := newTestLegacyPathEntryModal(field, func(value string, submitted bool) {
		results = append(results, LegacyPathEntryResult{Text: value, Submitted: submitted})
	})
	modal.saveDescription = true
	modal.SetText("b_skiff")
	var stack DialogStack
	stack.PushModalDialog(modal)
	t.Cleanup(modal.Dismiss)
	modal.keyboard.SimulateKeyPress(ebiten.KeyEnter)
	modal.Update()
	modal.Update()
	require.Empty(t, results, "input failure must not masquerade as a cancelled import")
	require.Same(t, modal, stack.PeekTopModalDialog())
	require.Equal(t, "b_skiff", modal.GetText())
	require.Equal(t, legacyEntryInputErrorMessage, modal.GetValidationMessage())
	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Dispose)
	modal.Draw(screen)
	requireImageRegionContainsColor(t, screen, modal.calculateRects().message, DialogStatusWarningColor)
	modal.keyboard.SimulateKeyPress(ebiten.KeyEscape)
	modal.Update()
	require.Equal(t, []LegacyPathEntryResult{{Text: "b_skiff", Submitted: false}}, results)
}
