// Package scenarios defines and persists scripted debug scenarios without
// depending on game or presentation state.
package scenarios

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

const (
	// CurrentSchemaVersion is the only scenario schema understood by this phase.
	CurrentSchemaVersion = 2

	// BaselineCurrent starts from the state active when the scenario is launched.
	BaselineCurrent BaselineType = "current"
	// BaselineInit starts from the game's initial state.
	BaselineInit BaselineType = "init"
	// BaselineSnapshot starts from one reviewed named save snapshot.
	BaselineSnapshot BaselineType = "snapshot"

	// StepCommand invokes one logical debug command.
	StepCommand StepType = "command"
	// StepNote records documentation in the ordered scenario flow.
	StepNote StepType = "note"
)

var (
	// ErrInvalidDefinition indicates a malformed or unsupported scenario definition.
	ErrInvalidDefinition = errors.New("invalid scenario definition")
	// ErrInvalidParameters indicates invalid launch parameter values.
	ErrInvalidParameters = errors.New("invalid scenario parameters")

	slugPattern      = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	commandIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*(\.[a-z][a-z0-9-]*)+$`)
)

// BaselineType identifies how the scenario obtains its starting state.
type BaselineType string

// Baseline identifies the state from which a scenario starts.
type Baseline struct {
	Type      BaselineType `json:"type"`
	Reference string       `json:"reference,omitempty"`
}

// StepType identifies one supported ordered scenario operation.
type StepType string

// Definition is the versioned serialized form of a scripted scenario.
type Definition struct {
	SchemaVersion int                             `json:"schemaVersion"`
	ID            string                          `json:"id"`
	Name          string                          `json:"name"`
	Description   string                          `json:"description"`
	Baseline      Baseline                        `json:"baseline"`
	Parameters    []debugging.ParameterDefinition `json:"parameters,omitempty"`
	Steps         []Step                          `json:"steps"`
}

// Step is one command or note in scenario order. Fields not belonging to its
// selected type are rejected by Validate.
type Step struct {
	Type      StepType                   `json:"type"`
	CommandID string                     `json:"commandId,omitempty"`
	Arguments map[string]CommandArgument `json:"arguments,omitempty"`
	Note      string                     `json:"note,omitempty"`
}

// CommandArgument is either a JSON literal or an explicit launch parameter reference.
// References serialize as {"parameter":"parameter-id"}; literals serialize directly.
type CommandArgument struct {
	Literal   json.RawMessage `json:"-"`
	Parameter string          `json:"-"`
}

// Invocation is a command step with every parameter reference replaced by a literal.
type Invocation struct {
	CommandID string                     `json:"commandId"`
	Arguments map[string]json.RawMessage `json:"arguments,omitempty"`
}

// BoundStep preserves scenario order while exposing concrete command invocations.
type BoundStep struct {
	Type       StepType
	Invocation *Invocation
	Note       string
}

// MarshalJSON preserves literal JSON types and gives references an explicit shape.
func (argument CommandArgument) MarshalJSON() ([]byte, error) {
	if argument.Parameter != "" {
		if len(argument.Literal) != 0 {
			return nil, fmt.Errorf("%w: command argument has both literal and parameter", ErrInvalidDefinition)
		}

		return json.Marshal(struct {
			Parameter string `json:"parameter"`
		}{Parameter: argument.Parameter})
	}
	if !json.Valid(argument.Literal) {
		return nil, fmt.Errorf("%w: command argument literal is not valid JSON", ErrInvalidDefinition)
	}

	return append([]byte(nil), argument.Literal...), nil
}

// UnmarshalJSON decodes the reserved one-field parameter reference shape or a literal.
func (argument *CommandArgument) UnmarshalJSON(data []byte) error {
	var object map[string]json.RawMessage
	if json.Unmarshal(data, &object) == nil && len(object) == 1 {
		if raw, ok := object["parameter"]; ok {
			var parameter string
			if err := json.Unmarshal(raw, &parameter); err != nil || parameter == "" {
				return fmt.Errorf("%w: parameter reference requires a non-empty string", ErrInvalidDefinition)
			}
			argument.Literal = nil
			argument.Parameter = parameter

			return nil
		}
	}
	if !json.Valid(data) {
		return fmt.Errorf("%w: command argument literal is not valid JSON", ErrInvalidDefinition)
	}
	argument.Literal = append(argument.Literal[:0], data...)
	argument.Parameter = ""

	return nil
}

// DecodeDefinition strictly decodes one definition, including rejection of unknown fields.
func DecodeDefinition(data []byte) (Definition, error) {
	var definition Definition
	if err := repositoryfs.DecodeStrictJSON(data, &definition); err != nil {
		return Definition{}, fmt.Errorf("%w: decode JSON: %w", ErrInvalidDefinition, err)
	}
	if err := definition.Validate(); err != nil {
		return Definition{}, err
	}

	return definition, nil
}

// Validate checks the complete static scenario schema.
func (definition Definition) Validate() error {
	if definition.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("%w: unsupported schema version %d", ErrInvalidDefinition, definition.SchemaVersion)
	}
	if !slugPattern.MatchString(definition.ID) {
		return fmt.Errorf("%w: ID %q is not a slug", ErrInvalidDefinition, definition.ID)
	}
	if strings.TrimSpace(definition.Name) == "" || strings.TrimSpace(definition.Description) == "" {
		return fmt.Errorf("%w: name and description are required", ErrInvalidDefinition)
	}
	if err := definition.Baseline.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidDefinition, err)
	}

	parameters := make(map[string]debugging.ParameterDefinition, len(definition.Parameters))
	for _, parameter := range definition.Parameters {
		if _, exists := parameters[parameter.ID]; exists {
			return fmt.Errorf("%w: duplicate parameter %q", ErrInvalidDefinition, parameter.ID)
		}
		if err := validateParameterDefinition(parameter); err != nil {
			return err
		}
		parameters[parameter.ID] = parameter
	}
	if len(definition.Steps) == 0 {
		return fmt.Errorf("%w: at least one step is required", ErrInvalidDefinition)
	}
	for index, step := range definition.Steps {
		if err := validateStep(step, parameters); err != nil {
			return fmt.Errorf("%w: step %d: %w", ErrInvalidDefinition, index+1, err)
		}
	}

	return nil
}

// Validate checks the discriminated baseline object independently of repository state.
func (baseline Baseline) Validate() error {
	switch baseline.Type {
	case BaselineCurrent, BaselineInit:
		if baseline.Reference != "" {
			return fmt.Errorf("baseline type %q forbids reference", baseline.Type)
		}
	case BaselineSnapshot:
		if baseline.Reference == "" {
			return errors.New("snapshot baseline requires reference")
		}
		reference, err := save_snapshot.ParseReference(baseline.Reference)
		if err != nil || reference.Source == save_snapshot.SourceRollback {
			return errors.New("snapshot baseline reference must be project:<id> or personal:<id>")
		}
	default:
		return fmt.Errorf("unsupported baseline type %q", baseline.Type)
	}

	return nil
}

// BindWithValues also returns the detached launch values actually used, including
// defaults and values not referenced by a step. An empty map is known, not unknown.
func (definition Definition) BindWithValues(values map[string]json.RawMessage) ([]BoundStep, map[string]json.RawMessage, error) {
	if err := definition.Validate(); err != nil {
		return nil, nil, err
	}
	parameters := make(map[string]debugging.ParameterDefinition, len(definition.Parameters))
	boundValues := make(map[string]json.RawMessage, len(definition.Parameters))
	for _, parameter := range definition.Parameters {
		parameters[parameter.ID] = parameter
	}
	for id := range values {
		if _, exists := parameters[id]; !exists {
			return nil, nil, fmt.Errorf("%w: unknown parameter %q", ErrInvalidParameters, id)
		}
	}
	for _, parameter := range definition.Parameters {
		value, supplied := values[parameter.ID]
		if !supplied {
			value = parameter.Default
		}
		if len(value) == 0 {
			if parameter.Required {
				return nil, nil, fmt.Errorf("%w: parameter %q is required", ErrInvalidParameters, parameter.ID)
			}

			continue
		}
		if err := validateParameterValue(parameter, value); err != nil {
			return nil, nil, fmt.Errorf("%w: parameter %q: %w", ErrInvalidParameters, parameter.ID, err)
		}
		boundValues[parameter.ID] = append(json.RawMessage(nil), value...)
	}

	steps := make([]BoundStep, 0, len(definition.Steps))
	for _, step := range definition.Steps {
		bound := BoundStep{Type: step.Type, Note: step.Note}
		if step.Type == StepCommand {
			arguments := make(map[string]json.RawMessage, len(step.Arguments))
			for id, argument := range step.Arguments {
				if argument.Parameter != "" {
					value, exists := boundValues[argument.Parameter]
					if !exists {
						return nil, nil, fmt.Errorf("%w: parameter %q has no value", ErrInvalidParameters, argument.Parameter)
					}
					arguments[id] = append(json.RawMessage(nil), value...)
				} else {
					arguments[id] = append(json.RawMessage(nil), argument.Literal...)
				}
			}
			bound.Invocation = &Invocation{CommandID: step.CommandID, Arguments: arguments}
		}
		steps = append(steps, bound)
	}

	return steps, boundValues, nil
}

func validateStep(step Step, parameters map[string]debugging.ParameterDefinition) error {
	switch step.Type {
	case StepCommand:
		if !commandIDPattern.MatchString(step.CommandID) || step.Note != "" {
			return errors.New("command step requires commandId and forbids note")
		}
		for id, argument := range step.Arguments {
			if id == "" {
				return errors.New("command argument ID is required")
			}
			if argument.Parameter != "" {
				if len(argument.Literal) != 0 {
					return fmt.Errorf("argument %q has both a literal and parameter reference", id)
				}
				if _, exists := parameters[argument.Parameter]; !exists {
					return fmt.Errorf("argument %q references missing parameter %q", id, argument.Parameter)
				}
			} else if !json.Valid(argument.Literal) {
				return fmt.Errorf("argument %q has an invalid JSON literal", id)
			}
		}
	case StepNote:
		if strings.TrimSpace(step.Note) == "" || step.CommandID != "" || len(step.Arguments) != 0 {
			return errors.New("note step requires note and forbids command fields")
		}
	default:
		return fmt.Errorf("unsupported step type %q", step.Type)
	}

	return nil
}

func validateParameterDefinition(parameter debugging.ParameterDefinition) error {
	if !slugPattern.MatchString(parameter.ID) || strings.TrimSpace(parameter.Name) == "" {
		return fmt.Errorf("%w: parameter requires a slug ID and name", ErrInvalidDefinition)
	}
	if !supportedParameterKind(parameter.Kind) {
		return fmt.Errorf("%w: parameter %q has unsupported kind %q", ErrInvalidDefinition, parameter.ID, parameter.Kind)
	}
	if parameter.Minimum != nil && parameter.Maximum != nil && *parameter.Minimum > *parameter.Maximum {
		return fmt.Errorf("%w: parameter %q minimum exceeds maximum", ErrInvalidDefinition, parameter.ID)
	}
	if parameter.Pattern != "" {
		if _, err := regexp.Compile(parameter.Pattern); err != nil {
			return fmt.Errorf("%w: parameter %q has invalid pattern: %w", ErrInvalidDefinition, parameter.ID, err)
		}
	}
	choiceValues := make(map[string]struct{}, len(parameter.Choices))
	for _, choice := range parameter.Choices {
		if choice.Value == "" || choice.Label == "" {
			return fmt.Errorf("%w: parameter %q has an incomplete choice", ErrInvalidDefinition, parameter.ID)
		}
		if _, exists := choiceValues[choice.Value]; exists {
			return fmt.Errorf("%w: parameter %q has duplicate choice %q", ErrInvalidDefinition, parameter.ID, choice.Value)
		}
		choiceValues[choice.Value] = struct{}{}
	}
	if len(parameter.Default) > 0 {
		if err := validateParameterValue(parameter, parameter.Default); err != nil {
			return fmt.Errorf("%w: parameter %q has invalid default: %w", ErrInvalidDefinition, parameter.ID, err)
		}
	}

	return nil
}

func supportedParameterKind(kind debugging.ParameterKind) bool {
	switch kind {
	case debugging.ParameterString, debugging.ParameterInteger, debugging.ParameterIntegerList,
		debugging.ParameterBoolean, debugging.ParameterEnum, debugging.ParameterCharacter,
		debugging.ParameterLocation, debugging.ParameterMonster, debugging.ParameterEquipment:
		return true
	default:
		return false
	}
}

func validateParameterValue(parameter debugging.ParameterDefinition, raw json.RawMessage) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return errors.New("null is not allowed")
	}
	switch parameter.Kind {
	case debugging.ParameterInteger:
		return validateInteger(parameter, raw)
	case debugging.ParameterIntegerList:
		var values []int
		if err := json.Unmarshal(raw, &values); err != nil || len(values) == 0 {
			return errors.New("must be a non-empty integer list")
		}
		for _, value := range values {
			encoded, err := json.Marshal(value)
			if err != nil {
				panic("marshal integer: " + err.Error())
			}
			if err = validateInteger(parameter, encoded); err != nil {
				return err
			}
		}
	case debugging.ParameterBoolean:
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return errors.New("must be a boolean")
		}
	case debugging.ParameterString, debugging.ParameterEnum, debugging.ParameterCharacter,
		debugging.ParameterLocation, debugging.ParameterMonster, debugging.ParameterEquipment:
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return errors.New("must be a string")
		}
		if parameter.Pattern != "" && !regexp.MustCompile(parameter.Pattern).MatchString(value) {
			return errors.New("does not match the required pattern")
		}
		if len(parameter.Choices) > 0 {
			matched := false
			for _, choice := range parameter.Choices {
				matched = matched || choice.Value == value
			}
			if !matched {
				return fmt.Errorf("unknown choice %q", value)
			}
		}
	default:
		return fmt.Errorf("unsupported kind %q", parameter.Kind)
	}

	return nil
}

func validateInteger(parameter debugging.ParameterDefinition, raw json.RawMessage) error {
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return errors.New("must be an integer")
	}
	if parameter.Minimum != nil && value < *parameter.Minimum {
		return fmt.Errorf("must be at least %d", *parameter.Minimum)
	}
	if parameter.Maximum != nil && value > *parameter.Maximum {
		return fmt.Errorf("must be at most %d", *parameter.Maximum)
	}

	return nil
}
