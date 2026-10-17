package main

import (
	"fmt"
	"image"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

type gameplaySettingDefinition struct {
	label  string
	detail string
}

const (
	displayPresetStatusRow = 2
	displayPresetsPerLine  = 3
	displayResizingRow     = 4
)

const (
	gameplayRepeatedDivisionIndex = iota
	gameplayHiddenDiscoveriesIndex
	gameplayUnlitDungeonsIndex
	gameplayBatchMerchantTheftIndex
	gameplaySmoothSurfaceMovementIndex
	gameplayConcealedTurnSlotsIndex
	gameplayFlexibleSextantUseIndex
)

func gameplaySettings() [audioSettingsGameplayCount]gameplaySettingDefinition {
	return [...]gameplaySettingDefinition{
		{
			label: "Repeated Monster Division",
			detail: "Original: surviving Slimes and Gargoyles can divide repeatedly. " +
				"Off keeps Redux's one-child split limit; On restores repeated division.",
		},
		{
			label: "Remember Hidden Dungeon Discoveries",
			detail: "Remember discovered hidden doors and pits between visits. " +
				"On: keep discoveries. Off: reset each fresh visit.",
		},
		{
			label: "Show Subdued Unlit Dungeons",
			detail: "Original: an unlit dungeon viewport is black. " +
				"On shows subdued geometry without granting light; Off restores binary darkness.",
		},
		{
			label: "Batch Merchant Theft",
			detail: "On: one Faulinei theft per paid batch. Off: original theft per item, bundle or patient; " +
				"theft can stop a batch early. Charity requires a separate review.",
		},
		{
			label: "Smooth Surface Movement",
			detail: "Scroll while walking, riding, flying or boating. " +
				"Mounts animate 25% faster; automatic sailing follows its cadence. " +
				"Gameplay timing, combat and dungeons are unchanged.",
		},
		{
			label: "Concealed Turn Slots",
			detail: "On reserves blank NEXT slots for concealed enemies. " +
				"Off compacts the four-action forecast to avoid revealing hidden actors. " +
				"Scheduling is unchanged.",
		},
		{
			label: "Flexible Sextant Use",
			detail: "Redux: day or night, including surface settlement interiors. Sextant required. " +
				"No dungeons or Underworld. Off: night world-map use.",
		},
	}
}

// Update processes input and updates dialog state.
func (d *ConfigurationDialog) Update() {
	if !d.isVisible || !d.hasFocus {
		return
	}
	if d.updateDisplayWorkflow() {
		return
	}
	if d.updateAudioDiagnostics() {
		return
	}
	if key, owned := d.navigation.Poll(d.keyboard); owned {
		if key != nil {
			d.navigateConfigurationRows(*key)
		}

		return
	}
	switch d.currentTab {
	case audioSettingsGameplayTab:
		d.updateGameplayTab()
	case audioSettingsAudioTab:
		d.updateAudioTab()
	case audioSettingsDisplayTab:
		d.updateDisplayTab()
	default:
		panic("unsupported Configuration section")
	}
}

func (d *ConfigurationDialog) navigateConfigurationRows(key ebiten.Key) {
	if key != ebiten.KeyUp && key != ebiten.KeyDown {
		return // These compact setting lists have no paging action.
	}
	switch d.currentTab {
	case audioSettingsGameplayTab:
		d.updateGameplayControl(key)
	case audioSettingsAudioTab:
		d.updateAudioControl(key)
	case audioSettingsDisplayTab:
		if key == ebiten.KeyUp {
			d.focusedDisplayIndex = max(0, d.focusedDisplayIndex-1)
		} else if d.stagedDisplay.Mode == config.ModeWindowed {
			d.focusedDisplayIndex = min(configurationDisplayRowCount-1, d.focusedDisplayIndex+1)
		}
	}
}

func (d *ConfigurationDialog) updateGameplayTab() {
	boundKeys := []ebiten.Key{
		ebiten.KeyEscape,
		ebiten.KeyEnter,
		ebiten.KeyNumpadEnter,
		ebiten.KeyTab,
		ebiten.KeySpace,
	}
	boundKey := d.keyboard.GetBoundKeyPressed(&boundKeys)
	if boundKey == nil || !d.keyboard.TryToRegisterKeyPress(*boundKey) {
		return
	}
	d.updateGameplayControl(*boundKey)
}

func (d *ConfigurationDialog) updateGameplayControl(key ebiten.Key) {
	//nolint:exhaustive // The keyboard lookup restricts input to this tab's bindings.
	switch key {
	case ebiten.KeyEscape:
		d.onCancelClicked()
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		d.onOKClicked()
	case ebiten.KeyTab:
		d.cycleConfigurationTab()
	case ebiten.KeyUp:
		if d.focusedGameplayIndex > 0 {
			d.focusedGameplayIndex--
		}
	case ebiten.KeyDown:
		if d.focusedGameplayIndex < audioSettingsGameplayCount-1 {
			d.focusedGameplayIndex++
		}
	case ebiten.KeySpace:
		d.toggleFocusedGameplaySetting()
	default:
		return
	}
}

func (d *ConfigurationDialog) updateAudioTab() {
	boundKeys := []ebiten.Key{
		ebiten.KeyD,
		ebiten.KeyEscape,
		ebiten.KeyEnter,
		ebiten.KeyNumpadEnter,
		ebiten.KeyLeft,
		ebiten.KeyRight,
		ebiten.KeyTab,
	}
	boundKey := d.keyboard.GetBoundKeyPressed(&boundKeys)
	if boundKey == nil || !d.keyboard.TryToRegisterKeyPress(*boundKey) {
		return
	}

	//nolint:exhaustive // The keyboard lookup above restricts input to this tab's bindings.
	switch *boundKey {
	case ebiten.KeyD:
		d.openAudioDiagnostics()
	case ebiten.KeyEscape:
		d.onCancelClicked()
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		d.onOKClicked()
	case ebiten.KeyTab:
		d.cycleConfigurationTab()
	default:
		d.updateAudioControl(*boundKey)
	}
}

func (d *ConfigurationDialog) cycleConfigurationTab() {
	d.navigation.Reset()
	previous := d.currentTab
	direction := 1
	if d.keyboard.IsModifierPressed(input.ModifierShift) {
		direction = -1
	}
	d.currentTab = (d.currentTab + direction + len(d.tabNames)) % len(d.tabNames)
	if previous == audioSettingsAudioTab {
		d.leaveAudioTab()
	}
	if d.currentTab == audioSettingsAudioTab {
		d.enablePreviewAudio()
		d.updateSliderFocus()
	}
}

func (d *ConfigurationDialog) updateAudioControl(key ebiten.Key) {
	//nolint:exhaustive // This helper receives only directional keys from updateAudioTab.
	switch key {
	case ebiten.KeyUp:
		if d.focusedSliderIndex > 0 {
			d.focusedSliderIndex--
			d.updateSliderFocus()
		}
	case ebiten.KeyDown:
		if d.focusedSliderIndex < audioSettingsSliderCount-1 {
			d.focusedSliderIndex++
			d.updateSliderFocus()
		}
	case ebiten.KeyLeft:
		d.adjustFocusedSlider(-1)
	case ebiten.KeyRight:
		d.adjustFocusedSlider(1)
	default:
		return
	}
}

// Draw renders the dialog to the screen.
func (d *ConfigurationDialog) Draw(screen *ebiten.Image) {
	if !d.isVisible || screen == nil {
		return
	}
	d.ensureLayout(screen.Bounds().Dx(), screen.Bounds().Dy())
	if d.displayPhase == configurationDisplayConfirming {
		d.drawDisplayConfirmation(screen)

		return
	}
	widgets.FillClassicPixelRoundedRect(
		screen,
		d.layout.cardRect,
		d.layout.pixelUnit,
		ucolor.DialogBackground.RGBA(),
	)
	d.titleOutput.Reset()
	d.titleOutput.AppendLine().Text("Configuration", widgets.DialogHeaderColor).Done()
	d.titleOutput.DrawPanelContent(
		screen,
		image.Pt(d.layout.titleRect.Min.X, centeredAudioOutputY(d.layout.titleRect, d.titleOutput)),
		d.layout.titleRect.Dx(),
		etext.AlignCenter,
	)
	d.drawTabs(screen)
	switch d.currentTab {
	case audioSettingsGameplayTab:
		d.drawGameplayContent(screen)
	case audioSettingsAudioTab:
		if d.audioReport != nil {
			d.drawAudioDiagnostics(screen)
		} else {
			d.drawAudioContent(screen)
		}
	case audioSettingsDisplayTab:
		d.drawDisplayContent(screen)
	default:
		panic("unsupported Configuration section")
	}
	d.drawFooter(screen)
	widgets.DrawClassicPixelRoundedFrame(
		screen,
		d.layout.cardRect,
		d.layout.pixelUnit,
		widgets.DialogRuleColor,
	)
}

func (d *ConfigurationDialog) drawDisplayContent(screen *ebiten.Image) {
	labels, values := d.displayContentRows()
	for index, rect := range d.layout.displayRows {
		if index == displayPresetStatusRow {
			if d.stagedDisplay.Mode == config.ModeWindowed {
				d.drawWindowPresetAvailability(screen, rect)
			}

			continue
		}
		selected := index == d.focusedDisplayIndex &&
			(index == configurationDisplayModeRow || d.stagedDisplay.Mode == config.ModeWindowed)
		textColor := widgets.DialogValueColor
		labelColor := widgets.DialogLabelColor
		if selected {
			widgets.FillClassicPixelRoundedRect(
				screen, d.layout.displaySelections[index], d.layout.pixelUnit, widgets.DialogSelectedBackground,
			)
			textColor = widgets.DialogSelectedText
			labelColor = widgets.DialogSelectedLabelColor
		}
		labelRect := d.layout.displayLabels[index]
		d.labelOutput.Reset()
		d.labelOutput.AppendLine().Text(labels[index], labelColor).Done()
		d.labelOutput.DrawPanelContent(
			screen,
			image.Pt(labelRect.Min.X, centeredAudioOutputY(labelRect, d.labelOutput)),
			labelRect.Dx(),
			etext.AlignStart,
		)
		valueRect := d.layout.displayValues[index]
		d.valueOutput.Reset()
		d.valueOutput.AppendLine().Text(values[index], textColor).Done()
		d.valueOutput.DrawPanelContent(
			screen,
			image.Pt(valueRect.Min.X, centeredAudioOutputY(valueRect, d.valueOutput)),
			valueRect.Dx(),
			etext.AlignEnd,
		)
	}
	widgets.DrawClassicPixelRoundedFrame(
		screen, d.layout.bodyRect, d.layout.pixelUnit, widgets.DialogRuleColor,
	)
}

func (d *ConfigurationDialog) displayContentRows() ([5]string, [5]string) {
	labels := [...]string{"Display Mode", "Window Size", "", "Aspect", "Resize"}
	values := [...]string{
		"<  " + displayModeLabel(d.stagedDisplay.Mode) + "  >",
		d.displaySizeLabel(),
		"",
		"16:9  LOCKED",
		"Enabled",
	}
	if d.stagedDisplay.Mode != config.ModeWindowed {
		labels[configurationDisplaySizeRow] = "Screen Size"
		values[displayResizingRow] = "Desktop managed"
	}

	return labels, values
}

func (d *ConfigurationDialog) drawWindowPresetAvailability(screen *ebiten.Image, rect image.Rectangle) {
	choices := d.windowSizeChoices()
	unavailable := false
	d.presetOutput.Reset()
	for start := 0; start < len(choices); start += displayPresetsPerLine {
		line := d.presetOutput.AppendLine()
		if start == 0 {
			line.Text("Presets: ", widgets.DialogLabelColor)
		} else {
			line.Text("         ", widgets.DialogLabelColor)
		}
		for index := start; index < min(start+displayPresetsPerLine, len(choices)); index++ {
			if choices[index].custom {
				continue
			}
			color := widgets.DialogValueColor
			suffix := ""
			if !choices[index].available {
				color = ucolor.UITextDisabled.RGBA()
				unavailable, suffix = true, "*"
			}
			if index > start {
				line.Text("  ", color)
			}
			line.Text(
				fmt.Sprintf("%dx%d%s", choices[index].X, choices[index].Y, suffix),
				color,
			)
		}
		line.Done()
	}
	if unavailable {
		d.presetOutput.AppendLine().Text("* Too large for this display", widgets.DialogControlHintColor).Done()
	}
	d.presetOutput.DrawPanelContent(
		screen,
		image.Pt(rect.Min.X, centeredAudioOutputY(rect, d.presetOutput)),
		rect.Dx(),
		etext.AlignCenter,
	)
}

func (d *ConfigurationDialog) drawDisplayConfirmation(screen *ebiten.Image) {
	// This view owns the whole body, not the Gameplay detail column.
	d.detailOutput.SetMaxCharsPerLine(max(1, d.detailOutput.Font.CharsForWidth(d.layout.bodyRect.Dx())))
	widgets.FillClassicPixelRoundedRect(
		screen, d.layout.cardRect, d.layout.pixelUnit, ucolor.DialogBackground.RGBA(),
	)
	d.titleOutput.Reset()
	d.titleOutput.AppendLine().Text("Keep Display Settings?", widgets.DialogHeaderColor).Done()
	d.titleOutput.DrawPanelContent(
		screen,
		image.Pt(d.layout.titleRect.Min.X, centeredAudioOutputY(d.layout.titleRect, d.titleOutput)),
		d.layout.titleRect.Dx(),
		etext.AlignCenter,
	)
	d.detailOutput.Reset()
	d.detailOutput.AppendLine().
		Text("Display Mode: "+displayModeLabel(d.stagedDisplay.Mode), widgets.DialogValueColor).
		Done()
	logicalWidth, logicalHeight := d.displayManager.LogicalSize()
	d.detailOutput.AppendLine().Text(
		fmt.Sprintf("Game Area: %d x %d (16:9)", logicalWidth, logicalHeight),
		widgets.DialogValueColor,
	).Done()
	d.detailOutput.AppendLine().Text(
		fmt.Sprintf(
			"Settings will revert in %d seconds.",
			(d.displayConfirmationTicks+configurationTicksPerSecond-1)/configurationTicksPerSecond,
		),
		widgets.DialogStatusWarningColor,
	).Done()
	selection := "[ Keep ]        Revert"
	if !d.keepDisplaySettings {
		selection = "Keep        [ Revert ]"
	}
	d.detailOutput.AppendLine().Text(selection, widgets.DialogValueColor).Done()
	d.detailOutput.DrawPanelContent(
		screen,
		image.Pt(d.layout.bodyRect.Min.X, centeredAudioOutputY(d.layout.bodyRect, d.detailOutput)),
		d.layout.bodyRect.Dx(),
		etext.AlignCenter,
	)
	d.footerOutput.Reset()
	widgets.AppendControlHints(d.footerOutput.AppendLine(), "Left/Right: Select").Done()
	widgets.AppendControlHints(d.footerOutput.AppendLine(), "Enter: Confirm  Esc: Revert").Done()
	d.footerOutput.DrawPanelContent(
		screen,
		image.Pt(d.layout.footerRect.Min.X, centeredAudioOutputY(d.layout.footerRect, d.footerOutput)),
		d.layout.footerRect.Dx(),
		etext.AlignCenter,
	)
	widgets.DrawClassicPixelRoundedFrame(
		screen, d.layout.cardRect, d.layout.pixelUnit, widgets.DialogRuleColor,
	)
}

func (d *ConfigurationDialog) drawTabs(screen *ebiten.Image) {
	for index, tabName := range d.tabNames {
		textColor := widgets.DialogControlHintColor
		if index == d.currentTab {
			widgets.FillClassicPixelRoundedRect(
				screen,
				d.layout.tabRects[index],
				d.layout.pixelUnit,
				widgets.DialogSelectedBackground,
			)
			textColor = widgets.DialogSelectedText
		}
		d.tabOutput.Reset()
		d.tabOutput.AppendLine().Text(tabName, textColor).Done()
		d.tabOutput.DrawPanelContent(
			screen,
			image.Pt(
				d.layout.tabRects[index].Min.X,
				centeredAudioOutputY(d.layout.tabRects[index], d.tabOutput),
			),
			d.layout.tabRects[index].Dx(),
			etext.AlignCenter,
		)
	}
}

func (d *ConfigurationDialog) drawGameplayContent(screen *ebiten.Image) {
	settings := gameplaySettings()
	for index, rect := range d.layout.gameplayRows {
		textColor := widgets.DialogValueColor
		markColor := widgets.DialogCheckedMarkColor
		if index == d.focusedGameplayIndex {
			widgets.FillClassicPixelRoundedRect(
				screen, rect, d.layout.pixelUnit, widgets.DialogSelectedBackground,
			)
			textColor = widgets.DialogSelectedText
			markColor = widgets.DialogSelectedCheckedMarkColor
		}
		d.gameplayOutput.Reset()
		d.gameplayOutput.AppendLine().Text(settings[index].label, textColor).WordWrap().Done()
		inset := max(1, d.layout.pixelUnit*audioSettingsCenterDivisor)
		top := centeredAudioOutputY(rect, d.gameplayOutput)
		d.labelOutput.Reset()
		d.labelOutput.AppendLine().
			Text("[", textColor).Text(d.gameplayCheckmark(index), markColor).
			Text("]", textColor).
			Done()
		d.labelOutput.DrawPanelContent(screen, image.Pt(rect.Min.X+inset, top), d.gameplayCheckboxWidth(), etext.AlignStart)
		d.gameplayOutput.DrawPanelContent(
			screen,
			image.Pt(rect.Min.X+inset+d.gameplayCheckboxWidth(), top),
			d.gameplayLabelWidth(rect.Dx(), d.layout.pixelUnit),
			etext.AlignStart,
		)
	}

	d.prepareGameplayDetail()
	inset := max(1, d.layout.pixelUnit*audioSettingsCenterDivisor)
	d.detailOutput.DrawPanelContent(
		screen,
		image.Pt(d.layout.gameplayDetail.Min.X+inset, d.layout.gameplayDetail.Min.Y+inset),
		max(1, d.layout.gameplayDetail.Dx()-inset*audioSettingsCenterDivisor),
		etext.AlignStart,
	)
	widgets.DrawClassicPixelRoundedFrame(
		screen, d.layout.gameplayListRect, d.layout.pixelUnit, widgets.DialogRuleColor,
	)
	widgets.DrawClassicPixelRoundedFrame(
		screen, d.layout.gameplayDetail, d.layout.pixelUnit, widgets.DialogRuleColor,
	)
}

// prepareGameplayDetail uses the measured layout's wrapping capacity without
// rasterizing. Selection and staged settings remain owned by Update.
func (d *ConfigurationDialog) prepareGameplayDetail() {
	inset := max(1, d.layout.pixelUnit*audioSettingsCenterDivisor)
	d.detailOutput.SetMaxCharsPerLine(max(1,
		d.detailOutput.Font.CharsForWidth(d.layout.gameplayDetail.Dx()-inset*audioSettingsCenterDivisor)))
	d.detailOutput.Reset()
	selected := gameplaySettings()[d.focusedGameplayIndex]
	d.detailOutput.AppendLine().Text(selected.label, widgets.DialogHeaderColor).WordWrap().Done()
	d.detailOutput.AppendLine().Text(selected.detail, widgets.DialogValueColor).WordWrap().Done()
}

func (d *ConfigurationDialog) drawAudioContent(screen *ebiten.Image) {
	sliders := []*widgets.Slider{d.musicSlider, d.sfxSlider, d.ambientSlider}
	labels := []string{"Music Volume", "SFX Volume", "Ambient Volume"}
	for index, slider := range sliders {
		d.labelOutput.Reset()
		d.labelOutput.AppendLine().Text(labels[index], widgets.DialogLabelColor).Done()
		d.labelOutput.DrawPanelContent(
			screen,
			d.layout.sliderLabelRects[index].Min,
			d.layout.sliderLabelRects[index].Dx(),
			etext.AlignStart,
		)

		d.valueOutput.Reset()
		d.valueOutput.AppendLine().Text(strconv.Itoa(slider.GetValue()), widgets.DialogValueColor).Done()
		d.valueOutput.DrawPanelContent(
			screen,
			d.layout.sliderValueRects[index].Min,
			d.layout.sliderValueRects[index].Dx(),
			etext.AlignEnd,
		)
		slider.Draw(screen)
	}
}

func (d *ConfigurationDialog) drawFooter(screen *ebiten.Image) {
	d.footerOutput.Reset()
	switch {
	case d.audioReport != nil:
		hints := []string{"Up/Down: Scroll  PageUp/PageDown: Page", "R: Refresh captured state", "D/Esc: Back to Sound"}
		for _, hint := range hints {
			widgets.AppendControlHints(d.footerOutput.AppendLine(), hint).Done()
		}
	case d.displayPhase == configurationDisplayApplying:
		d.footerOutput.AppendLine().
			Text("Applying display settings...", widgets.DialogStatusWarningColor).
			Done()
	case d.displayPhase == configurationDisplayReverting:
		d.footerOutput.AppendLine().
			Text("Restoring previous display settings...", widgets.DialogStatusWarningColor).
			Done()
	case d.saveError != "":
		d.footerOutput.AppendLine().
			Text(d.saveError, widgets.DialogStatusWarningColor).
			WordWrap().
			Done()
		widgets.AppendControlHints(d.footerOutput.AppendLine(), d.configurationErrorGuidance()).Done()
	default:
		var adjust string
		switch d.currentTab {
		case audioSettingsGameplayTab:
			adjust = "Space: Toggle"
		case audioSettingsAudioTab:
			adjust = "Left/Right: Adjust  D: Diagnostics"
		case audioSettingsDisplayTab:
			adjust = "Left/Right: Change"
		default:
			panic("unsupported Configuration section")
		}
		for _, hint := range []string{"Tab/Shift-Tab: Section  Up/Down: Select", adjust, "Enter: Apply  Esc: Cancel"} {
			widgets.AppendControlHints(d.footerOutput.AppendLine(), hint).Done()
		}
	}
	d.footerOutput.DrawPanelContent(
		screen,
		image.Pt(
			d.layout.footerRect.Min.X,
			centeredAudioOutputY(d.layout.footerRect, d.footerOutput),
		),
		d.layout.footerRect.Dx(),
		etext.AlignCenter,
	)
}

func (d *ConfigurationDialog) configurationErrorGuidance() string {
	if d.displayRollbackRetry {
		return "Enter: Retry restore  Esc: Cancel"
	}

	return "Enter: Retry  Esc: Cancel"
}

func (d *ConfigurationDialog) gameplayCheckmark(index int) string {
	if d.gameplaySettingValue(index) {
		return "X"
	}

	return " "
}

func (d *ConfigurationDialog) gameplaySettingValue(index int) bool {
	switch index {
	case gameplayRepeatedDivisionIndex:
		return d.stagedGameplay.AllowRepeatedMonsterDivision
	case gameplayHiddenDiscoveriesIndex:
		return d.stagedGameplay.PersistHiddenDungeonDiscoveries
	case gameplayUnlitDungeonsIndex:
		return d.stagedGameplay.ShowSubduedUnlitDungeons
	case gameplaySmoothSurfaceMovementIndex:
		return d.stagedGameplay.SmoothSurfaceMovement
	case gameplayBatchMerchantTheftIndex:
		return d.stagedGameplay.BatchMerchantTheft
	case gameplayConcealedTurnSlotsIndex:
		return d.stagedGameplay.ShowConcealedTurnOrderSlots
	case gameplayFlexibleSextantUseIndex:
		return d.stagedGameplay.FlexibleSextantUse
	default:
		panic("unsupported Gameplay setting index")
	}
}

func (d *ConfigurationDialog) toggleFocusedGameplaySetting() {
	switch d.focusedGameplayIndex {
	case gameplayRepeatedDivisionIndex:
		d.stagedGameplay.AllowRepeatedMonsterDivision = !d.stagedGameplay.AllowRepeatedMonsterDivision
	case gameplayHiddenDiscoveriesIndex:
		d.stagedGameplay.PersistHiddenDungeonDiscoveries = !d.stagedGameplay.PersistHiddenDungeonDiscoveries
	case gameplayUnlitDungeonsIndex:
		d.stagedGameplay.ShowSubduedUnlitDungeons = !d.stagedGameplay.ShowSubduedUnlitDungeons
	case gameplaySmoothSurfaceMovementIndex:
		d.stagedGameplay.SmoothSurfaceMovement = !d.stagedGameplay.SmoothSurfaceMovement
	case gameplayBatchMerchantTheftIndex:
		d.stagedGameplay.BatchMerchantTheft = !d.stagedGameplay.BatchMerchantTheft
	case gameplayConcealedTurnSlotsIndex:
		d.stagedGameplay.ShowConcealedTurnOrderSlots = !d.stagedGameplay.ShowConcealedTurnOrderSlots
	case gameplayFlexibleSextantUseIndex:
		d.stagedGameplay.FlexibleSextantUse = !d.stagedGameplay.FlexibleSextantUse
	default:
		panic("unsupported Gameplay setting index")
	}
	d.saveError = ""
}

func centeredAudioOutputY(rect image.Rectangle, output *text.Output) int {
	lineCount := output.GetPanelContentLineCount()
	if lineCount == 0 {
		lineCount = 1
	}
	contentHeight := int(math.Ceil(output.GetTotalHeightForLines(lineCount)))

	return rect.Min.Y + max(0, (rect.Dy()-contentHeight)/audioSettingsCenterDivisor)
}
