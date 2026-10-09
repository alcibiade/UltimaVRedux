package scenarios

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
)

func TestBindWithValuesRetainsResolvedLaunchInputs(t *testing.T) {
	definition := Definition{
		SchemaVersion: CurrentSchemaVersion, ID: "resolved", Name: "Resolved", Description: "Retain launch inputs.",
		Baseline: CurrentBaseline(),
		Parameters: []debugging.ParameterDefinition{
			{ID: "required", Name: "Required", Kind: debugging.ParameterInteger, Required: true},
			{ID: "flag", Name: "Flag", Kind: debugging.ParameterBoolean, Default: json.RawMessage(`false`)},
			{ID: "empty", Name: "Empty", Kind: debugging.ParameterString, Default: json.RawMessage(`""`)},
			{ID: "absent", Name: "Absent", Kind: debugging.ParameterString},
		},
		Steps: []Step{{Type: StepCommand, CommandID: "test.command", Arguments: map[string]CommandArgument{"value": {Parameter: "required"}}}},
	}
	_, _, err := definition.BindWithValues(nil)
	require.ErrorIs(t, err, ErrInvalidParameters)
	input := map[string]json.RawMessage{"required": json.RawMessage(`0`)}
	steps, values, err := definition.BindWithValues(input)
	require.NoError(t, err)
	encoded, err := json.Marshal(values)
	require.NoError(t, err)
	assert.JSONEq(t, `{"required":0,"flag":false,"empty":""}`, string(encoded))
	input["required"][0] = '1'
	steps[0].Invocation.Arguments["value"][0] = '2'
	definition.Parameters[1].Default[0] = 't'
	assert.Equal(t, "0", string(values["required"]))
	assert.Equal(t, "false", string(values["flag"]))
	definition.Parameters = nil
	definition.Steps = []Step{{Type: StepNote, Note: "No launch inputs."}}
	_, values, err = definition.BindWithValues(nil)
	require.NoError(t, err)
	require.NotNil(t, values)
	assert.Empty(t, values)
}

func TestDefinitionDecodeValidateAndBind(t *testing.T) {
	t.Parallel()
	data := []byte(`{
  "schemaVersion": 2,
  "id": "starvation-check",
  "name": "Starvation check",
  "description": "Runs a seeded starvation command.",
  "baseline": {"type":"init"},
  "parameters": [
    {"id":"seed","name":"Seed","kind":"integer","required":true,"minimum":0,"maximum":10},
    {"id":"enabled","name":"Enabled","kind":"boolean","required":false,"default":true}
  ],
  "steps": [
    {"type":"note","note":"Set up deterministic state."},
    {"type":"command","commandId":"scenario.starvation.trigger","arguments":{"seed":{"parameter":"seed"},"enabled":{"parameter":"enabled"},"label":"fixed"}}
  ]
}`)
	definition, err := DecodeDefinition(data)
	if err != nil {
		t.Fatalf("DecodeDefinition() error = %v", err)
	}
	steps, err := definition.Bind(map[string]json.RawMessage{"seed": json.RawMessage(`7`)})
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if len(steps) != 2 || steps[0].Type != StepNote || steps[0].Note == "" {
		t.Fatalf("Bind() note step = %#v", steps)
	}
	invocation := steps[1].Invocation
	if invocation == nil || invocation.CommandID != "scenario.starvation.trigger" {
		t.Fatalf("Bind() invocation = %#v", invocation)
	}
	assertJSONEqual(t, invocation.Arguments["seed"], `7`)
	assertJSONEqual(t, invocation.Arguments["enabled"], `true`)
	assertJSONEqual(t, invocation.Arguments["label"], `"fixed"`)

	// Bound arguments must not alias caller-owned buffers.
	seed := json.RawMessage(`8`)
	bound, err := definition.Bind(map[string]json.RawMessage{"seed": seed})
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	seed[0] = '9'
	assertJSONEqual(t, bound[1].Invocation.Arguments["seed"], `8`)
}

func TestDefinitionValidationRejectsMalformedDefinitions(t *testing.T) {
	t.Parallel()
	minimum, maximum := 0, 10
	valid := validDefinition()
	tests := []struct {
		name   string
		mutate func(*Definition)
	}{
		{name: "schema", mutate: func(d *Definition) { d.SchemaVersion = 1 }},
		{name: "scenario slug", mutate: func(d *Definition) { d.ID = "../escape" }},
		{name: "metadata", mutate: func(d *Definition) { d.Description = "" }},
		{name: "baseline", mutate: func(d *Definition) { d.Baseline = Baseline{Type: "save"} }},
		{name: "current reference", mutate: func(d *Definition) { d.Baseline.Reference = "project:save" }},
		{name: "snapshot missing reference", mutate: func(d *Definition) { d.Baseline = Baseline{Type: BaselineSnapshot} }},
		{name: "snapshot rollback reference", mutate: func(d *Definition) { d.Baseline = SnapshotBaseline("rollback:save") }},
		{name: "no steps", mutate: func(d *Definition) { d.Steps = nil }},
		{name: "duplicate parameter", mutate: func(d *Definition) { d.Parameters = append(d.Parameters, d.Parameters[0]) }},
		{name: "parameter slug", mutate: func(d *Definition) { d.Parameters[0].ID = "bad_id" }},
		{name: "unsupported kind", mutate: func(d *Definition) { d.Parameters[0].Kind = "number" }},
		{name: "reversed bounds", mutate: func(d *Definition) { d.Parameters[0].Minimum, d.Parameters[0].Maximum = &maximum, &minimum }},
		{name: "invalid default type", mutate: func(d *Definition) { d.Parameters[0].Default = json.RawMessage(`"x"`) }},
		{name: "missing parameter reference", mutate: func(d *Definition) { d.Steps[0].Arguments["seed"] = CommandArgument{Parameter: "missing"} }},
		{name: "command missing ID", mutate: func(d *Definition) { d.Steps[0].CommandID = "" }},
		{name: "command with note", mutate: func(d *Definition) { d.Steps[0].Note = "invalid" }},
		{name: "note with command", mutate: func(d *Definition) { d.Steps = []Step{{Type: StepNote, Note: "note", CommandID: "x"}} }},
		{name: "unsupported step", mutate: func(d *Definition) { d.Steps[0].Type = "wait" }},
		{name: "invalid literal", mutate: func(d *Definition) { d.Steps[0].Arguments["seed"] = CommandArgument{Literal: json.RawMessage(`{`)} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition := valid
			definition.Parameters = append([]debugging.ParameterDefinition(nil), valid.Parameters...)
			definition.Steps = cloneSteps(valid.Steps)
			test.mutate(&definition)
			if err := definition.Validate(); !errors.Is(err, ErrInvalidDefinition) {
				t.Fatalf("Validate() error = %v, want ErrInvalidDefinition", err)
			}
		})
	}
}

func TestDecodeDefinitionRejectsUnknownAndTrailingJSON(t *testing.T) {
	t.Parallel()
	for _, data := range []string{
		`{"schemaVersion":2,"id":"valid-id","name":"Name","description":"Description","baseline":{"type":"current"},"steps":[{"type":"note","note":"hello"}],"unknown":true}`,
		`{"schemaVersion":2,"id":"valid-id","name":"Name","description":"Description","baseline":{"type":"current"},"steps":[{"type":"note","note":"hello"}]} {}`,
		`{"schemaVersion":2,"id":"valid-id","name":"Name","description":"Description","baseline":{"type":"current"},"steps":[{"type":"note","note":"hello"}]} {`,
		`{"schemaVersion":2,"id":"valid-id","name":"Name","description":"Description","baseline":{"type":"current","unknown":true},"steps":[{"type":"note","note":"hello"}]}`,
		`{"schemaVersion":1,"id":"valid-id","name":"Name","description":"Description","baseline":"current","steps":[{"type":"note","note":"hello"}]}`,
	} {
		if _, err := DecodeDefinition([]byte(data)); !errors.Is(err, ErrInvalidDefinition) {
			t.Fatalf("DecodeDefinition(%q) error = %v", data, err)
		}
	}
}

func TestBindSupportsAllParameterKindsAndConstraints(t *testing.T) {
	t.Parallel()
	minimum, maximum := 1, 3
	definition := Definition{
		SchemaVersion: CurrentSchemaVersion,
		ID:            "parameter-kinds",
		Name:          "Parameter kinds",
		Description:   "Exercises every catalog parameter kind.",
		Baseline:      CurrentBaseline(),
		Parameters: []debugging.ParameterDefinition{
			{ID: "text", Name: "Text", Kind: debugging.ParameterString, Required: true, Pattern: `^[a-z]+$`},
			{ID: "count", Name: "Count", Kind: debugging.ParameterInteger, Required: true, Minimum: &minimum, Maximum: &maximum},
			{ID: "counts", Name: "Counts", Kind: debugging.ParameterIntegerList, Required: true, Minimum: &minimum, Maximum: &maximum},
			{ID: "flag", Name: "Flag", Kind: debugging.ParameterBoolean, Required: true},
			{ID: "mode", Name: "Mode", Kind: debugging.ParameterEnum, Required: true, Choices: []debugging.Choice{{Value: "safe", Label: "Safe"}}},
			{ID: "character", Name: "Character", Kind: debugging.ParameterCharacter, Required: true},
			{ID: "location", Name: "Location", Kind: debugging.ParameterLocation, Required: true},
			{ID: "monster", Name: "Monster", Kind: debugging.ParameterMonster, Required: true},
			{ID: "equipment", Name: "Equipment", Kind: debugging.ParameterEquipment, Required: true},
		},
		Steps: []Step{{Type: StepNote, Note: "Parameters are launch-only in this test."}},
	}
	values := map[string]json.RawMessage{
		"text": json.RawMessage(`"valid"`), "count": json.RawMessage(`2`), "counts": json.RawMessage(`[1,3]`),
		"flag": json.RawMessage(`true`), "mode": json.RawMessage(`"safe"`), "character": json.RawMessage(`"avatar"`),
		"location": json.RawMessage(`"britain"`), "monster": json.RawMessage(`"orc"`), "equipment": json.RawMessage(`"sword"`),
	}
	if _, err := definition.Bind(values); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}

	values["mode"] = json.RawMessage(`"unsafe"`)
	if _, err := definition.Bind(values); !errors.Is(err, ErrInvalidParameters) {
		t.Fatalf("Bind(invalid enum) error = %v", err)
	}
}

func TestBindRejectsMissingUnknownAndInvalidValues(t *testing.T) {
	t.Parallel()
	definition := validDefinition()
	tests := []map[string]json.RawMessage{
		nil,
		{"seed": json.RawMessage(`11`)},
		{"seed": json.RawMessage(`true`)},
		{"seed": json.RawMessage(`1`), "unknown": json.RawMessage(`2`)},
	}
	for _, values := range tests {
		if _, err := definition.Bind(values); !errors.Is(err, ErrInvalidParameters) {
			t.Fatalf("Bind(%v) error = %v, want ErrInvalidParameters", values, err)
		}
	}
}

func validDefinition() Definition {
	minimum, maximum := 0, 10
	return Definition{
		SchemaVersion: CurrentSchemaVersion,
		ID:            "valid-id",
		Name:          "Valid",
		Description:   "Valid definition.",
		Baseline:      CurrentBaseline(),
		Parameters: []debugging.ParameterDefinition{{
			ID: "seed", Name: "Seed", Kind: debugging.ParameterInteger, Required: true, Minimum: &minimum, Maximum: &maximum,
		}},
		Steps: []Step{{
			Type: StepCommand, CommandID: "scenario.test", Arguments: map[string]CommandArgument{"seed": {Parameter: "seed"}},
		}},
	}
}

func cloneSteps(steps []Step) []Step {
	cloned := make([]Step, len(steps))
	for index, step := range steps {
		cloned[index] = step
		cloned[index].Arguments = make(map[string]CommandArgument, len(step.Arguments))
		for id, argument := range step.Arguments {
			cloned[index].Arguments[id] = argument
		}
	}
	return cloned
}

func assertJSONEqual(t *testing.T, actual json.RawMessage, expected string) {
	t.Helper()
	var actualValue any
	var expectedValue any
	if err := json.Unmarshal(actual, &actualValue); err != nil {
		t.Fatalf("unmarshal actual JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(expected), &expectedValue); err != nil {
		t.Fatalf("unmarshal expected JSON: %v", err)
	}
	if actualText, expectedText := mustJSON(t, actualValue), mustJSON(t, expectedValue); actualText != expectedText {
		t.Fatalf("JSON = %s, want %s", actualText, expectedText)
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}
	return string(data)
}
