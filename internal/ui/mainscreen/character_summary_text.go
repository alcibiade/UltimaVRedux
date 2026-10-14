package mainscreen

import (
	"fmt"
	"image/color"
	"math"

	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/text"
)

type characterTextKey struct {
	name          [party_state.NMaxPlayerNameSize]byte
	hp, maxHP     uint16
	mp            byte
	statusLabel   string
	statusWidth   int
	rgba          [4]uint32
	font          *text.UltimaFont
	size, spacing float64
}

type characterTextRow struct {
	key         characterTextKey
	left, right *text.Output
	builds      uint64
	statusScale float64
}

func (c *CharacterSummary) prepareCharacterText(
	index int, character *party_state.PlayerCharacter, textColor color.Color,
) *characterTextRow {
	var key characterTextKey
	key.name, key.hp, key.maxHP = character.Name, character.GetCurrentHp(), character.GetMaxHp()
	key.mp = character.GetCurrentMp()
	key.statusLabel, _ = c.characterTextStatus(index, character)
	if key.statusLabel == characterPossessedLabel {
		key.statusWidth = characterRightTextBounds(c.bounds).Dx()
	}
	key.rgba[0], key.rgba[1], key.rgba[2], key.rgba[3] = textColor.RGBA()
	key.font, key.size, key.spacing = c.output.Font, c.output.Font.TextFace.Size, c.output.GetLineSpacing()
	row := &c.textRows[index]
	if row.left != nil && row.key == key {
		return row
	}
	if row.left == nil || row.key.font != key.font || row.key.size != key.size || row.key.spacing != key.spacing {
		row.left = text.NewOutput(key.font, key.spacing, characterTextLines, mainScreenMaxCharsPerLine)
		row.right = text.NewOutput(key.font, key.spacing, characterTextLines, mainScreenMaxCharsPerLine)
	}
	row.left.Reset()
	row.left.AppendLine().Text(character.GetNameAsString(), textColor).Done()
	row.left.AppendLine().Text(fmt.Sprintf("%d/%dHP", key.hp, key.maxHP), textColor).Done()
	row.right.Reset()
	row.statusScale = fitCharacterStatusScale(key.font, key.statusLabel, key.statusWidth)
	row.right.AddStyledText([]text.ColorSegment{{
		Text: key.statusLabel, Color: textColor, FontType: text.FontTypeNormal, FontScale: row.statusScale,
	}}, etext.AlignEnd)
	row.right.AppendLine().Text(fmt.Sprintf("%dMP", key.mp), textColor).Done()
	row.key = key
	row.builds++

	return row
}

// Fit only the expanded combat label into the existing status cell. Segment
// scaling leaves the shared font, MP line, line spacing and portrait layout
// unchanged. Remeasure the derived face because glyph advances are rounded;
// multiplying the original width by a scale alone can still overflow the cell.
func fitCharacterStatusScale(font *text.UltimaFont, label string, width int) float64 {
	if width <= 0 {
		return text.FontScaleNormal
	}
	face := *font.TextFace
	measured, _ := etext.Measure(label, &face, 0)
	if measured <= float64(width) {
		return text.FontScaleNormal
	}
	size := max(1, math.Floor(face.Size*float64(width)/measured))
	for {
		scale := size / font.TextFace.Size
		face.Size = font.TextFace.Size * scale
		measured, _ = etext.Measure(label, &face, 0)
		if measured <= float64(width) || size <= 1 {
			return scale
		}
		size--
	}
}
