package widgets

import (
	"fmt"
	"strings"
)

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

	splitCommandStrings := strings.Split(outputStr, " ")

	hasExactMatchNeedingMoreParams := false
	for _, match := range *primaryCommandMatches {
		if len(match.Matches) > 1 && len(splitCommandStrings) == 1 {
			commandStr := match.Matches[0].GetString()
			inputStr := splitCommandStrings[0]
			if strings.EqualFold(commandStr, inputStr) {
				hasExactMatchNeedingMoreParams = true

				break
			}
		}
	}

	if hasExactMatchNeedingMoreParams && bAddSpaceAtEndIfPerfectMatch {
		t.output.AppendText(" ")

		return
	}

	if nPrimaryCommandMatch == 1 && len(splitCommandStrings) > (*primaryCommandMatches)[0].GetNumberOfParameters() {
		return
	}

	if nPrimaryCommandMatch == 0 {
		return
	}

	hasTrailingSpace := len(outputStr) > 0 && outputStr[len(outputStr)-1] == ' '
	splitStr := strings.Split(outputStr, " ")

	var currentPosition int
	var currentToken string

	if hasTrailingSpace {
		currentPosition = len(splitStr) - 1
		currentToken = ""
	} else {
		currentPosition = len(splitStr) - 1
		currentToken = splitStr[currentPosition]
	}

	optionsMap := make(map[string]bool)

	for _, cmd := range *primaryCommandMatches {
		if currentPosition < len(cmd.Matches) {
			match := cmd.Matches[currentPosition]

			matchStr := match.GetString()

			if matchStr != "" {
				if strings.Contains(matchStr, " to ") {
					continue
				}
				if currentToken == "" || strings.HasPrefix(strings.ToUpper(matchStr), strings.ToUpper(currentToken)) {
					optionsMap[matchStr] = true
				}
			} else {
				partialMatches, err := match.GetPartialMatches(currentToken)
				if err != nil {
					continue
				}
				for _, opt := range partialMatches {
					optionsMap[opt] = true
				}
			}
		}
	}

	if len(optionsMap) == 0 {
		if currentToken != "" {
			for _, cmd := range *primaryCommandMatches {
				if currentPosition < len(cmd.Matches) {
					match := cmd.Matches[currentPosition]
					matchStr := match.GetString()

					if matchStr != "" && strings.Contains(matchStr, " to ") {
						isValid, err := match.PartiallyMatches(currentToken)
						if err == nil && isValid {
							if bAddSpaceAtEndIfPerfectMatch {
								t.output.AppendText(" ")
							}

							return
						}

						break
					}
				}
			}
		}

		for _, cmd := range *primaryCommandMatches {
			if currentPosition < len(cmd.Matches) {
				match := cmd.Matches[currentPosition]
				matchStr := match.GetString()
				if matchStr != "" && strings.Contains(matchStr, " to ") {
					t.TextInputCallbacks.AmbiguousAutoComplete(
						fmt.Sprintf("Enter value (%s)", matchStr),
					)

					return
				}
			}
		}

		return
	}

	options := make([]string, 0, len(optionsMap))
	for opt := range optionsMap {
		options = append(options, opt)
	}

	if len(options) > 1 && currentToken != "" {
		hasExactMatch := false
		var exactMatchOption string
		for _, opt := range options {
			if strings.EqualFold(opt, currentToken) {
				hasExactMatch = true
				exactMatchOption = opt

				break
			}
		}

		if !hasExactMatch {
			if !isTabKey {
				t.TextInputCallbacks.AmbiguousAutoComplete(
					"Ambiguous: " + strings.Join(options, ", "),
				)

				return
			}
		} else {
			if !isTabKey {
				options = []string{exactMatchOption}
			}
		}
	}

	if len(options) == 1 {
		option := options[0]

		if currentToken != "" {
			optionLower := strings.ToLower(option)
			currentTokenLower := strings.ToLower(currentToken)

			if optionLower == currentTokenLower {
				// Already complete
			} else if len(currentTokenLower) < len(optionLower) {
				suffix := option[len(currentToken):]
				t.output.AppendText(strings.ToUpper(suffix))
			}
		} else {
			t.output.AppendText(strings.ToUpper(option))
		}

		if bAddSpaceAtEndIfPerfectMatch {
			t.output.AppendText(" ")
		}

		return
	}

	commonPrefix := findLongestCommonPrefix(options)
	currentTokenLower := strings.ToLower(currentToken)

	if len(commonPrefix) > len(currentTokenLower) {
		suffix := commonPrefix[len(currentTokenLower):]
		t.output.AppendText(strings.ToUpper(suffix))
	}

	t.TextInputCallbacks.AmbiguousAutoComplete("Options: " + strings.Join(options, ", "))
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
