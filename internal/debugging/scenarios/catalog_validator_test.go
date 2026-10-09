package scenarios

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
)

func TestValidateAgainstCatalogAcceptsCompatibleDefinition(t *testing.T) {
	t.Parallel()
	catalog := newValidatorTestCatalog(t)
	definition := catalogDefinition()

	issues := ValidateAgainstCatalog(definition, catalog, CatalogValidationPolicy{
		AllowedExecutionModes: []debugging.ExecutionMode{debugging.ExecutionSynchronized},
	})

	if len(issues) != 0 {
		t.Fatalf("ValidateAgainstCatalog() issues = %#v", issues)
	}
}

func TestValidateAgainstCatalogRejectsStepsAfterTerminalCommand(t *testing.T) {
	t.Parallel()
	definition := catalogDefinition()
	definition.Steps = []Step{
		{Type: StepCommand, CommandID: "game.terminal"},
		Step{Type: StepNote, Note: "unreachable"},
		{Type: StepCommand, CommandID: "game.query"},
	}

	issues := ValidateAgainstCatalog(definition, newValidatorTestCatalog(t), CatalogValidationPolicy{
		AllowedExecutionModes: []debugging.ExecutionMode{debugging.ExecutionQuery},
	})

	assertIssueCodes(t, issues, IssueStepAfterTerminal, IssueStepAfterTerminal)
	if issues[0].Path != "steps[1]" || issues[1].Path != "steps[2]" {
		t.Fatalf("terminal issue paths = %q, %q", issues[0].Path, issues[1].Path)
	}
}

func TestValidateAgainstCatalogReportsStableCommandAndArgumentIssues(t *testing.T) {
	t.Parallel()
	catalog := newValidatorTestCatalog(t)
	definition := catalogDefinition()
	definition.Steps = []Step{
		{Type: StepCommand, CommandID: "missing.command"},
		{Type: StepCommand, CommandID: "game.query"},
		{
			Type:      StepCommand,
			CommandID: "game.change",
			Arguments: map[string]CommandArgument{
				"unknown": {Literal: json.RawMessage(`true`)},
			},
		},
	}

	issues := ValidateAgainstCatalog(definition, catalog, CatalogValidationPolicy{
		AllowedExecutionModes: []debugging.ExecutionMode{debugging.ExecutionSynchronized},
		ManagementCommandIDs:  []string{"game.query"},
	})

	want := []CatalogValidationIssue{
		{Path: "steps[0].commandId", Code: IssueCommandNotFound, Message: `command "missing.command" does not exist`},
		{Path: "steps[1].commandId", Code: IssueManagementCommand, Message: `command "game.query" is reserved for scenario management`},
		{Path: "steps[1].commandId", Code: IssueExecutionMode, Message: `command "game.query" has no allowed execution mode`},
		{Path: "steps[2].arguments.count", Code: IssueArgumentRequired, Message: `required argument "count" is missing`},
		{Path: "steps[2].arguments.mode", Code: IssueArgumentRequired, Message: `required argument "mode" is missing`},
		{Path: "steps[2].arguments.unknown", Code: IssueArgumentNotFound, Message: `command "game.change" has no argument "unknown"`},
	}
	if !reflect.DeepEqual(issues, want) {
		t.Fatalf("ValidateAgainstCatalog() issues = %#v, want %#v", issues, want)
	}
}

func TestValidateAgainstCatalogSupportsManagementPredicateAndAnyAllowedMode(t *testing.T) {
	t.Parallel()
	catalog := newValidatorTestCatalog(t)
	definition := catalogDefinition()
	definition.Steps[0].CommandID = "game.multi-mode"
	definition.Steps[0].Arguments = nil

	issues := ValidateAgainstCatalog(definition, catalog, CatalogValidationPolicy{
		AllowedExecutionModes: []debugging.ExecutionMode{debugging.ExecutionSynchronized},
		IsManagementCommand: func(commandID string) bool {
			return strings.HasSuffix(commandID, "multi-mode")
		},
	})

	assertIssueCodes(t, issues, IssueManagementCommand)
}

func TestValidateAgainstCatalogRejectsScenarioExcludedCommand(t *testing.T) {
	t.Parallel()
	catalog, err := debugging.NewCatalog([]debugging.CommandDefinition{{
		ID: "game.preference", Category: "Test", Name: "Preference", Description: "Persist preference.",
		ExecutionModes: []debugging.ExecutionMode{debugging.ExecutionLiveSafe}, ScenarioExcluded: true,
	}})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	definition := catalogDefinition()
	definition.Steps = []Step{{Type: StepCommand, CommandID: "game.preference"}}

	issues := ValidateAgainstCatalog(definition, catalog, CatalogValidationPolicy{
		AllowedExecutionModes: []debugging.ExecutionMode{debugging.ExecutionLiveSafe},
	})

	assertIssueCodes(t, issues, IssueScenarioExcluded)
}

func TestValidateAgainstCatalogRejectsRunCancellationStep(t *testing.T) {
	definition := validDefinition()
	definition.Steps = []Step{{Type: StepCommand, CommandID: debugging.CommandScenarioRunCancel,
		Arguments: map[string]CommandArgument{"runId": {Literal: json.RawMessage(`"run-1"`)}}}}
	issues := ValidateAgainstCatalog(definition, debugging.NewInitialCatalog(), CatalogValidationPolicy{})
	assertIssueCodes(t, issues, IssueScenarioExcluded)
}

func TestValidateAgainstCatalogUsesCatalogNormalizationForLiterals(t *testing.T) {
	t.Parallel()
	catalog := newValidatorTestCatalog(t)
	definition := catalogDefinition()
	definition.Steps[0].Arguments["count"] = CommandArgument{Literal: json.RawMessage(`9`)}
	definition.Steps[0].Arguments["mode"] = CommandArgument{Literal: json.RawMessage(`"unknown"`)}
	definition.Steps[0].Arguments["enabled"] = CommandArgument{Literal: json.RawMessage(`"yes"`)}

	issues := ValidateAgainstCatalog(definition, catalog, CatalogValidationPolicy{})

	assertIssueCodes(t, issues, IssueLiteralInvalid, IssueLiteralInvalid, IssueLiteralInvalid)
	for _, issue := range issues {
		if !strings.Contains(issue.Message, "invalid debug command request") {
			t.Errorf("issue message %q does not come from catalog normalization", issue.Message)
		}
	}
}

func TestValidateAgainstCatalogAllowsOmittedArgumentWithCatalogDefault(t *testing.T) {
	t.Parallel()
	catalog := newValidatorTestCatalog(t)
	definition := catalogDefinition()
	delete(definition.Steps[0].Arguments, "enabled")

	issues := ValidateAgainstCatalog(definition, catalog, CatalogValidationPolicy{})
	if len(issues) != 0 {
		t.Fatalf("ValidateAgainstCatalog() defaulted argument issues = %#v", issues)
	}
}

func TestValidateAgainstCatalogRequiresRuntimeManagedArgumentsToRemainOmitted(t *testing.T) {
	t.Parallel()
	catalog := newValidatorTestCatalog(t)
	definition := catalogDefinition()

	issues := ValidateAgainstCatalog(definition, catalog, CatalogValidationPolicy{})
	if len(issues) != 0 {
		t.Fatalf("ValidateAgainstCatalog() omitted runtime argument issues = %#v", issues)
	}

	definition.Steps[0].Arguments["expectedFingerprint"] = CommandArgument{Literal: json.RawMessage(`"stale"`)}
	issues = ValidateAgainstCatalog(definition, catalog, CatalogValidationPolicy{})
	assertIssueCodes(t, issues, IssueArgumentRuntimeManaged)
}

func TestValidateAgainstCatalogDefersDynamicOptionMembership(t *testing.T) {
	t.Parallel()
	catalog := newValidatorTestCatalog(t)
	definition := catalogDefinition()
	definition.Steps[0].Arguments["liveTarget"] = CommandArgument{Literal: json.RawMessage(`"future-party-member"`)}

	issues := ValidateAgainstCatalog(definition, catalog, CatalogValidationPolicy{})
	if len(issues) != 0 {
		t.Fatalf("ValidateAgainstCatalog() dynamic option issues = %#v", issues)
	}

	definition.Steps[0].Arguments["liveTarget"] = CommandArgument{Literal: json.RawMessage(`7`)}
	issues = ValidateAgainstCatalog(definition, catalog, CatalogValidationPolicy{})
	assertIssueCodes(t, issues, IssueLiteralInvalid)
}

func TestValidateAgainstCatalogChecksParameterReferences(t *testing.T) {
	t.Parallel()
	catalog := newValidatorTestCatalog(t)
	definition := catalogDefinition()
	definition.Parameters = append(definition.Parameters,
		debugging.ParameterDefinition{ID: "optional", Name: "Optional", Kind: debugging.ParameterBoolean},
		debugging.ParameterDefinition{ID: "wrong-kind", Name: "Wrong kind", Kind: debugging.ParameterString, Required: true},
	)
	definition.Steps[0].Arguments["enabled"] = CommandArgument{Parameter: "optional"}
	definition.Steps[0].Arguments["count"] = CommandArgument{Parameter: "wrong-kind"}
	definition.Steps[0].Arguments["mode"] = CommandArgument{Parameter: "missing"}

	issues := ValidateAgainstCatalog(definition, catalog, CatalogValidationPolicy{})

	assertIssueCodes(t, issues, IssueParameterKind, IssueParameterValueRequired, IssueParameterNotFound)
	if issues[0].Path != "steps[0].arguments.count" || issues[1].Path != "steps[0].arguments.enabled" ||
		issues[2].Path != "steps[0].arguments.mode" {
		t.Fatalf("issue paths = %#v", issues)
	}
}

func TestValidateAgainstCatalogChecksReferenceConstraintsAndDefaults(t *testing.T) {
	t.Parallel()
	catalog := newValidatorTestCatalog(t)
	definition := catalogDefinition()
	definition.Parameters[0].Minimum = nil
	definition.Parameters[0].Maximum = intPointer(8)
	definition.Parameters[1].Choices = []debugging.Choice{{Value: "safe", Label: "Safe"}, {Value: "other", Label: "Other"}}
	definition.Parameters[1].Default = json.RawMessage(`"other"`)

	issues := ValidateAgainstCatalog(definition, catalog, CatalogValidationPolicy{})

	assertIssueCodes(t, issues, IssueParameterConstraint, IssueParameterConstraint, IssueParameterDefaultInvalid)
	if issues[0].Path != "steps[0].arguments.count" || issues[1].Path != "steps[0].arguments.mode" {
		t.Fatalf("constraint issue paths = %#v", issues)
	}
}

func TestValidateAgainstCatalogDoesNotMutateDefinition(t *testing.T) {
	t.Parallel()
	catalog := newValidatorTestCatalog(t)
	definition := catalogDefinition()
	before, err := json.Marshal(definition)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	_ = ValidateAgainstCatalog(definition, catalog, CatalogValidationPolicy{})

	after, err := json.Marshal(definition)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("definition mutated:\nbefore %s\nafter  %s", before, after)
	}
}

func TestValidateAgainstCatalogUsesScenarioChoicesInsteadOfLiveCharacterTokens(t *testing.T) {
	t.Parallel()
	catalog := newValidatorTestCatalog(t)
	definition := catalogDefinition()
	definition.Steps[0].Arguments["target"] = CommandArgument{Literal: json.RawMessage(`"2"`)}

	issues := ValidateAgainstCatalog(definition, catalog, CatalogValidationPolicy{})

	if len(issues) != 0 {
		t.Fatalf("ValidateAgainstCatalog() generic position issues = %#v", issues)
	}
	definition.Steps[0].Arguments["target"] = CommandArgument{Literal: json.RawMessage(`""`)}
	assertIssueCodes(t, ValidateAgainstCatalog(definition, catalog, CatalogValidationPolicy{}), IssueLiteralInvalid)
	definition.Steps[0].Arguments["target"] = CommandArgument{Literal: json.RawMessage(`"1.QXZhdGFy"`)}
	assertIssueCodes(t, ValidateAgainstCatalog(definition, catalog, CatalogValidationPolicy{}), IssueLiteralInvalid)
}

func TestValidateAgainstCatalogRequiresCatalog(t *testing.T) {
	t.Parallel()
	issues := ValidateAgainstCatalog(catalogDefinition(), nil, CatalogValidationPolicy{})
	assertIssueCodes(t, issues, IssueCatalogRequired)
	if issues[0].Path != "$" {
		t.Fatalf("catalog issue path = %q", issues[0].Path)
	}
}

func newValidatorTestCatalog(t *testing.T) *debugging.Catalog {
	t.Helper()
	minimum, maximum := 1, 5
	catalog, err := debugging.NewCatalog([]debugging.CommandDefinition{
		{
			ID: "game.change", Category: "Game", Name: "Change", Description: "Changes test state.",
			ExecutionModes: []debugging.ExecutionMode{debugging.ExecutionSynchronized},
			Parameters: []debugging.ParameterDefinition{
				{ID: "count", Name: "Count", Kind: debugging.ParameterInteger, Required: true, Minimum: &minimum, Maximum: &maximum},
				{ID: "mode", Name: "Mode", Kind: debugging.ParameterEnum, Required: true, Choices: []debugging.Choice{{Value: "safe", Label: "Safe"}, {Value: "fast", Label: "Fast"}}},
				{
					ID: "enabled", Name: "Enabled", Kind: debugging.ParameterBoolean,
					Required: true, Default: json.RawMessage(`true`),
				},
				{
					ID: "target", Name: "Target", Kind: debugging.ParameterCharacter, OptionsProvider: "party",
					Choices:               []debugging.Choice{{Value: "avatar", Label: "Avatar"}},
					ScenarioChoices:       []debugging.Choice{{Value: "1", Label: "Party Position 1"}, {Value: "2", Label: "Party Position 2"}},
					ScenarioValueResolver: debugging.ScenarioValueResolverActivePartyPosition,
				},
				{ID: "liveTarget", Name: "Live Target", Kind: debugging.ParameterCharacter, OptionsProvider: "party"},
				{ID: "expectedFingerprint", Name: "Expected Fingerprint", Kind: debugging.ParameterString, Required: true, ScenarioRuntime: true},
			},
		},
		{ID: "game.query", Category: "Game", Name: "Query", Description: "Queries test state.", ExecutionModes: []debugging.ExecutionMode{debugging.ExecutionQuery}},
		{
			ID: "game.terminal", Category: "Game", Name: "Terminal", Description: "Terminates a test scenario.",
			ExecutionModes: []debugging.ExecutionMode{debugging.ExecutionQuery}, ScenarioTerminal: true,
		},
		{ID: "game.multi-mode", Category: "Game", Name: "Multi mode", Description: "Uses multiple modes.", ExecutionModes: []debugging.ExecutionMode{debugging.ExecutionQuery, debugging.ExecutionSynchronized}},
	})
	if err != nil {
		t.Fatalf("debugging.NewCatalog() error = %v", err)
	}

	return catalog
}

func catalogDefinition() Definition {
	minimum, maximum := 1, 5
	return Definition{
		SchemaVersion: CurrentSchemaVersion,
		ID:            "catalog-validation",
		Name:          "Catalog validation",
		Description:   "Validates a scenario against a command catalog.",
		Baseline:      CurrentBaseline(),
		Parameters: []debugging.ParameterDefinition{
			{ID: "count", Name: "Count", Kind: debugging.ParameterInteger, Required: true, Minimum: &minimum, Maximum: &maximum},
			{ID: "mode", Name: "Mode", Kind: debugging.ParameterEnum, Required: true, Choices: []debugging.Choice{{Value: "safe", Label: "Safe"}}},
		},
		Steps: []Step{{
			Type:      StepCommand,
			CommandID: "game.change",
			Arguments: map[string]CommandArgument{
				"count":   {Parameter: "count"},
				"mode":    {Parameter: "mode"},
				"enabled": {Literal: json.RawMessage(`true`)},
				"target":  {Literal: json.RawMessage(`"1"`)},
			},
		}},
	}
}

func assertIssueCodes(t *testing.T, issues []CatalogValidationIssue, want ...string) {
	t.Helper()
	got := make([]string, len(issues))
	for index, issue := range issues {
		got[index] = issue.Code
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issue codes = %v, want %v; issues = %#v", got, want, issues)
	}
}

func intPointer(value int) *int {
	return &value
}
