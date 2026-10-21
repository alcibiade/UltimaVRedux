package grammar

import (
	"strings"
)

type ExecuteFunc func(string, *TextCommand)

type TextCommand struct {
	Matches        Matches
	ExecuteCommand ExecuteFunc
}

func NewTextCommand(matches Matches, executeFunc ExecuteFunc) *TextCommand {
	textCommand := TextCommand{}
	textCommand.Matches = matches
	textCommand.ExecuteCommand = executeFunc

	return &textCommand
}

func (t *TextCommand) GetNumberOfParameters() int {
	return len(t.Matches)
}

func (t *TextCommand) GetTextCommandIfAtLeastPartialMatch(command string) *TextCommand {
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

func (t *TextCommand) GetTextCommandIfPerfectMatch(command string) *TextCommand {
	splitCommand := splitCommand(command)

	if len(splitCommand) != len(t.Matches) {
		return nil
	}

	return t.GetTextCommandIfAtLeastPartialMatch(command)
}

func splitCommand(command string) []string {
	splitCommand := strings.Split(command, " ")
	if splitCommand[len(splitCommand)-1] == "" {
		splitCommand = splitCommand[:len(splitCommand)-1]
	}

	return splitCommand
}
