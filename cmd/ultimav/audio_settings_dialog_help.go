package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

// Each entry is an Output row with natural wrapping; empty entries are real
// paragraph-spacing rows. Do not embed newlines or unsupported punctuation.
func combatSpeedHelpLines(spec config.NumericSettingSpec) []string {
	return []string{
		"Change how quickly combat actions and animations play.", "",
		fmt.Sprintf("%.1fx is normal speed.", spec.Default),
		"Higher values make combat faster.", "",
		fmt.Sprintf("Choose %.1fx to %.1fx.", spec.Min, spec.Max),
		fmt.Sprintf("Left/Right changes speed by %.1fx.", spec.Step), "",
		fmt.Sprintf("Default: %.1fx", spec.Default),
	}
}

func actionPopupsHelpLines(spec config.NumericSettingSpec) []string {
	choices := make([]string, 0, int((spec.Max-spec.Min)/spec.Step)+1)
	for value := int(spec.Min); value <= int(spec.Max); value += int(spec.Step) {
		choice := strconv.Itoa(value)
		if value == 0 {
			choice = gameplaySettingOffLabel
		}
		choices = append(choices, choice)
	}
	last := len(choices) - 1
	choiceText := strings.Join(choices[:last], ", ") + " or " + choices[last]

	return []string{
		"Choose how many action messages can appear over the map at once.", "",
		"Off hides these popups.",
		"Messages still appear in the lower-right text area.", "",
		"Choose " + choiceText + ".", "",
		"Default: " + strconv.Itoa(int(spec.Default)),
	}
}

func (setting gameplaySettingDefinition) detailLines() []string {
	if setting.helpLines != nil {
		return setting.helpLines
	}

	return []string{setting.detail}
}

// Measurement and drawing populate the same Output, including blank rows and
// wrapping. This avoids estimating paragraph height from concatenated strings.
func populateGameplayDetail(output *text.Output, setting gameplaySettingDefinition, value string) {
	output.Reset()
	output.AppendLine().Text(setting.label, widgets.DialogHeaderColor).WordWrap().Done()
	for _, line := range setting.detailLines() {
		output.AppendLine().Text(line, widgets.DialogValueColor).WordWrap().Done()
	}
	output.AppendLine().Text("Value: ", widgets.DialogLabelColor).Text(value, widgets.DialogValueColor).Done()
}
