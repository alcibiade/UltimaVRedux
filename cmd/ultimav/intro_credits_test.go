package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/bradhannah/berad-sdk/modules/berad-audio/authoring"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestAcknowledgmentsRetainOriginalAndSuppliedAudioCredits(t *testing.T) {
	recordings := audio.RecordingCredits()
	document := buildAcknowledgments(recordings)
	var content strings.Builder
	for _, paragraph := range document {
		content.WriteString(paragraph.Heading + "\n" + paragraph.Label + "\n" + paragraph.Value + "\n")
	}
	for _, value := range []string{"Original Ultima V", "IBM Version by", "John Fachini", "Herman Miller",
		"Produced and Designed by", "Lord British", "Artwork by", "Denis Loubet and Doug Wike",
		"Kenneth Arnold", "Ultima V Redux", "Project creator and developer", "Brad Hannah", "Music & Sound Attribution",
		"Source attribution not recorded.", "Processed by", "Audacity", "Special Thanks"} {
		require.Contains(t, content.String(), value)
	}
	for _, recording := range recordings {
		require.Contains(t, content.String(), recording.Name)
		if recording.Provenance.License != "unknown" {
			require.Contains(t, content.String(), recording.Provenance.License, "freeform attribution must survive verbatim")
		}
	}
	require.NotContains(t, content.String(), "ImportedFrom")
	require.NotContains(t, content.String(), "Historical bundled asset")
	require.Contains(t, document[len(document)-1].Value, "thank you")
	slices.Reverse(recordings)
	require.Equal(t, document, buildAcknowledgments(recordings))
}

func TestIntroAcknowledgmentsLeaveTitleAudioOwnershipAlone(t *testing.T) {
	scene := newIntroInputTestScene()
	manager, err := audio.NewGameAudioManagerWithConfig(nil)
	require.NoError(t, err)
	scene.audioManager = manager
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	scene.Activate()
	track := manager.GetMusicManager().GetCurrentTrack()
	require.NotEmpty(t, track)
	for range 2 {
		scene.openAcknowledgments()
		require.True(t, manager.PlaybackSnapshot().Enabled)
		require.Equal(t, track, manager.GetMusicManager().GetCurrentTrack())
		scene.keyboard.SimulateKeyPress(ebiten.KeyEscape)
		require.NoError(t, scene.Update(nil))
		require.True(t, manager.PlaybackSnapshot().Enabled)
		require.Equal(t, track, manager.GetMusicManager().GetCurrentTrack())
		require.Zero(t, scene.dialogStack.Count())
	}
}

func TestIntroAcknowledgmentsHeldEnterEscapeNeverLeakAcrossModal(t *testing.T) {
	scene := newIntroInputTestScene()
	scene.nCurrentSelection = introChoiceAcknowledgments
	held := map[ebiten.Key]bool{}
	scene.keyboard = input.NewKeyboardWithKeyStateSource(-1, func(key ebiten.Key) bool { return held[key] })
	held[ebiten.KeyEnter] = true
	scene.keyboard.SetAllowKeyPressImmediately()
	require.NoError(t, scene.Update(nil))
	dialog := scene.dialogStack.PeekTopModalDialog()
	require.IsType(t, &widgets.CreditsDialog{}, dialog)
	// Cross the title's ten-second idle threshold while Enter remains held.
	for range 10_000/introUpdateInterval + 2 {
		require.NoError(t, scene.Update(nil))
		require.Same(t, dialog, scene.dialogStack.PeekTopModalDialog())
	}
	clear(held)
	require.NoError(t, scene.Update(nil))
	held[ebiten.KeyEscape] = true
	scene.keyboard.SetAllowKeyPressImmediately()
	require.NoError(t, scene.Update(nil))
	for range 120 {
		require.NoError(t, scene.Update(nil))
		require.Zero(t, scene.dialogStack.Count())
	}
	require.True(t, scene.keyboard.IsWaitingForRelease())
	require.Equal(t, introChoiceAcknowledgments, scene.nCurrentSelection)
}

func TestAcknowledgmentsGroupOnlyIdenticalSourceAndProcessingCredits(t *testing.T) {
	base := audio.RecordingProvenance{SourceCreator: "Composer", SourceURL: "https://example.org/source", License: "Credit one"}
	otherLicense, processed := base, base
	otherLicense.License = "Credit two"
	processed.Processing = []authoring.ProcessingStep{{Description: "Edited", Tool: "Editor", PerformedBy: "Adapter"}}
	recordings := []audio.RecordingCredit{
		{Name: "B", Provenance: base}, {Name: "A", Provenance: base},
		{Name: "C", Provenance: otherLicense}, {Name: "D", Provenance: processed},
	}
	groups := groupRecordingCredits(recordings, false)
	require.Len(t, groups, 3)
	require.Equal(t, []string{"A", "B"}, groups[0].names)
	require.Equal(t, "Credit two", groups[1].source.License)
	require.Equal(t, processed.Processing, groups[2].source.Processing)
	var document []text.CreditParagraph
	for _, group := range groups {
		document = appendRecordingAttribution(document, group)
	}
	require.Contains(t, document, text.CreditParagraph{Label: "Processed by", Value: "Adapter"})
}

func TestIntroAcknowledgmentsDispatchModalIsolationAndFreshReopen(t *testing.T) {
	scene := newIntroInputTestScene()
	scene.nCurrentSelection = introChoiceAcknowledgments
	scene.notice = "Acknowledgments is not yet available."
	scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
	require.NoError(t, scene.Update(nil))
	dialog, ok := scene.dialogStack.PeekTopModalDialog().(*widgets.CreditsDialog)
	require.True(t, ok)
	require.Empty(t, scene.notice)
	require.Nil(t, scene.audioManager, "credits need neither audio players nor gameplay")
	scene.activateSelection(nil)
	require.Equal(t, 1, scene.dialogStack.Count())
	scene.keyboard.SimulateKeyPress(ebiten.KeySpace)
	require.NoError(t, scene.Update(nil))
	for range 1000 {
		require.NoError(t, scene.Update(nil))
	}
	require.Same(t, dialog, scene.dialogStack.PeekTopModalDialog())
	require.Nil(t, scene.returnView)
	require.True(t, scene.menuVisible)
	require.Equal(t, scene.elapsedMs, scene.menuIdleStartedMs)
	for _, key := range []ebiten.Key{ebiten.KeyEnter, ebiten.KeyNumpadEnter, ebiten.KeyEnd, ebiten.KeyHome} {
		scene.keyboard.SimulateKeyPress(key)
		require.NoError(t, scene.Update(nil))
		require.Same(t, dialog, scene.dialogStack.PeekTopModalDialog())
	}
	scene.keyboard.SimulateKeyPress(ebiten.KeyEscape)
	require.NoError(t, scene.Update(nil))
	require.Equal(t, 0, scene.dialogStack.Count())
	require.Equal(t, introChoiceAcknowledgments, scene.nCurrentSelection)
	dialog.Update() // Closing twice must not remove a different instance.
	scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
	require.NoError(t, scene.Update(nil))
	require.IsType(t, &widgets.CreditsDialog{}, scene.dialogStack.PeekTopModalDialog())
	require.NotSame(t, dialog, scene.dialogStack.PeekTopModalDialog())
	scene.keyboard.SimulateKeyPress(ebiten.KeyEscape)
	require.NoError(t, scene.Update(nil))
}
