package main

import (
	"errors"
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/character_creation"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

// Models the Cocoa handoff: once a native editor has started, raw character
// input remains empty after Blur, but the next native field can receive text.
type creationNativeSession struct {
	focused     *creationNativeField
	nativeUsed  bool
	pendingText string
}

func (session *creationNativeSession) rawCharacters() []rune {
	if session.nativeUsed {
		return nil
	}

	return []rune(session.pendingText)
}

type creationNativeField struct {
	session     *creationNativeSession
	value       string
	start       int
	end         int
	preedit     string
	preeditEnd  int
	handle      func(*creationNativeField) (bool, error)
	lateCommit  string
	setCalls    int
	handleCalls int
	blurCalls   int
	lastBounds  image.Rectangle
}

func (field *creationNativeField) Focus() {
	if previous := field.session.focused; previous != nil && previous != field {
		previous.Blur()
	}
	field.session.focused = field
}

func (field *creationNativeField) Blur() {
	if !field.IsFocused() {
		return
	}
	field.blurCalls++
	field.session.focused = nil
	field.preedit = ""
	field.value += field.lateCommit
	field.lateCommit = ""
	field.session.pendingText = ""
}

func (field *creationNativeField) IsFocused() bool { return field.session.focused == field }

func (field *creationNativeField) HandleInputWithBounds(bounds image.Rectangle) (bool, error) {
	field.lastBounds = bounds
	field.handleCalls++
	field.session.nativeUsed = true
	if field.handle != nil {
		handle := field.handle
		field.handle = nil

		return handle(field)
	}
	if field.session.pendingText == "" {
		return false, nil
	}
	value := field.session.pendingText
	field.session.pendingText = ""
	field.value = field.value[:field.start] + value + field.value[field.end:]
	field.start += len(value)
	field.end = field.start

	return true, nil
}

func (field *creationNativeField) Text() string { return field.value }

func (field *creationNativeField) TextForRendering() string {
	if field.preedit != "" {
		return field.value[:field.start] + field.preedit + field.value[field.end:]
	}

	return field.value
}

func (field *creationNativeField) Selection() (int, int) { return field.start, field.end }

func (field *creationNativeField) CompositionSelection() (int, int, bool) {
	return 0, field.preeditEnd, field.preedit != ""
}

func (field *creationNativeField) SetTextAndSelection(value string, start, end int) {
	field.setCalls++
	field.value, field.start, field.end = value, start, end
	field.preedit = ""
}

func nativeCharacterCreationFixture(
	t *testing.T, callbacks CharacterCreationCallbacks,
) (*characterCreationSceneFixture, *creationNativeField, *bool) {
	t.Helper()
	fixture := newCharacterCreationSceneFixture(t, callbacks)
	field := &creationNativeField{session: new(creationNativeSession)}
	focused := true
	fixture.scene.nameField = field
	fixture.scene.appendInputChars = nil
	fixture.scene.isWindowFocused = func() bool { return focused }
	fixture.scene.Activate()
	fixture.update() // Acquire focus before delivering input.

	return fixture, field, &focused
}

func TestCharacterCreationNameAfterPriorNativeEditor(t *testing.T) {
	for _, priorEditor := range []string{"save description", "Tavern Subject"} {
		t.Run(priorEditor, func(t *testing.T) {
			var results []character_creation.Result
			fixture, field, _ := nativeCharacterCreationFixture(t, CharacterCreationCallbacks{
				OnComplete: func(result character_creation.Result) { results = append(results, result) },
				OnCancel:   func() { t.Fatal("unexpected cancellation") },
			})
			prior := &creationNativeField{session: field.session}
			prior.Focus()
			_, err := prior.HandleInputWithBounds(image.Rect(0, 0, 1, 8))
			require.NoError(t, err)
			prior.Blur()
			fixture.update()
			field.session.pendingText = "Avatar"
			require.Empty(t, field.session.rawCharacters())
			require.Nil(t, fixture.scene.appendInputChars, "production uses only native name input")
			fixture.update()
			require.Equal(t, "Avatar", string(fixture.scene.name))
			require.False(t, field.lastBounds.Empty())
			assert.Zero(t, fixture.scene.source.Calls())
			fixture.press(ebiten.KeyEnter)
			require.Equal(t, characterCreationSex, fixture.scene.state)
			require.False(t, field.IsFocused())
			nameInputCalls := field.handleCalls
			fixture.press(ebiten.KeyF)
			require.Equal(t, characterCreationIntroduction, fixture.scene.state)
			assert.Zero(t, fixture.scene.source.Calls())
			fixture.press(ebiten.KeySpace)
			for _, key := range []ebiten.Key{
				ebiten.KeyA, ebiten.KeyB, ebiten.KeyB, ebiten.KeyA, ebiten.KeyB, ebiten.KeyA, ebiten.KeyA,
			} {
				fixture.press(key)
			}
			fixture.press(ebiten.KeySpace)
			require.Len(t, results, 1)
			assert.Equal(t, character_creation.Result{
				Name: "Avatar", Gender: party_state.Female,
				Bonuses:    character_creation.AttributeBonuses{Strength: 4, Intelligence: 8},
				RandomSeed: 42, RandomSourceCalls: 64,
			}, results[0])
			assert.False(t, field.IsFocused())
			assert.Equal(t, nameInputCalls, field.handleCalls, "choices never poll native text")
		})
	}
}

func TestCharacterCreationNativeNameFiltersCommittedTextAndRetainsSelection(t *testing.T) {
	fixture, field, _ := nativeCharacterCreationFixture(t, CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) {}, OnCancel: func() {},
	})
	field.session.pendingText = "A\nv-áa1t a!rXYZ"
	fixture.update()
	assert.Equal(t, "AvatarXY", field.Text())
	assert.Equal(t, "AvatarXY", string(fixture.scene.name))
	start, end := field.Selection()
	assert.Equal(t, 8, start)
	assert.Equal(t, 8, end)
	field.SetTextAndSelection(field.Text(), 1, 4)
	value, start, end := fixture.scene.namePresentation()
	assert.Equal(t, "AvatarXY", value)
	assert.Equal(t, 1, start)
	assert.Equal(t, 4, end)
	fixture.press(ebiten.KeyBackspace)
	assert.Equal(t, "AarXY", field.Text(), "delete selection once")
	field.session.pendingText = "é\u0301B"
	fixture.update()
	assert.Equal(t, "ABarXY", field.Text())
	start, end = field.Selection()
	assert.Equal(t, 2, start)
	assert.Equal(t, 2, end)
	fixture.press(ebiten.KeyHome)
	fixture.press(ebiten.KeyDelete)
	assert.Equal(t, "BarXY", string(fixture.scene.name))
}

func TestCharacterCreationNativePreeditOwnsControlsThroughRelease(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyEnter, ebiten.KeyNumpadEnter, ebiten.KeyEscape, ebiten.KeyBackspace} {
		t.Run(key.String(), func(t *testing.T) {
			fixture, field, _ := nativeCharacterCreationFixture(t, CharacterCreationCallbacks{
				OnComplete: func(character_creation.Result) {},
				OnCancel:   func() { t.Fatal("preedit must not cancel creation") },
			})
			held := make(map[ebiten.Key]bool)
			fixture.scene.keyboard = input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return held[key] })
			field.session.pendingText = "Av"
			fixture.update()
			field.preedit, field.preeditEnd = "é", len("é")
			sets := field.setCalls
			held[key] = true
			fixture.update()
			assert.Equal(t, sets, field.setCalls, "do not reset a live dead-key composition")
			value, start, end := fixture.scene.namePresentation()
			assert.Equal(t, "Avé", value)
			assert.Equal(t, 2, start)
			assert.Equal(t, 3, end, "render selection uses glyph positions, not UTF-8 bytes")
			field.handle = func(field *creationNativeField) (bool, error) {
				field.preedit = ""
				field.value, field.start, field.end = "Avé", len("Avé"), len("Avé")

				return true, nil
			}
			fixture.update()
			fixture.update() // Same held control must not now confirm/cancel/delete.
			assert.Equal(t, "Av", string(fixture.scene.name))
			assert.Equal(t, characterCreationName, fixture.scene.state)
			held[key] = false
			fixture.update()
			fixture.press(ebiten.KeyEnter)
			assert.Equal(t, characterCreationSex, fixture.scene.state)
		})
	}
}

func TestCharacterCreationNativeNameFocusAndLateCommitCleanup(t *testing.T) {
	fixture, field, focused := nativeCharacterCreationFixture(t, CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) {}, OnCancel: func() {},
	})
	held := make(map[ebiten.Key]bool)
	fixture.scene.keyboard = input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return held[key] })
	field.session.pendingText = "Avatar"
	fixture.update()
	field.preedit, field.lateCommit = "é", "Leak"
	*focused = false
	fixture.update()
	assert.False(t, field.IsFocused())
	assert.Equal(t, "Avatar", field.Text())
	held[ebiten.KeyEnter] = true
	fixture.update()
	*focused = true
	fixture.update()
	assert.Equal(t, characterCreationName, fixture.scene.state)
	assert.False(t, field.IsFocused())
	held[ebiten.KeyEnter] = false
	fixture.update()
	assert.True(t, field.IsFocused())
	fixture.press(ebiten.KeyEnter)
	assert.Equal(t, characterCreationSex, fixture.scene.state)
	assert.False(t, field.IsFocused())
	fixture.press(ebiten.KeyEscape)
	fixture.update()
	assert.Equal(t, "Avatar", field.Text())
	assert.True(t, field.IsFocused())
	field.lateCommit = "Leak"
	fixture.scene.PrepareForSceneReplacement()
	assert.False(t, field.IsFocused())
	assert.Equal(t, "Avatar", field.Text())
	fixture.update()
	assert.False(t, field.IsFocused(), "inactive scenes cannot reclaim input")
	fixture.scene.Activate()
	fixture.update()
	assert.True(t, field.IsFocused())
	fixture.scene.Dispose()
	blurs := field.blurCalls
	fixture.scene.Dispose()
	assert.False(t, field.IsFocused())
	assert.Equal(t, blurs, field.blurCalls)
}

func TestCharacterCreationNativeDeletionDoesNotDeleteTwice(t *testing.T) {
	fixture, field, _ := nativeCharacterCreationFixture(t, CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) {}, OnCancel: func() {},
	})
	field.session.pendingText = "Avatar"
	fixture.update()
	field.handle = func(field *creationNativeField) (bool, error) {
		field.value, field.start, field.end = "Avata", 5, 5

		return true, nil
	}
	fixture.press(ebiten.KeyBackspace)
	assert.Equal(t, "Avata", string(fixture.scene.name))
	field.preedit = "é"
	field.handle = func(field *creationNativeField) (bool, error) {
		field.preedit = ""

		return false, nil // Session closed with its final preedit removed.
	}
	fixture.press(ebiten.KeyBackspace)
	assert.Equal(t, "Avata", string(fixture.scene.name), "last preedit deletion is still IME-owned")
	fixture.press(ebiten.KeyBackspace)
	assert.Equal(t, "Avat", string(fixture.scene.name), "the next fresh press edits committed text")
}

func TestCharacterCreationNativeNameCancelAndEmptyConfirm(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyEscape, ebiten.KeyEnter, ebiten.KeyNumpadEnter} {
		t.Run(key.String(), func(t *testing.T) {
			cancelled := 0
			fixture, field, _ := nativeCharacterCreationFixture(t, CharacterCreationCallbacks{
				OnComplete: func(character_creation.Result) { t.Fatal("unexpected completion") },
				OnCancel:   func() {},
			})
			fixture.scene.onCancel = func() {
				cancelled++
				assert.False(t, field.IsFocused(), "release native input before replacement callbacks")
			}
			if key == ebiten.KeyEscape {
				field.session.pendingText = "Avatar"
				fixture.update()
			}
			fixture.press(key)
			assert.Equal(t, 1, cancelled)
			assert.Equal(t, characterCreationCancelled, fixture.scene.state)
			assert.False(t, field.IsFocused())
			assert.Zero(t, fixture.scene.source.Calls())
			fixture.update()
			assert.Equal(t, 1, cancelled)
		})
	}
}

func TestCharacterCreationNativeNameErrorReleasesSession(t *testing.T) {
	fixture, field, _ := nativeCharacterCreationFixture(t, CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) {}, OnCancel: func() {},
	})
	want := errors.New("native input failed")
	field.handle = func(*creationNativeField) (bool, error) { return false, want }
	err := fixture.scene.Update(nil)
	require.ErrorIs(t, err, want)
	assert.False(t, field.IsFocused())
	assert.Equal(t, characterCreationName, fixture.scene.state)
}
