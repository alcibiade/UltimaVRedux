package main

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/character_creation"
	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestCharacterCreationSceneCompletesDeterministicWorkflow(t *testing.T) {
	var completed []character_creation.Result
	fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
		OnComplete: func(result character_creation.Result) { completed = append(completed, result) },
		OnCancel:   func() { t.Fatal("unexpected cancellation") },
	})
	scene := fixture.scene

	fixture.typeRunes("Avatar")
	assert.Equal(t, characterCreationName, scene.state)
	fixture.press(ebiten.KeyEnter)
	assert.Equal(t, characterCreationSex, scene.state)
	fixture.press(ebiten.KeyEscape)
	assert.Equal(t, characterCreationName, scene.state)
	assert.Equal(t, "Avatar", string(scene.name))
	fixture.press(ebiten.KeyEnter)
	fixture.press(ebiten.KeyF)
	assert.Equal(t, characterCreationIntroduction, scene.state)
	assert.Zero(t, scene.source.Calls())
	fixture.press(ebiten.KeySpace)
	assert.Equal(t, characterCreationFirstRound, scene.state)
	assert.Positive(t, scene.source.Calls())

	responses := []ebiten.Key{
		ebiten.KeyA, ebiten.KeyB, ebiten.KeyB, ebiten.KeyA,
		ebiten.KeyB, ebiten.KeyA, ebiten.KeyA,
	}
	wantStates := []characterCreationState{
		characterCreationFirstRound,
		characterCreationFirstRound,
		characterCreationFirstRound,
		characterCreationSecondRound,
		characterCreationSecondRound,
		characterCreationFinalRound,
		characterCreationConclusion,
	}
	for index, response := range responses {
		fixture.press(response)
		assert.Equal(t, index+1, scene.questionNumber)
		assert.Equal(t, wantStates[index], scene.state)
	}
	assert.Empty(t, completed)

	fixture.press(ebiten.KeySpace)
	require.Len(t, completed, 1)
	assert.Equal(t, characterCreationComplete, scene.state)
	assert.True(t, scene.terminal)
	want := character_creation.Result{
		Name:              "Avatar",
		Gender:            party_state.Female,
		Bonuses:           character_creation.AttributeBonuses{Strength: 4, Intelligence: 8},
		RandomSeed:        42,
		RandomSourceCalls: 64,
	}
	assert.Equal(t, want, completed[0])
	result, available := scene.Result()
	assert.True(t, available)
	assert.Equal(t, want, result)

	require.NoError(t, scene.Update(nil))
	assert.Len(t, completed, 1)
}

func TestCompletedCharacterCanRetryOrConfirmDiscard(t *testing.T) {
	cancelled, retried := 0, 0
	fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) { t.Fatal("completion must not replay") },
		OnCancel:   func() { cancelled++ },
	})
	scene := fixture.scene
	scene.transitionTo(characterCreationComplete)
	scene.hasResult = true
	scene.result = character_creation.Result{Name: "Avatar"}
	scene.configureTerminalAction(func() { retried++ }, "Save failed.", "Enter: Retry.")
	fixture.press(ebiten.KeyEscape)
	require.Equal(t, characterCreationDiscardConfirmation, scene.state)
	fixture.press(ebiten.KeyEscape)
	require.True(t, scene.terminal)
	require.True(t, scene.hasResult)
	require.Equal(t, "Avatar", scene.result.Name)
	fixture.press(ebiten.KeyNumpadEnter)
	require.Equal(t, 1, retried)
	fixture.press(ebiten.KeyEscape)
	fixture.press(ebiten.KeyY)
	require.Equal(t, 1, cancelled)
}

func TestCharacterCreationSceneCallbackCanDisposeSceneReentrantly(t *testing.T) {
	var scene *CharacterCreationScene
	completed := 0
	fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) {
			completed++
			scene.Dispose()
		},
		OnCancel: func() { t.Fatal("unexpected cancellation") },
	})
	scene = fixture.scene
	fixture.completeIdentityAndIntroduction()
	for _, response := range []ebiten.Key{
		ebiten.KeyA, ebiten.KeyB, ebiten.KeyB, ebiten.KeyA, ebiten.KeyB, ebiten.KeyA, ebiten.KeyA,
	} {
		fixture.press(response)
	}

	require.NotPanics(t, func() { fixture.press(ebiten.KeySpace) })
	assert.Equal(t, 1, completed)
	assert.True(t, scene.disposed)
	_, available := scene.Result()
	assert.True(t, available)
}

func TestCharacterCreationSceneCancellationCallbackCanReplaceSceneReentrantly(t *testing.T) {
	var game *Game
	replacement := &characterCreationReplacementScene{
		config: testCharacterCreationConfig(t.TempDir()),
	}
	fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) { t.Fatal("unexpected completion") },
		OnCancel: func() {
			game.replaceScene(replacement)
		},
	})
	game = NewGame(fixture.scene)
	fixture.scene.keyboard.SimulateKeyPress(ebiten.KeyEscape)

	require.NotPanics(t, func() { require.NoError(t, game.Update()) })
	assert.Same(t, replacement, game.currentScene)
	assert.True(t, fixture.scene.disposed)
}

func TestCharacterCreationSceneCancellationBoundaries(t *testing.T) {
	t.Run("name escape cancels", func(t *testing.T) {
		cancelled := 0
		fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
			OnComplete: func(character_creation.Result) { t.Fatal("unexpected completion") },
			OnCancel:   func() { cancelled++ },
		})
		fixture.press(ebiten.KeyEscape)
		assert.Equal(t, 1, cancelled)
		assert.Equal(t, characterCreationCancelled, fixture.scene.state)
		_, available := fixture.scene.Result()
		assert.False(t, available)
		fixture.press(ebiten.KeyEscape)
		assert.Equal(t, 1, cancelled)
	})

	t.Run("empty enter cancels", func(t *testing.T) {
		cancelled := 0
		fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
			OnComplete: func(character_creation.Result) { t.Fatal("unexpected completion") },
			OnCancel:   func() { cancelled++ },
		})
		fixture.press(ebiten.KeyEnter)
		assert.Equal(t, 1, cancelled)
	})

	t.Run("post identity escape confirms discard", func(t *testing.T) {
		cancelled := 0
		fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
			OnComplete: func(character_creation.Result) { t.Fatal("unexpected completion") },
			OnCancel:   func() { cancelled++ },
		})
		fixture.typeRunes("Avatar")
		fixture.press(ebiten.KeyEnter)
		fixture.press(ebiten.KeyM)
		fixture.press(ebiten.KeyEscape)
		assert.Equal(t, characterCreationDiscardConfirmation, fixture.scene.state)
		fixture.press(ebiten.KeyN)
		assert.Equal(t, characterCreationIntroduction, fixture.scene.state)
		fixture.press(ebiten.KeyEscape)
		fixture.press(ebiten.KeyY)
		assert.Equal(t, 1, cancelled)
		assert.Equal(t, characterCreationCancelled, fixture.scene.state)
	})
}

func TestCharacterCreationSceneNameEditingAndRuneFiltering(t *testing.T) {
	fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) {},
		OnCancel:   func() {},
	})
	fixture.queueRunes([]rune{'A', '\n', 'v', '-', 'á', 'a', '1', 't', ' ', 'a', '!', 'r', 'X', 'Y', 'Z'})
	fixture.update()
	assert.Equal(t, "AvatarXY", string(fixture.scene.name))

	fixture.press(ebiten.KeyBackspace)
	assert.Equal(t, "AvatarX", string(fixture.scene.name))
	fixture.press(ebiten.KeyEnter)
	assert.Equal(t, characterCreationSex, fixture.scene.state)
}

func TestCharacterCreationSceneInvalidAnswerDoesNotMutateQuestionOrRNG(t *testing.T) {
	fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) {},
		OnCancel:   func() {},
	})
	fixture.completeIdentityAndIntroduction()
	wantDilemma, available := fixture.scene.questionnaire.CurrentDilemma()
	require.True(t, available)
	wantCalls := fixture.scene.source.Calls()

	fixture.press(ebiten.KeyC)

	gotDilemma, available := fixture.scene.questionnaire.CurrentDilemma()
	require.True(t, available)
	assert.Equal(t, wantDilemma, gotDilemma)
	assert.Equal(t, wantCalls, fixture.scene.source.Calls())
	assert.Zero(t, fixture.scene.questionNumber)
	assert.Equal(t, characterCreationFirstRound, fixture.scene.state)
}

func TestCharacterCreationSceneDrainsPendingRunesAcrossTransitions(t *testing.T) {
	fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) {},
		OnCancel:   func() {},
	})
	fixture.typeRunes("Avatar")
	fixture.press(ebiten.KeyEnter)
	fixture.queueRunes([]rune{'f', 'a'})
	fixture.update()
	assert.Equal(t, characterCreationSex, fixture.scene.state, "late text cannot choose a physical option")
	fixture.press(ebiten.KeyF)
	assert.Equal(t, characterCreationIntroduction, fixture.scene.state)

	fixture.update()
	assert.Equal(t, characterCreationIntroduction, fixture.scene.state)
	assert.Zero(t, fixture.scene.questionNumber)
}

func TestCharacterCreationSceneWaitsForPhysicalKeyReleaseAcrossTransitions(t *testing.T) {
	fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) {},
		OnCancel:   func() {},
	})
	held := make(map[ebiten.Key]bool)
	fixture.scene.keyboard = input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool {
		return held[key]
	})
	fixture.typeRunes("Avatar")

	held[ebiten.KeyEnter] = true
	fixture.update()
	require.Equal(t, characterCreationSex, fixture.scene.state)
	fixture.update()
	require.Equal(t, characterCreationSex, fixture.scene.state)
	held[ebiten.KeyEnter] = false
	fixture.update()

	held[ebiten.KeyF] = true
	fixture.update()
	require.Equal(t, characterCreationIntroduction, fixture.scene.state)
	fixture.update()
	require.Equal(t, characterCreationIntroduction, fixture.scene.state)
	held[ebiten.KeyF] = false
	fixture.update()

	held[ebiten.KeySpace] = true
	fixture.update()
	require.Equal(t, characterCreationFirstRound, fixture.scene.state)
	fixture.update()
	require.Zero(t, fixture.scene.questionNumber)
	held[ebiten.KeySpace] = false
	fixture.update()
	require.Zero(t, fixture.scene.questionNumber)
}

func TestCharacterCreationSceneLifecycleAndConstructorContracts(t *testing.T) {
	callbacks := CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) {},
		OnCancel:   func() {},
	}
	fixture := newCharacterCreationSceneFixture(t, callbacks)
	require.NoError(t, fixture.scene.PreflightActivation())
	fixture.scene.PrepareInputForSceneActivation()
	fixture.scene.Activate()
	fixture.scene.Activate()
	assert.True(t, fixture.scene.active)
	fixture.scene.PrepareForSceneReplacement()
	assert.False(t, fixture.scene.active)
	fixture.scene.Dispose()
	fixture.scene.Dispose()
	require.Error(t, fixture.scene.PreflightActivation())
	require.NotPanics(t, func() { fixture.scene.Draw(nil) })

	require.Panics(t, func() {
		CreateCharacterCreationScene(
			testCharacterCreationConfig(t.TempDir()), nil, callbacks, newCharacterCreationIdentityPresentation(nil, 0),
		)
	})
	require.Panics(t, func() {
		CreateCharacterCreationScene(
			testCharacterCreationConfig(t.TempDir()),
			deterministic_random.New(1),
			CharacterCreationCallbacks{},
			newCharacterCreationIdentityPresentation(nil, 0),
		)
	})
	_, err := CreateCharacterCreationScene(
		nil, deterministic_random.New(1), callbacks, newCharacterCreationIdentityPresentation(nil, 0),
	)
	require.ErrorIs(t, err, errCharacterCreationSceneInvalid)

	advanced := deterministic_random.New(1)
	_ = advanced.Intn(2)
	_, err = CreateCharacterCreationScene(
		testCharacterCreationConfig(t.TempDir()), advanced, callbacks, newCharacterCreationIdentityPresentation(nil, 0),
	)
	require.ErrorIs(t, err, character_creation.ErrAdvancedRandomSource)

	_, err = CreateCharacterCreationScene(
		testCharacterCreationConfig(t.TempDir()), deterministic_random.New(1), callbacks,
		newCharacterCreationIdentityPresentation(nil, 0),
	)
	require.ErrorContains(t, err, "CREATE.16")
}

func TestCharacterCreationSceneDrawsAllPresentationStatesAtScaledAspectRatios(t *testing.T) {
	fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) {},
		OnCancel:   func() {},
	})
	// The logical canvas is fixed. Exercise state drawing once; scaling and
	// distinctive asset placement have separate positive pixel anchors below.
	screen := ebiten.NewImage(320, 200)
	t.Cleanup(screen.Deallocate)
	draw := func() {
		require.NotPanics(t, func() { fixture.scene.Draw(screen) })
	}

	draw()
	for _, size := range []image.Point{{1280, 720}, {200, 400}} {
		scaled := ebiten.NewImage(size.X, size.Y)
		t.Cleanup(scaled.Deallocate)
		require.NotPanics(t, func() { fixture.scene.Draw(scaled) })
		assertCharacterCreationPixel(t, scaled, 0, 0, ucolor.EGABlack)
	}
	fixture.typeRunes("Avatar")
	fixture.press(ebiten.KeyEnter)
	draw()
	fixture.press(ebiten.KeyF)
	draw()
	fixture.press(ebiten.KeySpace)
	draw()
	for range characterCreationQuestionCount {
		previous := fixture.scene.state
		fixture.press(ebiten.KeyA)
		if fixture.scene.state != previous && fixture.scene.state != characterCreationConclusion {
			draw()
		}
	}
	require.Equal(t, characterCreationConclusion, fixture.scene.state)
	draw()
	fixture.press(ebiten.KeyEscape)
	draw()
}

func TestCharacterCreationSceneDrawsTitleBackdropAndFixedCellIdentityPanel(t *testing.T) {
	fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) {},
		OnCancel:   func() {},
	})
	screen := newPresentationRaster(1280, 720)
	t.Cleanup(screen.Deallocate)
	screen.Fill(ucolor.EGARed)
	fixture.scene.identityPresentation.title = ebiten.NewImage(introLogicalWidth, introLogicalHeight)
	t.Cleanup(fixture.scene.identityPresentation.title.Deallocate)
	fixture.scene.identityPresentation.title.Fill(ucolor.EGADarkGray)

	fixture.scene.Draw(screen)

	assertCharacterCreationPixel(t, screen, 0, 0, ucolor.EGABlack)
	assertCharacterCreationPixel(t, fixture.scene.canvas, 10, 10, ucolor.EGADarkGray)
	assertCharacterCreationPixel(t, fixture.scene.canvas, 120, 120, ucolor.EGABlue)
	assertCharacterCreationPixel(t, fixture.scene.canvas, 120, 127, ucolor.EGAWhite)
	assertCharacterCreationPixel(t, fixture.scene.canvas, 7, 140, ucolor.EGAWhite)
	assertCharacterCreationPixel(t, fixture.scene.canvas, 312, 140, ucolor.EGAWhite)
	assertCharacterCreationPixel(t, fixture.scene.canvas, 8, 128, ucolor.EGABlack)
	assertCharacterCreationRectHasColor(t, fixture.scene.canvas, image.Rect(24, 136, 288, 144), ucolor.EGAWhite)
	assertCharacterCreationRectLacksColor(t, fixture.scene.canvas, image.Rect(8, 144, 312, 152), ucolor.EGAWhite)

	fixture.typeRunes("Avatar")
	fixture.press(ebiten.KeyEnter)
	fixture.scene.Draw(screen)
	assertCharacterCreationRectHasColor(t, fixture.scene.canvas, image.Rect(120, 152, 168, 160), ucolor.EGAWhite)
	assertCharacterCreationRectLacksColor(t, fixture.scene.canvas, image.Rect(168, 152, 176, 160), ucolor.EGAWhite)
	assertCharacterCreationRectHasColor(t, fixture.scene.canvas, image.Rect(64, 168, 264, 176), ucolor.EGAWhite)
	assertCharacterCreationRectHasColor(t, fixture.scene.canvas, image.Rect(264, 168, 272, 176), ucolor.EGAWhite)
}

func TestCharacterCreationIdentityFlameAdvancesWhileTyping(t *testing.T) {
	fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) {},
		OnCancel:   func() {},
	})
	firstFlame := ebiten.NewImage(1, 1)
	t.Cleanup(firstFlame.Deallocate)
	firstFlame.Fill(ucolor.EGARed)
	secondFlame := ebiten.NewImage(1, 1)
	t.Cleanup(secondFlame.Deallocate)
	secondFlame.Fill(ucolor.EGACyan)
	fixture.scene.identityPresentation.flameAnimation = sprites.NewImageAnimation(
		[]*ebiten.Image{firstFlame, secondFlame},
		100,
	)
	fixture.scene.identityPresentation.elapsedMs = 0
	screen := ebiten.NewImage(introLogicalWidth, introLogicalHeight)
	t.Cleanup(screen.Deallocate)

	fixture.scene.Draw(screen)
	assertCharacterCreationPixel(t, fixture.scene.canvas, introFlameX, introFlameY, ucolor.EGARed)
	for range 7 {
		fixture.update()
	}
	fixture.scene.Draw(screen)
	assertCharacterCreationPixel(t, fixture.scene.canvas, introFlameX, introFlameY, ucolor.EGACyan)

	fixture.typeRunes("Avatar")
	fixture.press(ebiten.KeyEnter)
	sexElapsedMs := fixture.scene.identityPresentation.elapsedMs
	fixture.press(ebiten.KeyF)
	assert.Equal(t, sexElapsedMs+introUpdateInterval, fixture.scene.identityPresentation.elapsedMs)
	elapsedMs := fixture.scene.identityPresentation.elapsedMs
	fixture.update()
	assert.Equal(t, elapsedMs, fixture.scene.identityPresentation.elapsedMs)
}

func TestCharacterCreationScenePlacesSemanticAssetsAtOriginalCoordinates(t *testing.T) {
	fixture := newCharacterCreationSceneFixture(t, CharacterCreationCallbacks{
		OnComplete: func(character_creation.Result) {},
		OnCancel:   func() {},
	})
	screen := ebiten.NewImage(320, 200)
	t.Cleanup(screen.Deallocate)
	// This pixel test owns mutable artwork; behavioural fixtures share only a
	// transparent placeholder and never mutate it.
	setCharacterCreationTestArtwork(t, fixture.scene)
	fixture.scene.assets.Oracle.Fill(ucolor.EGABlue)
	fixture.scene.assets.Incense.Fill(ucolor.EGACyan)
	fixture.scene.assets.Shrine.Fill(ucolor.EGAMagenta)
	for _, symbol := range fixture.scene.assets.VirtueSymbols {
		symbol.Fill(ucolor.EGARed)
	}

	fixture.typeRunes("Avatar")
	fixture.press(ebiten.KeyEnter)
	fixture.press(ebiten.KeyF)
	fixture.scene.Draw(screen)
	assertCharacterCreationPixel(t, fixture.scene.canvas, 0, 96, ucolor.EGABlue)

	fixture.press(ebiten.KeySpace)
	fixture.scene.Draw(screen)
	assertCharacterCreationPixel(t, fixture.scene.canvas, 16, 0, ucolor.EGACyan)
	assertCharacterCreationPixel(t, fixture.scene.canvas, 200, 0, ucolor.EGACyan)
	dilemma, available := fixture.scene.questionnaire.CurrentDilemma()
	require.True(t, available)
	leftPosition := characterCreationVirtuePositions[dilemma.OptionA]
	rightPosition := characterCreationVirtuePositions[dilemma.OptionB]
	assertCharacterCreationPixel(t, fixture.scene.canvas, leftPosition.X, leftPosition.Y, ucolor.EGARed)
	assertCharacterCreationPixel(t, fixture.scene.canvas, rightPosition.X+184, rightPosition.Y, ucolor.EGARed)

	fixture.scene.state = characterCreationConclusion
	fixture.scene.Draw(screen)
	assertCharacterCreationPixel(t, fixture.scene.canvas, 168, 100, ucolor.EGAMagenta)
}

func setCharacterCreationTestArtwork(t *testing.T, scene *CharacterCreationScene) {
	t.Helper()
	scene.assets.Oracle = ebiten.NewImage(1, 1)
	scene.assets.Incense = ebiten.NewImage(1, 1)
	scene.assets.Shrine = ebiten.NewImage(1, 1)
	t.Cleanup(scene.assets.Oracle.Deallocate)
	t.Cleanup(scene.assets.Incense.Deallocate)
	t.Cleanup(scene.assets.Shrine.Deallocate)
	symbol := ebiten.NewImage(1, 1)
	t.Cleanup(symbol.Deallocate)
	for index := range scene.assets.VirtueSymbols {
		scene.assets.VirtueSymbols[index] = symbol
	}
}

func assertCharacterCreationPixel(t *testing.T, source *ebiten.Image, x, y int, expected ucolor.EGAColor) {
	t.Helper()
	r, g, b, _ := source.At(x, y).RGBA()
	require.True(t, expected.Matches(uint8(r>>8), uint8(g>>8), uint8(b>>8)))
}

func assertCharacterCreationRectHasColor(
	t *testing.T,
	source *ebiten.Image,
	rect image.Rectangle,
	expected ucolor.EGAColor,
) {
	t.Helper()
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			r, g, b, _ := source.At(x, y).RGBA()
			if expected.Matches(uint8(r>>8), uint8(g>>8), uint8(b>>8)) {
				return
			}
		}
	}
	require.Failf(t, "expected color not found", "%v in %v", expected, rect)
}

func assertCharacterCreationRectLacksColor(
	t *testing.T,
	source *ebiten.Image,
	rect image.Rectangle,
	unexpected ucolor.EGAColor,
) {
	t.Helper()
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			r, g, b, _ := source.At(x, y).RGBA()
			require.Falsef(
				t,
				unexpected.Matches(uint8(r>>8), uint8(g>>8), uint8(b>>8)),
				"unexpected color at (%d,%d)",
				x,
				y,
			)
		}
	}
}

type characterCreationSceneFixture struct {
	t       *testing.T
	scene   *CharacterCreationScene
	pending []rune
}

type characterCreationReplacementScene struct {
	config *config.UltimaVConfiguration
}

func (*characterCreationReplacementScene) Update(*Game) error { return nil }

func (*characterCreationReplacementScene) Draw(*ebiten.Image) {}

func (scene *characterCreationReplacementScene) GetUltimaConfiguration() *config.UltimaVConfiguration {
	return scene.config
}

func (*characterCreationReplacementScene) InvalidateResolution() {}

func newCharacterCreationSceneFixture(
	t *testing.T,
	callbacks CharacterCreationCallbacks,
) *characterCreationSceneFixture {
	t.Helper()
	placeholder := ebiten.NewImage(1, 1)
	t.Cleanup(placeholder.Deallocate)
	assets := &sprites.CharacterCreationAssets{
		Oracle:        placeholder,
		Incense:       placeholder,
		VirtueSymbols: [character_creation.VirtueCount]*ebiten.Image{},
		Shrine:        placeholder,
		Glyphs:        make([]*ebiten.Image, 91),
		GlyphWidths:   make([]int, 91),
		Script:        sprites.CharacterCreationScript{},
	}
	for index := range assets.VirtueSymbols {
		assets.VirtueSymbols[index] = placeholder
	}
	var dilemmaTexts [sprites.CharacterCreationDilemmaCount]string
	for index := range dilemmaTexts {
		dilemmaTexts[index] = "Choose A or B."
	}
	var err error
	assets.Script, err = sprites.NewCharacterCreationScript(
		"A gypsy speaks.",
		"So be it.",
		dilemmaTexts,
	)
	require.NoError(t, err)
	fixture := &characterCreationSceneFixture{t: t, pending: nil}
	fixture.scene = newCharacterCreationScene(
		testCharacterCreationConfig(t.TempDir()),
		deterministic_random.New(42),
		callbacks,
		assets,
		newCharacterCreationIdentityPresentation(nil, 0),
	)
	fixture.scene.keyboard = newPresentationTestKeyboard()
	// Capture the canvas before Dispose clears the scene's reference. This also
	// covers reentrant replacement/disposal tests using the same fixture.
	t.Cleanup(fixture.scene.canvas.Deallocate)
	t.Cleanup(fixture.scene.Dispose)
	fixture.scene.appendInputChars = func(destination []rune) []rune {
		result := append(destination, fixture.pending...)
		fixture.pending = nil

		return result
	}

	return fixture
}

func (fixture *characterCreationSceneFixture) queueRunes(inputRunes []rune) {
	fixture.pending = append(fixture.pending, inputRunes...)
}

func (fixture *characterCreationSceneFixture) typeRunes(value string) {
	fixture.t.Helper()
	fixture.queueRunes([]rune(value))
	fixture.update()
}

func (fixture *characterCreationSceneFixture) press(key ebiten.Key) {
	fixture.t.Helper()
	if fixture.scene.keyboard == nil {
		return
	}
	fixture.scene.keyboard.SimulateKeyPress(key)
	fixture.update()
}

func (fixture *characterCreationSceneFixture) update() {
	fixture.t.Helper()
	require.NoError(fixture.t, fixture.scene.Update(nil))
}

func (fixture *characterCreationSceneFixture) completeIdentityAndIntroduction() {
	fixture.t.Helper()
	fixture.typeRunes("Avatar")
	fixture.press(ebiten.KeyEnter)
	fixture.press(ebiten.KeyF)
	fixture.press(ebiten.KeySpace)
	require.Equal(fixture.t, characterCreationFirstRound, fixture.scene.state)
}

func testCharacterCreationConfig(directory string) *config.UltimaVConfiguration {
	return &config.UltimaVConfiguration{
		SavedConfigData: &config.UltimaVConfigurationFlags{DataFilePath: directory},
	}
}
