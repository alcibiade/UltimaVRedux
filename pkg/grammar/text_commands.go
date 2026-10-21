package grammar

import (
	"fmt"
	"strings"
)

// TextCommands is the registered command set that typed input is matched against.
type TextCommands []TextCommand

// OneOrMoreCommandsMatch reports whether the input at least partially matches one command.
func (t *TextCommands) OneOrMoreCommandsMatch(command string) bool {
	return t.HowManyCommandsMatch(command) > 0
}

// HowManyCommandsMatch returns how many commands currently match the input.
func (t *TextCommands) HowManyCommandsMatch(command string) int {
	return len(*t.GetAllMatchingTextCommands(command))
}

// GetAllMatchingTextCommands returns the commands the input could still complete.
func (t *TextCommands) GetAllMatchingTextCommands(command string) *TextCommands {
	allMatches := make(TextCommands, 0)
	for _, checkCommand := range *t {
		m := checkCommand.getTextCommandIfAtLeastPartialMatch(command)
		if m == nil {
			continue
		}
		allMatches = append(allMatches, *m)
	}

	return &allMatches
}

// GetAllPerfectMatches returns the commands the input matches exactly.
func (t *TextCommands) GetAllPerfectMatches(command string) *TextCommands {
	allMatches := make(TextCommands, 0)
	for _, checkCommand := range *t {
		m := checkCommand.getTextCommandIfPerfectMatch(command)
		if m == nil {
			continue
		}
		allMatches = append(allMatches, *m)
	}

	return &allMatches
}

// GetFriendlyListOfAllCommands lists each distinct root command with its description, comma-separated.
func (t *TextCommands) GetFriendlyListOfAllCommands() string {
	// Use map to deduplicate command outputs (prevents duplicate root commands)
	seenCommands := make(map[string]bool)
	var uniqueCommands []string

	for _, textCommand := range *t {
		commandOutput := fmt.Sprintf("%s (%s)",
			textCommand.Matches[0].GetString(),
			textCommand.Matches[0].GetDescription())

		// Skip if already added
		if !seenCommands[commandOutput] {
			seenCommands[commandOutput] = true
			uniqueCommands = append(uniqueCommands, commandOutput)
		}
	}

	return strings.Join(uniqueCommands, ", ")
}
