package scenarios

import (
	"encoding/json"
	"testing"
)

func TestMigrateLegacyCharacterTokensCopiesOnlyRecognizedLiterals(t *testing.T) {
	t.Parallel()
	definition := validDefinition()
	definition.Steps = []Step{
		legacyCharacterStep(partyCharacterUpdateCommand, `"0.QXZhdGFy"`),
		legacyCharacterStep(partyEquipmentLoadoutUpdateCommand, `"5.SW9sbw"`),
		legacyCharacterStep("party.other.update", `"1.RHVwcmU"`),
	}

	migrated, migrations := MigrateLegacyCharacterTokens(definition)

	if got := string(definition.Steps[0].Arguments["character"].Literal); got != `"0.QXZhdGFy"` {
		t.Fatalf("input character literal mutated to %s", got)
	}
	if got := string(migrated.Steps[0].Arguments["character"].Literal); got != `"1"` {
		t.Fatalf("first migrated literal = %s, want %q", got, "1")
	}
	if got := string(migrated.Steps[1].Arguments["character"].Literal); got != `"6"` {
		t.Fatalf("second migrated literal = %s, want %q", got, "6")
	}
	if got := string(migrated.Steps[2].Arguments["character"].Literal); got != `"1.RHVwcmU"` {
		t.Fatalf("unrelated command literal = %s", got)
	}
	if len(migrations) != 2 {
		t.Fatalf("migrations = %#v, want 2", migrations)
	}
	want := Migration{Path: "$.steps[0].arguments.character", Code: MigrationLegacyCharacterToken, Before: "0.QXZhdGFy", After: "1"}
	if migrations[0] != want {
		t.Fatalf("first migration = %#v, want %#v", migrations[0], want)
	}
}

func TestMigrateLegacyCharacterTokensRejectsNearMatches(t *testing.T) {
	t.Parallel()
	tests := []string{
		`"6.QXZhdGFy"`, `"00.QXZhdGFy"`, `"0."`, `"0.not+url"`, `"0.A"`, `"1"`, `1`,
	}
	for _, literal := range tests {
		definition := validDefinition()
		definition.Steps = []Step{legacyCharacterStep(partyCharacterUpdateCommand, literal)}

		migrated, migrations := MigrateLegacyCharacterTokens(definition)

		if len(migrations) != 0 {
			t.Fatalf("literal %s migrations = %#v", literal, migrations)
		}
		if got := string(migrated.Steps[0].Arguments["character"].Literal); got != literal {
			t.Fatalf("literal %s changed to %s", literal, got)
		}
	}
}

func TestMigrateLegacyCharacterTokensMapsEverySupportedSlot(t *testing.T) {
	t.Parallel()
	for slot := range 6 {
		before := string(rune('0'+slot)) + ".QXZhdGFy"
		raw, err := json.Marshal(before)
		if err != nil {
			t.Fatalf("Marshal(%q) error = %v", before, err)
		}
		gotBefore, gotAfter, ok := migrateLegacyCharacterLiteral(raw)
		if !ok || gotBefore != before || gotAfter != string(rune('1'+slot)) {
			t.Fatalf("slot %d migration = (%q, %q, %t)", slot, gotBefore, gotAfter, ok)
		}
	}
}

func TestMigrateRuntimeWorldPreconditionsCopiesAndRemovesOnlyManagedArguments(t *testing.T) {
	t.Parallel()
	definition := validDefinition()
	definition.Steps = []Step{
		{
			Type: StepCommand, CommandID: worldPositionSetCommand,
			Arguments: map[string]CommandArgument{
				"location":                  {Literal: json.RawMessage(`"33"`)},
				"expectedLocationId":        {Literal: json.RawMessage(`2`)},
				"expectedFloor":             {Literal: json.RawMessage(`0`)},
				"expectedX":                 {Literal: json.RawMessage(`10`)},
				"expectedY":                 {Literal: json.RawMessage(`11`)},
				"expectedSourceFingerprint": {Parameter: "source"},
			},
		},
		{
			Type: StepCommand, CommandID: dungeonExplorerSetupCommand,
			Arguments: map[string]CommandArgument{
				"dungeon":                   {Literal: json.RawMessage(`"33"`)},
				"expectedLocationId":        {Literal: json.RawMessage(`33`)},
				"expectedFloor":             {Literal: json.RawMessage(`1`)},
				"expectedX":                 {Literal: json.RawMessage(`2`)},
				"expectedY":                 {Literal: json.RawMessage(`3`)},
				"expectedSourceFingerprint": {Literal: json.RawMessage(`"review"`)},
				"expectedDungeonRevision":   {Literal: json.RawMessage(`7`)},
			},
		},
		{
			Type: StepCommand, CommandID: "world.other",
			Arguments: map[string]CommandArgument{"expectedX": {Literal: json.RawMessage(`4`)}},
		},
	}

	migrated, migrations := MigrateRuntimeWorldPreconditions(definition)

	for stepIndex, argumentIDs := range [][]string{
		{"expectedLocationId", "expectedFloor", "expectedX", "expectedY", "expectedSourceFingerprint"},
		{"expectedLocationId", "expectedFloor", "expectedX", "expectedY", "expectedSourceFingerprint", "expectedDungeonRevision"},
	} {
		for _, argumentID := range argumentIDs {
			if _, exists := migrated.Steps[stepIndex].Arguments[argumentID]; exists {
				t.Fatalf("migrated step %d retained %s", stepIndex, argumentID)
			}
		}
	}
	if got := string(migrated.Steps[0].Arguments["location"].Literal); got != `"33"` {
		t.Fatalf("destination location = %s", got)
	}
	if got := string(migrated.Steps[2].Arguments["expectedX"].Literal); got != `4` {
		t.Fatalf("unrelated command expectedX = %s", got)
	}
	if got := string(definition.Steps[0].Arguments["expectedLocationId"].Literal); got != `2` {
		t.Fatalf("input definition mutated: expectedLocationId = %s", got)
	}
	if len(migrations) != 11 {
		t.Fatalf("migrations = %#v, want 11", migrations)
	}
	wantPaths := []string{
		"$.steps[0].arguments.expectedLocationId", "$.steps[0].arguments.expectedFloor",
		"$.steps[0].arguments.expectedX", "$.steps[0].arguments.expectedY",
		"$.steps[0].arguments.expectedSourceFingerprint",
		"$.steps[1].arguments.expectedLocationId", "$.steps[1].arguments.expectedFloor",
		"$.steps[1].arguments.expectedX", "$.steps[1].arguments.expectedY",
		"$.steps[1].arguments.expectedSourceFingerprint", "$.steps[1].arguments.expectedDungeonRevision",
	}
	for index, wantPath := range wantPaths {
		if migrations[index].Path != wantPath || migrations[index].Code != MigrationRuntimeWorldPrecondition ||
			migrations[index].After != "resolved at runtime" {
			t.Fatalf("migration %d = %#v, want path %q", index, migrations[index], wantPath)
		}
	}
	if migrations[4].Before != `{"parameter":"source"}` {
		t.Fatalf("parameter-reference migration before = %q", migrations[4].Before)
	}
}

func legacyCharacterStep(commandID, literal string) Step {
	return Step{
		Type: StepCommand, CommandID: commandID,
		Arguments: map[string]CommandArgument{"character": {Literal: json.RawMessage(literal)}},
	}
}
