package scenarios

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
)

const (
	// MigrationLegacyCharacterToken identifies a position migration from an opaque active-character token.
	MigrationLegacyCharacterToken MigrationCode = "legacy_character_token"
	// MigrationRuntimeWorldPrecondition identifies removal of a reviewed value now resolved from live state.
	MigrationRuntimeWorldPrecondition MigrationCode = "runtime_world_precondition"

	partyCharacterUpdateCommand        = "party.character.update"
	partyEquipmentLoadoutUpdateCommand = "party.equipment.loadout.update"
	worldPositionSetCommand            = "world.position.set"
)

// MigrationCode identifies one load-only scenario definition migration.
type MigrationCode string

// Migration describes one in-memory change requiring explicit persistence.
type Migration struct {
	Path   string        `json:"path"`
	Code   MigrationCode `json:"code"`
	Before string        `json:"before"`
	After  string        `json:"after"`
}

// MigrateDefinition applies every load-only migration without changing the stored source.
func MigrateDefinition(definition Definition) (Definition, []Migration) {
	migrated, migrations := MigrateLegacyCharacterTokens(definition)
	migrated, worldMigrations := MigrateRuntimeWorldPreconditions(migrated)

	return migrated, append(migrations, worldMigrations...)
}

// MigrateLegacyCharacterTokens returns a copy with legacy character identity
// literals replaced by their one-based active-party positions.
func MigrateLegacyCharacterTokens(definition Definition) (Definition, []Migration) {
	migrated := definition
	var migrations []Migration
	for stepIndex, step := range definition.Steps {
		if step.Type != StepCommand || (step.CommandID != partyCharacterUpdateCommand && step.CommandID != partyEquipmentLoadoutUpdateCommand) {
			continue
		}
		argument, exists := step.Arguments["character"]
		if !exists || argument.Parameter != "" {
			continue
		}
		before, after, ok := migrateLegacyCharacterLiteral(argument.Literal)
		if !ok {
			continue
		}
		if migrations == nil {
			migrated.Steps = append([]Step(nil), definition.Steps...)
		}
		migratedStep := step
		migratedStep.Arguments = make(map[string]CommandArgument, len(step.Arguments))
		for id, value := range step.Arguments {
			migratedStep.Arguments[id] = value
		}
		migratedStep.Arguments["character"] = CommandArgument{Literal: json.RawMessage(fmt.Sprintf("%q", after))}
		migrated.Steps[stepIndex] = migratedStep
		migrations = append(migrations, Migration{
			Path: fmt.Sprintf("$.steps[%d].arguments.character", stepIndex), Code: MigrationLegacyCharacterToken,
			Before: before, After: after,
		})
	}

	return migrated, migrations
}

// MigrateRuntimeWorldPreconditions removes persisted reviewed-state arguments
// that Scenario Lab now resolves from authoritative live state before execution.
func MigrateRuntimeWorldPreconditions(definition Definition) (Definition, []Migration) {
	migrated := definition
	var migrations []Migration
	for stepIndex, step := range definition.Steps {
		argumentIDs := runtimeWorldArgumentIDs(step.CommandID)
		if step.Type != StepCommand || len(argumentIDs) == 0 {
			continue
		}
		migratedStep := step
		stepCopied := false
		for _, argumentID := range argumentIDs {
			argument, exists := step.Arguments[argumentID]
			if !exists {
				continue
			}
			if migrations == nil {
				migrated.Steps = append([]Step(nil), definition.Steps...)
			}
			if !stepCopied {
				migratedStep.Arguments = make(map[string]CommandArgument, len(step.Arguments))
				for id, value := range step.Arguments {
					migratedStep.Arguments[id] = value
				}
				stepCopied = true
			}
			delete(migratedStep.Arguments, argumentID)
			payload, err := json.Marshal(argument)
			if err != nil {
				panic(fmt.Sprintf("encoding migrated scenario argument %q: %v", argumentID, err))
			}
			migrations = append(migrations, Migration{
				Path: fmt.Sprintf("$.steps[%d].arguments.%s", stepIndex, argumentID),
				Code: MigrationRuntimeWorldPrecondition, Before: string(payload), After: "resolved at runtime",
			})
		}
		if stepCopied {
			migrated.Steps[stepIndex] = migratedStep
		}
	}

	return migrated, migrations
}

func runtimeWorldArgumentIDs(commandID string) []string {
	switch commandID {
	case worldPositionSetCommand:
		return []string{"expectedLocationId", "expectedFloor", "expectedX", "expectedY", "expectedSourceFingerprint"}
	default:
		return nil
	}
}

func migrateLegacyCharacterLiteral(raw json.RawMessage) (string, string, bool) {
	var value string
	if json.Unmarshal(raw, &value) != nil || len(value) < 3 || value[1] != '.' || value[0] < '0' || value[0] > '5' {
		return "", "", false
	}
	suffix := value[2:]
	if suffix == "" {
		return "", "", false
	}
	if _, err := base64.RawURLEncoding.DecodeString(suffix); err != nil {
		return "", "", false
	}

	return value, strconv.Itoa(int(value[0]-'0') + 1), true
}
