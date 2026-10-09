package mainscreen

import (
	"image"
	"math"
	"strconv"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

const provisionFoodChars = 96

func (p *ProvisionChangePanel) hasFoodLoss() bool {
	return p.rows[0].Name == "Food" && p.rows[0].Lost > 0
}

func (p *ProvisionChangePanel) populateFoodLoss() {
	p.foodHeading.Reset()
	p.foodDetails.Reset()
	if !p.hasFoodLoss() {
		return
	}
	food := p.rows[0]
	p.foodHeading.AppendLine().Text("FOOD", widgets.DialogLabelColor).
		Text(" -"+strconv.FormatUint(food.Lost, 10), widgets.DialogDeltaNegative).Done()
	p.foodDetails.AppendLine().Text("Remaining: ", widgets.DialogLabelColor).
		Text(strconv.FormatUint(uint64(food.Total), 10), widgets.DialogValueColor).Done()
	if food.Gained > 0 {
		p.foodDetails.AppendLine().Text("Gained: ", widgets.DialogLabelColor).
			Text("+"+strconv.FormatUint(food.Gained, 10), widgets.DialogDeltaPositive).Done()
	}
}

func provisionOutputWidth(output *text.Output) int {
	width := 0
	for _, line := range strings.Split(output.GetOutputStr(false), "\n") {
		measured, _ := output.Font.MeasureString(line)
		width = max(width, int(math.Ceil(measured)))
	}

	return width
}

// Feature actual food losses beside the ordinary resource grid. Both share the
// original observer, expiry and scene-owned modal composition rules.
//
//nolint:funlen // Keep measured featured-card and companion-grid geometry together.
func (p *ProvisionChangePanel) layoutFoodLoss() {
	padding := max(1, p.viewport.Dx()/provisionPanelMarginDivisor)
	gap := padding * provisionPanelSides
	foodWidth := max(provisionOutputWidth(p.foodHeading), provisionOutputWidth(p.foodDetails)) + gap
	foodHeight := int(math.Ceil(p.foodHeading.GetTotalHeightForLines(1)+
		p.foodDetails.GetTotalHeightForLines(p.foodDetails.GetPanelContentLineCount()))) + padding*provisionPanelColumns
	count, cellWidth := 0, 0
	for index := 1; index < len(p.rows); index++ {
		if p.rows[index].Name != "" {
			count++
			cellWidth = max(cellWidth, provisionOutputWidth(p.outputs[index]))
		}
	}
	left := p.viewport.Min.X + (p.viewport.Dx()-foodWidth)/provisionPanelSides
	top := p.viewport.Min.Y + padding*provisionPanelTopInset
	if count == 0 {
		p.foodBounds = image.Rect(left, top, left+foodWidth, top+foodHeight)
		p.bounds = p.foodBounds

		return
	}
	available := p.viewport.Dx() - padding*provisionPanelOuterInsets
	stacked := foodWidth+gap+cellWidth+gap > available
	gridAvailable := available - foodWidth - gap
	if stacked {
		gridAvailable = available
	}
	columns := min(count, provisionPanelColumns, max(1, (gridAvailable-gap)/(cellWidth+gap)))
	gridWidth := max(provisionOutputWidth(p.header), columns*cellWidth+(columns-1)*gap) + gap
	rowHeight := int(math.Ceil(p.header.GetTotalHeightForLines(1)))
	rowCount := (count + columns - 1) / columns
	gridHeight := (rowCount+1)*rowHeight + (rowCount+provisionPanelSides)*padding
	width, height := foodWidth+gap+gridWidth, max(foodHeight, gridHeight)
	if stacked {
		width, height = max(foodWidth, gridWidth), foodHeight+gap+gridHeight
	}
	left = p.viewport.Min.X + (p.viewport.Dx()-width)/provisionPanelSides
	p.bounds = image.Rect(left, top, left+width, top+height)
	p.foodBounds = image.Rect(left, top, left+foodWidth, top+foodHeight)
	gridX, gridY := left+foodWidth+gap, top
	if stacked {
		gridX, gridY = left, top+foodHeight+gap
	}
	p.headerBounds = image.Rect(gridX+padding, gridY+padding, gridX+gridWidth-padding, gridY+padding+rowHeight)
	position := 0
	for index := 1; index < len(p.rows); index++ {
		if p.rows[index].Name == "" {
			continue
		}
		x := gridX + padding + (position%columns)*(cellWidth+gap)
		y := p.headerBounds.Max.Y + padding + (position/columns)*(rowHeight+padding)
		p.cells[index] = image.Rect(x, y, x+cellWidth, y+rowHeight)
		position++
	}
}
