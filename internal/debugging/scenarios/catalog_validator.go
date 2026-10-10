package scenarios

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
)

// Stable catalog validation issue codes.
const (
	IssueDefinitionInvalid       = "definition.invalid"
	IssueCatalogRequired         = "catalog.required"
	IssueCommandNotFound         = "command.not_found"
	IssueManagementCommand       = "command.management_excluded"
	IssueScenarioExcluded        = "command.scenario_excluded"
	IssueExecutionMode           = "command.execution_mode_not_allowed"
	IssueArgumentNotFound        = "argument.not_found"
	IssueArgumentRequired        = "argument.required"
	IssueArgumentRuntimeManaged  = "argument.runtime_managed"
	IssueLiteralInvalid          = "argument.literal_invalid"
	IssueParameterNotFound       = "parameter.not_found"
	IssueParameterValueRequired  = "parameter.value_required"
	IssueParameterKind           = "parameter.kind_incompatible"
	IssueParameterConstraint     = "parameter.constraint_incompatible"
	IssueParameterDefaultInvalid = "parameter.default_invalid"
	IssueStepAfterTerminal       = "step.after_terminal_command"
)

// CatalogValidationIssue describes one deterministic catalog compatibility problem.
type CatalogValidationIssue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CatalogValidationPolicy defines which commands a scenario may invoke. Management
// commands can be supplied as stable IDs, a predicate, or both.
type CatalogValidationPolicy struct {
	AllowedExecutionModes []debugging.ExecutionMode
	ManagementCommandIDs  []string
	IsManagementCommand   func(commandID string) bool
}

// ValidateAgainstCatalog checks a structurally valid scenario against an injected
// command catalog without changing either input.
func ValidateAgainstCatalog(
	definition Definition,
	catalog *debugging.Catalog,
	policy CatalogValidationPolicy,
) []CatalogValidationIssue {
	if catalog == nil {
		return []CatalogValidationIssue{{Path: "$", Code: IssueCatalogRequired, Message: "command catalog is required"}}
	}

	parameters := make(map[string]debugging.ParameterDefinition, len(definition.Parameters))
	for _, parameter := range definition.Parameters {
		parameters[parameter.ID] = parameter
	}
	allowedModes := make(map[debugging.ExecutionMode]struct{}, len(policy.AllowedExecutionModes))
	for _, mode := range policy.AllowedExecutionModes {
		allowedModes[mode] = struct{}{}
	}
	managementCommands := make(map[string]struct{}, len(policy.ManagementCommandIDs))
	for _, commandID := range policy.ManagementCommandIDs {
		managementCommands[commandID] = struct{}{}
	}
	issues := make([]CatalogValidationIssue, 0)
	terminalStep := -1
	terminalCommandID := ""
	for stepIndex, step := range definition.Steps {
		if terminalStep >= 0 {
			issues = append(issues, CatalogValidationIssue{
				Path:    fmt.Sprintf("steps[%d]", stepIndex),
				Code:    IssueStepAfterTerminal,
				Message: fmt.Sprintf("step follows terminal command %q at steps[%d]", terminalCommandID, terminalStep),
			})

			continue
		}
		if step.Type != StepCommand {
			continue
		}

		issues = append(issues, validateCatalogStep(
			stepIndex,
			step,
			catalog,
			parameters,
			allowedModes,
			managementCommands,
			policy.IsManagementCommand,
		)...)
		command, exists := catalog.Get(step.CommandID)
		if exists && command.ScenarioTerminal {
			terminalStep = stepIndex
			terminalCommandID = step.CommandID
		}
	}

	return issues
}

func validateCatalogStep(
	stepIndex int,
	step Step,
	catalog *debugging.Catalog,
	scenarioParameters map[string]debugging.ParameterDefinition,
	allowedModes map[debugging.ExecutionMode]struct{},
	managementCommands map[string]struct{},
	managementPredicate func(string) bool,
) []CatalogValidationIssue {
	stepPath := fmt.Sprintf("steps[%d]", stepIndex)
	command, exists := catalog.Get(step.CommandID)
	if !exists {
		return []CatalogValidationIssue{{
			Path: stepPath + ".commandId", Code: IssueCommandNotFound,
			Message: fmt.Sprintf("command %q does not exist", step.CommandID),
		}}
	}

	issues := validateCommandPolicy(
		stepPath,
		command,
		allowedModes,
		managementCommands,
		managementPredicate,
	)
	commandParameters, requiredIssues := indexCommandParameters(stepPath, command.Parameters, step.Arguments)
	issues = append(issues, requiredIssues...)

	argumentIDs := make([]string, 0, len(step.Arguments))
	for argumentID := range step.Arguments {
		argumentIDs = append(argumentIDs, argumentID)
	}
	sort.Strings(argumentIDs)
	for _, argumentID := range argumentIDs {
		issues = append(issues, validateCatalogArgument(
			stepPath,
			step.CommandID,
			argumentID,
			step.Arguments[argumentID],
			commandParameters,
			scenarioParameters,
		)...)
	}

	return issues
}

func validateCommandPolicy(
	stepPath string,
	command debugging.CommandDefinition,
	allowedModes map[debugging.ExecutionMode]struct{},
	managementCommands map[string]struct{},
	managementPredicate func(string) bool,
) []CatalogValidationIssue {
	issues := make([]CatalogValidationIssue, 0)
	if isManagementCommand(command.ID, managementCommands, managementPredicate) {
		issues = append(issues, CatalogValidationIssue{
			Path: stepPath + ".commandId", Code: IssueManagementCommand,
			Message: fmt.Sprintf("command %q is reserved for scenario management", command.ID),
		})
	}
	if command.ScenarioExcluded {
		issues = append(issues, CatalogValidationIssue{
			Path: stepPath + ".commandId", Code: IssueScenarioExcluded,
			Message: fmt.Sprintf("command %q is excluded from scenarios", command.ID),
		})
	}
	if len(allowedModes) > 0 && !HasAllowedExecutionMode(command.ExecutionModes, allowedModes) {
		issues = append(issues, CatalogValidationIssue{
			Path: stepPath + ".commandId", Code: IssueExecutionMode,
			Message: fmt.Sprintf("command %q has no allowed execution mode", command.ID),
		})
	}

	return issues
}

func indexCommandParameters(
	stepPath string,
	parameters []debugging.ParameterDefinition,
	arguments map[string]CommandArgument,
) (map[string]debugging.ParameterDefinition, []CatalogValidationIssue) {
	indexed := make(map[string]debugging.ParameterDefinition, len(parameters))
	issues := make([]CatalogValidationIssue, 0)
	for _, parameter := range parameters {
		indexed[parameter.ID] = parameter
		if _, supplied := arguments[parameter.ID]; !supplied && parameter.Required && len(parameter.Default) == 0 && !parameter.ScenarioRuntime {
			issues = append(issues, CatalogValidationIssue{
				Path: stepPath + ".arguments." + parameter.ID, Code: IssueArgumentRequired,
				Message: fmt.Sprintf("required argument %q is missing", parameter.ID),
			})
		}
	}

	return indexed, issues
}

func validateCatalogArgument(
	stepPath string,
	commandID string,
	argumentID string,
	argument CommandArgument,
	commandParameters map[string]debugging.ParameterDefinition,
	scenarioParameters map[string]debugging.ParameterDefinition,
) []CatalogValidationIssue {
	argumentPath := stepPath + ".arguments." + argumentID
	commandParameter, exists := commandParameters[argumentID]
	if !exists {
		return []CatalogValidationIssue{{
			Path: argumentPath, Code: IssueArgumentNotFound,
			Message: fmt.Sprintf("command %q has no argument %q", commandID, argumentID),
		}}
	}
	if commandParameter.ScenarioRuntime {
		return []CatalogValidationIssue{{
			Path: argumentPath, Code: IssueArgumentRuntimeManaged,
			Message: fmt.Sprintf("command argument %q is resolved from live state at scenario runtime", argumentID),
		}}
	}
	if argument.Parameter == "" {
		if err := normalizeCatalogValue(effectiveScenarioParameter(commandParameter), argument.Literal); err != nil {
			return []CatalogValidationIssue{{
				Path: argumentPath, Code: IssueLiteralInvalid, Message: err.Error(),
			}}
		}

		return nil
	}

	scenarioParameter, exists := scenarioParameters[argument.Parameter]
	if !exists {
		return []CatalogValidationIssue{{
			Path: argumentPath, Code: IssueParameterNotFound,
			Message: fmt.Sprintf("scenario parameter %q does not exist", argument.Parameter),
		}}
	}

	return validateParameterReference(argumentPath, scenarioParameter, effectiveScenarioParameter(commandParameter))
}

func effectiveScenarioParameter(parameter debugging.ParameterDefinition) debugging.ParameterDefinition {
	if len(parameter.ScenarioChoices) == 0 {
		return parameter
	}
	parameter.Choices = parameter.ScenarioChoices
	parameter.OptionsProvider = ""

	return parameter
}

func isManagementCommand(commandID string, commandIDs map[string]struct{}, predicate func(string) bool) bool {
	if _, exists := commandIDs[commandID]; exists {
		return true
	}

	return predicate != nil && predicate(commandID)
}

// HasAllowedExecutionMode reports whether a command advertises at least one mode
// accepted by the scenario policy.
func HasAllowedExecutionMode(modes []debugging.ExecutionMode, allowed map[debugging.ExecutionMode]struct{}) bool {
	for _, mode := range modes {
		if _, exists := allowed[mode]; exists {
			return true
		}
	}

	return false
}

func validateParameterReference(
	path string,
	source debugging.ParameterDefinition,
	target debugging.ParameterDefinition,
) []CatalogValidationIssue {
	issues := make([]CatalogValidationIssue, 0)
	if !source.Required && len(source.Default) == 0 {
		issues = append(issues, CatalogValidationIssue{
			Path: path, Code: IssueParameterValueRequired,
			Message: fmt.Sprintf("referenced parameter %q must be required or have a default", source.ID),
		})
	}
	if source.Kind != target.Kind {
		return append(issues, CatalogValidationIssue{
			Path: path, Code: IssueParameterKind,
			Message: fmt.Sprintf(
				"parameter %q kind %q is incompatible with command kind %q",
				source.ID,
				source.Kind,
				target.Kind,
			),
		})
	}
	if reason := incompatibleConstraints(source, target); reason != "" {
		issues = append(issues, CatalogValidationIssue{Path: path, Code: IssueParameterConstraint, Message: reason})
	}
	if len(source.Default) > 0 {
		if err := normalizeCatalogValue(target, source.Default); err != nil {
			issues = append(issues, CatalogValidationIssue{
				Path: path, Code: IssueParameterDefaultInvalid,
				Message: fmt.Sprintf("parameter %q default is invalid for the command argument: %v", source.ID, err),
			})
		}
	}

	return issues
}

func incompatibleConstraints(source, target debugging.ParameterDefinition) string {
	if reason := incompatibleNumericConstraints(source, target); reason != "" {
		return reason
	}
	if target.OptionsProvider != "" {
		return ""
	}
	if reason := incompatibleChoiceConstraints(source, target); reason != "" {
		return reason
	}

	return incompatiblePatternConstraints(source, target)
}

func incompatibleNumericConstraints(source, target debugging.ParameterDefinition) string {
	if target.Minimum != nil && (source.Minimum == nil || *source.Minimum < *target.Minimum) {
		return fmt.Sprintf("parameter %q permits values below command minimum %d", source.ID, *target.Minimum)
	}
	if target.Maximum != nil && (source.Maximum == nil || *source.Maximum > *target.Maximum) {
		return fmt.Sprintf("parameter %q permits values above command maximum %d", source.ID, *target.Maximum)
	}

	return ""
}

func incompatibleChoiceConstraints(source, target debugging.ParameterDefinition) string {
	if len(target.Choices) == 0 {
		return ""
	}
	if len(source.Choices) == 0 {
		return fmt.Sprintf("parameter %q does not restrict values to the command choices", source.ID)
	}
	for _, choice := range source.Choices {
		if !choiceExists(target.Choices, choice.Value) {
			return fmt.Sprintf("parameter %q choice %q is not accepted by the command", source.ID, choice.Value)
		}
	}

	return ""
}

func incompatiblePatternConstraints(source, target debugging.ParameterDefinition) string {
	if target.Pattern == "" {
		return ""
	}
	if len(source.Choices) == 0 {
		if source.Pattern == "" {
			return fmt.Sprintf("parameter %q does not enforce the command pattern", source.ID)
		}

		return ""
	}
	for _, choice := range source.Choices {
		raw, err := json.Marshal(choice.Value)
		if err != nil {
			panic("marshal scenario parameter choice: " + err.Error())
		}
		if err = normalizeCatalogValue(target, raw); err != nil {
			return fmt.Sprintf("parameter %q choice %q does not satisfy command constraints", source.ID, choice.Value)
		}
	}

	return ""
}

func choiceExists(choices []debugging.Choice, value string) bool {
	for _, choice := range choices {
		if choice.Value == value {
			return true
		}
	}

	return false
}

// normalizeCatalogValue delegates literal validation to Catalog.NormalizeRequest.
// Dynamic options are update-thread snapshots, so only their static constraints apply here.
func normalizeCatalogValue(parameter debugging.ParameterDefinition, value json.RawMessage) error {
	if parameter.OptionsProvider != "" {
		parameter.Choices = nil
	}
	parameter.Required = true
	parameter.Default = nil
	valueCatalog, err := debugging.NewCatalog([]debugging.CommandDefinition{{
		ID:             "scenario.value.validate",
		Category:       "Scenarios",
		Name:           "Validate scenario value",
		Description:    "Validates one scenario command argument.",
		Aliases:        nil,
		SearchTerms:    nil,
		Parameters:     []debugging.ParameterDefinition{parameter},
		ExecutionModes: []debugging.ExecutionMode{debugging.ExecutionQuery},
		Destructive:    false,
	}})
	if err != nil {
		panic("construct catalog value validator: " + err.Error())
	}
	_, err = valueCatalog.NormalizeRequest(debugging.Request{
		SessionID:               "scenario-validator",
		ID:                      "validate-value",
		CommandID:               "scenario.value.validate",
		ExpectedSceneGeneration: 0,
		ExpectedOptionsRevision: 0,
		Arguments:               map[string]json.RawMessage{parameter.ID: value},
	})

	if err != nil {
		return fmt.Errorf("normalize catalog value: %w", err)
	}

	return nil
}
