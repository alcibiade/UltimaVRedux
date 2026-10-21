// Package mainscreen provides UI components for the main game screen display.
//
// This package contains widgets specifically designed for the main game interface,
// including character stats, provisions, and status displays. Components integrate
// with the game state and redraw current values every frame.
//
// Key components:
//   - CharacterSummary: Party member stats and portraits
//   - ProvisionSummary: Inventory and resource display
//   - CharacterStatus: Character state indicators
package mainscreen

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// CharacterSummary renders party portraits, resources and detached combat status text.
type CharacterSummary struct {
	spriteSheet *sprites.SpriteSheet
	ultimaFont  *text.UltimaFont
	output      *text.Output
	bounds      image.Rectangle

	// Reusable draw options to reduce per-frame allocations (safety-first optimization)
	portraitDop *ebiten.DrawImageOptions

	textRows   [party_state.MaxCharactersInParty]characterTextRow
	portraits  [party_state.MaxCharactersInParty]characterPortrait
	combatText [party_state.MaxCharactersInParty]CharacterCombatText
}

const (
	// Status effect outline intensity (moderate visibility without overwhelming portrait)
	statusOutlineIntensity = 0.7
	// Character summary positioning constants
	// Active/inactive player text dimming
	inactivePlayerIntensity = 0.6
	characterRowPadding     = 2
	characterTextGap        = 4
	rightTextWidthPercent   = 35
	rightTextInsetPercent   = 4
	percentDivisor          = 100
	characterLayoutCenter   = 2
	characterTextLines      = 2
	characterOutputMaxLines = 10
	logicalAspectWidth      = 16
	logicalAspectHeight     = 9
	defaultSummaryStartX    = 0.77
	defaultSummaryStartY    = 0.025
	defaultSummaryEndX      = 0.99
	defaultSummaryEndY      = 0.48
)

// Status effect color mapping for visual feedback
// Colors match original Ultima V visual themes for status conditions
//
//nolint:gochecknoglobals // Read-only semantic palette shared by portrait outlines and roster text.
var statusColors = map[party_state.CharacterStatus]color.Color{
	party_state.Poisoned: ucolor.From(ucolor.EGAGreen).RGBA(),   // Green for poison
	party_state.Sleep:    ucolor.From(ucolor.EGAMagenta).RGBA(), // Magenta for sleep
	party_state.Charmed:  ucolor.From(ucolor.EGABlue).RGBA(),    // Blue for charm
	party_state.Dead:     ucolor.From(ucolor.EGARed).RGBA(),     // Red for death
	// party_state.Good has no entry (nil) - uses default coloring
}

// NewCharacterSummaryForWidth creates a summary scaled to the logical render width.
func NewCharacterSummaryForWidth(spriteSheet *sprites.SpriteSheet, screenWidth int) *CharacterSummary {
	characterSummary := new(CharacterSummary)

	characterSummary.spriteSheet = spriteSheet

	// Use centralized typography system for consistent font scaling
	characterSummary.ultimaFont, characterSummary.output = newMainScreenOutput(
		screenWidth,
		widgets.FontSizeXLarge,
		characterOutputMaxLines,
	)
	logicalHeight := screenWidth * logicalAspectHeight / logicalAspectWidth
	characterSummary.bounds = image.Rect(
		int(float64(screenWidth)*defaultSummaryStartX),
		int(float64(logicalHeight)*defaultSummaryStartY),
		int(float64(screenWidth)*defaultSummaryEndX),
		int(float64(logicalHeight)*defaultSummaryEndY),
	)

	// Initialize reusable draw options for safety-first performance optimization
	characterSummary.portraitDop = &ebiten.DrawImageOptions{}

	return characterSummary
}

// SetBounds assigns the content area reserved by the main-screen layout.
func (c *CharacterSummary) SetBounds(bounds image.Rectangle) {
	c.bounds = bounds
}

// getStatusColor returns the color to use for a character's status effect.
// Returns nil for Good status (uses default coloring), or the appropriate
// status color for visual feedback.
func (c *CharacterSummary) getStatusColor(status party_state.CharacterStatus) color.Color {
	return statusColors[status] // Returns nil for Good status (not in map)
}

// getEffectiveTextColor computes the text color based on character status, active player state,
// and solo move mode. Selection uses brightness; hues remain reserved for statuses.
// Returns the appropriate color with intensity adjustments for non-active players.
func (c *CharacterSummary) getEffectiveTextColor(
	status party_state.CharacterStatus,
	isActivePlayer, hasActivePlayer bool,
	isSoloMode, isSoloCharacter bool,
) color.Color {
	return selectedCharacterTextColor(
		c.getStatusColor(status), isActivePlayer, hasActivePlayer, isSoloMode, isSoloCharacter,
	)
}

func selectedCharacterTextColor(
	statusColor color.Color,
	isActivePlayer, hasActivePlayer, isSoloMode, isSoloCharacter bool,
) color.Color {
	isSelected := isActivePlayer
	hasSelection := hasActivePlayer
	if isSoloMode {
		isSelected = isSoloCharacter
		hasSelection = true
	}

	// Status effect colors (poison, asleep, etc.) take priority
	if statusColor != nil {
		if hasSelection && !isSelected {
			return ucolor.From(statusColor).Intensity(inactivePlayerIntensity).RGBA()
		}

		return statusColor
	}

	if hasSelection && !isSelected {
		return ucolor.From(ucolor.White).Intensity(inactivePlayerIntensity).RGBA()
	}

	return ucolor.White
}

// shouldApplyStatusOutline determines if a character status should have
// a portrait outline effect for visual emphasis.
func (c *CharacterSummary) shouldApplyStatusOutline(status party_state.CharacterStatus) bool {
	return status != party_state.Good
}

// Draw renders the character summary display to the screen.
//
// Displays party member portraits, names, HP/MP values, and status information
// for all active party members. Automatically handles input validation and
// error recovery for missing character data. Highlights the active player if specified.
//
// Parameters:
//   - partyState: Current party state containing character data (required)
//   - screen: Target screen image for rendering (required)
//   - activePlayerIndex: Index of active player (-1 for none/all, 0-5 for specific character)
//   - isSoloMode: Whether solo move mode is active in combat (subdued text for non-solo characters)
//   - soloCharacterIndex: Index of solo character in solo mode (only valid if isSoloMode is true)
func (c *CharacterSummary) Draw(partyState *party_state.PartyState, screen *ebiten.Image,
	activePlayerIndex int, isSoloMode bool, soloCharacterIndex int,
) {
	// Error handling: validate inputs
	if partyState == nil {
		logging.Errorf(logging.CategoryUI, "CharacterSummary.Draw called with nil partyState, skipping draw")

		return
	}
	if screen == nil {
		logging.Errorf(logging.CategoryUI, "CharacterSummary.Draw called with nil screen, skipping draw")

		return
	}

	if c.bounds.Empty() {
		return
	}
	rowHeight := c.bounds.Dy() / party_state.MaxCharactersInParty
	rightTextBounds := characterRightTextBounds(c.bounds)
	rightTextLeft := rightTextBounds.Min.X
	rightTextWidth := rightTextBounds.Dx()

	for index := range party_state.MaxCharactersInParty {
		// Error handling: validate character data
		if index >= partyState.GetCharacterCount() {
			c.portraits[index].dispose()
			logging.Warnf(logging.CategoryUI, "CharacterSummary character index %d out of bounds, skipping", index)

			continue
		}

		character := partyState.GetCharacter(index)

		if character.PartyMap != party_state.PartyMapActive {
			c.portraits[index].dispose()

			continue
		}

		rowTop := c.bounds.Min.Y + index*rowHeight
		portraitSize := max(rowHeight-characterRowPadding*characterLayoutCenter, 1)
		c.drawCharacterPortrait(index, character, screen, rowTop, portraitSize)

		// Determine if this character is the active player
		// activePlayerIndex -1 means no active player, so no character should be highlighted
		isActivePlayer := activePlayerIndex >= 0 && activePlayerIndex == index
		hasActivePlayer := activePlayerIndex >= 0

		// Determine solo mode coloring
		isSoloCharacter := isSoloMode && soloCharacterIndex == index

		// Get text color based on status effects, active player status, and solo mode
		textColor := c.characterTextColor(
			index, character, isActivePlayer, hasActivePlayer, isSoloMode, isSoloCharacter,
		)

		textTop := rowTop +
			(rowHeight-int(c.output.GetTotalHeightForLines(characterTextLines)))/characterLayoutCenter
		leftTextX := c.bounds.Min.X + portraitSize + characterTextGap
		leftTextWidth := max(rightTextLeft-characterTextGap-leftTextX, 1)
		row := c.prepareCharacterText(index, character, textColor)
		row.left.DrawPanelContent(screen, image.Pt(leftTextX, textTop), leftTextWidth, etext.AlignStart)
		row.right.DrawPanelContent(screen, image.Pt(rightTextLeft, textTop), rightTextWidth, etext.AlignEnd)
	}
}

func (c *CharacterSummary) drawCharacterPortrait(
	index int, character *party_state.PlayerCharacter, screen *ebiten.Image, rowTop, portraitSize int,
) {
	source := c.spriteSheet.GetSprite(character.GetKeySpriteIndex())
	portrait := c.prepareCharacterPortrait(index, source, character.GetStatus())
	c.portraitDop.GeoM.Reset()
	c.portraitDop.GeoM.Scale(
		float64(portraitSize)/float64(portrait.Bounds().Dx()),
		float64(portraitSize)/float64(portrait.Bounds().Dy()),
	)
	c.portraitDop.GeoM.Translate(float64(c.bounds.Min.X), float64(rowTop+characterRowPadding))
	screen.DrawImage(portrait, c.portraitDop)
}

func characterSummaryStatusName(character *party_state.PlayerCharacter) string {
	return characterStatusName(character.GetStatus())
}

func characterStatusName(status party_state.CharacterStatus) string {
	if status == party_state.Poisoned {
		return "Pois."
	}

	if name, ok := status.FriendlyName(); ok {
		return name
	}

	logging.Warnf(logging.CategoryUI, "Invalid character status %d", status)

	return "Unknown"
}

func characterRightTextBounds(bounds image.Rectangle) image.Rectangle {
	rightTextWidth := bounds.Dx() * rightTextWidthPercent / percentDivisor
	rightInset := max(bounds.Dx()*rightTextInsetPercent/percentDivisor, 1)

	return image.Rect(bounds.Max.X-rightInset-rightTextWidth, bounds.Min.Y, bounds.Max.X-rightInset, bounds.Max.Y)
}
