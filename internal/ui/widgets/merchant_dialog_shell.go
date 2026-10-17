package widgets

import (
	"fmt"
	"image"
	stdcolor "image/color"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
)

const (
	merchantShellStartPercentX       = 0.10
	merchantShellEndPercentX         = 0.90
	merchantShellStartPercentY       = 0.10
	merchantShellEndPercentY         = 0.90
	merchantShellHeaderRatio         = 0.12
	merchantShellFooterRatio         = 0.10
	merchantShellMerchantPaneRatio   = 0.33
	merchantShellPortraitRatio       = 0.25
	merchantShellPaddingDivisor      = 64
	merchantShellMinimumPadding      = 4
	merchantShellTextInsetUnits      = 2
	merchantShellCenterDivisor       = 2
	merchantShellLineSpacing         = 1.15
	merchantShellMinimumTextChars    = 8
	merchantShellHeaderReservedLines = 2
	merchantShellFooterMaximumLines  = 2
	merchantShellSpeechHintLines     = 2
)

// MerchantDialogHeaderView is finalized display data for the common merchant header.
type MerchantDialogHeaderView struct {
	ShopIdentity string
	Owner        string
	PartyGold    string
}

// MerchantDialogExchangeView contains the latest finalized merchant exchange.
type MerchantDialogExchangeView struct {
	Text string
}

// MerchantDialogFooterView is finalized keyboard guidance.
type MerchantDialogFooterView struct {
	Guidance string
}

type merchantFooterSegment struct {
	Text  string
	Color stdcolor.Color
}

// MerchantDialogShellConfig configures the presentation-only merchant shell.
type MerchantDialogShellConfig struct {
	Header         MerchantDialogHeaderView
	Exchange       MerchantDialogExchangeView
	Footer         MerchantDialogFooterView
	MerchantSprite *ebiten.Image
}

type merchantDialogShellLayout struct {
	header       image.Rectangle
	body         image.Rectangle
	merchantPane image.Rectangle
	portrait     image.Rectangle
	exchange     image.Rectangle
	workspace    image.Rectangle
	footer       image.Rectangle
}

// MerchantDialogShell draws shared merchant presentation without owning input or transactions.
type MerchantDialogShell struct {
	header         MerchantDialogHeaderView
	exchange       MerchantDialogExchangeView
	footer         MerchantDialogFooterView
	merchantSprite *ebiten.Image

	outerBorder    *Border
	layout         merchantDialogShellLayout
	headerText     image.Rectangle
	portraitSprite image.Rectangle
	portraitText   image.Rectangle
	exchangeText   image.Rectangle
	footerText     image.Rectangle

	headerOutput     *text.Output
	ownerOutput      *text.Output
	exchangeOutput   *text.Output
	footerOutput     *text.Output
	bodyFont         *text.UltimaFont
	exchangeHint     *text.Output
	exchangePage     int
	exchangeCapacity int

	lastScreenWidth  int
	lastScreenHeight int
}

// NewMerchantDialogShell creates the reusable compact merchant shell.
func NewMerchantDialogShell(config MerchantDialogShellConfig) *MerchantDialogShell {
	var emptyLayout merchantDialogShellLayout
	var emptyBounds image.Rectangle
	shell := &MerchantDialogShell{
		header:           config.Header,
		exchange:         config.Exchange,
		footer:           config.Footer,
		merchantSprite:   config.MerchantSprite,
		outerBorder:      nil,
		layout:           emptyLayout,
		headerText:       emptyBounds,
		portraitSprite:   emptyBounds,
		portraitText:     emptyBounds,
		exchangeText:     emptyBounds,
		footerText:       emptyBounds,
		headerOutput:     nil,
		ownerOutput:      nil,
		exchangeOutput:   nil,
		footerOutput:     nil,
		bodyFont:         nil,
		exchangeHint:     nil,
		exchangePage:     0,
		exchangeCapacity: 0,
		lastScreenWidth:  0,
		lastScreenHeight: 0,
	}
	shell.ensureLayout()

	return shell
}

// Refresh replaces common display data without changing family-owned state.
func (s *MerchantDialogShell) Refresh(
	header MerchantDialogHeaderView,
	exchange MerchantDialogExchangeView,
	footer MerchantDialogFooterView,
) {
	s.header = header
	if s.exchange.Text != exchange.Text {
		s.exchangePage = 0
	}
	s.exchange = exchange
	s.footer = footer
	s.populateOutputs()
}

// SetMerchantSprite replaces the presentation sprite drawn beside the merchant name.
func (s *MerchantDialogShell) SetMerchantSprite(sprite *ebiten.Image) {
	s.merchantSprite = sprite
}

// DrawBackground draws the opaque shell backing before family-owned workspace content.
func (s *MerchantDialogShell) DrawBackground(screen *ebiten.Image) {
	if screen == nil {
		return
	}
	s.ensureLayout()
	for _, bounds := range s.connectedFrameBounds() {
		FillClassicPixelRoundedRect(screen, bounds, s.pixelUnit(), MerchantDialogBackgroundColor)
	}
}

// DrawCommonRegions draws shell frames, common text, and the optional merchant sprite.
func (s *MerchantDialogShell) DrawCommonRegions(screen *ebiten.Image) {
	if screen == nil {
		return
	}
	s.ensureLayout()
	for _, bounds := range s.connectedFrameBounds() {
		DrawClassicPixelRoundedFrame(screen, bounds, s.pixelUnit(), DialogRuleColor)
	}
	s.drawHeader(screen)
	s.drawMerchantSprite(screen)
	s.ownerOutput.DrawPanelContent(screen, s.portraitText.Min, s.portraitText.Dx(), etext.AlignCenter)
	s.drawExchange(screen)
	s.footerOutput.DrawPanelContent(
		screen,
		image.Pt(s.footerText.Min.X, verticallyCenteredOutputY(s.footerText, s.footerOutput)),
		s.footerText.Dx(),
		etext.AlignCenter,
	)
}

// GetBounds returns the centered normalized outer bounds.
func (s *MerchantDialogShell) GetBounds() image.Rectangle {
	s.ensureLayout()

	return s.outerBorder.GetBounds()
}

// GetHeaderBounds returns the full-width common header region.
func (s *MerchantDialogShell) GetHeaderBounds() image.Rectangle {
	s.ensureLayout()

	return s.layout.header
}

// GetBodyBounds returns the complete two-column body region.
func (s *MerchantDialogShell) GetBodyBounds() image.Rectangle {
	s.ensureLayout()

	return s.layout.body
}

// GetMerchantPaneBounds returns the common portrait and latest-exchange column.
func (s *MerchantDialogShell) GetMerchantPaneBounds() image.Rectangle {
	s.ensureLayout()

	return s.layout.merchantPane
}

// GetWorkspaceBounds returns the family-owned body workspace.
func (s *MerchantDialogShell) GetWorkspaceBounds() image.Rectangle {
	s.ensureLayout()

	return s.layout.workspace
}

// GetPortraitBounds returns the common merchant portrait and name region.
func (s *MerchantDialogShell) GetPortraitBounds() image.Rectangle {
	s.ensureLayout()

	return s.layout.portrait
}

// GetExchangeBounds returns the bounded latest-exchange region.
func (s *MerchantDialogShell) GetExchangeBounds() image.Rectangle {
	s.ensureLayout()

	return s.layout.exchange
}

// GetFooterBounds returns the full-width keyboard-guidance footer.
func (s *MerchantDialogShell) GetFooterBounds() image.Rectangle {
	s.ensureLayout()

	return s.layout.footer
}

// GetHeaderView returns the shell's current detached header value.
func (s *MerchantDialogShell) GetHeaderView() MerchantDialogHeaderView {
	return s.header
}

// GetExchangeView returns the shell's current latest exchange.
func (s *MerchantDialogShell) GetExchangeView() MerchantDialogExchangeView {
	return s.exchange
}

// GetFooterView returns the shell's current detached footer value.
func (s *MerchantDialogShell) GetFooterView() MerchantDialogFooterView {
	return s.footer
}

// GetFontSize returns the semantic font size used throughout the shell.
func (s *MerchantDialogShell) GetFontSize() StandardFontSize {
	return FontSizeMainScreen
}

func (s *MerchantDialogShell) ensureLayout() {
	screenWidth, screenHeight := display.GetManager().GetCurrentSize()
	if screenWidth <= 0 || screenHeight <= 0 {
		return
	}
	if s.outerBorder != nil && screenWidth == s.lastScreenWidth && screenHeight == s.lastScreenHeight {
		return
	}
	s.lastScreenWidth = screenWidth
	s.lastScreenHeight = screenHeight
	s.outerBorder = NewOpaqueBorder(merchantDialogShellPlacement(), BorderWidthStandard, MerchantDialogBackgroundColor)
	s.rebuildOutputs()
	s.calculateLayout()
	s.populateOutputs()
}

func (s *MerchantDialogShell) rebuildOutputs() {
	_, s.headerOutput = GetScaledFontWithOutput(FontConfig{
		Size:                  FontSizeMainScreen,
		LineSpacing:           0,
		LineSpacingMultiplier: merchantShellLineSpacing,
		MaxLines:              merchantShellHeaderReservedLines,
		MaxChars:              merchantShellMinimumTextChars,
	})
	_, s.ownerOutput = GetScaledFontWithOutput(FontConfig{
		Size:                  FontSizeMainScreen,
		LineSpacing:           0,
		LineSpacingMultiplier: merchantShellLineSpacing,
		MaxLines:              1,
		MaxChars:              merchantShellMinimumTextChars,
	})
	s.bodyFont, s.exchangeOutput = GetScaledFontWithOutput(FontConfig{
		Size:                  FontSizeMainScreen,
		LineSpacing:           0,
		LineSpacingMultiplier: merchantShellLineSpacing,
		MaxLines:              merchantShellMinimumTextChars,
		MaxChars:              merchantShellMinimumTextChars,
	})
	s.exchangeHint = text.NewOutput(s.bodyFont, s.exchangeOutput.GetLineSpacing(),
		merchantShellSpeechHintLines, merchantShellMinimumTextChars)
	_, s.footerOutput = GetScaledFontWithOutput(FontConfig{
		Size:                  FontSizeMainScreen,
		LineSpacing:           0,
		LineSpacingMultiplier: merchantShellLineSpacing,
		MaxLines:              merchantShellFooterMaximumLines,
		MaxChars:              merchantShellMinimumTextChars,
	})
}

func (s *MerchantDialogShell) calculateLayout() {
	interior := s.outerBorder.GetInteriorBounds()
	padding := max(merchantShellMinimumPadding, min(interior.Dx(), interior.Dy())/merchantShellPaddingDivisor)
	content := insetRectangle(interior, padding)
	gap := padding
	headerHeight := max(
		int(math.Ceil(s.headerOutput.GetTotalHeightForLines(merchantShellHeaderReservedLines)))+padding,
		int(float64(content.Dy())*merchantShellHeaderRatio),
	)
	footerHeight := max(
		int(math.Ceil(s.footerOutput.GetFontHeight()))+padding,
		int(float64(content.Dy())*merchantShellFooterRatio),
	)

	s.layout.header = image.Rect(content.Min.X, content.Min.Y, content.Max.X, content.Min.Y+headerHeight)
	s.layout.footer = image.Rect(content.Min.X, content.Max.Y-footerHeight, content.Max.X, content.Max.Y)
	s.layout.body = image.Rect(content.Min.X, s.layout.header.Max.Y+gap, content.Max.X, s.layout.footer.Min.Y-gap)

	usableBodyWidth := max(1, s.layout.body.Dx()-gap)
	merchantPaneWidth := max(1, int(math.Round(float64(usableBodyWidth)*merchantShellMerchantPaneRatio)))
	s.layout.merchantPane = image.Rect(
		s.layout.body.Min.X,
		s.layout.body.Min.Y,
		s.layout.body.Min.X+merchantPaneWidth,
		s.layout.body.Max.Y,
	)
	s.layout.workspace = image.Rect(
		s.layout.merchantPane.Max.X+gap,
		s.layout.body.Min.Y,
		s.layout.body.Max.X,
		s.layout.body.Max.Y,
	)
	portraitHeight := max(1, int(math.Round(float64(s.layout.merchantPane.Dy()-gap)*merchantShellPortraitRatio)))
	s.layout.portrait = image.Rect(
		s.layout.merchantPane.Min.X,
		s.layout.merchantPane.Min.Y,
		s.layout.merchantPane.Max.X,
		s.layout.merchantPane.Min.Y+portraitHeight,
	)
	s.layout.exchange = image.Rect(
		s.layout.merchantPane.Min.X,
		s.layout.portrait.Max.Y+gap,
		s.layout.merchantPane.Max.X,
		s.layout.merchantPane.Max.Y,
	)
	s.calculateTextLayout()
	s.configureOutputCapacity()
}

func (s *MerchantDialogShell) calculateTextLayout() {
	textInset := max(merchantShellMinimumPadding, s.pixelUnit()*merchantShellTextInsetUnits)
	s.headerText = insetRectangle(s.layout.header, textInset)
	portraitContent := insetRectangle(s.layout.portrait, textInset)
	nameHeight := min(portraitContent.Dy(), int(math.Ceil(s.ownerOutput.GetFontHeight())))
	s.portraitSprite = image.Rect(
		portraitContent.Min.X,
		portraitContent.Min.Y,
		portraitContent.Max.X,
		max(portraitContent.Min.Y, portraitContent.Max.Y-nameHeight-textInset),
	)
	s.portraitText = image.Rect(
		portraitContent.Min.X,
		max(portraitContent.Min.Y, portraitContent.Max.Y-nameHeight),
		portraitContent.Max.X,
		portraitContent.Max.Y,
	)
	s.exchangeText = insetRectangle(s.layout.exchange, textInset)
	s.footerText = insetRectangle(s.layout.footer, textInset)
}

func (s *MerchantDialogShell) configureOutputCapacity() {
	s.headerOutput.SetMaxCharsPerLine(max(
		merchantShellMinimumTextChars,
		s.headerOutput.Font.CharsForWidth(max(1, s.headerText.Dx())),
	))
	s.ownerOutput.SetMaxCharsPerLine(max(
		merchantShellMinimumTextChars,
		s.ownerOutput.Font.CharsForWidth(max(1, s.portraitText.Dx())),
	))
	s.exchangeCapacity = max(1, utf8.RuneCountInString(s.exchange.Text)+1)
	s.exchangeOutput = text.NewOutput(
		s.bodyFont,
		s.exchangeOutput.GetLineSpacing(),
		s.exchangeCapacity,
		max(merchantShellMinimumTextChars, s.bodyFont.CharsForWidth(max(1, s.exchangeText.Dx()))),
	)
	footerCapacity := min(
		merchantShellFooterMaximumLines,
		max(1, s.footerOutput.GetLineCapacity(max(1, s.footerText.Dy()))),
	)
	s.footerOutput = text.NewOutput(
		s.footerOutput.Font,
		s.footerOutput.GetLineSpacing(),
		footerCapacity,
		max(merchantShellMinimumTextChars, s.footerOutput.Font.CharsForWidth(max(1, s.footerText.Dx()))),
	)
}

func (s *MerchantDialogShell) populateOutputs() {
	if s.headerOutput == nil || s.ownerOutput == nil || s.exchangeOutput == nil || s.footerOutput == nil {
		return
	}
	s.populateHeaderOutput()
	s.populateOwnerOutput()
	s.populateExchangeOutput()
	s.populateFooterOutput()
}

func (s *MerchantDialogShell) populateHeaderOutput() {
	s.headerOutput.Reset()
	maxChars := s.headerOutput.Font.CharsForWidth(max(1, s.headerText.Dx()))
	shop, _ := fitWrappedText(s.header.ShopIdentity, s.headerOutput.Font, s.headerOutput.GetLineSpacing(), maxChars, 1)
	s.headerOutput.AppendLine().WithAlignment(etext.AlignCenter).White(shop).Done()
	details, _ := fitWrappedText(
		merchantHeaderDetails(s.header),
		s.headerOutput.Font,
		s.headerOutput.GetLineSpacing(),
		maxChars,
		1,
	)
	label, value, found := strings.Cut(details, ": ")
	line := s.headerOutput.AppendLine().WithAlignment(etext.AlignCenter)
	if found {
		line.Text(label+": ", DialogLabelColor).Text(value, DialogValueColor)
	} else {
		line.Text(details, DialogValueColor)
	}
	line.Done()
}

func (s *MerchantDialogShell) populateOwnerOutput() {
	s.ownerOutput.Reset()
	owner, _ := fitWrappedText(
		s.header.Owner,
		s.ownerOutput.Font,
		s.ownerOutput.GetLineSpacing(),
		s.ownerOutput.Font.CharsForWidth(max(1, s.portraitText.Dx())),
		1,
	)
	s.ownerOutput.AppendLine().WithAlignment(etext.AlignCenter).Yellow(owner).Done()
}

func (s *MerchantDialogShell) populateExchangeOutput() {
	chars := max(1, s.bodyFont.CharsForWidth(s.exchangeText.Dx()))
	if capacity := utf8.RuneCountInString(s.exchange.Text) + 1; capacity > s.exchangeCapacity {
		s.exchangeCapacity = capacity
		s.exchangeOutput = text.NewOutput(s.bodyFont, s.exchangeOutput.GetLineSpacing(), capacity, chars)
	}
	s.exchangeOutput.Reset()
	s.exchangeOutput.SetMaxCharsPerLine(chars)
	latest := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s.exchange.Text, "\r\n", "\n"), "\r", "\n"))
	if latest != "" {
		s.exchangeOutput.AppendLine().White(latest).WordWrap().Done()
	}
	s.exchangePage = min(s.exchangePage, s.exchangePageCount()-1)
	s.populateExchangeHint()
}

func (s *MerchantDialogShell) exchangePageCapacity() int {
	capacity := max(1, s.exchangeOutput.GetLineCapacity(s.exchangeText.Dy()))
	if s.exchangeOutput.GetPanelContentLineCount() <= capacity {
		return capacity
	}
	hintHeight := int(math.Ceil(s.exchangeHint.GetTotalHeightForLines(merchantShellSpeechHintLines)))
	height := s.exchangeText.Dy() - hintHeight - s.pixelUnit()

	return max(1, s.exchangeOutput.GetLineCapacity(height))
}

func (s *MerchantDialogShell) exchangePageCount() int {
	capacity := s.exchangePageCapacity()

	return max(1, (s.exchangeOutput.GetPanelContentLineCount()+capacity-1)/capacity)
}

// PageExchange changes only the visible portion of the latest merchant speech.
// The containing dialog owns keyboard input and transaction state.
func (s *MerchantDialogShell) PageExchange(delta int) {
	s.ensureLayout()
	s.exchangePage = max(0, min(s.exchangePage+delta, s.exchangePageCount()-1))
	s.populateExchangeHint()
}

func (s *MerchantDialogShell) populateExchangeHint() {
	s.exchangeHint.Reset()
	if pages := s.exchangePageCount(); pages > 1 {
		s.exchangeHint.SetMaxCharsPerLine(len("Shift+Left/Right"))
		s.exchangeHint.AppendLine().Text("Shift+Left/Right", DialogControlKeyColor).Done()
		s.exchangeHint.AppendLine().Text("Speech ", DialogControlHintColor).
			Text(fmt.Sprintf("%d/%d", s.exchangePage+1, pages), DialogValueColor).Done()
	}
}

func (s *MerchantDialogShell) drawExchange(screen *ebiten.Image) {
	content := s.exchangeText
	if s.exchangePageCount() > 1 {
		hintHeight := int(math.Ceil(s.exchangeHint.GetTotalHeightForLines(merchantShellSpeechHintLines)))
		hint := image.Rect(content.Min.X, content.Max.Y-hintHeight, content.Max.X, content.Max.Y)
		s.exchangeHint.DrawPanelContent(screen, hint.Min, hint.Dx(), etext.AlignCenter)
		content.Max.Y = hint.Min.Y - s.pixelUnit()
	}
	clipBounds := content.Intersect(screen.Bounds())
	if clipBounds.Empty() {
		return
	}
	clipped, ok := screen.SubImage(clipBounds).(*ebiten.Image)
	if !ok {
		panic("merchant speech clip is not an image")
	}
	capacity := s.exchangePageCapacity()
	s.exchangeOutput.DrawPanelContentRange(clipped, content.Min, content.Dx(), etext.AlignStart,
		s.exchangePage*capacity, capacity)
}

func (s *MerchantDialogShell) populateFooterOutput() {
	s.footerOutput.Reset()
	capacity := min(
		merchantShellFooterMaximumLines,
		s.footerOutput.GetLineCapacity(max(1, s.footerText.Dy())),
	)
	guidance := fitMerchantFooterGuidance(
		s.footer.Guidance,
		s.footerOutput.Font,
		s.footerOutput.GetLineSpacing(),
		s.footerOutput.Font.CharsForWidth(max(1, s.footerText.Dx())),
		capacity,
	)
	for _, line := range parseMerchantFooterGuidance(guidance) {
		builder := s.footerOutput.AppendLine().WithAlignment(etext.AlignCenter)
		if len(line) == 0 {
			builder.Text("", DialogControlHintColor)
		}
		for _, segment := range line {
			builder.Text(segment.Text, segment.Color)
		}
		builder.WordWrap().Done()
	}
}

func fitMerchantFooterGuidance(
	guidance string,
	font *text.UltimaFont,
	lineSpacing float64,
	maxChars int,
	maxLines int,
) string {
	if maxLines <= 0 {
		return ""
	}
	guidance = strings.ReplaceAll(guidance, "\r\n", "\n")
	guidance = strings.ReplaceAll(guidance, "\r", "\n")
	lines := strings.Split(guidance, "\n")
	lines = lines[:min(len(lines), merchantShellFooterMaximumLines)]
	if len(lines) > maxLines {
		lines = []string{strings.Join(lines, " ")}
	}
	lineCapacity := maxLines
	if len(lines) > 1 {
		lineCapacity = 1
	}
	for index, line := range lines {
		if text.MeasureText(line, font, lineSpacing, maxChars).Lines <= lineCapacity {
			continue
		}
		lines[index], _ = fitWrappedText(line, font, lineSpacing, maxChars, lineCapacity)
	}

	return strings.Join(lines, "\n")
}

func parseMerchantFooterGuidance(guidance string) [][]merchantFooterSegment {
	guidance = strings.ReplaceAll(guidance, "\r\n", "\n")
	guidance = strings.ReplaceAll(guidance, "\r", "\n")
	sourceLines := strings.Split(guidance, "\n")
	sourceLines = sourceLines[:min(len(sourceLines), merchantShellFooterMaximumLines)]
	lines := make([][]merchantFooterSegment, 0, len(sourceLines))
	for _, sourceLine := range sourceLines {
		var segments []merchantFooterSegment
		for cursor := 0; cursor < len(sourceLine); {
			openingOffset := strings.IndexByte(sourceLine[cursor:], '[')
			if openingOffset < 0 {
				segments = append(segments, colonControlHintSegments(sourceLine[cursor:])...)

				break
			}
			opening := cursor + openingOffset
			closingOffset := strings.IndexByte(sourceLine[opening+1:], ']')
			if closingOffset < 0 {
				segments = append(segments, merchantFooterSegment{
					Text:  sourceLine[cursor:],
					Color: DialogControlHintColor,
				})

				break
			}
			if opening > cursor {
				segments = append(segments, colonControlHintSegments(sourceLine[cursor:opening])...)
			}
			closing := opening + 1 + closingOffset
			segments = append(segments, merchantFooterSegment{
				Text:  sourceLine[opening : closing+1],
				Color: DialogControlKeyColor,
			})
			cursor = closing + 1
		}
		lines = append(lines, segments)
	}

	return lines
}

func (s *MerchantDialogShell) drawHeader(screen *ebiten.Image) {
	lineCount := s.headerOutput.GetPanelContentLineCount()
	textHeight := int(math.Ceil(s.headerOutput.GetTotalHeightForLines(lineCount)))
	startY := s.headerText.Min.Y + max(0, (s.headerText.Dy()-textHeight)/merchantShellCenterDivisor)
	s.headerOutput.DrawPanelContent(screen, image.Pt(s.headerText.Min.X, startY), s.headerText.Dx(), etext.AlignCenter)
}

func (s *MerchantDialogShell) drawMerchantSprite(screen *ebiten.Image) {
	if s.merchantSprite == nil || s.portraitSprite.Empty() {
		return
	}
	spriteBounds := s.merchantSprite.Bounds()
	if spriteBounds.Empty() {
		return
	}
	scale := max(1, min(s.portraitSprite.Dx()/spriteBounds.Dx(), s.portraitSprite.Dy()/spriteBounds.Dy()))
	drawWidth := spriteBounds.Dx() * scale
	drawHeight := spriteBounds.Dy() * scale
	drawOptions := new(ebiten.DrawImageOptions)
	drawOptions.Filter = ebiten.FilterNearest
	drawOptions.GeoM.Translate(float64(-spriteBounds.Min.X), float64(-spriteBounds.Min.Y))
	drawOptions.GeoM.Scale(float64(scale), float64(scale))
	drawOptions.GeoM.Translate(
		float64(s.portraitSprite.Min.X+(s.portraitSprite.Dx()-drawWidth)/merchantShellCenterDivisor),
		float64(s.portraitSprite.Min.Y+(s.portraitSprite.Dy()-drawHeight)/merchantShellCenterDivisor),
	)
	screen.DrawImage(s.merchantSprite, drawOptions)
}

func (s *MerchantDialogShell) connectedFrameBounds() []image.Rectangle {
	overlap := max(1, s.pixelUnit())
	columnSplit := (s.layout.merchantPane.Max.X + s.layout.workspace.Min.X) / merchantShellCenterDivisor
	rowSplit := (s.layout.portrait.Max.Y + s.layout.exchange.Min.Y) / merchantShellCenterDivisor

	return []image.Rectangle{
		image.Rect(s.layout.header.Min.X, s.layout.header.Min.Y, s.layout.header.Max.X, s.layout.body.Min.Y+overlap),
		image.Rect(s.layout.merchantPane.Min.X, s.layout.body.Min.Y, columnSplit+overlap, rowSplit+overlap),
		image.Rect(s.layout.exchange.Min.X, rowSplit, columnSplit+overlap, s.layout.body.Max.Y),
		image.Rect(columnSplit, s.layout.body.Min.Y, s.layout.workspace.Max.X, s.layout.body.Max.Y),
		image.Rect(s.layout.footer.Min.X, s.layout.body.Max.Y-overlap, s.layout.footer.Max.X, s.layout.footer.Max.Y),
	}
}

func (s *MerchantDialogShell) pixelUnit() int {
	return getClassicUIPixelUnit(s.lastScreenWidth, s.lastScreenHeight)
}

func merchantDialogShellPlacement() sprites.PercentBasedPlacement {
	return sprites.PercentBasedPlacement{
		StartPercentX: merchantShellStartPercentX,
		EndPercentX:   merchantShellEndPercentX,
		StartPercentY: merchantShellStartPercentY,
		EndPercentY:   merchantShellEndPercentY,
	}
}

func merchantHeaderDetails(view MerchantDialogHeaderView) string {
	if gold := strings.TrimSpace(view.PartyGold); gold != "" {
		return "Party gold: " + gold
	}

	return ""
}

func verticallyCenteredOutputY(bounds image.Rectangle, output *text.Output) int {
	lineCount := output.GetPanelContentLineCount()
	textHeight := int(math.Ceil(output.GetTotalHeightForLines(lineCount)))

	return bounds.Min.Y + max(0, (bounds.Dy()-textHeight)/merchantShellCenterDivisor)
}

func insetRectangle(bounds image.Rectangle, inset int) image.Rectangle {
	inset = max(0, inset)
	maxInsetX := max(0, bounds.Dx()/merchantShellCenterDivisor)
	maxInsetY := max(0, bounds.Dy()/merchantShellCenterDivisor)
	insetX := min(inset, maxInsetX)
	insetY := min(inset, maxInsetY)

	return image.Rect(
		bounds.Min.X+insetX,
		bounds.Min.Y+insetY,
		bounds.Max.X-insetX,
		bounds.Max.Y-insetY,
	)
}
