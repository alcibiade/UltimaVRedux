package grammar

import (
	"fmt"
	"strings"
)

type TextCommands []TextCommand

func (t *TextCommands) GetNumOfCommands() int {
	return len(*t)
}

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

// ValidateNoAmbiguousCommands checks all registered commands for ambiguous
// overlap. Two commands with the same slot count are ambiguous if every
// position's match slot could accept overlapping input. This means a user
// could type a string that "perfectly matches" both commands, causing silent
// execution failure (GetAllPerfectMatches returns >1 result).
//
// Panics if any ambiguous pair is found — this is a programming bug in
// command registration that must be fixed before the program can run.
func (t *TextCommands) ValidateNoAmbiguousCommands() {
	commands := *t

	// Group commands by slot count
	bySlotCount := make(map[int][]int) // slot count -> indices
	for i, cmd := range commands {
		n := len(cmd.Matches)
		bySlotCount[n] = append(bySlotCount[n], i)
	}

	// For each group, check all pairs
	for slotCount, indices := range bySlotCount {
		for a := range indices {
			for b := a + 1; b < len(indices); b++ {
				cmdA := commands[indices[a]]
				cmdB := commands[indices[b]]

				if commandsOverlapAtAllPositions(cmdA, cmdB, slotCount) {
					panic(fmt.Sprintf(
						"grammar: ambiguous command overlap detected (%d slots)\n"+
							"  Command A: %s\n"+
							"  Command B: %s\n"+
							"Both commands could match the same input at every position, "+
							"causing silent execution failure.",
						slotCount,
						describeCommand(cmdA),
						describeCommand(cmdB),
					))
				}
			}
		}
	}
}

// commandsOverlapAtAllPositions returns true if every slot position in the two
// commands could accept overlapping input.
func commandsOverlapAtAllPositions(a, b TextCommand, slotCount int) bool {
	for i := range slotCount {
		if !a.Matches[i].CouldOverlapWith(b.Matches[i]) {
			return false
		}
	}

	return true
}

// describeCommand returns a human-readable description of a command's match
// pattern for use in error messages.
func describeCommand(cmd TextCommand) string {
	result := ""
	var resultSb130 strings.Builder
	for i, m := range cmd.Matches {
		if i > 0 {
			resultSb130.WriteString(" ")
		}

		desc := m.GetDescription()
		str := m.GetString()

		switch {
		case str != "" && desc != "":
			result += fmt.Sprintf("[%s (%s)]", str, desc)
		case str != "":
			result += fmt.Sprintf("[%s]", str)
		case desc != "":
			result += fmt.Sprintf("[%s]", desc)
		default:
			result += fmt.Sprintf("[%T]", m)
		}
	}
	result += resultSb130.String()

	return result
}

// CommandEntry represents a single command with its description
type CommandEntry struct {
	Command     string
	Description string
}

// GetAllCommandEntries returns all unique commands as individual entries
// Used for paginated command list display
func (t *TextCommands) GetAllCommandEntries() []CommandEntry {
	seenCommands := make(map[string]bool)
	var entries []CommandEntry

	for _, textCommand := range *t {
		cmd := textCommand.Matches[0].GetString()
		desc := textCommand.Matches[0].GetDescription()
		key := cmd + ":" + desc

		if !seenCommands[key] {
			seenCommands[key] = true
			entries = append(entries, CommandEntry{Command: cmd, Description: desc})
		}
	}

	return entries
}
