package mainscreen

import (
	"fmt"
	"image/color"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/text"
)

type characterTextKey struct {
	name          [party_state.NMaxPlayerNameSize]byte
	hp, maxHP     uint16
	mp            byte
	status        party_state.CharacterStatus
	rgba          [4]uint32
	font          *text.UltimaFont
	size, spacing float64
}

type characterTextRow struct {
	key         characterTextKey
	left, right *text.Output
	builds      uint64
}

func (c *CharacterSummary) prepareCharacterText(
	index int, character *party_state.PlayerCharacter, textColor color.Color,
) *characterTextRow {
	var key characterTextKey
	key.name, key.hp, key.maxHP = character.Name, character.GetCurrentHp(), character.GetMaxHp()
	key.mp, key.status = character.GetCurrentMp(), character.GetStatus()
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
	row.right.AppendLine().Text(characterSummaryStatusName(character), textColor).Done()
	row.right.AppendLine().Text(fmt.Sprintf("%dMP", key.mp), textColor).Done()
	row.key = key
	row.builds++

	return row
}
