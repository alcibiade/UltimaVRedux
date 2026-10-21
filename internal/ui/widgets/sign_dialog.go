package widgets

import (
	"image"
	stdcolor "image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

const (
	centerDivisor          = 2  // Divisor for centering calculations
	signCanvasColumns      = 16 // Original message-window width
	signCanvasRows         = 13 // Original message-window height
	signCanvasPaddingCells = 1  // Keep one source-character cell clear of the dialog border
)

// SignDialog displays one complete sign using source bitmap fonts. It renders
// once to a static buffer, then waits for any key to dismiss.
type SignDialog struct {
	// Display buffer (rendered once, reused every frame)
	buffer      *ebiten.Image
	drawOptions *ebiten.DrawImageOptions

	// Border and positioning
	border    *Border
	bounds    image.Rectangle
	pixelUnit int
	hint      staticViewerHint

	// State
	dismissed         bool
	waitingForRelease bool
	keyboard          *input.Keyboard

	// Sign positioning within the dialog.
	signOffsetX      int
	signOffsetY      int
	signBounds       image.Rectangle
	rawText          []byte
	translated       bool
	translationRows  [][]text.ColorSegment
	placement        sprites.PercentBasedPlacement
	scale            int
	lastScreenWidth  int
	lastScreenHeight int
}

// SignDialogConfig provides configuration for creating a sign dialog.
type SignDialogConfig struct {
	Placement sprites.PercentBasedPlacement
	Keyboard  *input.Keyboard
	Sign      *references.SignRecord
	RawText   []byte // Alternative: provide raw text directly
	Scale     int    // Optional fixed integer scale; zero selects the largest fitting scale
}

// NewSignDialog creates a static, single-dismissal sign display dialog.
func NewSignDialog(config SignDialogConfig) *SignDialog {
	// Determine raw text source
	var rawText []byte
	if config.Sign != nil {
		rawText = config.Sign.RawText
	} else {
		rawText = config.RawText
	}

	// Handle empty sign
	if len(rawText) == 0 {
		rawText = []byte("(empty sign)")
	}

	dialog := &SignDialog{
		buffer:            nil, // ensureLayout allocates the buffer and border
		drawOptions:       &ebiten.DrawImageOptions{},
		border:            nil,
		bounds:            image.Rect(0, 0, 0, 0),
		pixelUnit:         0,
		hint:              staticViewerHint{output: nil, width: 0},
		dismissed:         false,
		waitingForRelease: true,
		keyboard:          config.Keyboard,
		signOffsetX:       0,
		signOffsetY:       0,
		signBounds:        image.Rect(0, 0, 0, 0),
		rawText:           append([]byte(nil), rawText...),
		translated:        false,
		translationRows:   nil,
		placement:         config.Placement,
		scale:             config.Scale,
		lastScreenWidth:   0,
		lastScreenHeight:  0,
	}

	// Initialize keyboard debouncing
	if config.Keyboard != nil {
		config.Keyboard.PrepareForTransition(KeyboardTransitionDelayMs)
	}

	dialog.ensureLayout()

	return dialog
}

func (d *SignDialog) ensureLayout() {
	screenWidth, screenHeight := display.GetManager().GetCurrentSize()
	if d.buffer != nil && screenWidth == d.lastScreenWidth && screenHeight == d.lastScreenHeight {
		return
	}
	d.lastScreenWidth, d.lastScreenHeight = screenWidth, screenHeight
	d.border = newDialogBorder(d.placement)
	d.bounds = d.border.GetBounds()
	d.pixelUnit = getClassicUIPixelUnit(screenWidth, screenHeight)

	// Create buffer for static rendering
	if d.buffer != nil {
		d.buffer.Deallocate()
	}
	d.buffer = ebiten.NewImage(d.bounds.Dx(), d.bounds.Dy())

	// Position draw options for the buffer
	d.drawOptions.GeoM.Reset()
	d.drawOptions.GeoM.Translate(float64(d.bounds.Min.X), float64(d.bounds.Min.Y))

	scale := d.scale
	if scale <= 0 {
		scale = fittingSignScale(d.border.GetInteriorBounds())
	}
	d.renderSign(d.rawText, text.NewSignRenderer(float64(scale)))
}

func fittingSignScale(interior image.Rectangle) int {
	canvasWidth := (signCanvasColumns + signCanvasPaddingCells*2) * text.BitmapCharWidth
	canvasHeight := (signCanvasRows + signCanvasPaddingCells*2) * text.BitmapCharHeight
	scale := min(interior.Dx()/canvasWidth, interior.Dy()/canvasHeight)

	return max(scale, 1)
}

func (d *SignDialog) renderSign(rawText []byte, renderer *text.SignRenderer) {
	defer applyClassicPixelRoundedMask(d.buffer, d.pixelUnit)

	// Clear buffer with black
	vector.FillRect(d.buffer,
		0, 0,
		float32(d.bounds.Dx()), float32(d.bounds.Dy()),
		stdcolor.Black,
		false)

	var result *text.SignRenderResult
	if d.translated {
		result = renderer.RenderTranslatedSign(rawText)
		d.translationRows = result.TranslationRows
	} else {
		result = renderer.RenderSign(rawText)
	}
	defer result.Image.Deallocate()

	// Center the source artwork inside the border-safe interior.
	interior := d.border.GetInteriorBounds()
	d.signOffsetX = interior.Min.X - d.bounds.Min.X + (interior.Dx()-result.Width)/centerDivisor
	d.signOffsetY = interior.Min.Y - d.bounds.Min.Y + (interior.Dy()-result.Height)/centerDivisor
	d.signBounds = image.Rect(d.signOffsetX, d.signOffsetY, d.signOffsetX+result.Width, d.signOffsetY+result.Height)

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(d.signOffsetX), float64(d.signOffsetY))
	d.buffer.DrawImage(result.Image, opts)
}

// Draw implements Widget interface.
// Simply blits the pre-rendered buffer - no per-frame rendering.
func (d *SignDialog) Draw(screen *ebiten.Image) {
	d.ensureLayout()
	// Draw the pre-rendered content
	screen.DrawImage(d.buffer, d.drawOptions)

	DrawClassicPixelRoundedFrame(screen, d.bounds, d.pixelUnit, DialogRuleColor)
	d.hint.draw(screen, d.bounds, ControlHint{"Ctrl+R", "Runes"}, ControlHint{"Other keys", hintActionClose})
}

// Update implements Widget interface.
// Handles keyboard dismissal with proper debouncing.
func (d *SignDialog) Update() {
	d.ensureLayout()
	if d.dismissed {
		return
	}
	if d.keyboard == nil {
		return
	}

	// Phase 1: Wait for trigger keys to be released
	if d.waitingForRelease {
		if d.keyboard.IsWaitingForRelease() {
			d.keyboard.TryAcceptKey(0)

			return
		}
		d.waitingForRelease = false
	}

	// Phase 2: Accept any new keypress
	if d.keyboard.IsAnyKeyPressed() {
		d.keyboard.PrepareForTransition(KeyboardTransitionDelayMs)

		d.dismissed = true
	}
}

// IsDismissed returns true if the dialog has been dismissed by user input.
func (d *SignDialog) IsDismissed() bool {
	return d.dismissed
}
