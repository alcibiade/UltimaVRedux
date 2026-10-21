package widgets

import (
	"image"
	stdcolor "image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	pkgcolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

// AstronomyDialog displays a star map showing planets and Shadowlord positions.
// Uses static rendering - draws once to buffer, then blits buffer each frame.
type AstronomyDialog struct {
	// Display buffer (rendered once, reused every frame)
	buffer      *ebiten.Image
	drawOptions *ebiten.DrawImageOptions

	// Border and positioning
	border    *Border
	placement sprites.PercentBasedPlacement
	bounds    image.Rectangle
	pixelUnit int
	hint      staticViewerHint

	// State
	dismissed         bool
	waitingForRelease bool // Debouncing: wait for trigger keys to be released
	keyboard          *input.Keyboard

	// Data for rendering (set on creation, used during initial render)
	shadowlordPlanetIndices []int // Indices 0-7 of planets with Shadowlords
	spriteSheet             *sprites.SpriteSheet
	planetXPositions        [astronomyPlanetCount]int // Calculated X positions for each planet
	gameYear                uint16                    // Game year for planet calculations
	gameMonth               byte                      // Game month
	gameDay                 byte                      // Game day
	starPositions           [astronomyStarCount]image.Point
}

// AstronomyDialogConfig provides configuration for creating an astronomy dialog.
type AstronomyDialogConfig struct {
	Placement               sprites.PercentBasedPlacement
	Keyboard                *input.Keyboard
	SpriteSheet             *sprites.SpriteSheet
	ShadowlordPlanetIndices []int  // Planet indices (0-7) where Shadowlords are located
	GameYear                uint16 // Current game year for planet position calculation
	GameMonth               byte   // Current game month (1-13)
	GameDay                 byte   // Current game day (1-28)
	RandomIntInRange        func(minimum, maximum int) int
}

const (
	astronomyStarCount        = 80 // Number of random stars
	astronomyPlanetCount      = 8  // Number of planets
	astronomyYearCycle        = 100
	astronomyStarAreaRatio    = 0.9
	astronomyRenderedStarSize = 4

	astronomyTrackLength     = 22  // Track positions per planet
	astronomyTrackStop       = '#' // Valid planet position in an astronomyPlanetTracks row
	astronomyDaysPerMonth    = 28
	astronomyMonthsPerYear   = 13
	astronomyPlanetAreaRatio = 0.9  // Planets use the top 90% of the dialog height
	astronomyTrackRows       = 20.0 // Original track height in rows

	// Artwork is drawn at three times the original pixel size.
	astronomyPixelScale       = 3
	astronomyPlanetRadius     = 4  // 9x9 planet centre
	astronomyPlanetArmOffset  = 6  // Cardinal planet extensions
	astronomyCometOffset      = 15 // Comet head distance right of its planet
	astronomyCometHeadRadius  = 4  // 9-pixel-tall comet head
	astronomyCometHeadHeight  = 9
	astronomyCometTailLength  = 21
	astronomyCenterDivisor    = 2
	astronomyTelescopeSize    = 18 // Telescope sprite size in source pixels
	astronomyTelescopeScale   = 4.0
	astronomyTelescopePadding = 4 // Pixels between the telescope and the dialog bottom
)

// astronomyPlanetYPositions returns the fixed vertical track row of each planet.
// The planet-track functions hold the original coordinates; ref:ASTRO-0005.
func astronomyPlanetYPositions() [astronomyPlanetCount]int {
	return [astronomyPlanetCount]int{18, 17, 15, 13, 11, 8, 5, 1}
}

// astronomyPlanetStartIndex returns each planet's zero-based track index at the
// reference date.
func astronomyPlanetStartIndex() [astronomyPlanetCount]int {
	return [astronomyPlanetCount]int{18, 2, 8, 15, 11, 6, 4, 2}
}

// astronomyPlanetTracks returns each planet's movement pattern; astronomyTrackStop
// marks a valid position. Planet 0 is slowest (3 positions) and planet 7 is
// fastest (22 positions).
func astronomyPlanetTracks() [astronomyPlanetCount]string {
	return [astronomyPlanetCount]string{
		"....#......#......#...",
		"..#....#...#...#....#.",
		"..#..#..#..#..#..#..#.",
		".#.#.#.#.#.#.#.#.#.#.#",
		"#.#.#.#.##.#.##.#.##.#",
		".###.###.#####.###.###",
		".######.#######.######",
		"######################",
	}
}

// calculatePlanetXPosition calculates the one-based planet column based on game date.
// Uses the original date-driven track walk; ref:ASTRO-0005.
func calculatePlanetXPosition(planet int, year uint16, month, day byte) int {
	const refYear, refMonth, refDay = 39, 4, 5
	track := astronomyPlanetTracks()[planet]
	currentYear := int(year % astronomyYearCycle)
	currentMonth := int(month)
	currentDay := int(day)
	idx := astronomyPlanetStartIndex()[planet]
	for currentYear > refYear || (currentYear == refYear && currentMonth > refMonth) ||
		(currentYear == refYear && currentMonth == refMonth && currentDay > refDay) {
		currentDay--
		if currentDay == 0 {
			currentDay = astronomyDaysPerMonth
			currentMonth--
		}
		if currentMonth == 0 {
			currentMonth = astronomyMonthsPerYear
			currentYear--
		}
		for {
			idx--
			if idx < 0 {
				idx = astronomyTrackLength - 1
			}
			if track[idx] == astronomyTrackStop {
				break
			}
		}
	}

	return idx + 1
}

// NewAstronomyDialog creates a new astronomy display dialog.
// The dialog renders once on creation (static), then simply blits the buffer on each Draw().
func NewAstronomyDialog(config AstronomyDialogConfig) *AstronomyDialog {
	if config.RandomIntInRange == nil {
		panic("AstronomyDialog requires RandomIntInRange")
	}

	// Initialize keyboard debouncing - wait for current keys to be released
	if config.Keyboard != nil {
		config.Keyboard.PrepareForTransition(KeyboardTransitionDelayMs)
	}

	// Create border with opaque black interior
	border := newDialogBorder(config.Placement)
	bounds := border.GetBounds()
	screenWidth, screenHeight := display.GetManager().GetCurrentSize()

	dialog := &AstronomyDialog{
		buffer:                  ebiten.NewImage(bounds.Dx(), bounds.Dy()), // Static render target
		drawOptions:             &ebiten.DrawImageOptions{},
		border:                  border,
		placement:               config.Placement,
		bounds:                  bounds,
		pixelUnit:               getClassicUIPixelUnit(screenWidth, screenHeight),
		hint:                    staticViewerHint{output: nil, width: 0},
		dismissed:               false,
		waitingForRelease:       true, // Start waiting for trigger keys to be released
		keyboard:                config.Keyboard,
		shadowlordPlanetIndices: config.ShadowlordPlanetIndices,
		spriteSheet:             config.SpriteSheet,
		planetXPositions:        [astronomyPlanetCount]int{}, // Calculated below from the game date
		gameYear:                config.GameYear,
		gameMonth:               config.GameMonth,
		gameDay:                 config.GameDay,
		starPositions:           [astronomyStarCount]image.Point{}, // Generated below
	}

	// Calculate planet X positions based on game date
	for planet := range astronomyPlanetCount {
		dialog.planetXPositions[planet] = calculatePlanetXPosition(planet, config.GameYear, config.GameMonth, config.GameDay)
	}
	dialog.generateStars(config.RandomIntInRange)

	// Position draw options for the buffer
	dialog.drawOptions.GeoM.Translate(float64(bounds.Min.X), float64(bounds.Min.Y))

	// Render everything to buffer once
	dialog.renderToBuffer()

	return dialog
}

func (a *AstronomyDialog) generateStars(randomIntInRange func(int, int) int) {
	starAreaHeight := int(float64(a.bounds.Dy()) * astronomyStarAreaRatio)
	for index := range astronomyStarCount {
		a.starPositions[index] = image.Point{
			X: randomIntInRange(0, a.bounds.Dx()-astronomyRenderedStarSize),
			Y: randomIntInRange(0, starAreaHeight-astronomyRenderedStarSize),
		}
	}
}

// renderToBuffer draws all astronomy elements to the internal buffer.
// This is called once during initialization - not per-frame.
func (a *AstronomyDialog) renderToBuffer() {
	// Fill with solid black (opaque)
	vector.FillRect(a.buffer,
		0, 0,
		float32(a.bounds.Dx()), float32(a.bounds.Dy()),
		stdcolor.Black,
		false) // false = no anti-aliasing for pixel-perfect

	// Draw stars (80 random bright blue points)
	a.renderStars()

	// Draw planets (8 at calculated positions)
	a.renderPlanets()

	// Draw Shadowlord comets (red near planets)
	a.renderComets()

	// Draw telescope at bottom center
	a.renderTelescope()
	applyClassicPixelRoundedMask(a.buffer, a.pixelUnit)
}

// renderStars draws 80 random blue stars to the buffer.
func (a *AstronomyDialog) renderStars() {
	// Bright blue color for stars (EGA light blue)
	starColor := pkgcolor.EGALightBlue

	for _, star := range a.starPositions {
		// Draw 4x4 block for visibility (2x original 2x2)
		for dx := range astronomyRenderedStarSize {
			for dy := range astronomyRenderedStarSize {
				a.setPixelSafe(star.X+dx, star.Y+dy, starColor)
			}
		}
	}
}

// renderPlanets draws 8 planets at calculated positions based on game date.
func (a *AstronomyDialog) renderPlanets() {
	planetY := a.planetYPositions()
	for planet := range astronomyPlanetCount {
		centerX := astronomyPlanetX(a.planetXPositions[planet], a.bounds.Dx())
		// Draw planet as cross/plus shape
		a.drawPlanetCross(centerX, planetY[planet], stdcolor.White)
	}
}

// planetYPositions scales the original track rows (about 20 rows tall) into the
// planet area. Track Y offsets are used directly: screen Y grows downward; ref:ASTRO-0005.
func (a *AstronomyDialog) planetYPositions() [astronomyPlanetCount]int {
	yScale := float64(a.bounds.Dy()) * astronomyPlanetAreaRatio / astronomyTrackRows
	positions := astronomyPlanetYPositions()
	for planet, row := range positions {
		positions[planet] = int(float64(row) * yScale)
	}

	return positions
}

// drawPlanetCross draws a planet as a 9x9 center with cardinal extensions (3x original size).
func (a *AstronomyDialog) drawPlanetCross(centerX, centerY int, fill stdcolor.Color) {
	// 9x9 center (3x scale from original 3x3)
	for dx := -astronomyPlanetRadius; dx <= astronomyPlanetRadius; dx++ {
		for dy := -astronomyPlanetRadius; dy <= astronomyPlanetRadius; dy++ {
			a.setPixelSafe(centerX+dx, centerY+dy, fill)
		}
	}

	// Cardinal extensions at ±6 (3x scale from original ±2)
	a.setPixelSafe(centerX-astronomyPlanetArmOffset, centerY, fill)
	a.setPixelSafe(centerX+astronomyPlanetArmOffset, centerY, fill)
	a.setPixelSafe(centerX, centerY-astronomyPlanetArmOffset, fill)
	a.setPixelSafe(centerX, centerY+astronomyPlanetArmOffset, fill)
}

// renderComets draws red comets near planets where Shadowlords are located.
func (a *AstronomyDialog) renderComets() {
	planetY := a.planetYPositions()
	for _, planetIndex := range a.shadowlordPlanetIndices {
		if planetIndex < 0 || planetIndex >= astronomyPlanetCount {
			continue
		}

		centerX := astronomyPlanetX(a.planetXPositions[planetIndex], a.bounds.Dx())
		// Draw comet tail extending right from planet (offset 15 = 3x original 5)
		a.drawCometTail(centerX+astronomyCometOffset, planetY[planetIndex], pkgcolor.EGALightRed)
	}
}

// astronomyPlanetX maps one-based columns into the plot, leaving room for the
// six-pixel planet radius and the comet's 15-pixel offset plus 21-pixel tail.
func astronomyPlanetX(column, width int) int {
	const leftMargin, rightMargin, lastColumn = 12, 42, 22

	return leftMargin + (column-1)*max(0, width-leftMargin-rightMargin)/(lastColumn-1)
}

// drawCometTail draws a comet tail pattern at the given position (3x original size).
func (a *AstronomyDialog) drawCometTail(headX, headY int, fill stdcolor.Color) {
	// Comet head (3x scale: 9 pixels tall, original was 3)
	for dy := -astronomyCometHeadRadius; dy <= astronomyCometHeadRadius; dy++ {
		a.setPixelSafe(headX, headY+dy, fill)
	}

	// Tail extending right (3x scale: 21 pixels long, original was 7)
	for dx := 1; dx <= astronomyCometTailLength; dx++ {
		height := astronomyCometHeadHeight - (dx / astronomyPixelScale) // Tail tapers from 9 down to ~2
		for dy := -height / astronomyCenterDivisor; dy <= height/astronomyCenterDivisor; dy++ {
			a.setPixelSafe(headX+dx, headY+dy, fill)
		}
	}
}

// renderTelescope draws the telescope sprite at the bottom center (4x scaled).
func (a *AstronomyDialog) renderTelescope() {
	if a.spriteSheet == nil {
		return
	}

	// Get the telescope sprite (18x18 with border)
	telescopeSprite := a.spriteSheet.GetSprite(indexes.Telescope)

	// Scaled size: 72 pixels
	scaledSize := int(astronomyTelescopeSize * astronomyTelescopeScale)

	// Calculate center position at bottom of dialog
	bottomAreaY := a.bounds.Dy() - scaledSize - astronomyTelescopePadding
	centerX := (a.bounds.Dx() - scaledSize) / astronomyCenterDivisor

	// Draw telescope sprite to buffer (4x scaled)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(astronomyTelescopeScale, astronomyTelescopeScale)
	opts.GeoM.Translate(float64(centerX), float64(bottomAreaY))
	a.buffer.DrawImage(telescopeSprite, opts)
}

// setPixelSafe sets a pixel if within bounds.
func (a *AstronomyDialog) setPixelSafe(x, y int, c stdcolor.Color) {
	if x >= 0 && x < a.bounds.Dx() && y >= 0 && y < a.bounds.Dy() {
		a.buffer.Set(x, y, c)
	}
}

// Draw implements Widget interface.
// Simply blits the pre-rendered buffer - no per-frame rendering.
func (a *AstronomyDialog) Draw(screen *ebiten.Image) {
	// Draw the pre-rendered content
	screen.DrawImage(a.buffer, a.drawOptions)

	DrawClassicPixelRoundedFrame(screen, a.bounds, a.pixelUnit, DialogRuleColor)
	a.hint.draw(screen, a.bounds, ControlHint{"Any key", hintActionClose})
}

// Update implements Widget interface.
// Handles keyboard input for dismissal with proper debouncing.
func (a *AstronomyDialog) Update() {
	if a.keyboard == nil {
		return
	}

	// Phase 1: Wait for trigger keys to be released
	if a.waitingForRelease {
		if a.keyboard.IsWaitingForRelease() {
			// Still waiting for initial keys to be released - poll keyboard state
			a.keyboard.TryAcceptKey(0) // Trigger state update check

			return
		}
		// Keys released - transition to accepting new input
		a.waitingForRelease = false
	}

	// Phase 2: Accept any new keypress to dismiss
	if a.keyboard.IsAnyKeyPressed() {
		a.keyboard.ResetReleaseDebouncing()
		a.dismissed = true
	}
}

// IsDismissed returns true if the dialog has been dismissed by user input.
func (a *AstronomyDialog) IsDismissed() bool {
	return a.dismissed
}
