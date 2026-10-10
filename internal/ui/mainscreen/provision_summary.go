package mainscreen

import (
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

type ProvisionSummary struct {
	ultimaFont    *text.UltimaFont
	output        *text.Output
	compactFont   *text.UltimaFont
	compactOutput *text.Output
	minimumFont   *text.UltimaFont
	minimumOutput *text.Output
	bounds        image.Rectangle
	rowHeight     int
	labelWidth    int
	statusWidth   int
	calendarWidth int
	labelGap      int
	gap           int
	frameInset    int

	// Refresh flag for SystemCallbacks integration
	needsInventoryRefresh bool
}

const (
	provisionRowCount               = 4
	provisionColumnCount            = 2
	provisionCenterDivisor          = 2
	provisionOutputMaxLines         = 1
	provisionLabelGapAtReference    = 2
	provisionStatusCalendarGaps     = 3
	provisionFrameGutterAtReference = 2
	provisionFrameGutterCount       = 2
	maximumProvisionDate            = "13-28-65535"
)

type provisionResource struct {
	label      string
	value      string
	status     string
	labelColor color.Color
	valueColor color.Color
}

// Each row uses the same label and right-aligned quantity slots. Status slots
// are used only on the bottom row; Qr's status borrows the unused calendar space.
// Showing LOW or OUT cannot move a quantity or the next column.
type provisionCell struct {
	label  image.Rectangle
	value  image.Rectangle
	status image.Rectangle
}

// NewProvisionSummary creates a summary scaled to the current display width.
func NewProvisionSummary() *ProvisionSummary {
	width, _ := display.GetManager().GetCurrentSize()

	return NewProvisionSummaryForWidth(width)
}

// NewProvisionSummaryForWidth creates a summary scaled to the logical render width.
func NewProvisionSummaryForWidth(screenWidth int) *ProvisionSummary {
	var summary ProvisionSummary
	summary.rowHeight = ProvisionAmmoRowHeight(screenWidth)
	summary.ultimaFont, summary.output = newMainScreenOutput(screenWidth, widgets.FontSizeDefault, provisionOutputMaxLines)
	summary.compactFont, summary.compactOutput = newMainScreenOutput(
		screenWidth, widgets.FontSizeSmall, provisionOutputMaxLines)
	summary.minimumFont, summary.minimumOutput = newMainScreenOutput(
		screenWidth, widgets.FontSizeXSmall, provisionOutputMaxLines)
	summary.labelWidth = max(measuredProvisionWidth(summary.ultimaFont, "Ar"),
		measuredProvisionWidth(summary.ultimaFont, "Qr"), measuredProvisionWidth(summary.ultimaFont, "Ka"))
	summary.statusWidth = max(measuredProvisionWidth(summary.ultimaFont, "LOW"),
		measuredProvisionWidth(summary.ultimaFont, "OUT"))
	summary.calendarWidth = measuredProvisionWidth(summary.minimumFont, maximumProvisionDate)
	scale := float64(screenWidth) / mainScreenReferenceWidth
	summary.labelGap = max(1, int(math.Round(provisionLabelGapAtReference*scale)))
	spaceWidth, _ := summary.ultimaFont.MeasureString(" ")
	summary.gap = max(1, int(math.Ceil(spaceWidth/provisionCenterDivisor)))
	// Retain the established raster-frame clearance independently of text gaps.
	frameGutter := max(1, int(math.Round(provisionFrameGutterAtReference*scale)))
	summary.frameInset = measuredProvisionWidth(summary.ultimaFont, "0") + provisionFrameGutterCount*frameGutter

	return &summary
}

// ProvisionAmmoRowHeight reserves one normal-font summary line, including the
// established scaled baseline spacing. No compact typography is used for ammo.
func ProvisionAmmoRowHeight(screenWidth int) int {
	return int(math.Ceil(float64(widgets.FontSizeDefault) * widgets.MainScreenLineSpacingMultiplier *
		float64(screenWidth) / mainScreenReferenceWidth))
}

// ProvisionSummaryHeight reserves the same four normal-font rows at every resolution.
func ProvisionSummaryHeight(screenWidth int) int {
	return provisionRowCount * ProvisionAmmoRowHeight(screenWidth)
}

// SetBounds assigns the four-line content area reserved by the main-screen layout.
func (p *ProvisionSummary) SetBounds(bounds image.Rectangle) {
	p.bounds = bounds
}

// MarkInventoryChanged flags that the inventory display needs refresh.
func (p *ProvisionSummary) MarkInventoryChanged() {
	p.needsInventoryRefresh = true
}

// UpdateInventoryDisplay forces immediate refresh of inventory display.
func (p *ProvisionSummary) UpdateInventoryDisplay() {
	p.needsInventoryRefresh = true
}

// IsInventoryRefreshNeeded returns whether inventory display needs refresh.
func (p *ProvisionSummary) IsInventoryRefreshNeeded() bool {
	return p.needsInventoryRefresh
}

func (p *ProvisionSummary) Draw(gameState *game_state.GameState, screen *ebiten.Image) {
	if gameState == nil {
		logging.Warn(logging.CategoryUI, "ProvisionSummary.Draw called with nil gameState, skipping draw")

		return
	}
	if screen == nil {
		logging.Warn(logging.CategoryUI, "ProvisionSummary.Draw called with nil screen, skipping draw")

		return
	}
	p.needsInventoryRefresh = false
	if p.bounds.Empty() {
		return
	}

	resources := provisionResources(gameState)
	calendar := [provisionRowCount]string{
		gameState.DateTime.GetDateAsString(), gameState.DateTime.GetTimeAsString(), "", "",
	}
	for rowIndex, row := range p.rows() {
		cells, calendarCell := p.rowCells(row)
		for column, resource := range resources[rowIndex] {
			p.drawResource(screen, cells[column], resource)
		}
		if value := calendar[rowIndex]; value != "" {
			_, output := p.valueTextLayout(value, calendarCell.Dx())
			drawProvisionText(screen, output, calendarCell, value, ucolor.UITextPrimary.RGBA(), etext.AlignEnd)
		}
	}
}

func provisionResources(state *game_state.GameState) [provisionRowCount][provisionColumnCount]provisionResource {
	white := ucolor.UITextPrimary.RGBA()
	resources := [provisionRowCount][provisionColumnCount]provisionResource{
		{
			{label: "F", value: formatProvisionQuantity(state.PartyState.Inventory.Provisions.Food.Get()), status: "",
				labelColor: ucolor.EGALightGreen, valueColor: white},
			{label: "T", value: formatProvisionQuantity(state.PartyState.Inventory.Provisions.Torches.Get()), status: "",
				labelColor: ucolor.EGALightRed, valueColor: white},
		},
		{
			{label: "K", value: formatProvisionQuantity(state.PartyState.Inventory.Provisions.Keys.Get()), status: "",
				labelColor: ucolor.EGAYellow, valueColor: white},
			{label: "G", value: formatProvisionQuantity(state.PartyState.Inventory.Provisions.Gems.Get()), status: "",
				labelColor: ucolor.EGALightCyan, valueColor: white},
		},
		{
			{label: "$", value: formatProvisionQuantity(state.PartyState.Inventory.Gold.Get()), status: "",
				labelColor: ucolor.EGAYellow, valueColor: white},
			{label: "Ka", value: strconv.Itoa(state.PartyState.Karma.Value), status: "",
				labelColor: ucolor.EGALightMagenta, valueColor: white},
		},
		{},
	}
	for index, count := range SnapshotAmmunition(&state.PartyState.Inventory) {
		value, valueColor := ammunitionValue(count)
		quantity, status, _ := strings.Cut(value, " ")
		resources[provisionRowCount-1][index] = provisionResource{
			label: [2]string{"Ar", "Qr"}[index], value: quantity, status: status,
			labelColor: ucolor.EGALightCyan, valueColor: valueColor,
		}
	}

	return resources
}

func formatProvisionQuantity(value uint16) string {
	return strconv.FormatUint(uint64(value), 10)
}

func (p *ProvisionSummary) rows() [provisionRowCount]image.Rectangle {
	// The scene's left content edge already clears the rasterized frame. Its
	// asymmetric right edge needs a normal glyph plus gutters of frame clearance;
	// reserve that on the right instead of shifting the entire grid into the left frame.
	bounds := p.bounds
	bounds.Max.X -= p.frameInset
	var rows [provisionRowCount]image.Rectangle
	for index := range rows {
		bottom := bounds.Max.Y - (provisionRowCount-1-index)*p.rowHeight
		rows[index] = image.Rect(bounds.Min.X, bottom-p.rowHeight, bounds.Max.X, bottom)
	}

	return rows
}

func (p *ProvisionSummary) rowCells(bounds image.Rectangle) ([provisionColumnCount]provisionCell, image.Rectangle) {
	calendar := image.Rect(bounds.Max.X-p.calendarWidth, bounds.Min.Y, bounds.Max.X, bounds.Max.Y)
	// Budget two labels and quantities, with Ar's status between the columns.
	// Half-space gutters separate value/status, status/next label and the calendar.
	// Labels already have fixed two-character slots; their smaller internal gaps
	// leave room for complete four-digit values even at the odd logical width.
	// Qr's status needs no extra width: the calendar is absent on the ammo row.
	valueWidth := (calendar.Min.X - bounds.Min.X - provisionColumnCount*p.labelWidth -
		p.statusWidth - provisionColumnCount*p.labelGap - provisionStatusCalendarGaps*p.gap) / provisionColumnCount
	columnPitch := p.labelWidth + p.labelGap + valueWidth + p.statusWidth + provisionColumnCount*p.gap
	var cells [provisionColumnCount]provisionCell
	for index := range cells {
		left := bounds.Min.X + index*columnPitch
		valueLeft := left + p.labelWidth + p.labelGap
		valueRight := valueLeft + valueWidth
		statusLeft := valueRight + p.gap
		if index == len(cells)-1 {
			statusLeft = calendar.Min.X
		}
		cells[index] = provisionCell{
			label:  image.Rect(left, bounds.Min.Y, left+p.labelWidth, bounds.Max.Y),
			value:  image.Rect(valueLeft, bounds.Min.Y, valueRight, bounds.Max.Y),
			status: image.Rect(statusLeft, bounds.Min.Y, statusLeft+p.statusWidth, bounds.Max.Y),
		}
	}

	return cells, calendar
}

func (p *ProvisionSummary) drawResource(screen *ebiten.Image, cell provisionCell, resource provisionResource) {
	drawProvisionText(screen, p.output, cell.label, resource.label, resource.labelColor, etext.AlignStart)
	_, output := p.valueTextLayout(resource.value, cell.value.Dx())
	drawProvisionText(screen, output, cell.value, resource.value, resource.valueColor, etext.AlignEnd)
	if resource.status != "" {
		drawProvisionText(screen, p.output, cell.status, resource.status, resource.valueColor, etext.AlignStart)
	}
}

func drawProvisionText(
	screen *ebiten.Image,
	output *text.Output,
	bounds image.Rectangle,
	value string,
	valueColor color.Color,
	alignment etext.Align,
) {
	output.Reset()
	output.SetMaxCharsPerLine(max(1, output.Font.CharsForWidth(bounds.Dx())))
	output.AppendLine().Text(value, valueColor).WithAlignment(alignment).Done()
	y := bounds.Min.Y + (bounds.Dy()-int(math.Ceil(output.GetFontHeight())))/provisionCenterDivisor
	output.DrawPanelContent(screen, image.Pt(bounds.Min.X, y), bounds.Dx(), alignment)
}

func (p *ProvisionSummary) valueTextLayout(value string, availableWidth int) (*text.UltimaFont, *text.Output) {
	if measuredProvisionWidth(p.ultimaFont, value) <= availableWidth {
		return p.ultimaFont, p.output
	}
	if measuredProvisionWidth(p.compactFont, value) <= availableWidth {
		return p.compactFont, p.compactOutput
	}

	return p.minimumFont, p.minimumOutput
}

func measuredProvisionWidth(font *text.UltimaFont, value string) int {
	width, _ := font.MeasureString(value)

	return int(math.Ceil(width))
}
