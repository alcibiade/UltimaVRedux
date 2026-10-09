package grammar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- CouldOverlapWith unit tests ---

func TestMatchString_CouldOverlapWith_MatchString(t *testing.T) {
	tests := []struct {
		name    string
		a, b    MatchString
		overlap bool
	}{
		{
			name:    "identical strings overlap",
			a:       MatchString{Str: "combat"},
			b:       MatchString{Str: "combat"},
			overlap: true,
		},
		{
			name:    "prefix overlap - a is prefix of b",
			a:       MatchString{Str: "com"},
			b:       MatchString{Str: "combat"},
			overlap: true,
		},
		{
			name:    "prefix overlap - b is prefix of a",
			a:       MatchString{Str: "combat"},
			b:       MatchString{Str: "com"},
			overlap: true,
		},
		{
			name:    "case insensitive overlap",
			a:       MatchString{Str: "COMBAT"},
			b:       MatchString{Str: "combat"},
			overlap: true,
		},
		{
			name:    "no overlap - different strings",
			a:       MatchString{Str: "combat"},
			b:       MatchString{Str: "dungeon"},
			overlap: false,
		},
		{
			name:    "no overlap - different first char",
			a:       MatchString{Str: "list"},
			b:       MatchString{Str: "brit"},
			overlap: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.overlap, tt.a.CouldOverlapWith(tt.b),
				"a.CouldOverlapWith(b)")
			assert.Equal(t, tt.overlap, tt.b.CouldOverlapWith(tt.a),
				"b.CouldOverlapWith(a) — must be symmetric")
		})
	}
}

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

func TestMatchInt_CouldOverlapWith_MatchInt(t *testing.T) {
	tests := []struct {
		name    string
		a, b    MatchInt
		overlap bool
	}{
		{
			name:    "overlapping ranges",
			a:       MatchInt{IntMin: 0, IntMax: 15},
			b:       MatchInt{IntMin: 10, IntMax: 20},
			overlap: true,
		},
		{
			name:    "identical ranges",
			a:       MatchInt{IntMin: 0, IntMax: 15},
			b:       MatchInt{IntMin: 0, IntMax: 15},
			overlap: true,
		},
		{
			name:    "adjacent but touching",
			a:       MatchInt{IntMin: 0, IntMax: 10},
			b:       MatchInt{IntMin: 10, IntMax: 20},
			overlap: true,
		},
		{
			name:    "disjoint ranges",
			a:       MatchInt{IntMin: 0, IntMax: 5},
			b:       MatchInt{IntMin: 10, IntMax: 20},
			overlap: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.overlap, tt.a.CouldOverlapWith(tt.b))
			assert.Equal(t, tt.overlap, tt.b.CouldOverlapWith(tt.a),
				"must be symmetric")
		})
	}
}

func TestMatchString_CouldOverlapWith_MatchInt(t *testing.T) {
	t.Run("string that is a number in range", func(t *testing.T) {
		ms := MatchString{Str: "5"}
		mi := MatchInt{IntMin: 0, IntMax: 10}
		assert.True(t, ms.CouldOverlapWith(mi))
		assert.True(t, mi.CouldOverlapWith(ms))
	})

	t.Run("string that is not a number", func(t *testing.T) {
		ms := MatchString{Str: "combat"}
		mi := MatchInt{IntMin: 0, IntMax: 10}
		assert.False(t, ms.CouldOverlapWith(mi))
		assert.False(t, mi.CouldOverlapWith(ms))
	})
}

func TestMatchPlayerCharacter_CouldOverlapWith_Anything(t *testing.T) {
	pc := MatchPlayerCharacter{Description: "target"}

	t.Run("overlaps with MatchString", func(t *testing.T) {
		assert.True(t, pc.CouldOverlapWith(MatchString{Str: "anything"}))
	})

	t.Run("overlaps with MatchStringList", func(t *testing.T) {
		assert.True(t, pc.CouldOverlapWith(MatchStringList{Strings: []string{"x"}}))
	})

	t.Run("overlaps with MatchInt", func(t *testing.T) {
		assert.True(t, pc.CouldOverlapWith(MatchInt{IntMin: 0, IntMax: 6}))
	})

	t.Run("overlaps with another MatchPlayerCharacter", func(t *testing.T) {
		assert.True(t, pc.CouldOverlapWith(MatchPlayerCharacter{}))
	})
}

// --- ValidateNoAmbiguousCommands tests ---

func TestValidateNoAmbiguousCommands_NoOverlap(t *testing.T) {
	cmds := TextCommands{
		// 3-slot command: "combat list brit"
		*NewTextCommand([]Match{
			MatchString{Str: "combat"},
			MatchString{Str: "list"},
			MatchString{Str: "brit"},
		}, nil),
		// 3-slot command: "combat list dungeons" — different at position 2
		*NewTextCommand([]Match{
			MatchString{Str: "combat"},
			MatchString{Str: "list"},
			MatchString{Str: "dungeons"},
		}, nil),
	}

	// Should not panic
	assert.NotPanics(t, func() {
		cmds.ValidateNoAmbiguousCommands()
	})
}

func TestValidateNoAmbiguousCommands_DifferentSlotCount(t *testing.T) {
	cmds := TextCommands{
		// 2-slot command
		*NewTextCommand([]Match{
			MatchString{Str: "combat"},
			MatchString{Str: "list"},
		}, nil),
		// 3-slot command — different slot count, never compared
		*NewTextCommand([]Match{
			MatchString{Str: "combat"},
			MatchString{Str: "list"},
			MatchString{Str: "brit"},
		}, nil),
	}

	assert.NotPanics(t, func() {
		cmds.ValidateNoAmbiguousCommands()
	})
}

func TestValidateNoAmbiguousCommands_DetectsIdenticalCommands(t *testing.T) {
	cmds := TextCommands{
		*NewTextCommand([]Match{
			MatchString{Str: "combat"},
			MatchString{Str: "list"},
		}, nil),
		*NewTextCommand([]Match{
			MatchString{Str: "combat"},
			MatchString{Str: "list"},
		}, nil),
	}

	assert.Panics(t, func() {
		cmds.ValidateNoAmbiguousCommands()
	})
}

func TestValidateNoAmbiguousCommands_DetectsOverlappingMatchTypes(t *testing.T) {
	// Same slot count, all positions overlap:
	// Position 0: "combat" == "combat" ✓ overlap
	// Position 1: "brit" == "brit" ✓ overlap
	// Position 2: MatchInt{0-15} == MatchInt{0-15} ✓ overlap
	// Position 3: MatchStringList{dirs} vs MatchStringList{dirs} ✓ overlap
	cmds := TextCommands{
		*NewTextCommand([]Match{
			MatchString{Str: "combat"},
			MatchString{Str: "brit"},
			MatchInt{IntMin: 0, IntMax: 15},
			MatchStringList{Strings: []string{"north", "south", "east", "west"}},
		}, nil),
		*NewTextCommand([]Match{
			MatchString{Str: "combat"},
			MatchString{Str: "brit"},
			MatchInt{IntMin: 0, IntMax: 15},
			MatchStringList{Strings: []string{"north", "south", "east", "west"}},
		}, nil),
	}

	assert.Panics(t, func() {
		cmds.ValidateNoAmbiguousCommands()
	})
}

func TestValidateNoAmbiguousCommands_NoOverlap_IntVsStringList(t *testing.T) {
	// Position 2 differs: MatchInt vs MatchStringList{directions}
	// No direction name parses as an integer, so no overlap at position 2
	cmds := TextCommands{
		*NewTextCommand([]Match{
			MatchString{Str: "combat"},
			MatchString{Str: "dungeon"},
			MatchInt{IntMin: 0, IntMax: 6},
		}, nil),
		*NewTextCommand([]Match{
			MatchString{Str: "combat"},
			MatchString{Str: "dungeon"},
			MatchStringList{Strings: []string{"deceit", "despise", "shame"}},
		}, nil),
	}

	assert.NotPanics(t, func() {
		cmds.ValidateNoAmbiguousCommands()
	})
}

func TestValidateNoAmbiguousCommands_EmptyCommands(t *testing.T) {
	cmds := TextCommands{}

	assert.NotPanics(t, func() {
		cmds.ValidateNoAmbiguousCommands()
	})
}

func TestValidateNoAmbiguousCommands_SingleCommand(t *testing.T) {
	cmds := TextCommands{
		*NewTextCommand([]Match{
			MatchString{Str: "help"},
		}, nil),
	}

	assert.NotPanics(t, func() {
		cmds.ValidateNoAmbiguousCommands()
	})
}

func TestMatchString_CouldOverlapWith_MatchStringList(t *testing.T) {
	tests := []struct {
		name    string
		str     MatchString
		list    MatchStringList
		overlap bool
	}{
		{
			name:    "string shares first char with list item",
			str:     MatchString{Str: "south"},
			list:    MatchStringList{Strings: []string{"skeleton", "slime", "dragon"}},
			overlap: true, // "south" and "skeleton" share 's'
		},
		{
			name:    "string is prefix of list item",
			str:     MatchString{Str: "nor"},
			list:    MatchStringList{Strings: []string{"north", "south"}},
			overlap: true, // "nor" and "north" share 'n'
		},
		{
			name:    "no shared first char",
			str:     MatchString{Str: "west"},
			list:    MatchStringList{Strings: []string{"north", "south", "east"}},
			overlap: false, // 'w' not in {'n', 's', 'e'}
		},
		{
			name:    "case insensitive first char",
			str:     MatchString{Str: "EAST"},
			list:    MatchStringList{Strings: []string{"ettin", "dragon"}},
			overlap: true, // 'E' and 'e' match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.overlap, tt.str.CouldOverlapWith(tt.list),
				"MatchString.CouldOverlapWith(MatchStringList)")
			assert.Equal(t, tt.overlap, tt.list.CouldOverlapWith(tt.str),
				"MatchStringList.CouldOverlapWith(MatchString) — must be symmetric")
		})
	}
}
