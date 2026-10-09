//nolint:mnd // ref:CREATE-0002: original 320x200 composition and coordinates are immutable presentation data.
package main

import (
	"errors"
	"fmt"
	"image"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/character_creation"
	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

const (
	characterCreationFontSize        = 10
	characterCreationPromptLines     = 5
	characterCreationPromptLineChars = 44
	characterCreationNameLimit       = 8
	characterCreationQuestionCount   = 7
	characterCreationFirstRoundEnd   = 4
	characterCreationSecondRoundEnd  = 6
	characterCreationLeftArrowGlyph  = byte(1)
	characterCreationRightArrowGlyph = byte(2)
)

var errCharacterCreationSceneInvalid = errors.New("character creation scene configuration is invalid")

//nolint:gochecknoglobals // ref:CREATE-0002: immutable virtue-card placement coordinates.
var characterCreationVirtuePositions = [...]image.Point{
	image.Pt(40, 5), image.Pt(48, 7), image.Pt(48, 4), image.Pt(40, 10),
	image.Pt(40, 8), image.Pt(48, 0), image.Pt(40, 5), image.Pt(48, 6),
}

type characterCreationState uint8

const (
	characterCreationName characterCreationState = iota
	characterCreationSex
	characterCreationIntroduction
	characterCreationFirstRound
	characterCreationSecondRound
	characterCreationFinalRound
	characterCreationConclusion
	characterCreationDiscardConfirmation
	characterCreationComplete
	characterCreationCancelled
)

// CharacterCreationCallbacks hand detached results back to application orchestration.
type CharacterCreationCallbacks struct {
	OnComplete func(character_creation.Result)
	OnCancel   func()
}

type characterCreationIdentityPresentation struct {
	title          *ebiten.Image
	flameAnimation *sprites.ImageAnimation
	elapsedMs      int64
}

func newCharacterCreationIdentityPresentation(
	introSprites *sprites.IntroSprites,
	elapsedMs int64,
) *characterCreationIdentityPresentation {
	presentation := &characterCreationIdentityPresentation{
		title:          nil,
		flameAnimation: nil,
		elapsedMs:      max(elapsedMs, 0),
	}
	if introSprites != nil {
		presentation.title = introSprites.Ultima16Logo
		presentation.flameAnimation = introSprites.FlameAnimation
	}

	return presentation
}

func (presentation *characterCreationIdentityPresentation) advance() {
	presentation.elapsedMs += introUpdateInterval
}

func (presentation *characterCreationIdentityPresentation) draw(canvas *ebiten.Image) {
	if presentation.title != nil {
		drawIntroImage(canvas, presentation.title, 0, 0)
	}
	if presentation.flameAnimation != nil {
		drawIntroImage(
			canvas,
			presentation.flameAnimation.GetCurrentImageAtTime(presentation.elapsedMs),
			introFlameX,
			introFlameY,
		)
	}
}

// CharacterCreationScene owns the standalone illustrated identity and questionnaire workflow.
type CharacterCreationScene struct {
	assets               *sprites.CharacterCreationAssets
	proportionalAssets   *sprites.NarrativeIntroAssets
	config               *config.UltimaVConfiguration
	source               *deterministic_random.Source
	keyboard             *input.Keyboard
	appendInputChars     func([]rune) []rune
	canvas               *ebiten.Image
	identityPresentation *characterCreationIdentityPresentation
	identityOutput       *text.Output
	promptOutput         *text.Output
	discardOutput        *text.Output
	discardScreenWidth   int
	discardBounds        image.Rectangle
	dialogStack          widgets.DialogStack
	state                characterCreationState
	stateBeforeDiscard   characterCreationState
	name                 []rune
	gender               party_state.CharacterGender
	questionnaire        *character_creation.Questionnaire
	questionNumber       int
	result               character_creation.Result
	hasResult            bool
	onComplete           func(character_creation.Result)
	onCancel             func()
	terminalAction       func()
	terminalNotice       string
	active               bool
	terminal             bool
	disposed             bool
}

// CreateCharacterCreationScene loads optional original assets and creates a detached workflow scene.
func CreateCharacterCreationScene(
	gameConfig *config.UltimaVConfiguration,
	source *deterministic_random.Source,
	callbacks CharacterCreationCallbacks,
	identityPresentation *characterCreationIdentityPresentation,
) (*CharacterCreationScene, error) {
	if source == nil {
		panic("CreateCharacterCreationScene: random source is required")
	}
	if callbacks.OnComplete == nil || callbacks.OnCancel == nil {
		panic("CreateCharacterCreationScene: completion and cancellation callbacks are required")
	}
	if gameConfig == nil || gameConfig.SavedConfigData == nil ||
		strings.TrimSpace(gameConfig.SavedConfigData.DataFilePath) == "" {
		return nil, fmt.Errorf("%w: configured original-data directory is required", errCharacterCreationSceneInvalid)
	}
	if source.Calls() != 0 {
		return nil, fmt.Errorf(
			"%w: %w: source has %d prior calls",
			errCharacterCreationSceneInvalid,
			character_creation.ErrAdvancedRandomSource,
			source.Calls(),
		)
	}
	assets, err := sprites.LoadCharacterCreationAssets(gameConfig.SavedConfigData.DataFilePath)
	if err != nil {
		return nil, fmt.Errorf("creating character creation scene: %w", err)
	}

	return newCharacterCreationScene(gameConfig, source, callbacks, assets, identityPresentation), nil
}

func newCharacterCreationScene(
	gameConfig *config.UltimaVConfiguration,
	source *deterministic_random.Source,
	callbacks CharacterCreationCallbacks,
	assets *sprites.CharacterCreationAssets,
	identityPresentation *characterCreationIdentityPresentation,
) *CharacterCreationScene {
	if gameConfig == nil || source == nil || callbacks.OnComplete == nil || callbacks.OnCancel == nil ||
		assets == nil || identityPresentation == nil {
		panic("newCharacterCreationScene: all dependencies are required")
	}
	font := text.NewUltimaFont(characterCreationFontSize)

	return &CharacterCreationScene{
		assets: assets,
		proportionalAssets: &sprites.NarrativeIntroAssets{
			Shapes: nil, Glyphs: assets.Glyphs, GlyphWidths: assets.GlyphWidths, StoryBlocks: nil,
		},
		config:               gameConfig,
		source:               source,
		keyboard:             input.NewKeyboard(0),
		appendInputChars:     ebiten.AppendInputChars,
		canvas:               ebiten.NewImage(introLogicalWidth, introLogicalHeight),
		identityPresentation: identityPresentation,
		identityOutput: text.NewOutput(
			text.NewUltimaFont(text.BitmapCharHeight),
			text.BitmapCharHeight,
			1,
			introLogicalWidth/text.BitmapCharWidth,
		),
		promptOutput: text.NewOutput(
			font,
			font.TextFace.Metrics().HAscent*1.2,
			characterCreationPromptLines,
			characterCreationPromptLineChars,
		),
		dialogStack:        widgets.DialogStack{},
		discardOutput:      nil,
		discardScreenWidth: 0,
		discardBounds:      image.Rect(0, 0, 0, 0),
		state:              characterCreationName,
		stateBeforeDiscard: characterCreationName,
		name:               make([]rune, 0, characterCreationNameLimit),
		gender:             0,
		questionnaire:      nil,
		questionNumber:     0,
		result: character_creation.Result{
			Name: "", Gender: 0, Bonuses: character_creation.AttributeBonuses{
				Strength: 0, Dexterity: 0, Intelligence: 0,
			},
			RandomSeed: 0, RandomSourceCalls: 0,
		},
		hasResult:      false,
		onComplete:     callbacks.OnComplete,
		onCancel:       callbacks.OnCancel,
		terminalAction: nil,
		terminalNotice: "",
		active:         false,
		terminal:       false,
		disposed:       false,
	}
}

// Result returns the detached completed result even after scene disposal.
func (scene *CharacterCreationScene) Result() (character_creation.Result, bool) {
	return scene.result, scene.hasResult
}

func (scene *CharacterCreationScene) GetUltimaConfiguration() *config.UltimaVConfiguration {
	return scene.config
}

func (scene *CharacterCreationScene) InvalidateResolution() {}

func (scene *CharacterCreationScene) PreflightActivation() error {
	if scene.disposed {
		return fmt.Errorf("%w: disposed scene cannot be activated", errCharacterCreationSceneInvalid)
	}

	return nil
}

func (scene *CharacterCreationScene) PrepareInputForSceneActivation() {
	scene.drainInputRunes()
	scene.keyboard.PrepareForTransition(0)
}

func (scene *CharacterCreationScene) Activate() {
	if scene.disposed {
		panic("CharacterCreationScene.Activate: disposed scene cannot be activated")
	}
	if scene.active {
		return
	}
	scene.active = true
}

func (scene *CharacterCreationScene) PrepareForSceneReplacement() {
	scene.active = false
}

func (scene *CharacterCreationScene) Dispose() {
	if scene.disposed {
		return
	}
	scene.disposed = true
	scene.active = false
	scene.assets = nil
	scene.proportionalAssets = nil
	scene.source = nil
	scene.keyboard = nil
	scene.appendInputChars = nil
	scene.canvas = nil
	scene.identityPresentation = nil
	scene.identityOutput = nil
	scene.promptOutput = nil
	scene.discardOutput = nil
	scene.dialogStack = widgets.DialogStack{}
	scene.questionnaire = nil
	scene.onComplete = nil
	scene.onCancel = nil
	scene.terminalAction = nil
	scene.terminalNotice = ""
}

//nolint:cyclop // Explicit state dispatch keeps input ownership auditable.
func (scene *CharacterCreationScene) Update(_ *Game) error {
	if scene.disposed {
		return nil
	}
	if scene.state == characterCreationName || scene.state == characterCreationSex {
		// Original driver timing during blocking identity input is unavailable;
		// Redux explicitly continues the cosmetic flame animation while typing.
		scene.identityPresentation.advance()
	}
	scene.keyboard.ClearFrameConsumption()
	if modal := scene.dialogStack.PeekTopModalDialog(); modal != nil {
		modal.Update()

		return nil
	}
	if scene.keyboard.IsWaitingForRelease() {
		scene.drainInputRunes()

		return nil
	}
	if scene.terminal {
		scene.updateTerminal()

		return nil
	}

	switch scene.state {
	case characterCreationName:
		scene.updateName()
	case characterCreationSex:
		scene.updateSex()
	case characterCreationIntroduction:
		scene.updateFraming(characterCreationFirstRound)
	case characterCreationFirstRound, characterCreationSecondRound, characterCreationFinalRound:
		scene.updateDilemma()
	case characterCreationConclusion:
		scene.updateConclusion()
	case characterCreationDiscardConfirmation:
		scene.updateDiscardConfirmation()
	case characterCreationComplete, characterCreationCancelled:
		panic("CharacterCreationScene.Update: terminal state is not marked terminal")
	default:
		panic("CharacterCreationScene.Update: unknown state")
	}

	return nil
}

func (scene *CharacterCreationScene) updateTerminal() {
	if scene.state != characterCreationComplete && scene.state != characterCreationCancelled {
		panic("CharacterCreationScene.updateTerminal: non-terminal state is marked terminal")
	}
	if scene.state == characterCreationComplete && scene.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) {
		scene.requestDiscard()

		return
	}
	if scene.terminalAction == nil || (!scene.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter) &&
		!scene.keyboard.IsKeyPressedAndRegister(ebiten.KeyNumpadEnter)) {
		return
	}
	action := scene.terminalAction
	action()
}

func (scene *CharacterCreationScene) configureTerminalAction(action func(), notice string) {
	if !scene.terminal || action == nil {
		panic("CharacterCreationScene.configureTerminalAction: terminal scene and action are required")
	}
	scene.terminalAction = action
	scene.terminalNotice = notice
	scene.keyboard.PrepareForTransition(0)
}

func (scene *CharacterCreationScene) setTerminalNotice(notice string) {
	if !scene.terminal {
		panic("CharacterCreationScene.setTerminalNotice: terminal scene is required")
	}
	scene.terminalNotice = notice
	scene.keyboard.PrepareForTransition(0)
}

func (scene *CharacterCreationScene) updateName() {
	for _, inputRune := range scene.readInputRunes() {
		if ((inputRune >= 'A' && inputRune <= 'Z') || (inputRune >= 'a' && inputRune <= 'z')) &&
			len(scene.name) < characterCreationNameLimit {
			scene.name = append(scene.name, inputRune)
		}
	}
	if scene.keyboard.IsKeyPressedAndRegister(ebiten.KeyBackspace) {
		if len(scene.name) > 0 {
			scene.name = scene.name[:len(scene.name)-1]
		}

		return
	}
	if scene.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) {
		scene.cancel()

		return
	}
	if !scene.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter) &&
		!scene.keyboard.IsKeyPressedAndRegister(ebiten.KeyNumpadEnter) {
		return
	}
	if len(scene.name) == 0 {
		scene.cancel()

		return
	}
	scene.transitionTo(characterCreationSex)
}

func (scene *CharacterCreationScene) updateSex() {
	// REDUX DEVIATION: The original loops until M/F. Redux lets Escape return to
	// name entry so identity can be corrected without discarding the attempt.
	if scene.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) {
		scene.transitionTo(characterCreationName)

		return
	}
	response := scene.acceptedChoice('M', 'F', ebiten.KeyM, ebiten.KeyF)
	if response == 0 {
		return
	}
	if response == 'M' {
		scene.gender = party_state.Male
	} else {
		scene.gender = party_state.Female
	}
	scene.transitionTo(characterCreationIntroduction)
}

func (scene *CharacterCreationScene) updateFraming(next characterCreationState) {
	if scene.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) {
		scene.requestDiscard()

		return
	}
	if scene.acceptAnyKey() {
		questionnaire, err := character_creation.NewQuestionnaire(string(scene.name), scene.gender, scene.source)
		if err != nil {
			panic("CharacterCreationScene.updateFraming: validated scene inputs were rejected: " + err.Error())
		}
		scene.questionnaire = questionnaire
		scene.transitionTo(next)
	}
}

func (scene *CharacterCreationScene) updateDilemma() {
	if scene.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) {
		scene.requestDiscard()

		return
	}
	response := scene.acceptedChoice('A', 'B', ebiten.KeyA, ebiten.KeyB)
	if response == 0 || !scene.questionnaire.Answer(response) {
		return
	}
	scene.questionNumber++
	scene.advanceQuestionState()
}

func (scene *CharacterCreationScene) advanceQuestionState() {
	switch {
	case scene.questionNumber < characterCreationFirstRoundEnd:
		scene.transitionTo(characterCreationFirstRound)
	case scene.questionNumber < characterCreationSecondRoundEnd:
		scene.transitionTo(characterCreationSecondRound)
	case scene.questionNumber < characterCreationQuestionCount:
		scene.transitionTo(characterCreationFinalRound)
	default:
		scene.transitionTo(characterCreationConclusion)
	}
}

func (scene *CharacterCreationScene) updateConclusion() {
	if scene.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) {
		scene.requestDiscard()

		return
	}
	if !scene.acceptAnyKey() {
		return
	}
	result, complete := scene.questionnaire.Result()
	if !complete {
		panic("CharacterCreationScene.updateConclusion: questionnaire is incomplete")
	}
	scene.result = result
	scene.hasResult = true
	scene.state = characterCreationComplete
	scene.terminal = true
	scene.keyboard.PrepareForTransition(0)
	callback := scene.onComplete
	callback(result)
}

func (scene *CharacterCreationScene) updateDiscardConfirmation() {
	if scene.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) {
		scene.transitionTo(scene.stateBeforeDiscard)

		return
	}
	response := scene.acceptedChoice('Y', 'N', ebiten.KeyY, ebiten.KeyN)
	switch response {
	case 'Y':
		scene.cancel()
	case 'N':
		scene.transitionTo(scene.stateBeforeDiscard)
	}
}

func (scene *CharacterCreationScene) requestDiscard() {
	// REDUX DEVIATION: Original framing accepts Escape as any key and question
	// loops ignore it. Redux consistently confirms before abandoning progress.
	scene.stateBeforeDiscard = scene.state
	scene.transitionTo(characterCreationDiscardConfirmation)
}

func (scene *CharacterCreationScene) cancel() {
	scene.state = characterCreationCancelled
	scene.terminal = true
	scene.keyboard.PrepareForTransition(0)
	callback := scene.onCancel
	callback()
}

func (scene *CharacterCreationScene) transitionTo(next characterCreationState) {
	scene.state = next
	scene.terminal = next == characterCreationComplete || next == characterCreationCancelled
	scene.drainInputRunes()
	scene.keyboard.PrepareForTransition(0)
}

func (scene *CharacterCreationScene) acceptAnyKey() bool {
	pressedKey := scene.keyboard.GetBoundKeyPressed(&boundKeysAny)
	if pressedKey != nil && scene.keyboard.TryToRegisterKeyPress(*pressedKey) {
		return true
	}

	return len(scene.readInputRunes()) != 0
}

func (scene *CharacterCreationScene) acceptedChoice(
	first, second rune,
	firstKey, secondKey ebiten.Key,
) rune {
	if scene.keyboard.IsKeyPressedAndRegister(firstKey) {
		return first
	}
	if scene.keyboard.IsKeyPressedAndRegister(secondKey) {
		return second
	}

	return scene.choiceRune(scene.readInputRunes(), first, second)
}

func (scene *CharacterCreationScene) readInputRunes() []rune {
	if scene.appendInputChars == nil {
		return nil
	}

	return scene.appendInputChars(nil)
}

func (scene *CharacterCreationScene) drainInputRunes() {
	_ = scene.readInputRunes()
}

func (scene *CharacterCreationScene) choiceRune(inputRunes []rune, first, second rune) rune {
	for _, inputRune := range inputRunes {
		switch inputRune {
		case first, first + ('a' - 'A'):
			return first
		case second, second + ('a' - 'A'):
			return second
		}
	}

	return 0
}

//nolint:cyclop // Explicit finite-state rendering mirrors the input state machine.
func (scene *CharacterCreationScene) Draw(screen *ebiten.Image) {
	if scene.disposed || scene.canvas == nil {
		return
	}
	// REDUX DEVIATION: Original new_char draws into the live title buffer. Redux
	// redraws the same title assets from scene-owned presentation state.
	screen.Fill(color.EGABlack)
	scene.canvas.Fill(color.Black)
	drawState := scene.state
	if drawState == characterCreationDiscardConfirmation {
		drawState = scene.stateBeforeDiscard
	}
	switch drawState {
	case characterCreationName:
		scene.drawName()
	case characterCreationSex:
		scene.drawSex()
	case characterCreationIntroduction:
		scene.drawIntroduction()
	case characterCreationFirstRound, characterCreationSecondRound, characterCreationFinalRound:
		scene.drawDilemma()
	case characterCreationConclusion, characterCreationComplete:
		scene.drawConclusion()
	case characterCreationCancelled:
		// A cancellation callback normally replaces the scene synchronously.
	case characterCreationDiscardConfirmation:
		panic("CharacterCreationScene.Draw: discard confirmation cannot be its own backdrop")
	default:
		panic("CharacterCreationScene.Draw: unknown state")
	}
	if scene.terminalNotice != "" {
		scene.promptOutput.Reset()
		line := scene.promptOutput.AppendLine().Yellow(scene.terminalNotice)
		if scene.state == characterCreationComplete {
			widgets.AppendControlHints(line.Text(" ", widgets.DialogControlHintColor), "Esc: Discard.")
		}
		line.Done()
		scene.promptOutput.DrawPanelContent(scene.canvas, image.Pt(8, 176), 304, textv2.AlignCenter)
	}
	drawNarrativeCanvas(screen, scene.canvas)
	if scene.state == characterCreationDiscardConfirmation {
		scene.drawDiscardConfirmation(screen)
	}
	scene.dialogStack.Draw(screen)
}

func (scene *CharacterCreationScene) drawName() {
	scene.drawIdentityPanel(true)
}

func (scene *CharacterCreationScene) drawSex() {
	scene.drawIdentityPanel(false)
}

func (scene *CharacterCreationScene) drawIdentityPanel(enteringName bool) {
	scene.identityPresentation.draw(scene.canvas)
	scene.drawIdentityFrame()
	vector.FillRect(scene.canvas, 120, 120, 81, 7, color.EGABlue, false)
	vector.FillRect(scene.canvas, 120, 127, 81, 1, color.EGAWhite, false)
	vector.FillRect(scene.canvas, 8, 128, 304, 64, color.EGABlack, false)

	scene.drawIdentityLine(image.Pt(24, 136), "By what name shalt thou be known?")
	scene.drawIdentityLine(image.Pt(112, 152), ":")
	scene.drawIdentityLine(image.Pt(120, 152), string(scene.name))
	if enteringName {
		scene.drawIdentityLine(image.Pt(120+len(scene.name)*text.BitmapCharWidth, 152), "_")
	}
	if !enteringName {
		scene.drawIdentityLine(image.Pt(64, 168), "Art thou Male or Female? ")
		scene.drawIdentityLine(image.Pt(264, 168), "_")
	}
}

func (scene *CharacterCreationScene) drawIdentityLine(point image.Point, line string) {
	scene.identityOutput.Reset()
	scene.identityOutput.AppendLine().IBMWithColor(line, color.EGAWhite).Done()
	scene.identityOutput.DrawPanelContent(
		scene.canvas,
		point,
		introLogicalWidth-point.X,
		textv2.AlignStart,
	)
}

func (scene *CharacterCreationScene) drawIdentityFrame() {
	top := string([]byte{123}) + strings.Repeat(string([]byte{127}), 38) + string([]byte{124})
	bottom := string([]byte{125}) + strings.Repeat(string([]byte{127}), 38) + string([]byte{126})
	scene.drawIdentityFrameLine(image.Pt(0, 120), top)
	for y := 128; y < 192; y += text.BitmapCharHeight {
		scene.drawIdentityFrameLine(image.Pt(0, y), string([]byte{127}))
		scene.drawIdentityFrameLine(image.Pt(312, y), string([]byte{127}))
	}
	scene.drawIdentityFrameLine(image.Pt(0, 192), bottom)

	vector.StrokeLine(scene.canvas, 7, 127, 312, 127, 1, color.EGAWhite, false)
	vector.StrokeLine(scene.canvas, 7, 127, 7, 192, 1, color.EGAWhite, false)
	vector.StrokeLine(scene.canvas, 312, 127, 312, 192, 1, color.EGAWhite, false)
	vector.StrokeLine(scene.canvas, 7, 192, 312, 192, 1, color.EGAWhite, false)
	scene.drawIdentityCopyright()
}

func (scene *CharacterCreationScene) drawIdentityFrameLine(point image.Point, line string) {
	scene.identityOutput.Reset()
	scene.identityOutput.AppendLine().IBMWithColor(line, color.EGABlue).Done()
	scene.identityOutput.DrawPanelContent(scene.canvas, point, introLogicalWidth-point.X, textv2.AlignStart)
}

func (scene *CharacterCreationScene) drawIdentityCopyright() {
	vector.FillRect(scene.canvas, 40, 192, 232, 8, color.EGABlack, false)
	vector.FillRect(scene.canvas, 8, 193, 33, 7, color.EGABlue, false)
	vector.FillRect(scene.canvas, 272, 193, 40, 7, color.EGABlue, false)
	vector.StrokeLine(scene.canvas, 8, 192, 40, 192, 1, color.EGAWhite, false)
	vector.StrokeLine(scene.canvas, 272, 192, 311, 192, 1, color.EGAWhite, false)

	scene.drawIdentityFrameLine(image.Pt(40, 192), string([]byte{characterCreationRightArrowGlyph}))
	scene.drawIdentityLine(image.Pt(48, 192), "Copyright 1988 Lord British")
	scene.drawIdentityFrameLine(image.Pt(264, 192), string([]byte{characterCreationLeftArrowGlyph}))
	vector.StrokeLine(scene.canvas, 40, 192, 45, 195, 1, color.EGAWhite, false)
	vector.StrokeLine(scene.canvas, 45, 196, 40, 199, 1, color.EGAWhite, false)
	vector.StrokeLine(scene.canvas, 271, 192, 266, 195, 1, color.EGAWhite, false)
	vector.StrokeLine(scene.canvas, 266, 196, 271, 199, 1, color.EGAWhite, false)
}

func (scene *CharacterCreationScene) drawIntroduction() {
	drawIntroImage(scene.canvas, scene.assets.Oracle, 0, 96)
	layout := storyText(-1, 0, 9, 0, 320, 175, 320, 89, 200)
	drawIntroNarrativeText(scene.canvas, scene.proportionalAssets, scene.assets.Script.Introduction, layout)
}

func (scene *CharacterCreationScene) drawDilemma() {
	dilemma, available := scene.questionnaire.CurrentDilemma()
	if !available {
		panic("CharacterCreationScene.drawDilemma: question is unavailable")
	}
	drawIntroImage(scene.canvas, scene.assets.Incense, 16, 0)
	drawIntroImage(scene.canvas, scene.assets.Incense, 200, 0)
	leftPosition := characterCreationVirtuePositions[dilemma.OptionA]
	rightPosition := characterCreationVirtuePositions[dilemma.OptionB]
	drawIntroImage(
		scene.canvas,
		scene.assets.VirtueSymbols[dilemma.OptionA],
		leftPosition.X,
		leftPosition.Y,
	)
	drawIntroImage(
		scene.canvas,
		scene.assets.VirtueSymbols[dilemma.OptionB],
		rightPosition.X+184,
		rightPosition.Y,
	)
	question := scene.assets.Script.Dilemma(int(dilemma.OptionA), int(dilemma.OptionB))
	layout := storyText(-1, 0, 152, 0, 320, 0, 320, 200, 200)
	drawIntroNarrativeText(scene.canvas, scene.proportionalAssets, question, layout)
}

func (scene *CharacterCreationScene) drawConclusion() {
	drawIntroImage(scene.canvas, scene.assets.Shrine, 168, 100)
	layout := storyText(-1, 0, 0, 0, 320, 0, 166, 90, 200)
	layout.spaceWidth = 4
	drawIntroNarrativeText(scene.canvas, scene.proportionalAssets, scene.assets.Script.Conclusion, layout)
}

func (scene *CharacterCreationScene) drawDiscardConfirmation(screen *ebiten.Image) {
	pixelUnit, padding := scene.prepareDiscardConfirmation(screen.Bounds())
	scene.drawPreparedDiscardConfirmation(screen, pixelUnit, padding)
}

func (scene *CharacterCreationScene) prepareDiscardConfirmation(screen image.Rectangle) (int, int) {
	const title = "Discard this character?"
	const choices = "Y / N"
	const hint = "Escape continues creation."
	width, height := screen.Dx(), screen.Dy()
	if scene.discardOutput == nil || scene.discardScreenWidth != width {
		_, scene.discardOutput = widgets.GetScaledFontWithOutputForWidth(widgets.FontConfig{
			Size: widgets.FontSizeLarge, LineSpacing: 0, LineSpacingMultiplier: 1.3,
			MaxLines: 3, MaxChars: max(len(title), len(choices), len(hint)),
		}, width)
		scene.discardOutput.AppendLine().White(title).Done()
		scene.discardOutput.AppendLine().Yellow(choices).Done()
		scene.discardOutput.AppendLine().LightGray(hint).Done()
		scene.discardScreenWidth = width
	}
	output := scene.discardOutput
	textWidth := 0.0
	for _, line := range [...]string{title, choices, hint} {
		lineWidth, _ := output.Font.MeasureString(line)
		textWidth = max(textWidth, lineWidth)
	}
	pixelUnit := widgets.ClassicUIPixelUnit(width, height)
	padding := max(pixelUnit*4, int(math.Ceil(output.GetFontHeight())))
	cardWidth := int(math.Ceil(textWidth)) + padding*2
	cardHeight := int(math.Ceil(output.GetTotalHeightForLines(3))) + padding*2
	origin := screen.Min.Add(image.Pt((width-cardWidth)/2, (height-cardHeight)/2))
	scene.discardBounds = image.Rectangle{Min: origin, Max: origin.Add(image.Pt(cardWidth, cardHeight))}

	return pixelUnit, padding
}

func (scene *CharacterCreationScene) drawPreparedDiscardConfirmation(screen *ebiten.Image, pixelUnit, padding int) {
	widgets.FillClassicPixelRoundedRect(screen, scene.discardBounds, pixelUnit, color.EGABlack)
	widgets.DrawClassicPixelRoundedFrame(screen, scene.discardBounds, pixelUnit, widgets.DialogRuleColor)
	scene.discardOutput.DrawPanelContent(screen, scene.discardBounds.Min.Add(image.Pt(padding, padding)),
		scene.discardBounds.Dx()-padding*2, textv2.AlignCenter)
}
