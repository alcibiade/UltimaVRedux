import { choicesForParameter, validateParameter, type OptionSets } from "./command_forms"
import type { CommandDefinition, CommandInvocation, CommandResult, ParameterDefinition, ParameterKind } from "./protocol"

export type ScenarioSource = "project" | "personal"
export type ScenarioBaselineType = "current" | "init" | "snapshot"
export interface ScenarioBaseline { type: ScenarioBaselineType; reference?: string }
export type ScenarioTask = "library" | "quick-setups" | "latest-run" | "details" | "editor" | "run-review"
export type ScenarioParameterKind = Extract<ParameterKind, "string" | "integer" | "boolean">

export interface ScenarioParameter extends ParameterDefinition {
  kind: ScenarioParameterKind
}

export interface ScenarioParameterReference {
  parameter: string
}

export interface ScenarioCommandStep {
  type: "command"
  commandId: string
  arguments?: Record<string, unknown | ScenarioParameterReference>
}

export interface ScenarioNoteStep {
  type: "note"
  note: string
}

export type ScenarioStep = ScenarioCommandStep | ScenarioNoteStep

export interface ScenarioDefinition {
  schemaVersion: 2
  id: string
  name: string
  description: string
  baseline: ScenarioBaseline
  parameters?: ParameterDefinition[]
  steps: ScenarioStep[]
}

export type ScenarioMigrationCode = "legacy_character_token" | "runtime_world_precondition"

export interface ScenarioMigration {
  path: string
  code: ScenarioMigrationCode
  before: string
  after: string
}

export interface ScenarioComposerDraft {
  source: ScenarioSource
  savedReference?: string
  loadedRevision?: string
  originalDefinition: ScenarioDefinition
  definition: ScenarioDefinition
  launchValues: Record<string, unknown>
  migrations: ScenarioMigration[]
}

const scenarioManagementCommandIDs = new Set([
  "scenario.definition.list",
  "scenario.definition.get",
  "scenario.definition.validate",
  "scenario.definition.create",
  "scenario.definition.update",
  "scenario.definition.delete",
  "scenario.definition.duplicate",
  "scenario.run.start",
  "scenario.run.status",
  "scenario.run.cancel",
  "save.snapshot.list",
  "save.snapshot.get",
  "save.snapshot.capture",
  "save.snapshot.load",
  "save.snapshot.delete",
])

const scenarioExecutionModes = new Set([
  "live_safe",
  "synchronized_live",
  "workflow",
  "coordinated_alteration",
  "scene_restart",
])

export interface ScenarioDefinitionSummary {
  reference: string
  source: ScenarioSource
  id: string
  name: string
  description: string
  baseline: ScenarioBaseline
  revision: string
  parameters?: ParameterDefinition[]
  steps: ScenarioStep[]
  migrations?: ScenarioMigration[]
}

export interface ScenarioStepDisplay {
  commandId?: string
  commandName: string
  detail: string
  fullDetail: string
  unknownCommand: boolean
}

export interface ScenarioDefinitionListResult {
  definitions: ScenarioDefinitionSummary[]
}

export interface ScenarioDefinitionGetResult {
  reference: string
  revision: string
  definition: ScenarioDefinition
  migrations?: ScenarioMigration[]
}

export interface ScenarioDefinitionSaveResult {
  reference: string
  definition: ScenarioDefinitionSummary
}

export interface ScenarioValidationIssue {
  path: string
  code: string
  message: string
}

export interface ScenarioDefinitionValidationResult {
  valid: boolean
  issues: ScenarioValidationIssue[]
}

export type ScenarioRunState = "queued" | "running" | "succeeded" | "failed" | "cancelled"

export interface ScenarioRunStepStatus {
  number: number
  type: "command" | "note"
  commandId?: string
  note?: string
  state: "pending" | "succeeded" | "failed" | "skipped"
  result?: CommandResult
}

export interface ScenarioRunStatus {
  runId: string
  reference: string
  definitionId: string
  definitionRevision: string
  baseline: ScenarioBaseline
  baselineReference?: string
  baselineRevision?: string
  rollback?: { reference: string; revision: string }
  launch?: {
    parameterValues: Record<string, unknown>
    baselineApplied: boolean
    rng?: { seed: string; sourceCalls: string }
  }
  state: ScenarioRunState
  startedAt: string
  finishedAt?: string
  currentProcess: boolean
  nextStep: number
  totalSteps: number
  initialSceneGeneration: number
  sceneGeneration: number
  steps: ScenarioRunStepStatus[]
  errorCode?: string
  errorMessage?: string
}

export interface ScenarioRunStatusResult {
  run: ScenarioRunStatus | null
  runs: ScenarioRunStatus[]
  capacity: number
  historyError?: string
}

export function newScenarioComposerDraft(): ScenarioComposerDraft {
  const definition: ScenarioDefinition = {
    schemaVersion: 2,
    id: "new-scenario",
    name: "New Scenario",
    description: "Describe the scripted scenario.",
    baseline: { type: "current" },
    steps: [],
  }
  return {
    source: "personal",
    originalDefinition: cloneJSON(definition),
    definition,
    launchValues: {},
    migrations: [],
  }
}

export function scenarioStepFromInvocation(invocation: CommandInvocation, command?: CommandDefinition): ScenarioCommandStep {
	const parameters = new Map((command?.parameters ?? []).map((parameter) => [parameter.id, parameter]))
	const argumentsByID = Object.fromEntries(Object.entries(cloneRecord(invocation.arguments)).flatMap(([id, value]) => {
		const parameter = parameters.get(id)
		if (parameter?.scenarioRuntime) return []
		if (!parameter?.scenarioValueResolver) return [[id, value]]
		const resolved = capturedScenarioValue(parameter, value)
		return resolved === undefined ? [] : [[id, resolved]]
	}))
	return {
		type: "command",
		commandId: invocation.commandId,
		arguments: argumentsByID,
	}
}

export function scenarioTerminalStepIndex(steps: ScenarioStep[], commands: CommandDefinition[]): number {
  return steps.findIndex((step) => step.type === "command" && commands.find((command) => command.id === step.commandId)?.scenarioTerminal)
}

export function insertScenarioStep(
  steps: ScenarioStep[],
  step: ScenarioStep,
  commands: CommandDefinition[],
  requestedIndex = steps.length,
): { steps: ScenarioStep[]; index: number } {
  const terminalIndex = scenarioTerminalStepIndex(steps, commands)
  const maximumIndex = terminalIndex < 0 ? steps.length : terminalIndex
  const insertingTerminal = step.type === "command" && commands.find((command) => command.id === step.commandId)?.scenarioTerminal
  const index = insertingTerminal ? steps.length : Math.max(0, Math.min(requestedIndex, maximumIndex))
  return { steps: [...steps.slice(0, index), step, ...steps.slice(index)], index }
}

export function hydrateScenarioComposer(data: ScenarioDefinitionGetResult): ScenarioComposerDraft {
  const source = data.reference.startsWith("project:") ? "project" : "personal"
  const definition = cloneJSON(data.definition)
  const migrations = cloneJSON(data.migrations ?? [])
  const originalDefinition = cloneJSON(definition)
  for (const migration of migrations) restoreMigratedValue(originalDefinition, migration)
  return {
    source,
    savedReference: data.reference,
    loadedRevision: data.revision,
    originalDefinition,
    definition,
    launchValues: Object.fromEntries((definition.parameters ?? []).flatMap((parameter) =>
      parameter.default === undefined ? [] : [[parameter.id, cloneJSON(parameter.default)]],
    )),
    migrations,
  }
}

export function scenarioDraftIsDirty(draft: ScenarioComposerDraft): boolean {
  return scenarioDraftHasPendingMigrations(draft)
    || JSON.stringify(draft.definition) !== JSON.stringify(draft.originalDefinition)
}

export function scenarioDraftHasPendingMigrations(draft: ScenarioComposerDraft): boolean {
  return draft.migrations.some((migration) => migratedValue(draft.originalDefinition, migration) === migration.before)
}

export function scenarioIDFromName(name: string): string {
  const slug = name.toLocaleLowerCase().trim().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "")
  if (!slug) return ""
  return /^[a-z]/.test(slug) ? slug : `scenario-${slug}`
}

export function scenarioSourceLabel(source: ScenarioSource): string {
  return source === "project" ? "Project / Shared" : "Personal / Local"
}

export function scenarioBaselineLabel(baseline: ScenarioBaseline): string {
  if (baseline.type === "snapshot") return baseline.reference ?? "Named snapshot"
  return baseline.type === "init" ? "INIT.GAM" : "Current state"
}

export function scenarioStepDisplay(
  step: ScenarioStep,
  scenarioParameters: ParameterDefinition[],
  commands: CommandDefinition[],
  optionSets: OptionSets,
): ScenarioStepDisplay {
  if (step.type === "note") {
    const fullDetail = step.note
    return {
      commandName: "Note",
      detail: truncateScenarioValue(fullDetail),
      fullDetail,
      unknownCommand: false,
    }
  }

  const command = commands.find((candidate) => candidate.id === step.commandId)
  const commandParameters = new Map((command?.parameters ?? []).map((parameter) => [parameter.id, parameter]))
  const launchParameters = new Map(scenarioParameters.map((parameter) => [parameter.id, parameter]))
  const argumentIDs = Object.keys(step.arguments ?? {}).sort((left, right) => {
    const leftIndex = command?.parameters?.findIndex((parameter) => parameter.id === left) ?? -1
    const rightIndex = command?.parameters?.findIndex((parameter) => parameter.id === right) ?? -1
    if (leftIndex === -1 && rightIndex === -1) return left.localeCompare(right)
    if (leftIndex === -1) return 1
    if (rightIndex === -1) return -1
    return leftIndex - rightIndex
  })
  const argumentDetails = argumentIDs.map((argumentID) => {
    const parameter = commandParameters.get(argumentID)
    const label = parameter?.name ?? argumentID
    const value = step.arguments?.[argumentID]
    if (isScenarioParameterReference(value)) {
      const launchParameter = launchParameters.get(value.parameter)
      const detail = `${label}: $${launchParameter?.name ?? value.parameter} [${value.parameter}]`
      return { full: detail, compact: detail }
    }
    const choice = parameter === undefined ? undefined : scenarioParameterChoices(parameter, optionSets)
      .find((candidate) => candidate.value === String(value))
    if (choice) {
      const serialized = formatScenarioLiteral(value)
      const detail = `${label}: ${choice.label}${choice.label === serialized ? "" : ` (${serialized})`}`
      return { full: detail, compact: detail }
    }
    const serialized = formatScenarioLiteral(value)
    return { full: `${label}: ${serialized}`, compact: `${label}: ${truncateScenarioValue(serialized)}` }
  })
  const fullDetail = argumentDetails.map((detail) => detail.full).join(" · ") || "No arguments"
  const detail = argumentDetails.map((argument) => argument.compact).join(" · ") || "No arguments"

  return {
    commandId: step.commandId,
    commandName: command?.name ?? "Unknown command",
    detail,
    fullDetail,
    unknownCommand: command === undefined,
  }
}

export function cloneScenarioDefinition(definition: ScenarioDefinition): ScenarioDefinition {
  return cloneJSON(definition)
}

export function serializeScenarioDefinition(draft: ScenarioComposerDraft): string {
  const definition: ScenarioDefinition = {
    schemaVersion: 2,
    id: draft.definition.id,
    name: draft.definition.name,
    description: draft.definition.description,
    baseline: draft.definition.baseline,
    ...(draft.definition.parameters?.length ? { parameters: draft.definition.parameters } : {}),
    steps: draft.definition.steps,
  }
  return JSON.stringify(definition)
}

export function serializeScenarioLaunchValues(values: Record<string, unknown>): string {
  return JSON.stringify(values)
}

export function scenarioComposerSupportsParameter(kind: ParameterKind): kind is ScenarioParameterKind {
  return kind === "string" || kind === "integer" || kind === "boolean"
}

export function scenarioSafeCommands(commands: CommandDefinition[]): CommandDefinition[] {
  return commands.filter((command) => !scenarioManagementCommandIDs.has(command.id)
		&& !command.scenarioExcluded
    && command.executionModes.length > 0
    && command.executionModes.some((mode) => scenarioExecutionModes.has(mode)))
}

export function scenarioCommandArguments(command: CommandDefinition): Record<string, unknown> | undefined {
	const entries = (command.parameters ?? []).filter((parameter) => !parameter.scenarioRuntime).flatMap((parameter) =>
    parameter.default === undefined ? [] : [[parameter.id, cloneJSON(parameter.default)] as [string, unknown]],
  )
  return entries.length ? Object.fromEntries(entries) : undefined
}

export function compatibleScenarioParameters(
  commandParameter: ParameterDefinition,
  scenarioParameters: ParameterDefinition[],
): ParameterDefinition[] {
  return scenarioParameters.filter((parameter) => scenarioParameterCompatibilityIssue(parameter, commandParameter) === undefined)
}

export function scenarioParameterCompatibilityIssue(
  source: ParameterDefinition,
  target: ParameterDefinition,
): string | undefined {
	target = scenarioPlanningParameter(target)
  if (!source.required && source.default === undefined) return "Launch parameter must be required or have a default."
  if (source.kind !== target.kind) return `Kind ${source.kind} does not match ${target.kind}.`
  if (target.minimum !== undefined && (source.minimum === undefined || source.minimum < target.minimum)) {
    return `Minimum must be at least ${target.minimum}.`
  }
  if (target.maximum !== undefined && (source.maximum === undefined || source.maximum > target.maximum)) {
    return `Maximum must be at most ${target.maximum}.`
  }
  if (!target.optionsProvider && target.choices?.length) {
    const accepted = new Set(target.choices.map((choice) => choice.value))
    if (!source.choices?.length || source.choices.some((choice) => !accepted.has(choice.value))) {
      return "Choices must be restricted to values accepted by the command."
    }
  }
  const targetPattern = target.pattern
  if (targetPattern && !source.pattern && !source.choices?.length) return "A compatible validation pattern is required."
  if (targetPattern && source.choices?.some((choice) => !matchesPattern(choice.value, targetPattern))) {
    return "One or more choices do not match the command pattern."
  }
  if (source.default !== undefined) {
    const defaultIssue = scenarioValueIssue(target, source.default)
    if (defaultIssue) return `Default value: ${defaultIssue}`
  }
  return undefined
}

export function scenarioValueIssue(
  parameter: ParameterDefinition,
  value: unknown,
  optionSets: OptionSets = {},
): string | undefined {
	parameter = scenarioPlanningParameter(parameter)
  const input = value === undefined || value === null
    ? ""
    : parameter.kind === "integer_list" && Array.isArray(value)
      ? value.join(",")
      : String(value)
  return validateParameter(parameter, input, optionSets)
}

export function scenarioLaunchValueIssues(
  parameters: ParameterDefinition[],
  values: Record<string, unknown>,
): Record<string, string> {
  return Object.fromEntries(parameters.flatMap((parameter) => {
    const value = values[parameter.id] ?? parameter.default
    const issue = scenarioValueIssue(parameter, value)
    return issue ? [[parameter.id, issue]] : []
  }))
}

export function scenarioLiteralValue(parameter: ParameterDefinition, raw: string): unknown {
  if (raw === "") return undefined
  if (parameter.kind === "integer") return Number(raw)
  if (parameter.kind === "boolean") return raw === "true"
  if (parameter.kind === "integer_list") {
    const values = raw.split(",").map((value) => Number(value.trim()))
    return values.every(Number.isInteger) ? values : raw
  }
  return raw
}

export function scenarioLiteralDisplay(value: unknown): string {
  if (value === undefined || value === null) return ""
  if (Array.isArray(value)) return value.join(",")
  return String(value)
}

export function scenarioParameterChoices(parameter: ParameterDefinition, optionSets: OptionSets): ReturnType<typeof choicesForParameter> {
  return choicesForParameter(scenarioPlanningParameter(parameter), optionSets)
}

export function scenarioPlanningParameter(parameter: ParameterDefinition): ParameterDefinition {
	if (!parameter.scenarioChoices?.length) return parameter
	return { ...parameter, choices: parameter.scenarioChoices, optionsProvider: undefined }
}

function capturedScenarioValue(parameter: ParameterDefinition, value: unknown): unknown {
	if (parameter.scenarioValueResolver !== "active-party-position" || typeof value !== "string") return undefined
	if (parameter.scenarioChoices?.some((choice) => choice.value === value)) return value
	const opaqueIndex = Number(value.split(".", 1)[0])
	if (!Number.isInteger(opaqueIndex) || opaqueIndex < 0) return undefined
	const position = String(opaqueIndex + 1)
	return parameter.scenarioChoices?.some((choice) => choice.value === position) ? position : undefined
}

function restoreMigratedValue(definition: ScenarioDefinition, migration: ScenarioMigration): void {
  const match = migration.path.match(/^\$\.steps\[(\d+)]\.arguments\.([A-Za-z][A-Za-z0-9]*)$/)
  if (!match) return
  const step = definition.steps[Number(match[1])]
  if (step?.type !== "command") return
  if (migration.code === "legacy_character_token") {
    if (match[2] !== "character") return
    step.arguments = { ...step.arguments, character: migration.before }
    return
  }
  try {
    step.arguments = { ...step.arguments, [match[2]]: JSON.parse(migration.before) }
  } catch {
    return
  }
}

function migratedValue(definition: ScenarioDefinition, migration: ScenarioMigration): unknown {
  const match = migration.path.match(/^\$\.steps\[(\d+)]\.arguments\.([A-Za-z][A-Za-z0-9]*)$/)
  if (!match) return undefined
  const step = definition.steps[Number(match[1])]
  if (step?.type !== "command") return undefined
  const value = step.arguments?.[match[2]]
  return migration.code === "runtime_world_precondition" ? JSON.stringify(value) : value
}

function matchesPattern(value: string, pattern: string): boolean {
  try {
    return new RegExp(pattern).test(value)
  } catch {
    return false
  }
}

function cloneRecord(values: Record<string, unknown>): Record<string, unknown> {
  return Object.fromEntries(Object.entries(values).map(([key, value]) => [key, cloneJSON(value)]))
}

function cloneJSON<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

function isScenarioParameterReference(value: unknown): value is ScenarioParameterReference {
  return typeof value === "object" && value !== null && Object.keys(value).length === 1
    && typeof (value as { parameter?: unknown }).parameter === "string"
}

function formatScenarioLiteral(value: unknown): string {
  if (typeof value === "string") return JSON.stringify(value)
  if (typeof value === "boolean") return value ? "TRUE" : "FALSE"
  if (value === undefined) return "UNSET"
  return JSON.stringify(value)
}

function truncateScenarioValue(value: string): string {
  const maximumLength = 48
  return value.length <= maximumLength ? value : `${value.slice(0, maximumLength - 3)}...`
}
