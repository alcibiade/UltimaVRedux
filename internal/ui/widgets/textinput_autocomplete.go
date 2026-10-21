package widgets

import (
	"fmt"
	"strings"

	"github.com/bradhannah/UltimaVRedux/pkg/grammar"
)

// rangePromptMarker in a slot's text marks a value-range prompt such as "1 to 99" rather than a word to complete.
const rangePromptMarker = " to "

func (t *TextInput) tryToAutoComplete(bAddSpaceAtEndIfPerfectMatch bool, isTabKey bool) {
	outputStr := t.output.GetOutputStr(false)

	if outputStr == "" {
		// Use paginated command list if callback is set, otherwise fall back to original behavior
		if t.TextInputCallbacks.ShowCommandList != nil {
			t.TextInputCallbacks.ShowCommandList()
		} else {
			t.TextInputCallbacks.AmbiguousAutoComplete("Available Commands: " + t.textCommands.GetFriendlyListOfAllCommands())
		}

		return
	}

	nPrimaryCommandMatch := t.textCommands.HowManyCommandsMatch(outputStr)
	primaryCommandMatches := t.textCommands.GetAllMatchingTextCommands(outputStr)

	splitStr := strings.Split(outputStr, " ")

	if hasExactMatchNeedingMoreParams(primaryCommandMatches, splitStr) && bAddSpaceAtEndIfPerfectMatch {
		t.output.AppendText(" ")

		return
	}

	if nPrimaryCommandMatch == 1 && len(splitStr) > (*primaryCommandMatches)[0].GetNumberOfParameters() {
		return
	}

	if nPrimaryCommandMatch == 0 {
		return
	}

	// The token being completed is the last word; it is empty after a trailing space.
	currentPosition := len(splitStr) - 1
	currentToken := splitStr[currentPosition]

	optionsMap := collectCompletionOptions(primaryCommandMatches, currentPosition, currentToken)
	if len(optionsMap) == 0 {
		t.promptForRangeValue(primaryCommandMatches, currentPosition, currentToken, bAddSpaceAtEndIfPerfectMatch)

		return
	}

	options := make([]string, 0, len(optionsMap))
	for opt := range optionsMap {
		options = append(options, opt)
	}

	options, handled := t.narrowAmbiguousOptions(options, currentToken, isTabKey)
	if handled {
		return
	}

	if len(options) == 1 {
		t.completeSingleOption(options[0], currentToken, bAddSpaceAtEndIfPerfectMatch)

		return
	}

	t.completeCommonPrefix(options, currentToken)
	t.TextInputCallbacks.AmbiguousAutoComplete("Options: " + strings.Join(options, ", "))
}

// completeCommonPrefix appends whatever all options share beyond the typed token.
func (t *TextInput) completeCommonPrefix(options []string, token string) {
	commonPrefix := findLongestCommonPrefix(options)
	tokenLower := strings.ToLower(token)

	if len(commonPrefix) > len(tokenLower) {
		t.output.AppendText(strings.ToUpper(commonPrefix[len(tokenLower):]))
	}
}

// hasExactMatchNeedingMoreParams reports whether the single typed word is a complete first word of a longer command.
func hasExactMatchNeedingMoreParams(matches *grammar.TextCommands, splitCommandStrings []string) bool {
	if len(splitCommandStrings) != 1 {
		return false
	}

	for _, match := range *matches {
		if len(match.Matches) > 1 && strings.EqualFold(match.Matches[0].GetString(), splitCommandStrings[0]) {
			return true
		}
	}

	return false
}

// collectCompletionOptions gathers the distinct words that could fill the slot at position.
func collectCompletionOptions(matches *grammar.TextCommands, position int, token string) map[string]bool {
	optionsMap := make(map[string]bool)

	for _, cmd := range *matches {
		if position >= len(cmd.Matches) {
			continue
		}

		match := cmd.Matches[position]
		matchStr := match.GetString()

		if matchStr == "" {
			partialMatches, err := match.GetPartialMatches(token)
			if err != nil {
				continue
			}
			for _, opt := range partialMatches {
				optionsMap[opt] = true
			}

			continue
		}

		if strings.Contains(matchStr, rangePromptMarker) {
			continue
		}
		if token == "" || strings.HasPrefix(strings.ToUpper(matchStr), strings.ToUpper(token)) {
			optionsMap[matchStr] = true
		}
	}

	return optionsMap
}

// promptForRangeValue handles a slot with no word options: a value-range slot either accepts the
// typed value (adding a space) or is described to the player.
func (t *TextInput) promptForRangeValue(
	matches *grammar.TextCommands, position int, token string, addSpace bool,
) {
	var rangeMatch grammar.Match
	rangeStr := ""

	for _, cmd := range *matches {
		if position >= len(cmd.Matches) {
			continue
		}

		matchStr := cmd.Matches[position].GetString()
		if matchStr != "" && strings.Contains(matchStr, rangePromptMarker) {
			rangeMatch = cmd.Matches[position]
			rangeStr = matchStr

			break
		}
	}

	if rangeMatch == nil {
		return
	}

	if token != "" {
		isValid, err := rangeMatch.PartiallyMatches(token)
		if err == nil && isValid {
			if addSpace {
				t.output.AppendText(" ")
			}

			return
		}
	}

	t.TextInputCallbacks.AmbiguousAutoComplete(fmt.Sprintf("Enter value (%s)", rangeStr))
}

// narrowAmbiguousOptions reports ambiguity to the player when the typed token matches no option exactly
// (handled is true), and on a non-Tab key collapses the options to an exact match.
func (t *TextInput) narrowAmbiguousOptions(options []string, token string, isTabKey bool) ([]string, bool) {
	if len(options) <= 1 || token == "" || isTabKey {
		return options, false
	}

	for _, opt := range options {
		if strings.EqualFold(opt, token) {
			return []string{opt}, false
		}
	}

	t.TextInputCallbacks.AmbiguousAutoComplete("Ambiguous: " + strings.Join(options, ", "))

	return options, true
}

// completeSingleOption appends the rest of the only remaining option, upper-cased, as the game's prompt shows it.
func (t *TextInput) completeSingleOption(option, token string, addSpace bool) {
	switch {
	case token == "":
		t.output.AppendText(strings.ToUpper(option))
	case len(token) < len(option):
		t.output.AppendText(strings.ToUpper(option[len(token):]))
	}

	if addSpace {
		t.output.AppendText(" ")
	}
}

func findLongestCommonPrefix(options []string) string {
	if len(options) == 0 {
		return ""
	}
	if len(options) == 1 {
		return strings.ToLower(options[0])
	}

	prefix := strings.ToLower(options[0])
	for _, opt := range options[1:] {
		optLower := strings.ToLower(opt)
		commonLen := 0
		minLen := len(prefix)
		if len(optLower) < minLen {
			minLen = len(optLower)
		}
		for i := range minLen {
			if prefix[i] == optLower[i] {
				commonLen++
			} else {
				break
			}
		}
		prefix = prefix[:commonLen]
		if prefix == "" {
			return ""
		}
	}

	return prefix
}
