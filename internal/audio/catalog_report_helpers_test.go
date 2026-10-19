package audio

import (
	"html"
	"strings"
)

// Test-only report expectations retain the prior game regression vocabulary;
// production report rendering is owned by the SDK.
func needsRecordingBrief(moment CatalogMoment) bool {
	return moment.GenerationBrief != "" && moment.TriggerCoverage != catalogDeclaredOnly && moment.TriggerCoverage != catalogNoReference &&
		(moment.Availability == CatalogMissing || moment.AssignmentReview == CatalogIncorrect)
}
func briefAnchor(id string) string {
	return "brief-" + strings.Map(func(character rune) rune {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' {
			return character
		}
		return '-'
	}, strings.ToLower(id))
}
func tableCell(value string) string {
	return strings.NewReplacer("|", "\\|", "\r", "", "\n", "<br>").Replace(html.EscapeString(value))
}
