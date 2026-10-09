package widgets

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	gametext "github.com/bradhannah/UltimaVRedux/internal/text"
)

const merchantModalInsetDivisor = 48

type merchantModalFrameConfig struct {
	placement             sprites.PercentBasedPlacement
	headerRatio           float64
	footerRatio           float64
	minimumInset          int
	lineSpacingMultiplier float64
	footerMinimumLines    int
}

type merchantModalFrame struct {
	config merchantModalFrameConfig
	border *Border

	headerBounds image.Rectangle
	bodyBounds   image.Rectangle
	footerBounds image.Rectangle
	headerText   image.Rectangle
	bodyText     image.Rectangle
	footerText   image.Rectangle
	inset        int

	lastScreenWidth  int
	lastScreenHeight int
}

func newMerchantModalFrame(config merchantModalFrameConfig) *merchantModalFrame {
	if config.headerRatio <= 0 || config.footerRatio <= 0 || config.minimumInset < 0 ||
		config.lineSpacingMultiplier <= 0 || config.footerMinimumLines <= 0 {
		panic("merchantModalFrame: invalid layout configuration")
	}

	return &merchantModalFrame{config: config} //nolint:exhaustruct // Layout initializes on the first ensureLayout.
}

func (f *merchantModalFrame) ensureLayout() bool {
	width, height := display.GetManager().GetCurrentSize()
	if width <= 0 || height <= 0 {
		return false
	}
	if f.border != nil && width == f.lastScreenWidth && height == f.lastScreenHeight {
		return false
	}

	f.lastScreenWidth, f.lastScreenHeight = width, height
	f.border = NewOpaqueBorder(f.config.placement, BorderWidthStandard, MerchantDialogBackgroundColor)
	f.calculateLayout()

	return true
}

func (f *merchantModalFrame) calculateLayout() {
	interior := f.border.GetInteriorBounds()
	f.inset = max(f.config.minimumInset, min(interior.Dx(), interior.Dy())/merchantModalInsetDivisor)
	content := insetRectangle(interior, f.inset)

	font := GetScaledFont(FontSizeMainScreen)
	metrics := font.TextFace.Metrics()
	fontHeight := metrics.HAscent + metrics.HDescent
	lineSpacing := gametext.GetScaledNumberToResolution(
		display.GetManager(),
		float64(FontSizeMainScreen)*f.config.lineSpacingMultiplier,
	)
	footerTextHeight := fontHeight + float64(f.config.footerMinimumLines-1)*lineSpacing
	headerHeight := max(
		int(math.Ceil(fontHeight))+f.inset*2,
		int(float64(content.Dy())*f.config.headerRatio),
	)
	footerHeight := max(
		int(math.Ceil(footerTextHeight))+f.inset*2,
		int(float64(content.Dy())*f.config.footerRatio),
	)

	f.headerBounds = image.Rect(
		content.Min.X,
		content.Min.Y,
		content.Max.X,
		min(content.Max.Y, content.Min.Y+headerHeight),
	)
	f.footerBounds = image.Rect(
		content.Min.X,
		max(f.headerBounds.Max.Y, content.Max.Y-footerHeight),
		content.Max.X,
		content.Max.Y,
	)
	f.bodyBounds = image.Rect(content.Min.X, f.headerBounds.Max.Y, content.Max.X, f.footerBounds.Min.Y)
	f.headerText = insetRectangle(f.headerBounds, f.inset)
	f.bodyText = insetRectangle(f.bodyBounds, f.inset)
	f.footerText = insetRectangle(f.footerBounds, f.inset)
}

func (f *merchantModalFrame) draw(screen *ebiten.Image) {
	if screen == nil {
		return
	}
	for _, bounds := range [...]image.Rectangle{f.headerBounds, f.bodyBounds, f.footerBounds} {
		FillClassicPixelRoundedRect(screen, bounds, f.pixelUnit(), MerchantDialogBackgroundColor)
	}
	DrawClassicPixelRoundedFrame(screen, f.headerBounds, f.pixelUnit(), DialogRuleColor)
	DrawClassicPixelRoundedFrame(screen, f.bodyBounds, f.pixelUnit(), DialogRuleColor)
	DrawClassicPixelRoundedFrame(screen, f.footerBounds, f.pixelUnit(), DialogRuleColor)
}

func (f *merchantModalFrame) bounds() image.Rectangle {
	return f.border.GetBounds()
}

func (f *merchantModalFrame) pixelUnit() int {
	return getClassicUIPixelUnit(f.lastScreenWidth, f.lastScreenHeight)
}
