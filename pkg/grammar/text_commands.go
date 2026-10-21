package grammar

import (
	"fmt"
	"strings"
)

type TextCommands []TextCommand

// OneOrMoreCommandsMatch
// Indicates if the string provide at least partially match the command
func (t *TextCommands) OneOrMoreCommandsMatch(command string) bool {
	return t.HowManyCommandsMatch(command) > 0
}

// HowManyCommandsMatch
// Returns the number of commands that currently match the text entered
func (t *TextCommands) HowManyCommandsMatch(command string) int {
	return len(*t.GetAllMatchingTextCommands(command))
}

func (t *TextCommands) GetAllMatchingTextCommands(command string) *TextCommands {
	allMatches := make(TextCommands, 0)
	for _, checkCommand := range *t {
		m := checkCommand.GetTextCommandIfAtLeastPartialMatch(command)
		if m == nil {
			continue
		}
		allMatches = append(allMatches, *m)
	}

	return &allMatches
}

func (t *TextCommands) GetAllPerfectMatches(command string) *TextCommands {
	allMatches := make(TextCommands, 0)
	for _, checkCommand := range *t {
		m := checkCommand.GetTextCommandIfPerfectMatch(command)
		if m == nil {
			continue
		}
		allMatches = append(allMatches, *m)
	}

	return &allMatches
}

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
