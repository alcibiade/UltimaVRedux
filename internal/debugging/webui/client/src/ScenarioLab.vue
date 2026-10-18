<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue"
import { invalidateInstrumentReview, useInstrumentDraft, useInstrumentError } from "./instrument_session"
import { validateDraft } from "./command_forms"
import { vOwnedModal } from "./owned_modal"

import type { ScenarioStatusData } from "./approved_tool_views"
import type { OptionSets } from "./command_forms"
import type { Choice, CommandDefinition, ParameterDefinition } from "./protocol"
import ScenarioArgumentEditor from "./ScenarioArgumentEditor.vue"
import ScenarioStepSummary from "./ScenarioStepSummary.vue"
import {
  compatibleScenarioParameters,
  cloneScenarioDefinition,
  scenarioCommandArguments,
  scenarioComposerSupportsParameter,
  scenarioDraftIsDirty,
  scenarioDraftHasPendingMigrations,
  scenarioLaunchValueIssues,
  scenarioParameterCompatibilityIssue,
  scenarioSafeCommands,
	scenarioBaselineLabel,
	insertScenarioStep,
	scenarioTerminalStepIndex,
  scenarioSourceLabel,
  scenarioValueIssue,
} from "./scenarios"
import type { SaveSnapshotSummary } from "./saves"
import type {
  ScenarioComposerDraft,
  ScenarioDefinitionListResult,
  ScenarioRunStatusResult,
  ScenarioRunStatus,
  ScenarioSource,
  ScenarioStep,
  ScenarioTask,
  ScenarioValidationIssue,
} from "./scenarios"

const props = defineProps<{
  data?: ScenarioStatusData
  updatedAt?: Date
  pendingStatus: boolean
  pendingDirect: boolean
  pendingHourly: boolean
  pendingGuards: boolean
  pendingWaterfall: boolean
  pendingBridgeTroll: boolean
  pendingWord: boolean
  pendingBlackthornInterrogation: boolean
  pendingLordBritishFinale: boolean
  pendingHoleUp: boolean
  pendingKillParty: boolean
  pendingDeathSequence: boolean
  pendingDefinitionList: boolean
  pendingDefinitionGet: boolean
  pendingDefinitionSave: boolean
  pendingRunStart: boolean
  pendingRunCancel?: boolean
  historicalRun?: ScenarioRunStatus
  historicalRunIssue?: string
  reviewedBaselineRevision?: string
  reviewedSceneGeneration?: number
  wordChoices: Choice[]
  composer: ScenarioComposerDraft
  definitionList?: ScenarioDefinitionListResult
  runStatus?: ScenarioRunStatusResult
  selectedRunId?: string
  validationIssues: ScenarioValidationIssue[]
  activeTask: ScenarioTask
  commands: CommandDefinition[]
  optionSets: OptionSets
  snapshots: SaveSnapshotSummary[]
}>()

const emit = defineEmits<{
  refresh: []
  run: [operation: { commandID: string; argumentsByID: Record<string, string> }]
  updateComposer: [draft: ScenarioComposerDraft]
  createDefinition: [source: ScenarioSource]
  listDefinitions: []
  getDefinition: [reference: string]
  playDefinition: [reference: string]
  saveDefinition: []
  deleteDefinition: []
  duplicateDefinition: [value: { destinationSource: ScenarioSource; destinationId: string; destinationName: string }]
  reviewRun: []
  startRun: []
  reviewAgain: [run: ScenarioRunStatus]
  cancelRemaining: [run: ScenarioRunStatus]
  reviewRollback: [run: ScenarioRunStatus]
  selectRun: [runId: string]
  updateActiveTask: [task: ScenarioTask]
}>()

const search = useInstrumentDraft("scenario.search", "")
const sourceFilter = useInstrumentDraft<"all" | ScenarioSource>("scenario.source", "all")
const seed = useInstrumentDraft<number | string>("scenario.seed", 12345)
const word = useInstrumentDraft("scenario.word", "fallax")
const campingHours = useInstrumentDraft("scenario.hours", 8)
const campingGuard = useInstrumentDraft("scenario.guard", "")
const forceWanderer = useInstrumentDraft("scenario.wanderer", false)
const forceAmbush = useInstrumentDraft("scenario.ambush", false)
const finaleWoodenBox = useInstrumentDraft("scenario.box", true)
type ReviewedQuickSetup = {
  operation: "direct" | "hourly" | "bridge" | "blackthorn" | "finale"
  seed: number | string
  woodenBox: boolean
  partyFingerprint: string
}
const armed = ref<ReviewedQuickSetup>()
const armedDeath = ref<{ operation: "kill-party" | "death-sequence"; fingerprint: string }>()
const reviewPanel = ref<"project-write" | "delete" | "duplicate">()
const reportError = useInstrumentError()
const quickIssue = ref("")
function cancelQuickReview(): void {
  armed.value = undefined
  armedDeath.value = undefined
}
invalidateInstrumentReview(() => { cancelQuickReview(); reviewPanel.value = undefined })
watch([seed, finaleWoodenBox, () => props.activeTask, () => props.data?.partyFingerprint, () => props.data?.wipeReview?.fingerprint], cancelQuickReview, { flush: "sync" })
const confirmation = ref("")
const duplicateSource = ref<ScenarioSource>("personal")
const duplicateID = ref("")
const duplicateName = ref("")
const duplicateProjectConfirmation = ref("")
const newCommandSearch = useInstrumentDraft("scenario.commandSearch", "")
const newCommandID = useInstrumentDraft("scenario.commandID", "")
const newCommandArguments = useInstrumentDraft<Record<string, unknown>>("scenario.commandArguments", {})
const newCommandLiteralBackups = useInstrumentDraft("scenario.commandBackups", new Map<string, unknown>()).value
const newStepInsertionIndex = useInstrumentDraft<number | undefined>("scenario.insertion", undefined)
const launchParameterIssue = ref("")
const stepOperationIssue = ref("")
const pendingReplacement = ref<{ index: number; commandID: string; incompatible: string[] }>()
const stepSessionKeys = useInstrumentDraft<string[]>("scenario.stepKeys", [])
const expandedReferences = useInstrumentDraft("scenario.expanded", new Set<string>())
const literalBackups = useInstrumentDraft("scenario.literalBackups", new Map<string, unknown>()).value
const nextStepSessionID = useInstrumentDraft("scenario.nextStepKey", 1)
const builderDefinition = useInstrumentDraft("scenario.builderDefinition", "")
let nextParameterSessionID = 0
const parameterSessionKeys = ref((props.composer.definition.parameters ?? []).map(() => ++nextParameterSessionID))

const blocked = computed(() => !props.data || props.data.combatActive || props.data.dialogActive
  || props.data.gameplayWorkflowActive || props.data.scriptedSceneActive || props.data.finaleActive || props.data.allPartyDead)
const dirty = computed(() => scenarioDraftIsDirty(props.composer))
const pendingMigrations = computed(() => scenarioDraftHasPendingMigrations(props.composer))
const runHistory = computed(() => props.runStatus?.runs?.length ? props.runStatus.runs : props.runStatus?.run ? [props.runStatus.run] : [])
const selectedRun = computed(() => runHistory.value.find((run) => run.runId === props.selectedRunId) ?? runHistory.value[0])
const unsupportedParameters = computed(() => (props.composer.definition.parameters ?? [])
  .filter((parameter) => !scenarioComposerSupportsParameter(parameter.kind)))
const filteredDefinitions = computed(() => (props.definitionList?.definitions ?? []).filter((definition) => {
  if (sourceFilter.value !== "all" && definition.source !== sourceFilter.value) return false
  const query = search.value.trim().toLocaleLowerCase()
  return !query || [definition.name, definition.id, definition.reference, definition.description]
    .some((value) => value.toLocaleLowerCase().includes(query))
}))
const projectDefinitions = computed(() => filteredDefinitions.value.filter((definition) => definition.source === "project"))
const personalDefinitions = computed(() => filteredDefinitions.value.filter((definition) => definition.source === "personal"))
const deleteConfirmationValid = computed(() => props.composer.source === "project"
  ? confirmation.value === props.composer.definition.id
  : confirmation.value === props.composer.definition.name)
const duplicateConfirmationValid = computed(() => Boolean(duplicateID.value && duplicateName.value)
  && (duplicateSource.value !== "project" || duplicateProjectConfirmation.value === duplicateID.value))
const safeCommands = computed(() => scenarioSafeCommands(props.commands))
const filteredSafeCommands = computed(() => {
  const query = newCommandSearch.value.trim().toLocaleLowerCase()
  if (!query) return safeCommands.value
  return safeCommands.value.filter((command) => [
    command.name,
    command.id,
    command.category,
    command.description,
    ...(command.aliases ?? []),
    ...(command.searchTerms ?? []),
  ].some((value) => value.toLocaleLowerCase().includes(query)))
})
const safeCommandGroups = computed(() => {
  const groups = new Map<string, CommandDefinition[]>()
  for (const command of filteredSafeCommands.value) {
    const commands = groups.get(command.category) ?? []
    commands.push(command)
    groups.set(command.category, commands)
  }
  return [...groups].map(([category, commands]) => ({ category, commands }))
})
const commandIndex = computed(() => new Map(props.commands.map((command) => [command.id, command])))
const newCommand = computed(() => commandIndex.value.get(newCommandID.value))
const stagedCommandParameters = computed(() => (newCommand.value?.parameters ?? []).filter((parameter) => !parameter.scenarioRuntime))
const stagedCommandReady = computed(() => stagedCommandParameters.value.every((parameter) => stagedCommandParameterIssue(parameter) === undefined))
const terminalStepIndex = computed(() => scenarioTerminalStepIndex(props.composer.definition.steps, props.commands))
const stagedCommandInsertable = computed(() => stagedCommandReady.value
  && !(newCommand.value?.scenarioTerminal && terminalStepIndex.value >= 0))
const maximumInsertionIndex = computed(() => terminalStepIndex.value < 0 ? props.composer.definition.steps.length : terminalStepIndex.value)
const plannedInsertionIndex = computed(() => newCommand.value?.scenarioTerminal
  ? props.composer.definition.steps.length
  : Math.max(0, Math.min(newStepInsertionIndex.value ?? maximumInsertionIndex.value, maximumInsertionIndex.value)))
const insertionOptions = computed(() => newCommand.value?.scenarioTerminal
  ? [{ index: props.composer.definition.steps.length, label: "AFTER ALL SETUP STEPS · FINAL" }]
  : Array.from({ length: maximumInsertionIndex.value + 1 }, (_, index) => ({
    index,
    label: insertionLabel(index),
  })))
const launchValueIssues = computed(() => scenarioLaunchValueIssues(props.composer.definition.parameters ?? [], props.composer.launchValues))
const launchValuesValid = computed(() => Object.keys(launchValueIssues.value).length === 0)
const baselineReady = computed(() => props.composer.definition.baseline.type !== "snapshot"
  || availableBaselineSnapshots.value.some((snapshot) => snapshot.reference === props.composer.definition.baseline.reference))

watch(() => props.composer.originalDefinition, () => {
  const identity = JSON.stringify([props.composer.savedReference, props.composer.originalDefinition])
  if (builderDefinition.value === identity) return
  builderDefinition.value = identity
  reviewPanel.value = undefined
  newCommandSearch.value = ""
  newCommandID.value = ""
  newCommandArguments.value = {}
  newCommandLiteralBackups.clear()
  newStepInsertionIndex.value = undefined
  launchParameterIssue.value = ""
  stepOperationIssue.value = ""
  stepSessionKeys.value = []
  parameterSessionKeys.value = (props.composer.definition.parameters ?? []).map(() => ++nextParameterSessionID)
  literalBackups.clear()
}, { immediate: true })

watch(() => props.composer.definition.steps, (steps, previousSteps) => {
  const previousKeys = [...stepSessionKeys.value]
  const keysByStep = new Map(previousSteps.flatMap((step, index) => previousKeys[index] ? [[step, previousKeys[index]]] : []))
  const usedKeys = new Set<string>()
  const nextKeys = steps.map((step, index) => {
    const matched = keysByStep.get(step)
    if (matched) {
      usedKeys.add(matched)
      return matched
    }
    const replacement = steps.length === previousSteps.length ? previousKeys[index] : undefined
    if (replacement && !usedKeys.has(replacement)) {
      usedKeys.add(replacement)
      return replacement
    }
    return newStepSessionKey()
  })
  stepSessionKeys.value = nextKeys
})

function navigate(task: ScenarioTask): void {
  emit("updateActiveTask", task)
  if (task === "library") emit("listDefinitions")
}

function createDraft(source: ScenarioSource): void {
  emit("createDefinition", source)
}

function selectDefinition(reference: string): void {
  emit("getDefinition", reference)
}

function toggleDefinitionSummary(reference: string): void {
  const expanded = new Set(expandedReferences.value)
  if (expanded.has(reference)) expanded.delete(reference)
  else expanded.add(reference)
  expandedReferences.value = expanded
}

function updateDefinition(field: "id" | "name" | "description", value: string): void {
  emitDraft({ definition: { ...props.composer.definition, [field]: value } })
}

function updateBaselineType(type: "current" | "init" | "snapshot"): void {
  const reference = type === "snapshot" ? availableBaselineSnapshots.value[0]?.reference : undefined
  emitDraft({ definition: { ...props.composer.definition, baseline: { type, ...(reference ? { reference } : {}) } } })
}

function updateBaselineReference(reference: string): void {
  emitDraft({ definition: { ...props.composer.definition, baseline: { type: "snapshot", reference } } })
}

const availableBaselineSnapshots = computed(() => props.snapshots.filter((snapshot) =>
  snapshot.source !== "rollback" && snapshot.compatibility.compatible
    && (props.composer.source !== "project" || snapshot.source === "project"),
))

function updateSource(source: ScenarioSource): void {
  emitDraft({ source })
}

function requestSave(): void {
  if (props.composer.source === "project") {
    reviewPanel.value = "project-write"
    return
  }
  emit("saveDefinition")
}

function openDeleteReview(): void {
  confirmation.value = ""
  reviewPanel.value = "delete"
}

function openDuplicateReview(): void {
  if (pendingMigrations.value) return
  duplicateSource.value = "personal"
  duplicateID.value = `${props.composer.definition.id}-copy`
  duplicateName.value = `${props.composer.definition.name} Copy`
  duplicateProjectConfirmation.value = ""
  reviewPanel.value = "duplicate"
}

function confirmDuplicate(): void {
  if (!duplicateConfirmationValid.value) return
  emit("duplicateDefinition", {
    destinationSource: duplicateSource.value,
    destinationId: duplicateID.value,
    destinationName: duplicateName.value,
  })
  reviewPanel.value = undefined
}

function stage(operation: "direct" | "hourly" | "bridge" | "blackthorn" | "finale"): void {
  quickIssue.value = ""
  if (["direct", "hourly", "bridge"].includes(operation)) {
    const commandID = operation === "direct" ? "scenario.starvation.trigger" : operation === "hourly" ? "scenario.starvation.force-hourly" : "scenario.bridge-troll.start"
    const command = commandIndex.value.get(commandID)
    const issue = !String(seed.value).trim() ? "Enter a whole number. Blank is not zero."
      : command ? Object.values(validateDraft(command, { seed: String(seed.value) }, props.optionSets)).join(" ") : "The setup is unavailable in the current catalog."
    if (issue) {
      cancelQuickReview()
      quickIssue.value = `Seed: ${issue}`
      reportError(quickIssue.value)
      return
    }
  }
  if (armed.value?.operation !== operation) {
    armed.value = {
      operation,
      seed: seed.value,
      woodenBox: finaleWoodenBox.value,
      partyFingerprint: props.data?.partyFingerprint ?? "",
    }
    return
  }
  const reviewed = armed.value
  const commands = {
    direct: "scenario.starvation.trigger",
    hourly: "scenario.starvation.force-hourly",
    bridge: "scenario.bridge-troll.start",
    blackthorn: "scenario.blackthorn-interrogation.start",
    finale: "scenario.lord-british-finale.start",
  }
  emit("run", {
    commandID: commands[operation],
    argumentsByID: operation === "direct" || operation === "hourly" || operation === "bridge"
      ? { seed: String(reviewed.seed) }
      : operation === "finale"
        ? { expectedPartyFingerprint: reviewed.partyFingerprint, woodenBox: String(reviewed.woodenBox) }
        : { expectedPartyFingerprint: reviewed.partyFingerprint },
  })
  armed.value = undefined
}

function stageDeath(operation: "kill-party" | "death-sequence"): void {
  if (armedDeath.value?.operation !== operation) {
    armedDeath.value = { operation, fingerprint: props.data?.wipeReview.fingerprint ?? "" }
    return
  }
  emit("run", {
    commandID: operation === "kill-party" ? "death.kill-party" : "death.test-sequence",
    argumentsByID: { expectedFingerprint: armedDeath.value.fingerprint },
  })
  armedDeath.value = undefined
}

function addParameter(): void {
  const parameters = [...(props.composer.definition.parameters ?? [])]
  let suffix = parameters.length + 1
  while (parameters.some((parameter) => parameter.id === `parameter-${suffix}`)) suffix++
  parameters.push({ id: `parameter-${suffix}`, name: `Parameter ${suffix}`, kind: "string", required: false })
  parameterSessionKeys.value.push(++nextParameterSessionID)
  emitDraft({ definition: { ...props.composer.definition, parameters } })
}

function updateParameter(index: number, field: string, value: unknown): void {
  const parameters = [...(props.composer.definition.parameters ?? [])]
  const parameter = { ...parameters[index] }
  launchParameterIssue.value = ""
  if (field === "id") {
    const nextID = String(value)
    if (parameters.some((candidate, candidateIndex) => candidateIndex !== index && candidate.id === nextID)) {
      launchParameterIssue.value = `Parameter ID ${nextID} is already in use.`
      return
    }
    const previousID = parameter.id
    parameter.id = nextID
    parameters[index] = parameter
    const steps = props.composer.definition.steps.map((step) => {
      if (step.type !== "command" || !step.arguments) return step
      const argumentsByID = Object.fromEntries(Object.entries(step.arguments).map(([argumentID, argument]) => [
        argumentID,
        isParameterReference(argument) && argument.parameter === previousID ? { parameter: nextID } : argument,
      ]))
      const replacement = { ...step, arguments: argumentsByID }
      return replacement
    })
    const launchValues = { ...props.composer.launchValues }
    if (Object.hasOwn(launchValues, previousID)) {
      launchValues[nextID] = launchValues[previousID]
      delete launchValues[previousID]
    }
    emitDraft({ definition: { ...props.composer.definition, parameters, steps }, launchValues })
    return
  }
  if (field === "kind") {
    parameter.kind = value as "string" | "integer" | "boolean"
    delete parameter.minimum
    delete parameter.maximum
  } else if (field === "minimum" || field === "maximum") {
    if (value === "") delete parameter[field]
    else parameter[field] = Number(value)
  } else if (field === "default") {
    if (value === "") delete parameter.default
    else if (parameter.kind === "integer") parameter.default = Number(value)
    else if (parameter.kind === "boolean") parameter.default = value === true || value === "true"
    else parameter.default = value
  } else if (field === "required") parameter.required = Boolean(value)
  else Object.assign(parameter, { [field]: value })
  parameters[index] = parameter
  emitDraft({ definition: { ...props.composer.definition, parameters } })
}

function removeParameter(index: number): void {
  const parameters = [...(props.composer.definition.parameters ?? [])]
  const [removed] = parameters.splice(index, 1)
  parameterSessionKeys.value.splice(index, 1)
  const launchValues = { ...props.composer.launchValues }
  if (removed) delete launchValues[removed.id]
  const steps = !removed ? props.composer.definition.steps : props.composer.definition.steps.map((step, stepIndex) => {
    if (step.type !== "command" || !step.arguments) return step
    const command = commandIndex.value.get(step.commandId)
    const argumentsByID = { ...step.arguments }
    for (const [argumentID, argument] of Object.entries(argumentsByID)) {
      if (!isParameterReference(argument) || argument.parameter !== removed.id) continue
      const commandParameter = command?.parameters?.find((candidate) => candidate.id === argumentID)
      const backupKey = literalBackupKey(step, stepIndex, argumentID)
      const fallback = literalBackups.has(backupKey) ? literalBackups.get(backupKey) : commandParameter?.default
      if (fallback === undefined) delete argumentsByID[argumentID]
      else argumentsByID[argumentID] = fallback
    }
    const replacement = { ...step, arguments: Object.keys(argumentsByID).length ? argumentsByID : undefined }
    return replacement
  })
  emitDraft({ definition: { ...props.composer.definition, parameters, steps }, launchValues })
}

function moveStep(index: number, offset: number): void {
	const target = index + offset
	if (target < 0 || target >= props.composer.definition.steps.length) return
	if (terminalStepIndex.value >= 0 && ((index === terminalStepIndex.value && offset < 0) || (target === terminalStepIndex.value && offset > 0))) return
  const steps = [...props.composer.definition.steps]
  ;[steps[index], steps[target]] = [steps[target], steps[index]]
  stepOperationIssue.value = ""
  emitDraft({ definition: { ...props.composer.definition, steps } })
}

function removeStep(index: number): void {
  stepOperationIssue.value = ""
  emitDraft({ definition: { ...props.composer.definition, steps: props.composer.definition.steps.filter((_, i) => i !== index) } })
}

function addCommand(): void {
	const command = newCommand.value
	if (!command || !safeCommands.value.some((candidate) => candidate.id === command.id)) return
	if (!stagedCommandReady.value) {
		stepOperationIssue.value = "Complete the required command arguments before adding this step."
		return
	}
	if (command.scenarioTerminal && terminalStepIndex.value >= 0) {
		stepOperationIssue.value = "Only one final scenario command is allowed. Remove or replace the existing final step first."
		return
	}
	const argumentsByID = { ...(scenarioCommandArguments(command) ?? {}), ...newCommandArguments.value }
	const step: ScenarioStep = { type: "command", commandId: command.id, arguments: Object.keys(argumentsByID).length ? argumentsByID : undefined }
	const insertion = insertScenarioStep(props.composer.definition.steps, step, props.commands, plannedInsertionIndex.value)
	emitDraft({ definition: { ...props.composer.definition, steps: insertion.steps } })
	newStepInsertionIndex.value = insertion.index + 1
  newCommandID.value = ""
	newCommandArguments.value = {}
	newCommandLiteralBackups.clear()
	stepOperationIssue.value = ""
	focusScenarioStep(insertion.index)
}

function selectNewCommand(commandID: string): void {
	newCommandID.value = commandID
	const command = commandIndex.value.get(commandID)
	newCommandArguments.value = command ? { ...(scenarioCommandArguments(command) ?? {}) } : {}
	newCommandLiteralBackups.clear()
	stepOperationIssue.value = ""
}

function updateStagedArgumentSource(parameter: ParameterDefinition, source: string): void {
	const value = newCommandArguments.value[parameter.id]
	if (source !== "literal") {
		if (!isParameterReference(value)) newCommandLiteralBackups.set(parameter.id, value ?? parameter.default)
		setStagedArgument(parameter.id, { parameter: source })
		return
	}
	setStagedArgument(parameter.id, newCommandLiteralBackups.get(parameter.id) ?? parameter.default)
}

function setStagedArgument(parameterID: string, value: unknown): void {
	const argumentsByID = { ...newCommandArguments.value }
	if (value === undefined) delete argumentsByID[parameterID]
	else argumentsByID[parameterID] = value
	newCommandArguments.value = argumentsByID
	stepOperationIssue.value = ""
}

function stagedCommandArgumentSource(parameter: ParameterDefinition): string {
	const value = newCommandArguments.value[parameter.id]
	return isParameterReference(value) ? value.parameter : "literal"
}

function stagedCommandParameterValue(parameter: ParameterDefinition): unknown {
	const value = newCommandArguments.value[parameter.id]
	return isParameterReference(value) ? undefined : value ?? parameter.default
}

function stagedCommandParameterIssue(parameter: ParameterDefinition): string | undefined {
	const value = newCommandArguments.value[parameter.id]
	if (!isParameterReference(value)) return scenarioValueIssue(parameter, value ?? parameter.default, props.optionSets)
	const source = props.composer.definition.parameters?.find((candidate) => candidate.id === value.parameter)
	return source ? scenarioParameterCompatibilityIssue(source, parameter) : `Launch parameter ${value.parameter} does not exist.`
}

function duplicateStep(index: number): void {
	const steps = [...props.composer.definition.steps]
	const duplicate = cloneJSON(steps[index])
	if (duplicate?.type === "command" && commandIndex.value.get(duplicate.commandId)?.scenarioTerminal) {
		stepOperationIssue.value = "Final scenario commands cannot be duplicated."
		return
  }
  steps.splice(index + 1, 0, duplicate)
  stepOperationIssue.value = ""
  emitDraft({ definition: { ...props.composer.definition, steps } })
}

function addNote(): void {
	const insertion = insertScenarioStep(props.composer.definition.steps, { type: "note", note: "Describe this scenario checkpoint." }, props.commands, plannedInsertionIndex.value)
	emitDraft({ definition: { ...props.composer.definition, steps: insertion.steps } })
	newStepInsertionIndex.value = insertion.index + 1
	stepOperationIssue.value = ""
	focusScenarioStep(insertion.index)
}

function updateNote(index: number, note: string): void {
  const steps = [...props.composer.definition.steps]
  if (steps[index]?.type !== "note") return
  steps[index] = { type: "note", note }
  emitDraft({ definition: { ...props.composer.definition, steps } })
}

function updateArgumentSource(stepIndex: number, parameter: ParameterDefinition, source: string): void {
  const step = props.composer.definition.steps[stepIndex]
  if (step.type !== "command") return
  const value = step.arguments?.[parameter.id]
  const backupKey = literalBackupKey(step, stepIndex, parameter.id)
  if (source !== "literal") {
    if (!isParameterReference(value)) literalBackups.set(backupKey, value ?? parameter.default)
    setStepArgument(stepIndex, parameter.id, { parameter: source })
    return
  }
  const literal = literalBackups.has(backupKey) ? literalBackups.get(backupKey) : parameter.default
  setStepArgument(stepIndex, parameter.id, literal)
}

function updateLiteralArgument(stepIndex: number, argumentID: string, value: unknown): void {
  setStepArgument(stepIndex, argumentID, value)
}

function setStepArgument(stepIndex: number, argumentID: string, value: unknown): void {
  const steps = [...props.composer.definition.steps]
  const step = steps[stepIndex]
  if (step.type !== "command") return
  const argumentsByID = { ...step.arguments }
  if (value === undefined) delete argumentsByID[argumentID]
  else argumentsByID[argumentID] = value
  const replacement = { ...step, arguments: Object.keys(argumentsByID).length ? argumentsByID : undefined }
  steps[stepIndex] = replacement
  emitDraft({ definition: { ...props.composer.definition, steps } })
}

function updateLaunchValue(parameterID: string, kind: string, raw: string): void {
  const launchValues = { ...props.composer.launchValues }
  if (raw === "") delete launchValues[parameterID]
  else launchValues[parameterID] = kind === "integer" ? Number(raw) : kind === "boolean" ? raw === "true" : raw
  emitDraft({ launchValues })
}

function emitDraft(changes: Partial<ScenarioComposerDraft>): void {
  emit("updateComposer", { ...props.composer, ...changes })
}

function argumentMapping(value: unknown): string {
  return isParameterReference(value) ? value.parameter : "literal"
}

function commandForStep(step: ScenarioStep): CommandDefinition | undefined {
  return step.type === "command" ? commandIndex.value.get(step.commandId) : undefined
}

function commandParameterValue(step: ScenarioStep, parameter: ParameterDefinition): unknown {
  if (step.type !== "command") return undefined
  const value = step.arguments?.[parameter.id]
  return isParameterReference(value) ? undefined : value ?? parameter.default
}

function compatibleParameters(parameter: ParameterDefinition): ParameterDefinition[] {
  return compatibleScenarioParameters(parameter, props.composer.definition.parameters ?? [])
}

function unknownArgumentIDs(step: ScenarioStep, command?: CommandDefinition): string[] {
  if (step.type !== "command") return []
  const known = new Set((command?.parameters ?? []).map((parameter) => parameter.id))
  return Object.keys(step.arguments ?? {}).filter((argumentID) => !known.has(argumentID))
}

function requestCommandReplacement(index: number, commandID: string): void {
  const step = props.composer.definition.steps[index]
  const command = commandIndex.value.get(commandID)
  if (step.type !== "command" || !command || step.commandId === commandID) return
  const incompatible = incompatibleReplacementArguments(step, command)
  if (incompatible.length) {
    pendingReplacement.value = { index, commandID, incompatible }
    return
  }
  replaceCommand(index, command)
}

function confirmCommandReplacement(): void {
  if (!pendingReplacement.value) return
  const command = commandIndex.value.get(pendingReplacement.value.commandID)
  if (command) replaceCommand(pendingReplacement.value.index, command)
  pendingReplacement.value = undefined
}

function incompatibleReplacementArguments(step: Extract<ScenarioStep, { type: "command" }>, replacement: CommandDefinition): string[] {
  const previous = commandIndex.value.get(step.commandId)
  return Object.keys(step.arguments ?? {}).filter((argumentID) => {
    const oldParameter = previous?.parameters?.find((parameter) => parameter.id === argumentID)
    const newParameter = replacement.parameters?.find((parameter) => parameter.id === argumentID)
    if (!oldParameter || !newParameter || oldParameter.kind !== newParameter.kind) return true
    const value = step.arguments?.[argumentID]
    if (isParameterReference(value)) {
      const source = props.composer.definition.parameters?.find((parameter) => parameter.id === value.parameter)
      return !source || scenarioParameterCompatibilityIssue(source, newParameter) !== undefined
    }
    return scenarioValueIssue(newParameter, value, props.optionSets) !== undefined
  })
}

function replaceCommand(index: number, command: CommandDefinition): void {
	if (command.scenarioTerminal && (terminalStepIndex.value >= 0 || index !== props.composer.definition.steps.length - 1)) {
		stepOperationIssue.value = "A final scenario command must be the only final step. Move it to the end or remove the existing final step first."
		return
	}
  const steps = [...props.composer.definition.steps]
  const previous = steps[index]
  if (previous.type !== "command") return
  const incompatible = new Set(incompatibleReplacementArguments(previous, command))
  const argumentsByID = scenarioCommandArguments(command) ?? {}
  for (const [argumentID, value] of Object.entries(previous.arguments ?? {})) {
    if (!incompatible.has(argumentID)) argumentsByID[argumentID] = value
  }
  const replacement: ScenarioStep = { type: "command", commandId: command.id, arguments: Object.keys(argumentsByID).length ? argumentsByID : undefined }
  steps[index] = replacement
  stepOperationIssue.value = ""
  emitDraft({ definition: { ...props.composer.definition, steps } })
}

function stepIssues(index: number): ScenarioValidationIssue[] {
  const pathPrefixes = [`$.steps[${index}]`, `steps[${index}]`]
  const issues = props.validationIssues.filter((issue) => pathPrefixes.some((prefix) => issue.path.startsWith(prefix)))
  const step = props.composer.definition.steps[index]
  if (step.type !== "command") return issues
  const command = commandIndex.value.get(step.commandId)
  if (!command) return [...issues, { path: `$.steps[${index}].commandId`, code: "command.not_found", message: "Command is not present in the current catalog." }]
  for (const parameter of command.parameters ?? []) {
	if (parameter.scenarioRuntime) continue
    const value = step.arguments?.[parameter.id]
    let message: string | undefined
    if (isParameterReference(value)) {
      const source = props.composer.definition.parameters?.find((candidate) => candidate.id === value.parameter)
      message = source ? scenarioParameterCompatibilityIssue(source, parameter) : `Launch parameter ${value.parameter} does not exist.`
    } else {
      message = scenarioValueIssue(parameter, value ?? parameter.default, props.optionSets)
    }
    if (message) issues.push({ path: `$.steps[${index}].arguments.${parameter.id}`, code: "argument.invalid", message })
  }
  for (const argumentID of unknownArgumentIDs(step, command)) {
    issues.push({ path: `$.steps[${index}].arguments.${argumentID}`, code: "argument.not_found", message: "Argument is not defined by the selected command." })
  }
  return issues
}

function validationIssueMessage(issue: ScenarioValidationIssue): string {
	if (issue.code !== "step.after_terminal_command") return issue.message
	const match = issue.path.match(/steps\[(\d+)\]/)
	const stepNumber = match ? Number(match[1]) + 1 : undefined
	const terminalNumber = terminalStepIndex.value >= 0 ? terminalStepIndex.value + 1 : undefined
	if (stepNumber !== undefined && terminalNumber !== undefined) {
		return `Step ${stepNumber} follows final Step ${terminalNumber}. Move it before combat starts.`
	}

	return "This step follows a final combat-start step. Move it before combat starts."
}

function ensureStepSessionKeys(): void {
  while (stepSessionKeys.value.length < props.composer.definition.steps.length) stepSessionKeys.value.push(newStepSessionKey())
  if (stepSessionKeys.value.length > props.composer.definition.steps.length) stepSessionKeys.value.length = props.composer.definition.steps.length
}

function newStepSessionKey(): string {
  return `step-${nextStepSessionID.value++}`
}

function insertionLabel(index: number): string {
	if (props.composer.definition.steps.length === 0) return "FIRST STEP"
	if (index === 0) return "BEFORE STEP 01"
	if (index === props.composer.definition.steps.length && terminalStepIndex.value < 0) return `AFTER STEP ${String(index).padStart(2, "0")} · END`
	return `AFTER STEP ${String(index).padStart(2, "0")}`
}

function focusScenarioStep(index: number): void {
	void nextTick(() => {
		const step = document.querySelector<HTMLElement>(`[data-scenario-screen="editor"] [data-scenario-step="${index}"]`)
		if (typeof step?.scrollIntoView === "function") step.scrollIntoView({ block: "center" })
		step?.focus()
	})
}

function literalBackupKey(_step: object, index: number, argumentID: string): string {
  ensureStepSessionKeys()
  const step = props.composer.definition.steps[index]
  const scenarioIdentity = props.composer.savedReference ?? props.composer.definition.id
  const commandIdentity = step?.type === "command" ? step.commandId : "note"
  return `${scenarioIdentity}:${stepSessionKeys.value[index]}:${commandIdentity}:${argumentID}`
}

function cloneJSON<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

function isParameterReference(value: unknown): value is { parameter: string } {
  return typeof value === "object" && value !== null && Object.keys(value).length === 1
    && typeof (value as { parameter?: unknown }).parameter === "string"
}

function displayValue(value: unknown): string {
  return typeof value === "string" ? value : JSON.stringify(value)
}

function formatTimestamp(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? "UNKNOWN TIME" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "medium" }).format(date)
}

</script>

<template>
  <section class="approved-view scenario-lab" aria-label="Scenarios workspace" data-scenarios-workspace @keydown.esc="cancelQuickReview">
    <header class="approved-view-header">
      <div><span>REPEATABLE TEST WORKFLOWS</span><h3>Scenarios</h3></div>
      <button v-if="activeTask === 'quick-setups'" type="button" :disabled="pendingStatus" @click="emit('refresh')">{{ pendingStatus ? 'SYNCING...' : 'REFRESH' }}</button>
      <button v-else-if="activeTask === 'library'" type="button" :disabled="pendingDefinitionList" data-scenario-action="refresh-list" @click="emit('listDefinitions')">{{ pendingDefinitionList ? 'SYNCING...' : 'REFRESH LIBRARY' }}</button>
    </header>

    <nav class="scenario-tabs" aria-label="Scenario tasks">
      <button v-for="task in (['library', 'quick-setups', 'latest-run'] as ScenarioTask[])" :key="task" type="button" :aria-current="activeTask === task ? 'page' : undefined" :data-scenario-task="task" @click="navigate(task)">{{ task === 'quick-setups' ? 'QUICK SETUPS' : task === 'latest-run' ? 'RUN HISTORY' : 'LIBRARY' }}</button>
    </nav>

    <section v-if="activeTask === 'library'" class="scenario-library" data-scenario-screen="library">
      <header class="scenario-library-toolbar">
        <label>SEARCH<input v-model="search" type="search" data-scenario-search placeholder="NAME, ID, OR DESCRIPTION" /></label>
        <label>SOURCE<select v-model="sourceFilter" data-scenario-source-filter><option value="all">All sources</option><option value="project">Project / Shared</option><option value="personal">Personal / Local</option></select></label>
        <button type="button" data-scenario-action="new-personal" @click="createDraft('personal')">NEW PERSONAL</button>
        <button type="button" data-scenario-action="new-project" @click="createDraft('project')">NEW PROJECT</button>
      </header>
      <section v-for="group in [{ source: 'project' as const, items: projectDefinitions }, { source: 'personal' as const, items: personalDefinitions }]" :key="group.source" class="scenario-library-group" :data-scenario-source="group.source">
        <header><strong>{{ scenarioSourceLabel(group.source) }}</strong><small>{{ group.items.length }}</small></header>
        <article v-for="definition in group.items" :key="definition.reference" class="scenario-library-card">
          <button type="button" class="scenario-library-open" :disabled="pendingDefinitionGet || pendingDefinitionSave" :data-scenario-reference="definition.reference" @click="selectDefinition(definition.reference)">
            <strong>{{ definition.name }}</strong><span>{{ definition.description }}</span><small>{{ definition.reference }} · {{ scenarioBaselineLabel(definition.baseline).toUpperCase() }} BASELINE · {{ definition.parameters?.length ?? 0 }} PARAMETERS · {{ definition.steps.length }} STEPS</small>
          </button>
          <div class="scenario-library-card-actions">
            <button type="button" class="scenario-library-play" :disabled="pendingDefinitionGet || pendingDefinitionSave" :aria-label="`Run ${definition.name}`" :data-scenario-play="definition.reference" @click="emit('playDefinition', definition.reference)">▶ RUN</button>
            <button type="button" class="scenario-library-expand" :aria-expanded="expandedReferences.has(definition.reference)" :data-scenario-expand="definition.reference" @click="toggleDefinitionSummary(definition.reference)">{{ expandedReferences.has(definition.reference) ? 'HIDE STEPS' : 'SHOW STEPS' }}</button>
          </div>
          <ScenarioStepSummary v-if="expandedReferences.has(definition.reference)" :steps="definition.steps" :parameters="definition.parameters" :commands="commands" :option-sets="optionSets" />
        </article>
        <p v-if="group.items.length === 0">NO MATCHING {{ group.source.toUpperCase() }} DEFINITIONS</p>
      </section>
    </section>

    <section v-else-if="activeTask === 'details'" class="scenario-details" data-scenario-screen="details">
      <header><div><span>{{ scenarioSourceLabel(composer.source) }}</span><h4>{{ composer.definition.name }}</h4></div><code>{{ composer.savedReference }}</code></header>
      <p>{{ composer.definition.description }}</p>
      <section v-if="pendingMigrations" class="scenario-retained-warning" role="alert" data-scenario-migration-review>
        <strong>LEGACY CHARACTER TARGETS NEED REVIEW</strong>
        <p v-for="migration in composer.migrations" :key="`${migration.path}-${migration.before}`"><code>{{ migration.path }}</code>: <code>{{ migration.before }}</code> to <code>{{ migration.after }}</code></p>
        <p>The migrated target now follows whoever occupies that party position when the scenario runs. Review and save this revision explicitly.</p>
      </section>
      <dl>
        <div><dt>Reference</dt><dd>{{ composer.savedReference }}</dd></div><div><dt>Revision</dt><dd><code>{{ composer.loadedRevision }}</code></dd></div>
         <div><dt>Baseline</dt><dd>{{ scenarioBaselineLabel(composer.definition.baseline) }}</dd></div><div><dt>Parameters</dt><dd>{{ composer.definition.parameters?.length ?? 0 }}</dd></div><div><dt>Steps</dt><dd>{{ composer.definition.steps.length }}</dd></div>
      </dl>
      <section class="scenario-detail-plan">
        <header><strong>ORDERED STEPS</strong><small>{{ composer.definition.steps.length }}</small></header>
        <ScenarioStepSummary :steps="composer.definition.steps" :parameters="composer.definition.parameters" :commands="commands" :option-sets="optionSets" />
      </section>
      <p class="scenario-retained-warning">RUNS STOP ON FIRST FAILURE. COMPLETED MUTATIONS ARE RETAINED IN THE ACTIVE GAME STATE FOR DIAGNOSIS.</p>
      <footer class="scenario-detail-actions">
        <p v-if="dirty || pendingMigrations" class="scenario-retained-warning" role="status">SAVE FIRST: Save the edited definition and pending migrations before reviewing a run. Start executes the saved revision, not unsaved changes.</p>
        <button type="button" data-scenario-action="review-run" :disabled="dirty || pendingMigrations || unsupportedParameters.length > 0" @click="emit('reviewRun')">REVIEW &amp; RUN</button>
        <button type="button" data-scenario-action="edit" @click="emit('updateActiveTask', 'editor')">EDIT</button>
        <button type="button" data-scenario-action="duplicate" :disabled="pendingMigrations" @click="openDuplicateReview">DUPLICATE</button>
        <button type="button" class="danger-button" data-scenario-action="delete" @click="openDeleteReview">DELETE</button>
        <button type="button" data-scenario-action="back" @click="navigate('library')">BACK</button>
      </footer>
    </section>

    <div v-else-if="activeTask === 'editor'" class="scenario-composer" data-scenario-screen="editor">
      <fieldset class="scenario-composer-form" :disabled="pendingDefinitionSave">
        <section v-if="pendingMigrations" class="scenario-retained-warning" role="alert" data-scenario-migration-review>
          <strong>LEGACY CHARACTER TARGETS NEED REVIEW</strong>
          <p v-for="migration in composer.migrations" :key="`${migration.path}-${migration.before}`"><code>{{ migration.path }}</code>: <code>{{ migration.before }}</code> to <code>{{ migration.after }}</code></p>
          <p>The migrated target now follows whoever occupies that party position when the scenario runs. Review and save this revision explicitly.</p>
        </section>
        <section class="scenario-editor-section scenario-definition-fields">
          <header><strong>EDITOR</strong><small>{{ composer.savedReference ? `REVISION ${composer.loadedRevision?.slice(0, 12)}` : 'NEW DEFINITION' }}</small></header>
          <label>ID<input data-scenario-field="id" :value="composer.definition.id" :disabled="Boolean(composer.savedReference)" pattern="[a-z][a-z0-9]*(?:-[a-z0-9]+)*" required @input="updateDefinition('id', ($event.target as HTMLInputElement).value)" /></label>
          <label>NAME<input data-scenario-field="name" :value="composer.definition.name" required @input="updateDefinition('name', ($event.target as HTMLInputElement).value)" /></label>
          <label>DESCRIPTION<textarea data-scenario-field="description" :value="composer.definition.description" required @input="updateDefinition('description', ($event.target as HTMLTextAreaElement).value)" /></label>
          <label>BASELINE TYPE<select data-scenario-field="baseline" :value="composer.definition.baseline.type" @change="updateBaselineType(($event.target as HTMLSelectElement).value as 'current' | 'init' | 'snapshot')"><option value="current">Current state</option><option value="init">INIT.GAM</option><option value="snapshot">Named snapshot</option></select></label>
          <label v-if="composer.definition.baseline.type === 'snapshot'">NAMED SNAPSHOT<select data-scenario-field="baseline-reference" :value="composer.definition.baseline.reference" required @change="updateBaselineReference(($event.target as HTMLSelectElement).value)"><option value="">SELECT SNAPSHOT</option><option v-for="snapshot in availableBaselineSnapshots" :key="snapshot.reference" :value="snapshot.reference">{{ snapshot.name }} · {{ snapshot.reference }}</option></select><small v-if="availableBaselineSnapshots.length === 0">No compatible {{ composer.source }} snapshot is available.</small></label>
          <label>DESTINATION<select data-scenario-field="source" :value="composer.source" :disabled="Boolean(composer.savedReference)" @change="updateSource(($event.target as HTMLSelectElement).value as ScenarioSource)"><option value="project">Project / Shared</option><option value="personal">Personal / Local</option></select></label>
        </section>

        <section v-if="validationIssues.length" class="scenario-validation" role="alert" data-scenario-validation-issues>
          <header><strong>VALIDATION ISSUES</strong><small>{{ validationIssues.length }}</small></header>
          <p v-for="issue in validationIssues" :key="`${issue.path}-${issue.code}`"><code>{{ issue.path }}</code> · {{ validationIssueMessage(issue) }} <small>{{ issue.code }}</small></p>
        </section>

        <section class="scenario-editor-section">
          <header><strong>LAUNCH PARAMETERS</strong><button type="button" data-scenario-action="add-parameter" @click="addParameter">ADD PARAMETER</button></header>
          <p v-if="launchParameterIssue" class="error-line" data-scenario-parameter-issue>{{ launchParameterIssue }}</p>
          <p v-if="unsupportedParameters.length" class="error-line" data-scenario-unsupported-parameters>THIS EDITOR CANNOT EDIT OR LAUNCH: {{ unsupportedParameters.map((parameter) => `${parameter.id} (${parameter.kind})`).join(', ') }}.</p>
          <div v-for="(parameter, index) in composer.definition.parameters ?? []" :key="parameterSessionKeys[index]" class="scenario-parameter" :data-scenario-parameter="index">
            <input aria-label="Parameter ID" :value="parameter.id" @input="updateParameter(index, 'id', ($event.target as HTMLInputElement).value)" /><input aria-label="Parameter name" :value="parameter.name" @input="updateParameter(index, 'name', ($event.target as HTMLInputElement).value)" />
            <select aria-label="Parameter kind" :value="parameter.kind" :disabled="!scenarioComposerSupportsParameter(parameter.kind)" @change="updateParameter(index, 'kind', ($event.target as HTMLSelectElement).value)"><option v-if="!scenarioComposerSupportsParameter(parameter.kind)" :value="parameter.kind">{{ parameter.kind }}</option><option value="string">String</option><option value="integer">Integer</option><option value="boolean">Boolean</option></select>
            <label class="check-label"><input type="checkbox" :checked="parameter.required" @change="updateParameter(index, 'required', ($event.target as HTMLInputElement).checked)" /> REQUIRED</label>
            <input v-if="parameter.kind === 'integer'" aria-label="Minimum" type="number" placeholder="MIN" :value="parameter.minimum" @input="updateParameter(index, 'minimum', ($event.target as HTMLInputElement).value)" /><input v-if="parameter.kind === 'integer'" aria-label="Maximum" type="number" placeholder="MAX" :value="parameter.maximum" @input="updateParameter(index, 'maximum', ($event.target as HTMLInputElement).value)" />
            <select v-if="parameter.kind === 'boolean'" aria-label="Default value" :value="parameter.default === undefined ? '' : String(parameter.default)" @change="updateParameter(index, 'default', ($event.target as HTMLSelectElement).value)"><option value="">NO DEFAULT</option><option value="true">TRUE</option><option value="false">FALSE</option></select><input v-else aria-label="Default value" :type="parameter.kind === 'integer' ? 'number' : 'text'" placeholder="DEFAULT" :value="parameter.default" @input="updateParameter(index, 'default', ($event.target as HTMLInputElement).value)" />
            <button type="button" class="quiet-button" :data-remove-parameter="index" @click="removeParameter(index)">REMOVE</button>
          </div>
        </section>

        <section class="scenario-editor-section scenario-steps-editor">
		  <header><strong>ORDERED STEPS</strong><span><small>{{ composer.definition.steps.length }} STEPS</small><button type="button" data-scenario-action="add-note" @click="addNote">ADD NOTE AT POSITION</button></span></header>
		  <p v-if="stepOperationIssue" class="error-line scenario-step-operation-issue" data-scenario-step-operation-issue>{{ stepOperationIssue }}</p>
		  <section class="scenario-command-builder" data-scenario-command-builder>
			<div class="scenario-command-builder-controls">
			  <label>FIND COMMAND<input v-model="newCommandSearch" type="search" data-scenario-command-search placeholder="NAME, ID, CATEGORY, OR INTENT" /></label>
			  <label>COMMAND<select :value="newCommandID" aria-label="Command to add" data-scenario-command-choice @change="selectNewCommand(($event.target as HTMLSelectElement).value)"><option value="">{{ filteredSafeCommands.length ? 'SELECT COMMAND' : 'NO MATCHING COMMANDS' }}</option><optgroup v-for="group in safeCommandGroups" :key="group.category" :label="`${group.category} · ${group.commands.length}`"><option v-for="command in group.commands" :key="command.id" :value="command.id">{{ command.name }} · {{ command.id }}</option></optgroup></select></label>
			  <label>INSERT POSITION<select :value="plannedInsertionIndex" data-scenario-insertion-position @change="newStepInsertionIndex = Number(($event.target as HTMLSelectElement).value)"><option v-for="option in insertionOptions" :key="option.index" :value="option.index">{{ option.label }}</option></select></label>
			</div>
			<section v-if="newCommand" class="scenario-command-staging" data-scenario-command-staging>
			  <header><div><strong>{{ newCommand.name }}</strong><code>{{ newCommand.id }}</code></div><small>{{ newCommand.category }} · {{ newCommand.executionModes.join(' / ') }}{{ newCommand.scenarioTerminal ? ' · FINAL STEP' : '' }}</small></header>
			  <p>{{ newCommand.description }}</p>
			  <div v-if="stagedCommandParameters.length" class="scenario-arguments scenario-staged-arguments">
				<ScenarioArgumentEditor v-for="parameter in stagedCommandParameters" :key="parameter.id" :parameter="parameter" :value="stagedCommandParameterValue(parameter)" :source="stagedCommandArgumentSource(parameter)" :launch-parameters="compatibleParameters(parameter)" :option-sets="optionSets" @update-source="updateStagedArgumentSource(parameter, $event)" @update-literal="setStagedArgument(parameter.id, $event)" />
			  </div>
			  <p v-else class="scenario-no-arguments">NO COMMAND ARGUMENTS</p>
			  <footer><span>{{ newCommand.scenarioTerminal && terminalStepIndex >= 0 ? 'REMOVE THE EXISTING FINAL STEP FIRST' : stagedCommandReady ? 'READY TO INSERT' : 'COMPLETE REQUIRED ARGUMENTS' }}</span><button type="button" data-scenario-action="add-command" :disabled="!stagedCommandInsertable" @click="addCommand">ADD AT {{ insertionLabel(plannedInsertionIndex) }}</button></footer>
			</section>
		  </section>
          <article v-for="(step, index) in composer.definition.steps" :key="index" tabindex="-1" class="scenario-step" :class="[`scenario-step-${step.type}`, { 'has-issues': stepIssues(index).length > 0 }]" :data-scenario-step="index" :data-step-kind="step.type">
            <header><span>{{ String(index + 1).padStart(2, '0') }}</span><div class="scenario-step-title"><strong>{{ step.type === 'command' ? (commandForStep(step)?.name ?? 'UNKNOWN COMMAND') : 'NOTE' }}</strong><code v-if="step.type === 'command'">{{ step.commandId }}</code><small v-if="step.type === 'command'">{{ commandForStep(step)?.scenarioTerminal ? 'FINAL STEP · TERMINAL WORKFLOW' : commandForStep(step)?.executionModes.join(' / ') ?? 'NOT IN CATALOG' }}</small></div><div><button type="button" :disabled="index === 0 || (index === terminalStepIndex && index > 0)" :data-move-step-up="index" @click="moveStep(index, -1)">UP</button><button type="button" :disabled="index === composer.definition.steps.length - 1 || (index + 1 === terminalStepIndex)" :data-move-step-down="index" @click="moveStep(index, 1)">DOWN</button><button type="button" :disabled="step.type === 'command' && commandForStep(step)?.scenarioTerminal" :data-duplicate-step="index" @click="duplicateStep(index)">DUPLICATE</button><button type="button" :data-remove-step="index" @click="removeStep(index)">REMOVE</button></div></header>
            <template v-if="step.type === 'command'">
              <div class="scenario-command-replace"><label>COMMAND<select :aria-label="`Replace command for step ${index + 1}`" :value="step.commandId" @change="requestCommandReplacement(index, ($event.target as HTMLSelectElement).value)"><option v-if="!commandForStep(step)" :value="step.commandId">UNKNOWN · {{ step.commandId }}</option><option v-for="command in safeCommands" :key="command.id" :value="command.id">{{ command.name }} · {{ command.id }}</option></select></label></div>
              <div v-if="commandForStep(step)" class="scenario-arguments">
                <ScenarioArgumentEditor v-for="parameter in (commandForStep(step)?.parameters ?? []).filter((candidate) => !candidate.scenarioRuntime)" :key="parameter.id" :parameter="parameter" :value="commandParameterValue(step, parameter)" :source="argumentMapping(step.arguments?.[parameter.id])" :launch-parameters="compatibleParameters(parameter)" :option-sets="optionSets" @update-source="updateArgumentSource(index, parameter, $event)" @update-literal="updateLiteralArgument(index, parameter.id, $event)" />
                <div v-for="argumentID in unknownArgumentIDs(step, commandForStep(step))" :key="argumentID" class="scenario-unknown-argument"><code>{{ argumentID }}</code><span>{{ displayValue(step.arguments?.[argumentID]) }}</span><strong>NOT IN COMMAND CATALOG</strong><button type="button" @click="updateLiteralArgument(index, argumentID, undefined)">RESET</button></div>
                <p v-if="(commandForStep(step)?.parameters?.length ?? 0) === 0 && unknownArgumentIDs(step, commandForStep(step)).length === 0" class="scenario-no-arguments">NO COMMAND ARGUMENTS</p>
              </div>
              <section v-if="pendingReplacement?.index === index" class="scenario-replacement-review" role="alert" data-scenario-replacement-review><strong>REPLACE COMMAND?</strong><p>The following incompatible arguments will reset to the new command defaults or unset state: <code>{{ pendingReplacement.incompatible.join(', ') }}</code>.</p><button type="button" @click="pendingReplacement = undefined">CANCEL</button><button type="button" data-scenario-action="confirm-replace-command" @click="confirmCommandReplacement">REPLACE &amp; RESET</button></section>
            </template>
            <textarea v-else aria-label="Scenario note" :value="step.note" @input="updateNote(index, ($event.target as HTMLTextAreaElement).value)" />
            <div v-if="stepIssues(index).length" class="scenario-step-issues" role="alert" :data-scenario-step-issues="index"><p v-for="issue in stepIssues(index)" :key="`${issue.path}-${issue.code}-${issue.message}`"><code>{{ issue.path }}</code> · {{ validationIssueMessage(issue) }}</p></div>
          </article>
          <p v-if="composer.definition.steps.length === 0">NO STEPS · ADD A SCENARIO-SAFE CATALOG COMMAND OR NOTE</p>
        </section>
        <footer class="scenario-composer-actions"><span>{{ dirty ? 'UNSAVED CHANGES' : composer.savedReference ? `LOADED ${composer.savedReference}` : 'NEW DRAFT' }}</span><button type="button" data-scenario-action="editor-back" @click="navigate(composer.savedReference ? 'details' : 'library')">BACK</button><button type="button" data-scenario-action="editor-review-run" :disabled="!composer.savedReference || dirty || unsupportedParameters.length > 0" @click="emit('reviewRun')">REVIEW &amp; RUN</button><button type="button" data-scenario-action="save" :disabled="pendingDefinitionSave || composer.definition.steps.length === 0 || unsupportedParameters.length > 0 || !dirty" @click="requestSave">{{ pendingDefinitionSave ? 'VALIDATING...' : composer.savedReference ? 'SAVE REVISION' : `CREATE IN ${composer.source.toUpperCase()}` }}</button></footer>
      </fieldset>
    </div>

    <section v-else-if="activeTask === 'run-review'" class="scenario-run-review" data-scenario-screen="run-review">
      <header><div><span>MANDATORY REVIEW</span><h4>{{ composer.definition.name }}</h4></div><code>{{ composer.savedReference }}</code></header>
      <p>DEFINITION REVISION <code>{{ composer.loadedRevision }}</code></p>
      <p>REVIEWED SCENE {{ reviewedSceneGeneration }} <span v-if="reviewedBaselineRevision"> / BASELINE REVISION <code>{{ reviewedBaselineRevision }}</code></span></p>
      <p v-if="historicalRun" class="scenario-retained-warning">REVIEW AGAIN FROM {{ historicalRun.runId }}. Actual recorded launch values are prefilled, not current defaults. This is a new run, not a deterministic replay. Historical baseline {{ historicalRun.launch?.baselineApplied ? 'was applied' : 'was not applied' }}. Historical RNG seed {{ historicalRun.launch?.rng?.seed ?? 'unknown' }}, source calls {{ historicalRun.launch?.rng?.sourceCalls ?? 'unknown' }}. RNG provenance is diagnostic and is not reapplied.</p>
      <p v-if="historicalRunIssue" class="error-line" role="alert">{{ historicalRunIssue }}</p>
      <p v-if="!baselineReady" class="error-line" role="alert">The named baseline is unavailable. Select an available compatible baseline before starting.</p>
      <div class="scenario-review-grid"><div><span>BASELINE</span><strong>{{ scenarioBaselineLabel(composer.definition.baseline).toUpperCase() }}</strong><small>{{ composer.definition.baseline.type === 'init' ? 'Reloads INIT.GAM after capturing rollback.' : composer.definition.baseline.type === 'snapshot' ? 'Loads the reviewed named revision after capturing rollback.' : 'Uses the current active game state.' }}</small></div><div><span>READINESS</span><strong>{{ blocked ? 'BLOCKED' : 'READY' }}</strong><small>{{ blocked ? 'Resolve active combat, dialog, workflow, scripted scene, terminal outcome, or party death.' : 'No known scenario status blocker.' }}</small></div><div><span>CONSEQUENCE</span><strong>RETAINED MUTATIONS</strong><small>Successful earlier steps remain if a later step fails.</small></div></div>
      <section class="scenario-launch"><header><strong>LAUNCH VALUES</strong><small>{{ composer.definition.parameters?.length ?? 0 }} PARAMETERS</small></header><div v-for="parameter in composer.definition.parameters ?? []" :key="parameter.id" class="scenario-launch-parameter"><label :for="`scenario-launch-${parameter.id}`">{{ parameter.name }}<span>{{ parameter.required ? ' · REQUIRED' : ' · OPTIONAL' }}</span><small v-if="parameter.description">{{ parameter.description }}</small></label><select v-if="parameter.kind === 'boolean'" :id="`scenario-launch-${parameter.id}`" :value="String(composer.launchValues[parameter.id] ?? parameter.default ?? '')" @change="updateLaunchValue(parameter.id, parameter.kind, ($event.target as HTMLSelectElement).value)"><option value="">UNSET</option><option value="true">TRUE</option><option value="false">FALSE</option></select><input v-else :id="`scenario-launch-${parameter.id}`" :type="parameter.kind === 'integer' ? 'number' : 'text'" :min="parameter.minimum" :max="parameter.maximum" :required="parameter.required" :disabled="!scenarioComposerSupportsParameter(parameter.kind)" :value="composer.launchValues[parameter.id] ?? parameter.default ?? ''" @input="updateLaunchValue(parameter.id, parameter.kind, ($event.target as HTMLInputElement).value)" /><p v-if="launchValueIssues[parameter.id]" class="error-line" :data-scenario-launch-issue="parameter.id">{{ launchValueIssues[parameter.id] }}</p></div></section>
      <ScenarioStepSummary :steps="composer.definition.steps" :parameters="composer.definition.parameters" :commands="commands" :option-sets="optionSets" />
      <footer class="scenario-detail-actions"><button type="button" data-scenario-action="confirm-run" :disabled="pendingRunStart || blocked || !baselineReady || Boolean(historicalRunIssue) || unsupportedParameters.length > 0 || !launchValuesValid" @click="emit('startRun')">{{ pendingRunStart ? 'QUEUEING...' : 'RUN REVIEWED SCENARIO' }}</button><button type="button" @click="emit('updateActiveTask', 'details')">BACK TO DETAILS</button></footer>
    </section>

    <section v-else-if="activeTask === 'latest-run'" class="scenario-runs" data-scenario-screen="latest-run">
      <section v-if="selectedRun" class="scenario-history-actions" data-scenario-history-actions>
        <div class="scenario-detail-actions">
          <button type="button" data-scenario-review-again :disabled="!selectedRun.launch || pendingDefinitionGet" @click="emit('reviewAgain', selectedRun)">REVIEW AGAIN</button>
          <button v-if="selectedRun.currentProcess && ['queued', 'running'].includes(selectedRun.state)" type="button" data-scenario-cancel-remaining :disabled="pendingRunCancel || !commands.some((command) => command.id === 'scenario.run.cancel')" @click="emit('cancelRemaining', selectedRun)">{{ pendingRunCancel ? 'CANCELLING...' : 'CANCEL REMAINING STEPS' }}</button>
          <button v-if="selectedRun.rollback" type="button" data-scenario-review-rollback @click="emit('reviewRollback', selectedRun)">REVIEW ROLLBACK</button>
        </div>
        <p v-if="!selectedRun.launch" class="scenario-retained-warning">Historical launch values are unknown. Review Again is unsupported for this old record; current defaults are not the historical configuration.</p>
        <template v-else>
          <p>ACTUAL LAUNCH VALUES <code>{{ JSON.stringify(selectedRun.launch.parameterValues) }}</code></p>
          <p>BASELINE {{ selectedRun.launch.baselineApplied ? 'APPLIED' : 'NOT APPLIED' }} / RNG SEED {{ selectedRun.launch.rng?.seed ?? 'UNKNOWN' }} / SOURCE CALLS {{ selectedRun.launch.rng?.sourceCalls ?? 'UNKNOWN' }}</p>
        </template>
        <p>Cancel Remaining Steps stops future dispatch when processed. Completed effects, the applied baseline, and production workflows are preserved. Review Rollback opens a pinned Saves review; it never loads automatically.</p>
      </section>
      <div v-if="selectedRun" class="scenario-run-history" data-scenario-run-history>
        <nav class="scenario-run-list" aria-labelledby="scenario-run-list-title"><header><strong id="scenario-run-list-title">RECENT RUNS</strong><small>{{ runHistory.length }} / {{ runStatus?.capacity || runHistory.length }}</small></header><button v-for="run in runHistory" :key="run.runId" type="button" :data-scenario-run="run.runId" :data-run-state="run.state" :aria-current="selectedRun.runId === run.runId ? 'true' : undefined" @click="emit('selectRun', run.runId)"><span :class="run.state === 'failed' ? 'failure' : run.state === 'succeeded' ? 'success' : ''">{{ run.state.toUpperCase() }}</span><strong>{{ run.reference }}</strong><small>{{ run.nextStep }} / {{ run.totalSteps }} · <time :datetime="run.finishedAt ?? run.startedAt">{{ formatTimestamp(run.finishedAt ?? run.startedAt) }}</time></small></button></nav>
        <section class="scenario-run-status scenario-run-detail" :data-run-state="selectedRun.state" :data-selected-run="selectedRun.runId"><header><strong>{{ selectedRun.reference }}</strong><span :class="selectedRun.state === 'failed' ? 'failure' : 'success'">{{ selectedRun.state.toUpperCase() }}</span></header><p>{{ selectedRun.nextStep }} / {{ selectedRun.totalSteps }} · {{ scenarioBaselineLabel(selectedRun.baseline).toUpperCase() }} BASELINE · SCENE {{ selectedRun.sceneGeneration }}</p><dl class="scenario-run-timestamps"><div><dt>STARTED</dt><dd><time :datetime="selectedRun.startedAt">{{ formatTimestamp(selectedRun.startedAt) }}</time></dd></div><div v-if="selectedRun.finishedAt"><dt>FINISHED</dt><dd><time :datetime="selectedRun.finishedAt">{{ formatTimestamp(selectedRun.finishedAt) }}</time></dd></div></dl><p v-if="selectedRun.baselineRevision"><code>{{ selectedRun.baselineReference }} @ {{ selectedRun.baselineRevision }}</code></p><p v-if="selectedRun.rollback">ROLLBACK {{ selectedRun.rollback.reference }} @ {{ selectedRun.rollback.revision }}</p><p class="scenario-retained-warning">{{ selectedRun.currentProcess ? 'RUN MUTATIONS REMAIN IN THE ACTIVE GAME STATE AFTER COMPLETION.' : 'MUTATIONS MAY HAVE AFFECTED THE GAME STATE ASSOCIATED WITH THE ORIGINAL LAUNCH.' }}</p><p v-if="selectedRun.errorCode" class="error-line">{{ selectedRun.errorCode }} :: {{ selectedRun.errorMessage }}</p><ol><li v-for="step in selectedRun.steps" :key="step.number" class="scenario-run-step" :class="[`state-${step.state}`, { 'is-current': selectedRun.currentProcess && selectedRun.state === 'running' && step.number === selectedRun.nextStep }]" :data-run-step="step.number" :data-step-state="step.state"><span>{{ String(step.number).padStart(2, '0') }}</span><strong>{{ step.commandId ?? step.result?.commandId ?? step.type.toUpperCase() }}</strong><em :class="step.state === 'failed' ? 'failure' : ''">{{ step.state.toUpperCase() }}</em><p v-if="step.note">{{ step.note }}</p><p v-if="step.result?.summary">{{ step.result.summary }}</p><p v-if="step.result && !step.result.success" class="error-line">{{ step.result.errorCode }} :: {{ step.result.errorMessage }}</p><ul v-if="step.result?.changes?.length" class="scenario-run-changes"><li v-for="change in step.result.changes" :key="change.path"><code>{{ change.path }}</code> {{ displayValue(change.before) }} → {{ displayValue(change.after) }}</li></ul></li></ol><footer class="scenario-detail-actions"><button type="button" data-scenario-action="run-again" @click="emit('playDefinition', selectedRun.reference)">RUN AGAIN</button><button type="button" data-scenario-action="open-from-run" @click="emit('getDefinition', selectedRun.reference)">OPEN SCENARIO</button><button type="button" @click="navigate('library')">LIBRARY</button></footer></section>
      </div>
      <p v-else class="empty-state">NO DURABLE SCENARIO RUNS YET</p>
      <p v-if="runStatus?.historyError" class="error-line">RUN HISTORY ERROR :: {{ runStatus.historyError }}</p>
    </section>

    <section v-else class="scenario-quick-setups" data-scenario-screen="quick-setups">
      <p class="scenario-utility-note">QUICK SETUPS ARE IMMUTABLE BUILT-IN TEST UTILITIES. THEY ARE NOT SAVED SCENARIO SCRIPTS.</p>
      <p v-if="quickIssue" class="error-line" role="alert">{{ quickIssue }}</p>
      <section v-if="armed || armedDeath" class="locker-review" data-quick-review>
        <p v-if="armed">REVIEWED {{ armed.operation.toUpperCase() }}: {{ ['direct', 'hourly', 'bridge'].includes(armed.operation) ? `SEED ${armed.seed}` : armed.operation === 'finale' ? `WOODEN BOX ${armed.woodenBox ? 'OWNED' : 'ABSENT'}` : `PARTY ${armed.partyFingerprint}` }}</p>
        <p v-else-if="armedDeath">REVIEWED {{ armedDeath.operation }}: PARTY {{ armedDeath.fingerprint }}</p>
        <button type="button" @click="cancelQuickReview">CANCEL REVIEW</button>
      </section>
      <template v-if="data"><div class="scenario-context"><div><span>LOCATION</span><strong>{{ data.locationName }}</strong><small>{{ data.mapType }} · FLOOR {{ data.floor }}</small></div><div><span>TIME</span><strong>{{ data.time }}</strong><small>{{ data.date }}</small></div><div><span>FOOD</span><strong>{{ data.food }}</strong><small>{{ data.allPartyDead ? 'PARTY DEAD' : 'PARTY ACTIVE' }}</small></div><div><span>CONTEXT</span><strong>{{ blocked ? 'BLOCKED' : 'READY' }}</strong><small>{{ data.gameplayTerminal ? 'TERMINAL FINALE' : data.finaleActive ? 'FINALE HANDOFF' : data.combatActive ? 'COMBAT' : data.dialogActive ? 'DIALOG' : data.gameplayWorkflowActive ? 'WORKFLOW' : data.scriptedSceneActive ? 'SCRIPTED SCENE' : 'SAFE TO RUN' }}</small></div></div>
        <section class="scenario-party"><header><strong>ACTIVE PARTY BASELINE</strong><small>{{ updatedAt ? `SYNC ${updatedAt.toLocaleTimeString([], { hour12: false })}` : 'WAITING' }}</small></header><div v-for="member in data.party" :key="member.partyPosition" class="scenario-party-row"><strong>P{{ member.partyPosition }} · {{ member.name }}</strong><span>HP {{ member.currentHp }}/{{ member.maximumHp }}</span><span>MP {{ member.currentMp }}</span><span>XP {{ member.experience }}</span><small>{{ member.status }}</small></div></section>
        <div class="scenario-grid">
          <section class="scenario-card hole-up-card"><header><strong>HOLE UP</strong><small>PRODUCTION CAMPING WORKFLOW</small></header><p>Queue camping with explicit encounter branches against this reviewed active party.</p><label>HOURS<input v-model.number="campingHours" type="number" min="1" max="24" /></label><label>GUARD<select v-model="campingGuard"><option value="">No guard</option><option v-for="member in data.party" :key="member.rosterIndex" :value="String(member.rosterIndex)">{{ member.name }}</option></select></label><label><input v-model="forceWanderer" type="checkbox" /> FORCE WANDERER</label><label><input v-model="forceAmbush" type="checkbox" /> FORCE AMBUSH</label><button type="button" :disabled="blocked || pendingHoleUp || campingHours < 1 || campingHours > 24" @click="emit('run', { commandID: 'command-test.hole-up', argumentsByID: { hours: String(campingHours), forceWanderer: String(forceWanderer), forceAmbush: String(forceAmbush), expectedPartyFingerprint: data.partyFingerprint, ...(campingGuard ? { guardRosterIndex: campingGuard } : {}) } })">{{ pendingHoleUp ? 'QUEUEING...' : 'QUEUE CAMPING' }}</button></section>
          <section class="scenario-card danger-card"><header><strong>DIRECT STARVATION</strong><small>FOOD/TIME PRESERVED</small></header><p>Apply one seeded 1-8 HP starvation roll to every living active character.</p><label>SEED<input v-model="seed" type="number" min="0" max="2147483647" /></label><button type="button" :disabled="blocked || pendingDirect" @click="stage('direct')">{{ pendingDirect ? 'RUNNING...' : armed?.operation === 'direct' ? 'CONFIRM DAMAGE' : 'REVIEW DAMAGE' }}</button></section>
          <section class="scenario-card danger-card"><header><strong>HOURLY STARVATION</strong><small>DESTRUCTIVE SETUP</small></header><p>Set food to zero, set the hour to 12, and execute the authentic hourly meal check.</p><label>SEED<input v-model="seed" type="number" min="0" max="2147483647" /></label><button type="button" :disabled="blocked || pendingHourly" @click="stage('hourly')">{{ pendingHourly ? 'RUNNING...' : armed?.operation === 'hourly' ? 'CONFIRM FOOD/TIME/DAMAGE' : 'REVIEW HOURLY CHECK' }}</button></section>
          <section class="scenario-card"><header><strong>TOWN GUARD HOSTILITY</strong><small>{{ data.guards.supported ? 'SMALL MAP' : 'UNAVAILABLE' }}</small></header><p v-if="data.guards.supported">Criminal {{ data.guards.criminal ? 'YES' : 'NO' }} · Hostile {{ data.guards.hostilePotentialGuardCount }}/{{ data.guards.potentialGuardCount }}</p><p v-else>{{ data.guards.unavailableReason }}</p><button type="button" :disabled="blocked || pendingGuards || !data.guards.supported || data.guards.potentialGuardCount === 0" @click="emit('run', { commandID: 'scenario.guards.activate', argumentsByID: {} })">{{ pendingGuards ? 'ACTIVATING...' : 'ACTIVATE GUARDS' }}</button></section>
          <section class="scenario-card waterfall-setup-card"><header><strong>WATERFALL ENTRY</strong><small>AUTHENTIC TRIGGER</small></header><p>Move one step before the original waterfall tile. Walk south once to begin the production hazard workflow.</p><button type="button" :disabled="blocked || pendingWaterfall" @click="emit('run', { commandID: 'scenario.waterfall.setup', argumentsByID: {} })">{{ pendingWaterfall ? 'POSITIONING...' : 'SET UP WATERFALL' }}</button></section>
          <section class="scenario-card bridge-troll-card"><header><strong>BRIDGE TROLL</strong><small>PRODUCTION SNEAK / ATTACK</small></header><p>Move onto an original troll bridge, force the first sneak check to fail, and remove all gold. Refuse the toll or try to pay to enter combat.</p><label>SEED<input v-model="seed" type="number" min="0" max="2147483647" /></label><button type="button" :disabled="blocked || pendingBridgeTroll" @click="stage('bridge')">{{ pendingBridgeTroll ? 'STARTING...' : armed?.operation === 'bridge' ? 'CONFIRM GOLD / DEX / POSITION' : 'REVIEW SNEAK + ATTACK' }}</button></section>
          <section class="scenario-card danger-card blackthorn-interrogation-card"><header><strong>BLACKTHORN INTERROGATION</strong><small>PRODUCTION PRISON WORKFLOW</small></header><p>Open the Shrine of Honesty, move to Blackthorn's Palace, and begin the full mantra interrogation. Answers can ruin the shrine and martyr a companion.</p><button type="button" class="danger-button" :disabled="blocked || pendingBlackthornInterrogation" @click="stage('blackthorn')">{{ pendingBlackthornInterrogation ? 'STARTING...' : armed?.operation === 'blackthorn' ? 'CONFIRM PALACE / SHRINE / PARTY' : 'REVIEW INTERROGATION' }}</button></section>
          <section class="scenario-card danger-card lord-british-finale-card"><header><strong>LORD BRITISH FINALE</strong><small>TERMINAL PRODUCTION WORKFLOW</small></header><p>Begin the rescue-room sequence and ending. Sandalwood Box ownership determines whether an accepted Yes answer can reach the success ending.</p><label>SANDALWOOD BOX<select v-model="finaleWoodenBox"><option :value="true">OWNED</option><option :value="false">ABSENT</option></select></label><button type="button" class="danger-button" :disabled="blocked || pendingLordBritishFinale" @click="stage('finale')">{{ pendingLordBritishFinale ? 'STARTING...' : armed?.operation === 'finale' ? 'CONFIRM TERMINAL FINALE' : 'REVIEW ENDING' }}</button></section>
          <section class="scenario-card word-setup-card"><header><strong>WORD OF POWER</strong><small>PRODUCTION YELL</small></header><p>Move beside the original dungeon entrance, then use Yell normally.</p><label for="scenario-word">WORD<select id="scenario-word" v-model="word"><option v-for="choice in wordChoices" :key="choice.value" :value="choice.value">{{ choice.label }}</option></select></label><button type="button" :disabled="blocked || pendingWord" @click="emit('run', { commandID: 'scenario.word.setup', argumentsByID: { word } })">{{ pendingWord ? 'POSITIONING...' : 'SET UP DUNGEON' }}</button></section>
          <section class="scenario-card danger-card death-wipe-card"><header><strong>KILL ACTIVE PARTY</strong><small>ATOMIC WIPE ONLY</small></header><p>Set the reviewed active party to zero HP and Dead without starting resurrection.</p><small>{{ data.wipeReview.members.length }} MEMBERS · REVIEW {{ data.wipeReview.fingerprint.slice(0, 12) }}</small><button type="button" class="danger-button" :disabled="blocked || pendingKillParty || data.wipeReview.members.length === 0" @click="stageDeath('kill-party')">{{ pendingKillParty ? 'WIPING...' : armedDeath?.operation === 'kill-party' ? 'CONFIRM DESTRUCTIVE WIPE' : 'REVIEW PARTY WIPE' }}</button></section>
          <section class="scenario-card danger-card death-sequence-card"><header><strong>DEATH / RESURRECTION</strong><small>PRODUCTION WORKFLOW</small></header><p>Atomically wipe the reviewed party, then queue the production resurrection workflow.</p><small>{{ data.wipeReview.members.map((member) => `${member.name} ${member.hp} HP`).join(' · ') }}</small><button type="button" class="danger-button" :disabled="blocked || pendingDeathSequence || data.wipeReview.members.length === 0" @click="stageDeath('death-sequence')">{{ pendingDeathSequence ? 'QUEUEING...' : armedDeath?.operation === 'death-sequence' ? 'CONFIRM WIPE + WORKFLOW' : 'REVIEW DEATH SEQUENCE' }}</button></section>
        </div></template><p v-else class="empty-state">WAITING FOR SCENARIO STATUS</p>
    </section>

    <div v-if="reviewPanel" v-owned-modal="() => reviewPanel = undefined" class="scenario-owned-modal">
    <section v-if="reviewPanel === 'project-write'" class="scenario-review-panel" role="dialog" aria-modal="true" aria-labelledby="scenario-project-review-title" data-scenario-review="project-write"><h4 id="scenario-project-review-title">{{ composer.savedReference ? 'REVIEW SHARED PROJECT UPDATE' : 'CREATE SHARED PROJECT SCENARIO' }}</h4><p v-if="composer.savedReference">You are replacing revision <code>{{ composer.loadedRevision }}</code> for <strong>{{ composer.savedReference }}</strong>. Other developers may consume this shared definition.</p><p v-else>This creates <strong>project:{{ composer.definition.id }}</strong> in the repository-tracked shared scenario library.</p><div><button type="button" @click="reviewPanel = undefined">KEEP EDITING</button><button type="button" data-scenario-action="confirm-project-write" @click="reviewPanel = undefined; emit('saveDefinition')">{{ composer.savedReference ? 'VALIDATE & UPDATE PROJECT' : 'VALIDATE & CREATE PROJECT SCENARIO' }}</button></div></section>
    <section v-if="reviewPanel === 'delete'" class="scenario-review-panel" role="dialog" aria-modal="true" aria-labelledby="scenario-delete-title" data-scenario-review="delete"><h4 id="scenario-delete-title">DELETE {{ composer.savedReference }}</h4><p>{{ composer.source === 'project' ? `Type the exact ID ${composer.definition.id} to delete this shared definition.` : `Type the scenario name ${composer.definition.name} to confirm deletion.` }}</p><label>CONFIRMATION<input v-model="confirmation" data-scenario-delete-confirmation /></label><div><button type="button" @click="reviewPanel = undefined">CANCEL</button><button type="button" class="danger-button" data-scenario-action="confirm-delete" :disabled="!deleteConfirmationValid" @click="reviewPanel = undefined; emit('deleteDefinition')">DELETE DEFINITION</button></div></section>
    <section v-if="reviewPanel === 'duplicate'" class="scenario-review-panel" role="dialog" aria-modal="true" aria-labelledby="scenario-duplicate-title" data-scenario-review="duplicate"><h4 id="scenario-duplicate-title">DUPLICATE REVIEWED REVISION</h4><p>The source revision is fixed at <code>{{ composer.loadedRevision }}</code>. The destination must not already exist.</p><label>DESTINATION<select v-model="duplicateSource"><option value="personal">Personal / Local</option><option value="project">Project / Shared</option></select></label><label>ID<input v-model="duplicateID" data-scenario-duplicate-id pattern="[a-z][a-z0-9]*(?:-[a-z0-9]+)*" /></label><label>NAME<input v-model="duplicateName" data-scenario-duplicate-name /></label><p v-if="duplicateSource === 'project'" class="scenario-retained-warning">This duplicate will be repository-tracked and shared with other developers.</p><label v-if="duplicateSource === 'project'">TYPE NEW PROJECT ID<input v-model="duplicateProjectConfirmation" data-scenario-duplicate-project-confirmation :placeholder="duplicateID" /></label><div><button type="button" @click="reviewPanel = undefined">CANCEL</button><button type="button" data-scenario-action="confirm-duplicate" :disabled="!duplicateConfirmationValid" @click="confirmDuplicate">{{ duplicateSource === 'project' ? 'CREATE SHARED PROJECT DUPLICATE' : 'CREATE PERSONAL DUPLICATE' }}</button></div></section>
    </div>
  </section>
</template>
