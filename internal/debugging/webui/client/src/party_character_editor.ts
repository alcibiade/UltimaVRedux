import type { Choice, CommandDefinition } from "./protocol"
import type { CommandDraft, OptionSets } from "./command_forms"

export const partyCharacterCommandID = "party.character.update"
export const partyCharacterFieldIDs = ["strength", "dexterity", "intelligence", "currentHp", "currentMp", "experience", "level", "status", "partyPosition"] as const

export interface PartyCharacterMetadata {
  partyPosition: number
  name: string
  class: string
  status: string
  strength: number
  dexterity: number
  intelligence: number
  currentMp: number
  currentHp: number
  maximumHp: number
  experience: number
  level: number
  karma: number
}

export interface PartyAlterationPreviewChange {
  label: string
  before: string | number
  after: string | number
}

export function characterChoices(command: CommandDefinition, optionSets: OptionSets): Choice[] {
  const parameter = command.parameters?.find((candidate) => candidate.id === "character")
  return parameter?.optionsProvider ? optionSets[parameter.optionsProvider] ?? [] : []
}

export function positionChoices(command: CommandDefinition, optionSets: OptionSets, draft: CommandDraft): Choice[] {
  const parameter = command.parameters?.find((candidate) => candidate.id === "partyPosition")
  const choices = parameter?.optionsProvider ? optionSets[parameter.optionsProvider] ?? [] : parameter?.choices ?? []
  const currentPosition = selectedCharacterMetadata(command, optionSets, draft)?.partyPosition
  if (!currentPosition) return choices

  return currentPosition === 1
    ? choices.filter((choice) => choice.value === "1")
    : choices.filter((choice) => choice.value !== "1")
}

export function selectedCharacterChoice(command: CommandDefinition, optionSets: OptionSets, draft: CommandDraft): Choice | undefined {
  return characterChoices(command, optionSets).find((choice) => choice.value === draft.character)
}

export function selectedCharacterMetadata(
  command: CommandDefinition,
  optionSets: OptionSets,
  draft: CommandDraft,
): PartyCharacterMetadata | undefined {
  return selectedCharacterChoice(command, optionSets, draft)?.metadata as PartyCharacterMetadata | undefined
}

export function hydratePartyCharacterDraft(
  command: CommandDefinition,
  characterValue: string,
  optionSets: OptionSets,
): CommandDraft {
  const choice = characterChoices(command, optionSets).find((candidate) => candidate.value === characterValue)
  const metadata = choice?.metadata as PartyCharacterMetadata | undefined
  if (!metadata) return { character: characterValue }

  return { character: characterValue }
}

export function reconcilePartyCharacterDraft(
  command: CommandDefinition,
  existing: CommandDraft,
  existingDirtyFields: string[],
  optionSets: OptionSets,
): { draft: CommandDraft; dirtyFields: string[] } {
  const choices = characterChoices(command, optionSets)
  const selected = choices.some((choice) => choice.value === existing.character) ? existing.character : choices[0]?.value ?? ""
  if (!selected) return { draft: { character: "" }, dirtyFields: [] }

  const draft = hydratePartyCharacterDraft(command, selected, optionSets)
  const dirtyFields = selected === existing.character
    ? existingDirtyFields.filter((fieldID) => partyCharacterFieldIDs.includes(fieldID as typeof partyCharacterFieldIDs[number]))
    : []
  for (const fieldID of dirtyFields) draft[fieldID] = existing[fieldID] ?? draft[fieldID]

  return { draft, dirtyFields }
}

export function partyFieldIsDirty(
  command: CommandDefinition,
  optionSets: OptionSets,
  draft: CommandDraft,
  fieldID: string,
  value: string,
): boolean {
  const baseline = hydratePartyCharacterDraft(command, draft.character ?? "", optionSets)
  return value !== "" && baseline[fieldID] !== value
}

export function serializePartyCharacterDraft(draft: CommandDraft, dirtyFields: string[]): Record<string, string | number> {
  const argumentsByID: Record<string, string | number> = { character: draft.character }
  for (const fieldID of dirtyFields) {
    const value = draft[fieldID]
    if (value === undefined || value === "") continue
    if (fieldID === "experience" && dirtyFields.includes("level")) continue
    argumentsByID[fieldID] = ["strength", "dexterity", "intelligence", "currentHp", "currentMp", "experience", "level"].includes(fieldID) ? Number(value) : value
  }

  return argumentsByID
}

export function partyAlterationPreview(
  command: CommandDefinition,
  optionSets: OptionSets,
  draft: CommandDraft,
  dirtyFields: string[],
): PartyAlterationPreviewChange[] {
  const metadata = selectedCharacterMetadata(command, optionSets, draft)
  if (!metadata) return []
  const dirty = new Set(dirtyFields)
  const changes: PartyAlterationPreviewChange[] = []
  for (const [parameterID, label, metadataID] of [
    ["strength", "Strength", "strength"],
    ["dexterity", "Dexterity", "dexterity"],
    ["intelligence", "Intelligence", "intelligence"],
  ] as const) {
    if (dirty.has(parameterID)) changes.push({ label, before: metadata[metadataID], after: Number(draft[parameterID]) })
  }
  for (const [parameterID, label, metadataID] of [
    ["currentHp", "Current HP", "currentHp"],
    ["currentMp", "Current MP", "currentMp"],
    ["experience", "Experience", "experience"],
  ] as const) {
    if (dirty.has(parameterID) && !(parameterID === "experience" && dirty.has("level"))) {
      changes.push({ label, before: metadata[metadataID], after: Number(draft[parameterID]) })
    }
  }
  if (dirty.has("level")) {
    const level = Number(draft.level)
    const maximumHp = level * 30
    const experience = level <= 1 ? 0 : 100 * (2 ** (level - 2))
    changes.push({ label: "Level", before: metadata.level, after: level })
    changes.push({ label: "Maximum HP", before: metadata.maximumHp, after: maximumHp })
    changes.push({ label: "Experience", before: metadata.experience, after: experience })
    if (!dirty.has("currentHp") && metadata.currentHp > maximumHp) {
      changes.push({ label: "Current HP", before: metadata.currentHp, after: maximumHp })
    }
  }
  if (dirty.has("status")) {
    const statusParameter = command.parameters?.find((parameter) => parameter.id === "status")
    const status = statusParameter?.choices?.find((choice) => choice.value === draft.status)?.label ?? draft.status
    changes.push({ label: "Status", before: metadata.status, after: status })
  }
  if (dirty.has("partyPosition")) appendPartyOrderPreview(changes, command, optionSets, draft, metadata)
  if (dirty.has("intelligence") && !dirty.has("currentMp") && metadata.currentMp > Number(draft.intelligence)) {
    changes.push({ label: "Current MP", before: metadata.currentMp, after: Number(draft.intelligence) })
  }

  return changes.filter((change) => change.before !== change.after)
}

export function partyResourceErrors(
  command: CommandDefinition,
  optionSets: OptionSets,
  draft: CommandDraft,
  dirtyFields: string[],
): Record<string, string> {
  const metadata = selectedCharacterMetadata(command, optionSets, draft)
  if (!metadata) return {}
  const errors: Record<string, string> = {}
  const currentHP = Number(draft.currentHp)
  const level = Number(draft.level)
  const maximumHP = dirtyFields.includes("level") && Number.isInteger(level) && level >= 1 && level <= 8
    ? level * 30
    : metadata.maximumHp
  if (dirtyFields.includes("currentHp") && Number.isInteger(currentHP) && (currentHP < 1 || currentHP > maximumHP)) {
    errors.currentHp = `Must be between 1 and ${maximumHP}.`
  }
  const currentMP = Number(draft.currentMp)
  const intelligence = dirtyFields.includes("intelligence") ? Number(draft.intelligence) : metadata.intelligence
  if (dirtyFields.includes("currentMp") && Number.isInteger(currentMP) && Number.isInteger(intelligence) && (currentMP < 0 || currentMP > intelligence)) {
    errors.currentMp = `Must be between 0 and ${intelligence}.`
  }

  return errors
}

function appendPartyOrderPreview(
  changes: PartyAlterationPreviewChange[],
  command: CommandDefinition,
  optionSets: OptionSets,
  draft: CommandDraft,
  metadata: PartyCharacterMetadata,
): void {
  const destination = positionChoices(command, optionSets, draft).find((choice) => choice.value === draft.partyPosition)
  const destinationMetadata = destination?.metadata as PartyCharacterMetadata | undefined
  const destinationPosition = Number(draft.partyPosition)
  changes.push({ label: "Party position", before: metadata.partyPosition, after: destinationPosition })
  if (metadata.partyPosition === destinationPosition || !destinationMetadata) return
  changes.push({ label: `Position ${metadata.partyPosition}`, before: metadata.name, after: destinationMetadata.name })
  changes.push({ label: `Position ${destinationPosition}`, before: destinationMetadata.name, after: metadata.name })
}
