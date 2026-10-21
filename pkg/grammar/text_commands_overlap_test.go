package grammar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- CouldOverlapWith unit tests ---

func TestMatchStringList_CouldOverlapWith_MatchStringList(t *testing.T) {
	directions := MatchStringList{Strings: []string{"north", "south", "east", "west"}}
	monsters := MatchStringList{Strings: []string{"skeleton", "slime", "ettin", "wisp"}}
	otherDirs := MatchStringList{Strings: []string{"up", "down"}}

	// With first-char overlap detection, "south" and "skeleton" share "s",
	// "east" and "ettin" share "e". So these lists DO overlap — a user typing
	// "s" or "e" would match items from both lists.
	t.Run("directions vs monsters with shared first char", func(t *testing.T) {
		assert.True(t, directions.CouldOverlapWith(monsters),
			"should detect overlap: south/skeleton share 's', east/ettin share 'e'")
		assert.True(t, monsters.CouldOverlapWith(directions),
			"must be symmetric")
	})

	t.Run("directions vs unrelated list", func(t *testing.T) {
		assert.False(t, directions.CouldOverlapWith(otherDirs))
	})

	t.Run("identical lists", func(t *testing.T) {
		assert.True(t, directions.CouldOverlapWith(directions))
	})
}

// --- ValidateNoAmbiguousCommands tests ---

// words is a command slot that accepts exactly these strings.
func words(values ...string) MatchStringList {
	return MatchStringList{Strings: values}
}

// --- ValidateNoAmbiguousCommands tests ---

func TestValidateNoAmbiguousCommands_NoOverlap(t *testing.T) {
	cmds := TextCommands{
		// 3-slot command: "combat list brit"
		*NewTextCommand([]Match{words("combat"), words("list"), words("brit")}, nil),
		// 3-slot command: "combat list dungeons" — different at position 2
		*NewTextCommand([]Match{words("combat"), words("list"), words("dungeons")}, nil),
	}

	assert.NotPanics(t, func() {
		cmds.ValidateNoAmbiguousCommands()
	})
}

func TestValidateNoAmbiguousCommands_DifferentSlotCount(t *testing.T) {
	cmds := TextCommands{
		// 2-slot command
		*NewTextCommand([]Match{words("combat"), words("list")}, nil),
		// 3-slot command — different slot count, never compared
		*NewTextCommand([]Match{words("combat"), words("list"), words("brit")}, nil),
	}

	assert.NotPanics(t, func() {
		cmds.ValidateNoAmbiguousCommands()
	})
}

func TestValidateNoAmbiguousCommands_DetectsIdenticalCommands(t *testing.T) {
	cmds := TextCommands{
		*NewTextCommand([]Match{words("combat"), words("list")}, nil),
		*NewTextCommand([]Match{words("combat"), words("list")}, nil),
	}

	assert.Panics(t, func() {
		cmds.ValidateNoAmbiguousCommands()
	})
}

func TestValidateNoAmbiguousCommands_DetectsOverlapAtEveryPosition(t *testing.T) {
	directions := words("north", "south", "east", "west")
	cmds := TextCommands{
		*NewTextCommand([]Match{words("combat"), words("brit"), directions}, nil),
		// Position 2 differs as a list but shares first characters, so typing "s" matches both.
		*NewTextCommand([]Match{words("combat"), words("brit"), words("skeleton", "ettin")}, nil),
	}

	assert.Panics(t, func() {
		cmds.ValidateNoAmbiguousCommands()
	})
}

func TestValidateNoAmbiguousCommands_SingleCommand(t *testing.T) {
	cmds := TextCommands{
		*NewTextCommand([]Match{words("help")}, nil),
	}

	assert.NotPanics(t, func() {
		cmds.ValidateNoAmbiguousCommands()
	})
}

func TestTextCommandsMatchPartialInputWordByWord(t *testing.T) {
	cmds := TextCommands{
		*NewTextCommand([]Match{words("combat"), words("list"), words("brit")}, nil),
		*NewTextCommand([]Match{words("combat"), words("list"), words("dungeons")}, nil),
		*NewTextCommand([]Match{words("help")}, nil),
	}

	assert.Equal(t, 2, cmds.HowManyCommandsMatch("com"))
	assert.Equal(t, 2, cmds.HowManyCommandsMatch("combat list "), "a trailing space does not add a word")
	assert.Equal(t, 1, cmds.HowManyCommandsMatch("combat list d"))
	assert.True(t, cmds.OneOrMoreCommandsMatch("help"))
	assert.False(t, cmds.OneOrMoreCommandsMatch("combat help"), "every typed word must match its slot")
	assert.False(t, cmds.OneOrMoreCommandsMatch("help me"), "more words than slots never match")

	matches := *cmds.GetAllMatchingTextCommands("combat list b")
	if assert.Len(t, matches, 1) {
		assert.Equal(t, []string{"brit"}, matches[0].Matches[2].(MatchStringList).Strings)
	}
}
