package widgets

import (
	"fmt"
	"image"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
)

const characterHealthConditionColumn = 2

func (s *characterRosterDialogShell) healthColumns(
	row image.Rectangle, options []CharacterSelectionOption,
) [4]image.Rectangle {
	labels := [3]string{"HP", "Missing", "Condition"}
	widths := [3]int{}
	for index, label := range labels {
		width, _ := s.bodyFont.MeasureString(label)
		widths[index] = int(math.Ceil(width))
	}
	for _, option := range options {
		for index, value := range characterHealthValues(option) {
			width, _ := s.bodyFont.MeasureString(value)
			widths[index] = max(widths[index], int(math.Ceil(width)))
		}
	}
	columns := [4]image.Rectangle{}
	right := row.Max.X
	for index := len(widths) - 1; index >= 0; index-- {
		columns[index+1] = image.Rect(right-widths[index], row.Min.Y, right, row.Max.Y)
		right -= widths[index] + s.layout.padding
	}
	columns[0] = image.Rect(row.Min.X, row.Min.Y, max(row.Min.X, right), row.Max.Y)

	return columns
}

func characterHealthValues(option CharacterSelectionOption) [3]string {
	if option.Health == nil {
		panic("healing roster requires health for every candidate")
	}
	health := option.Health
	missing := strconv.Itoa(int(health.Missing))
	if health.Dead {
		missing = "-"
	}

	return [3]string{fmt.Sprintf("%d/%d", health.Current, health.Maximum), missing, option.Status}
}

func (s *characterRosterDialogShell) drawHealthRoster(screen *ebiten.Image, view characterRosterDialogView) {
	header := s.layout.rosterBodyRect
	header.Max.Y = header.Min.Y
	header.Min.Y -= int(math.Ceil(s.headerOutput.GetTotalHeightForLines(1)))
	for index, rect := range s.healthColumns(header, view.options) {
		s.headerOutput.Reset()
		s.headerOutput.SetMaxCharsPerLine(characterRosterDialogMaximumChars)
		s.headerOutput.AppendLine().Text([4]string{"Name", "HP", "Missing", "Condition"}[index], DialogLabelColor).Done()
		s.headerOutput.DrawPanelContent(screen, rect.Min, rect.Dx(), etext.AlignStart)
	}
	for index, option := range view.options {
		row := s.layout.rowRects[index]
		columns := s.healthColumns(row, view.options)
		textY := row.Min.Y + max(0, (row.Dy()-int(math.Ceil(s.rowOutput.GetFontHeight())))/characterRosterDialogCenterDivisor)
		selected := index == view.selected
		s.appendRosterRow(option, selected, columns[0].Dx())
		s.rowOutput.DrawPanelContent(screen, image.Pt(columns[0].Min.X, textY), columns[0].Dx(), etext.AlignStart)
		for field, value := range characterHealthValues(option) {
			colour := characterRosterRowColor(option.State, selected)
			if field == characterHealthConditionColumn {
				colour = option.statusColor(colour)
			}
			s.rowOutput.Reset()
			s.rowOutput.AppendLine().Text(value, colour).Done()
			rect := columns[field+1]
			s.rowOutput.DrawPanelContent(screen, image.Pt(rect.Min.X, textY), rect.Dx(), etext.AlignStart)
		}
	}
}
