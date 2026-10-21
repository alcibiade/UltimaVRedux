package grammar

import (
	"strings"
)

// ExecuteFunc runs a matched command with the full typed input.
type ExecuteFunc func(string, *TextCommand)

// TextCommand is a registered command: one word slot per typed word, plus the callback that runs it.
type TextCommand struct {
	Matches        Matches
	ExecuteCommand ExecuteFunc
}

// NewTextCommand builds a command from its word slots and the callback that runs it.
func NewTextCommand(matchList Matches, execute ExecuteFunc) *TextCommand {
	textCommand := TextCommand{Matches: matchList, ExecuteCommand: execute}

	return &textCommand
}

// GetNumberOfParameters returns how many typed words the command expects.
func (t *TextCommand) GetNumberOfParameters() int {
	return len(t.Matches)
}

func (t *TextCommand) getTextCommandIfAtLeastPartialMatch(command string) *TextCommand {
	splitCommand := splitCommand(command)

	nSplitCommands := len(splitCommand)
	if nSplitCommands > len(t.Matches) {
		return nil
	}
	for i := range nSplitCommands {
		bMatched, err := t.Matches[i].PartiallyMatches(splitCommand[i])
		if err != nil {
			return nil
		}
		if bMatched && i == nSplitCommands-1 {
			return t
		} else if !bMatched {
			return nil
		}
	}

	return nil
}

func (t *TextCommand) getTextCommandIfPerfectMatch(command string) *TextCommand {
	splitCommand := splitCommand(command)

	if len(splitCommand) != len(t.Matches) {
		return nil
	}

	return t.getTextCommandIfAtLeastPartialMatch(command)
}

func splitCommand(command string) []string {
	splitCommand := strings.Split(command, " ")
	if splitCommand[len(splitCommand)-1] == "" {
		splitCommand = splitCommand[:len(splitCommand)-1]
	}

	return splitCommand
}
