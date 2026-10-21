package widgets

import (
	"fmt"
	"image"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

const (
	interactionTranscriptDefaultCenterY = 0.70
	interactionTranscriptPaddingUnits   = 3
	interactionTranscriptGapUnits       = 2
	interactionTranscriptPromptMarker   = "  > "
	interactionTranscriptMaxWidth       = 0.66
	interactionTranscriptMinInputChars  = 4
)

type interactionTranscriptMode int

const (
	interactionTranscriptIdle interactionTranscriptMode = iota
	interactionTranscriptYesNo
	interactionTranscriptUppercaseYesNo
	interactionTranscriptText
	interactionTranscriptAcknowledgement
	interactionTranscriptContinue
)

// InteractionTranscriptConfig sizes one compact staged-interaction transcript.
type InteractionTranscriptConfig struct {
	MaxLines          int
	MaxInputLength    int
	WidthSamples      []string
	CenterYPercent    float64
	PanelPlacement    *sprites.PercentBasedPlacement
	ClearTextOnEscape bool
}

type interactionTranscriptEntry struct {
	message  string
	prompt   string
	response string
}

type interactionTranscriptLayout struct {
	panelRect      image.Rectangle
	transcriptRect image.Rectangle
	footerRect     image.Rectangle
	pixelUnit      int
	lineGap        int
}

// InteractionTranscript is a compact persistent transcript for staged workflows.
// It captures raw prompt results; the Event Queue owns sequencing and gameplay changes.
type InteractionTranscript struct {
	keyboard *input.Keyboard
	font     *text.UltimaFont

	transcriptOutput *text.Output
	footerOutput     *text.Output
	textInput        *TextInput
	entries          []interactionTranscriptEntry

	mode              interactionTranscriptMode
	activePrompt      string
	response          string
	submitted         bool
	inputComplete     bool
	clearTextOnEscape bool
	maxInputLength    int
	maxLines          int
	widthSamples      []string
	centerYPercent    float64
	panelPlacement    *sprites.PercentBasedPlacement
	controlHint       string
	stackedInput      bool
	layout            interactionTranscriptLayout
	lastWidth         int
	lastHeight        int
}

var _ Widget = (*InteractionTranscript)(nil)

// NewInteractionTranscript creates a compact persistent transcript for one staged workflow.
func NewInteractionTranscript(
	keyboard *input.Keyboard,
	config InteractionTranscriptConfig,
) *InteractionTranscript {
	if keyboard == nil {
		panic("NewInteractionTranscript: keyboard is required")
	}
	if config.MaxLines <= 0 {
		panic("NewInteractionTranscript: max lines must be positive")
	}
	if config.MaxInputLength <= 0 {
		panic("NewInteractionTranscript: max input length must be positive")
	}
	if len(config.WidthSamples) == 0 {
		panic("NewInteractionTranscript: width samples are required")
	}
	centerYPercent := config.CenterYPercent
	if centerYPercent == 0 {
		centerYPercent = interactionTranscriptDefaultCenterY
	}
	if centerYPercent <= 0 || centerYPercent >= 1 {
		panic("NewInteractionTranscript: center Y percent must be between zero and one")
	}
	var panelPlacement *sprites.PercentBasedPlacement
	if config.PanelPlacement != nil {
		placement := *config.PanelPlacement
		if placement.StartPercentX < 0 || placement.EndPercentX > 1 ||
			placement.StartPercentY < 0 || placement.EndPercentY > 1 ||
			placement.StartPercentX >= placement.EndPercentX ||
			placement.StartPercentY >= placement.EndPercentY {
			panic("NewInteractionTranscript: panel placement must be a non-empty screen region")
		}
		panelPlacement = &placement
	}

	dialog := &InteractionTranscript{
		keyboard:          keyboard,
		font:              nil,
		transcriptOutput:  nil,
		footerOutput:      nil,
		textInput:         nil,
		entries:           nil,
		mode:              interactionTranscriptIdle,
		activePrompt:      "",
		response:          "",
		submitted:         false,
		inputComplete:     false,
		clearTextOnEscape: config.ClearTextOnEscape,
		maxInputLength:    config.MaxInputLength,
		maxLines:          config.MaxLines,
		widthSamples:      append([]string(nil), config.WidthSamples...),
		centerYPercent:    centerYPercent,
		panelPlacement:    panelPlacement,
		controlHint:       "",
		stackedInput:      false,
		layout: interactionTranscriptLayout{
			panelRect:      image.Rect(0, 0, 0, 0),
			transcriptRect: image.Rect(0, 0, 0, 0),
			footerRect:     image.Rect(0, 0, 0, 0),
			pixelUnit:      0,
			lineGap:        0,
		},
		lastWidth:  0,
		lastHeight: 0,
	}
	dialog.textInput = newInteractionTranscriptTextInput(keyboard, config.MaxInputLength)

	width, height := display.GetManager().GetCurrentSize()
	dialog.ensureLayout(width, height)

	return dialog
}

func newInteractionTranscriptTextInput(keyboard *input.Keyboard, maxInputLength int) *TextInput {
	textInput := NewTextInput(
		sprites.PercentBasedPlacement{
			StartPercentX: 0,
			EndPercentX:   0,
			StartPercentY: 0,
			EndPercentY:   0,
		},
		GetScaledLineHeight(FontSizeLarge),
		maxInputLength,
		nil,
		TextInputCallbacks{
			AmbiguousAutoComplete: nil,
			OnEnter:               nil,
			OnCommandExecuted:     nil,
			ShowCommandList:       nil,
		},
		keyboard,
	)
	textInput.SetInputColors(TextInputColors{
		DefaultColor:          ucolor.DialogInputTextColor,
		NoMatchesColor:        ucolor.DialogInputTextColor,
		OneMatchColor:         ucolor.DialogInputTextColor,
		MoreThanOneMatchColor: ucolor.DialogInputTextColor,
	})
	textInput.SetPromptPrefix(interactionTranscriptPromptMarker)
	textInput.SetPromptColor(ucolor.DialogInputTextColor)
	textInput.SetCursorVisible(false)

	return textInput
}

// AppendLine adds an informational or outcome line to the bounded transcript.
func (d *InteractionTranscript) AppendLine(message string) {
	d.entries = append(d.entries, interactionTranscriptEntry{
		message:  message,
		prompt:   "",
		response: "",
	})
	d.rebuildTranscriptOutput()
}

// AppendToLastLine extends the latest informational line for staged animation.
func (d *InteractionTranscript) AppendToLastLine(suffix string) {
	if len(d.entries) == 0 || d.entries[len(d.entries)-1].prompt != "" {
		panic("InteractionTranscript.AppendToLastLine requires an informational line")
	}
	d.entries[len(d.entries)-1].message += suffix
	d.rebuildTranscriptOutput()
}

// AppendResponse adds a completed non-text-selector response to the transcript.
func (d *InteractionTranscript) AppendResponse(prompt, response string) {
	d.entries = append(d.entries, interactionTranscriptEntry{
		message:  "",
		prompt:   prompt,
		response: response,
	})
	d.rebuildTranscriptOutput()
}

// BeginYesNo activates a Yes/No prompt while retaining prior transcript lines.
func (d *InteractionTranscript) BeginYesNo(prompt string) {
	d.beginPrompt(interactionTranscriptYesNo, prompt)
	d.textInput.SetAutoReplaceMode(true)
	if err := d.textInput.SetInputValidationRegex("(?i)^[yn]?$"); err != nil {
		panic(fmt.Sprintf("configure interaction transcript yes/no input: %v", err))
	}
}

// BeginUppercaseYesNo activates an original-style prompt that accepts only Y or N.
func (d *InteractionTranscript) BeginUppercaseYesNo(prompt string) {
	d.beginPrompt(interactionTranscriptUppercaseYesNo, prompt)
	d.textInput.SetAutoReplaceMode(false)
	if err := d.textInput.SetInputValidationRegex("^[YN]?$"); err != nil {
		panic(fmt.Sprintf("configure interaction transcript uppercase yes/no input: %v", err))
	}
}

// BeginText activates text entry with caller-supplied validation and the configured Escape policy.
func (d *InteractionTranscript) BeginText(prompt, validation string) {
	d.beginPrompt(interactionTranscriptText, prompt)
	d.textInput.SetAutoReplaceMode(false)
	if err := d.textInput.SetInputValidationRegex(validation); err != nil {
		panic(fmt.Sprintf("configure interaction transcript input: %v", err))
	}
}

// BeginAcknowledgement displays close guidance and waits for Enter or Escape.
func (d *InteractionTranscript) BeginAcknowledgement() {
	d.beginConfirmation(interactionTranscriptAcknowledgement)
}

// BeginContinue displays continuation guidance and waits for Enter or Escape.
func (d *InteractionTranscript) BeginContinue() {
	d.beginConfirmation(interactionTranscriptContinue)
}

func (d *InteractionTranscript) beginConfirmation(mode interactionTranscriptMode) {
	d.keyboard.PrepareForTransition(0)
	d.mode = mode
	d.controlHint = ""
	d.activePrompt = ""
	d.response = ""
	d.submitted = false
	d.inputComplete = false
	d.textInput.ClearText()
	d.textInput.SetCursorVisible(false)
	d.rebuildTranscriptOutput()
	d.rebuildFooterOutput()
	d.updateInputPlacement()
}

// InputComplete reports whether the active prompt or acknowledgement has completed.
func (d *InteractionTranscript) InputComplete() bool {
	return d.inputComplete
}

// Response returns the last completed prompt response.
func (d *InteractionTranscript) Response() string {
	return d.response
}

// Submitted distinguishes an entered response from cancelled text entry.
func (d *InteractionTranscript) Submitted() bool {
	return d.submitted
}

// SetControlHint displays presentation-owned guidance without taking input ownership.
func (d *InteractionTranscript) SetControlHint(hint string) {
	d.controlHint = strings.TrimSpace(hint)
	d.rebuildFooterOutput()
}

// Draw renders the transcript, active prompt, guidance, and enclosing frame.
func (d *InteractionTranscript) Draw(screen *ebiten.Image) {
	if screen == nil {
		return
	}
	d.ensureLayout(screen.Bounds().Dx(), screen.Bounds().Dy())

	FillClassicPixelRoundedRect(screen, d.layout.panelRect, d.layout.pixelUnit, ucolor.DialogBackground.RGBA())
	d.transcriptOutput.DrawScrollingBufferWithHeight(
		screen,
		d.layout.transcriptRect.Min,
		d.layout.transcriptRect.Dx(),
		d.layout.transcriptRect.Dy(),
		false,
		etext.AlignStart,
		etext.AlignStart,
	)
	if d.activePrompt != "" {
		d.textInput.Draw(screen)
	}
	d.footerOutput.DrawPanelContent(
		screen,
		image.Pt(d.layout.footerRect.Min.X, verticallyCenteredOutputY(d.layout.footerRect, d.footerOutput)),
		d.layout.footerRect.Dx(),
		etext.AlignCenter,
	)
	DrawClassicPixelRoundedFrame(screen, d.layout.panelRect, d.layout.pixelUnit, DialogRuleColor)
}

// Update captures raw input for the active presentation stage.
func (d *InteractionTranscript) Update() {
	width, height := display.GetManager().GetCurrentSize()
	d.ensureLayout(width, height)

	switch d.mode {
	case interactionTranscriptIdle:
		return
	case interactionTranscriptAcknowledgement, interactionTranscriptContinue:
		if d.consumeConfirmationKey() {
			d.mode = interactionTranscriptIdle
			d.inputComplete = true
			d.rebuildFooterOutput()
		}
	case interactionTranscriptYesNo, interactionTranscriptUppercaseYesNo, interactionTranscriptText:
		d.updatePromptInput()
	}
}

func (d *InteractionTranscript) beginPrompt(mode interactionTranscriptMode, prompt string) {
	if strings.TrimSpace(prompt) == "" {
		panic("InteractionTranscript prompt is required")
	}
	d.keyboard.PrepareForTransition(0)
	d.mode = mode
	d.controlHint = ""
	d.activePrompt = prompt
	d.response = ""
	d.submitted = false
	d.inputComplete = false
	d.textInput.ClearText()
	d.textInput.SetCursorVisible(true)
	d.rebuildTranscriptOutput()
	d.rebuildFooterOutput()
	d.updateInputPlacement()
}

func (d *InteractionTranscript) updatePromptInput() {
	if d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter) ||
		d.keyboard.IsKeyPressedAndRegister(ebiten.KeyNumpadEnter) {
		response := d.textInput.GetText()
		if d.mode == interactionTranscriptYesNo || d.mode == interactionTranscriptUppercaseYesNo {
			comparableResponse := response
			if d.mode == interactionTranscriptYesNo {
				comparableResponse = strings.ToLower(response)
			}
			switch comparableResponse {
			case "y":
				response = "Yes"
			case "n":
				response = "No"
			case "Y":
				response = "Yes"
			case "N":
				response = "No"
			default:
				return
			}
		}
		d.finishPrompt(response, true)

		return
	}
	if d.mode == interactionTranscriptText && d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) {
		if d.clearTextOnEscape {
			d.textInput.ClearText()

			return
		}
		d.finishPrompt("", false)

		return
	}
	d.textInput.Update()
}

func (d *InteractionTranscript) finishPrompt(response string, submitted bool) {
	prompt := d.activePrompt
	d.response = response
	d.submitted = submitted
	d.inputComplete = true
	d.mode = interactionTranscriptIdle
	d.activePrompt = ""
	d.textInput.SetCursorVisible(false)
	if submitted {
		d.entries = append(d.entries, interactionTranscriptEntry{
			message:  "",
			prompt:   prompt,
			response: response,
		})
	}
	d.rebuildTranscriptOutput()
	d.rebuildFooterOutput()
	d.updateInputPlacement()
}

func (d *InteractionTranscript) consumeConfirmationKey() bool {
	return d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter) ||
		d.keyboard.IsKeyPressedAndRegister(ebiten.KeyNumpadEnter) ||
		d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape)
}

func (d *InteractionTranscript) ensureLayout(width, height int) {
	if width <= 0 || height <= 0 || (d.lastWidth == width && d.lastHeight == height) {
		return
	}
	d.lastWidth = width
	d.lastHeight = height
	d.font = GetScaledFont(FontSizeLarge)
	fontPoint := GetScaledLineHeight(FontSizeLarge)
	lineSpacing := text.GetScaledNumberToResolution(display.GetManager(), messageFontLineSpacing)
	pixelUnit := getClassicUIPixelUnit(width, height)
	padding := max(pixelUnit*interactionTranscriptPaddingUnits, inputPromptMinimumPadding)
	gap := max(1, pixelUnit*interactionTranscriptGapUnits)
	charsPerLine, preferredContentWidth := d.preferredContentSize()
	measurementOutput := text.NewOutput(d.font, lineSpacing, d.maxLines, charsPerLine)
	transcriptHeight := int(math.Ceil(measurementOutput.GetTotalHeightForLines(d.maxLines)))
	footerHeight := int(math.Ceil(measurementOutput.GetFontHeight()))
	footerLines := 1
	var panelRect image.Rectangle
	if d.panelPlacement != nil {
		panelRect = percentPlacementToPixelRectangle(*d.panelPlacement, width, height)
		footerLines = 2
		footerHeight = int(math.Ceil(measurementOutput.GetTotalHeightForLines(footerLines)))
	} else {
		maximumPanelWidth := int(math.Round(interactionTranscriptMaxWidth * float64(width)))
		panelWidth := min(width-padding*2, maximumPanelWidth, preferredContentWidth+padding*2)
		panelHeight := padding*2 + transcriptHeight + gap + footerHeight
		centerX := int(math.Round(DialogHorizontalMapOnly.ToPercentage() * float64(width)))
		centerY := int(math.Round(d.centerYPercent * float64(height)))
		panelMinX := centeredPromptRectMinX(centerX, panelWidth, width, padding)
		panelMinY := min(max(padding, centerY-panelHeight/2), height-padding-panelHeight)
		panelRect = image.Rect(panelMinX, panelMinY, panelMinX+panelWidth, panelMinY+panelHeight)
	}
	contentRect := insetRectangle(panelRect, padding)
	characterWidth, _ := d.font.MeasureString("M")
	charsPerLine = min(charsPerLine, max(1, int(float64(contentRect.Dx())/characterWidth)))
	if d.panelPlacement != nil {
		transcriptHeight = max(1, contentRect.Dy()-gap-footerHeight)
	}
	effectiveMaxLines := d.maxLines
	if d.panelPlacement != nil {
		effectiveMaxLines = max(effectiveMaxLines, measurementOutput.GetLineCapacity(transcriptHeight))
	}
	transcriptRect := image.Rect(
		contentRect.Min.X,
		contentRect.Min.Y,
		contentRect.Max.X,
		contentRect.Min.Y+transcriptHeight,
	)
	footerRect := image.Rect(
		contentRect.Min.X,
		transcriptRect.Max.Y+gap,
		contentRect.Max.X,
		transcriptRect.Max.Y+gap+footerHeight,
	)
	d.layout = interactionTranscriptLayout{
		panelRect:      panelRect,
		transcriptRect: transcriptRect,
		footerRect:     footerRect,
		pixelUnit:      pixelUnit,
		lineGap:        gap,
	}

	d.transcriptOutput = text.NewOutput(d.font, lineSpacing, effectiveMaxLines, charsPerLine)
	d.footerOutput = text.NewOutput(d.font, fontPoint, footerLines, charsPerLine)

	d.textInput.SetFontPoint(fontPoint)
	d.rebuildTranscriptOutput()
	d.rebuildFooterOutput()
	d.updateInputPlacement()
}

func (d *InteractionTranscript) preferredContentSize() (int, int) {
	samples := append(append([]string(nil), d.widthSamples...),
		yesNoControlHint,
		textSubmitControlHint,
		DialogCloseHint,
		DialogContinueHint,
	)
	maxChars := 1
	maxWidth := 1.0
	for _, sample := range samples {
		maxChars = max(maxChars, len(sample))
		measuredWidth, _ := d.font.MeasureString(sample)
		maxWidth = max(maxWidth, measuredWidth)
	}

	return maxChars, int(math.Ceil(maxWidth))
}

func (d *InteractionTranscript) updateInputPlacement() {
	if d.font == nil || d.lastWidth <= 0 || d.lastHeight <= 0 {
		return
	}
	promptWidth, _ := d.font.MeasureString(d.activePrompt)
	minimumInputSample := interactionTranscriptPromptMarker +
		strings.Repeat("M", min(interactionTranscriptMinInputChars, d.maxInputLength))
	minimumInputWidth, _ := d.font.MeasureString(minimumInputSample)
	wasStacked := d.stackedInput
	d.stackedInput = d.activePrompt != "" && int(math.Ceil(promptWidth+minimumInputWidth)) > d.layout.transcriptRect.Dx()
	if d.stackedInput != wasStacked {
		d.rebuildTranscriptOutput()
	}
	visibleRows := d.transcriptOutput.GetVisibleScrollingLineCount(d.layout.transcriptRect.Dy())
	activeRow := max(0, visibleRows-1)
	rowY := d.layout.transcriptRect.Min.Y +
		int(math.Round(float64(activeRow)*d.transcriptOutput.GetLineSpacing()))
	lineHeight := int(math.Ceil(d.transcriptOutput.GetFontHeight()))
	inputRect := image.Rect(
		d.layout.transcriptRect.Min.X,
		rowY,
		d.layout.transcriptRect.Max.X,
		rowY+lineHeight,
	)
	if !d.stackedInput {
		inputRect.Min.X = min(inputRect.Max.X, inputRect.Min.X+int(math.Ceil(promptWidth)))
	}
	d.textInput.SetPlacement(pixelRectangleToPercentPlacement(inputRect, d.lastWidth, d.lastHeight))
}

func (d *InteractionTranscript) rebuildTranscriptOutput() {
	if d.transcriptOutput == nil {
		return
	}
	d.transcriptOutput.Reset()
	for _, entry := range d.entries {
		if entry.prompt == "" {
			d.transcriptOutput.AppendLine().Text(entry.message, DialogValueColor).WordWrap().Done()

			continue
		}
		d.transcriptOutput.AppendLine().
			Text(entry.prompt, DialogValueColor).
			Text(interactionTranscriptPromptMarker, ucolor.DialogInputTextColor).
			Text(entry.response, ucolor.DialogInputTextColor).
			WordWrap().
			Done()
	}
	if d.activePrompt != "" {
		d.transcriptOutput.AppendLine().Text(d.activePrompt, DialogValueColor).WordWrap().Done()
		if d.stackedInput {
			d.transcriptOutput.AppendLine().Text(" ", DialogValueColor).Done()
		}
	}
}

func (d *InteractionTranscript) rebuildFooterOutput() {
	if d.footerOutput == nil {
		return
	}
	d.footerOutput.Reset()
	hint := d.controlHint
	if hint == "" {
		switch d.mode {
		case interactionTranscriptYesNo, interactionTranscriptUppercaseYesNo:
			hint = yesNoControlHint
		case interactionTranscriptText:
			hint = textSubmitControlHint
			if d.clearTextOnEscape {
				hint = "Enter: Submit   Esc: Clear text"
			}
		case interactionTranscriptAcknowledgement:
			hint = DialogCloseHint
		case interactionTranscriptContinue:
			hint = DialogContinueHint
		case interactionTranscriptIdle:
			return
		}
	}
	AppendControlHints(d.footerOutput.AppendLine().WithAlignment(etext.AlignCenter), hint).WordWrap().Done()
}

func percentPlacementToPixelRectangle(placement sprites.PercentBasedPlacement, width, height int) image.Rectangle {
	return image.Rect(
		int(math.Round(float64(width)*placement.StartPercentX)),
		int(math.Round(float64(height)*placement.StartPercentY)),
		int(math.Round(float64(width)*placement.EndPercentX)),
		int(math.Round(float64(height)*placement.EndPercentY)),
	)
}
