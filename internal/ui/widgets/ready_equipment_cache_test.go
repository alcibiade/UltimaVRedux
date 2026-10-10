package widgets

import (
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/profiling"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestReadyRetainsCurrentViewAndMatchesFreshComposition(t *testing.T) {
	state, dialog, held := newReadyReviewFixture(t)
	dialog.spriteSheet = &sprites.SpriteSheet{SpriteImage: sprites.NewPngSprite(sprites.TileAtlasPNG())}
	t.Cleanup(dialog.Dismiss)
	metrics := profiling.Enable()
	t.Cleanup(profiling.Disable)
	actual, expected := ebiten.NewImage(1280, 720), ebiten.NewImage(1280, 720)
	defer actual.Deallocate()
	defer expected.Deallocate()
	before := state.PartyState.Inventory.LockerSnapshot()
	rngCalls := state.GetRNG().Calls()
	dialog.Draw(actual)
	content := dialog.content
	builds := dialog.contentBuilds
	counters := metrics.Snapshot().Counters
	for range 60 {
		// Other panels can evict every shared text image without invalidating
		// Ready's composed current view or causing it to access the text cache.
		text.ClearTextCache()
		dialog.Update()
		dialog.Draw(actual)
	}
	require.Equal(t, builds, dialog.contentBuilds)
	require.Same(t, content, dialog.content)
	require.Equal(t, counters["text_misses"], metrics.Snapshot().Counters["text_misses"],
		"unchanged Ready must not generate text images even after shared-cache eviction")
	require.Equal(t, counters["font_creates"], metrics.Snapshot().Counters["font_creates"])
	require.Equal(t, before, state.PartyState.Inventory.LockerSnapshot())
	require.Equal(t, rngCalls, state.GetRNG().Calls())
	mutations := []struct {
		name  string
		apply func()
	}{
		{"selection", func() { pressReadyNavigation(dialog, held, ebiten.KeyDown, false) }},
		{"character", func() { dialog.selectCharacter(1) }},
		{"warning", func() { dialog.warningMessage = "Cannot equip this item." }},
		{"ammunition", func() { state.PartyState.Inventory.Equipment.Set(references.Arrows, 17) }},
		{"live strength", func() { dialog.getSelectedCharacter().Strength++ }},
		{"live name", func() { dialog.getSelectedCharacter().Name[0] = 'Z' }},
		{"live portrait", func() { dialog.getSelectedCharacter().Class = party_state.Wizard }},
		{"live slot", func() { dialog.getSelectedCharacter().SetEquippedItem(references.SlotHelm, references.LeatherHelm) }},
		{"locked controls", func() { dialog.request.LockCharacter = true }},
		{"rebuilt candidates", dialog.updateAvailableItems},
		{"paged details", func() {
			dialog.getSelectedItem().CanEquip = false
			dialog.getSelectedItem().EquipError = strings.Repeat("Equipment unavailable. ", 40)
			dialog.cycleDetailPage(1)
		}},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			builds = dialog.contentBuilds
			mutation.apply()
			actual.Clear()
			dialog.Draw(actual)
			require.Equal(t, builds+1, dialog.contentBuilds, "changed presentation must repaint once")
			expected.Clear()
			dialog.drawParentContent(expected)
			got, want := make([]byte, 1280*720*4), make([]byte, 1280*720*4)
			actual.ReadPixels(got)
			expected.ReadPixels(want)
			require.Equal(t, want, got, "cached non-zero-origin composition must match a fresh direct render")
			dialog.Draw(actual)
			require.Equal(t, builds+1, dialog.contentBuilds, "changed view must settle immediately")
			require.Same(t, content, dialog.content, "selection changes repaint one owned target")
		})
	}
}

func TestReadyCacheConfirmationResizeAndDismissal(t *testing.T) {
	_, dialog, _ := newReadyReviewFixture(t)
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	dialog.Draw(screen)
	content, builds := dialog.content, dialog.contentBuilds
	dialog.showStrengthBreakdown()
	require.NotNil(t, dialog.confirmation)
	dialog.Draw(screen)
	require.Same(t, content, dialog.content)
	require.Equal(t, builds, dialog.contentBuilds, "child confirmation must not repaint the parent")
	dialog.confirmation = nil
	setResponsiveDialogTestResolution(t, 1788, 1005)
	dialog.ensureLayout()
	require.Nil(t, dialog.content, "resize releases the old target")
	resized := ebiten.NewImage(1788, 1005)
	defer resized.Deallocate()
	dialog.Draw(resized)
	require.NotSame(t, content, dialog.content)
	require.Equal(t, builds+1, dialog.contentBuilds)
	dialog.Dismiss()
	dialog.Dismiss()
	require.Nil(t, dialog.content, "dismissal is idempotent and releases graphics ownership")
	dialog.Draw(resized)
	require.Nil(t, dialog.content, "a dismissed dialog cannot recreate its target")
}
