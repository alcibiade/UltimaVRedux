import { describe, expect, it } from "vitest"

import {
  compatibleScenarioParameters,
  newScenarioComposerDraft,
  hydrateScenarioComposer,
  scenarioCommandArguments,
  scenarioDraftIsDirty,
  scenarioDraftHasPendingMigrations,
  scenarioIDFromName,
  scenarioComposerSupportsParameter,
  scenarioLaunchValueIssues,
  scenarioLiteralDisplay,
  scenarioLiteralValue,
  scenarioSafeCommands,
  insertScenarioStep,
  scenarioTerminalStepIndex,
  scenarioStepDisplay,
  scenarioStepFromInvocation,
  serializeScenarioDefinition,
} from "./scenarios"
import type { ScenarioStep } from "./scenarios"
import type { CommandDefinition, ExecutionMode } from "./protocol"

describe("scripted scenario helpers", () => {
  it("preserves typed invocation arguments and serializes the exact version 2 definition", () => {
    const draft = newScenarioComposerDraft()
    draft.definition = {
      schemaVersion: 2,
      id: "typed-camping",
      name: "Typed Camping",
      description: "Retains normalized invocation value types.",
      baseline: { type: "current" },
      parameters: [{ id: "hours", name: "Hours", kind: "integer", required: true, default: 8, minimum: 1, maximum: 24 }],
      steps: [scenarioStepFromInvocation({
        commandId: "command-test.hole-up",
        arguments: { hours: 8, forceWanderer: true, label: "night" },
      })],
    }
    const command = draft.definition.steps[0]
    if (command?.type !== "command") throw new Error("expected command step")
    command.arguments = { ...command.arguments, hours: { parameter: "hours" } }

    const serialized = serializeScenarioDefinition(draft)

    expect(JSON.parse(serialized)).toEqual({
      schemaVersion: 2,
      id: "typed-camping",
      name: "Typed Camping",
      description: "Retains normalized invocation value types.",
      baseline: { type: "current" },
      parameters: [{ id: "hours", name: "Hours", kind: "integer", required: true, default: 8, minimum: 1, maximum: 24 }],
      steps: [{
        type: "command",
        commandId: "command-test.hole-up",
        arguments: { hours: { parameter: "hours" }, forceWanderer: true, label: "night" },
      }],
    })
    expect(serialized).not.toContain('"8"')
    expect(serialized).not.toContain('"true"')
  })

  it("omits runtime-managed arguments from captured and newly authored scenario steps", () => {
	const command: CommandDefinition = {
		id: "party.fill",
		category: "Party",
		name: "Fill Party",
		description: "Fill the active party.",
		executionModes: ["coordinated_alteration"],
		destructive: true,
		parameters: [
			{ id: "expectedFingerprint", name: "Reviewed Roster", kind: "string", required: true, scenarioRuntime: true },
			{ id: "rosterIndex", name: "Roster Index", kind: "integer", required: false, default: 3 },
		],
	}

	expect(scenarioStepFromInvocation({
		commandId: command.id,
		arguments: { expectedFingerprint: "transient-hash", rosterIndex: 4 },
	}, command)).toEqual({
		type: "command",
		commandId: command.id,
		arguments: { rosterIndex: 4 },
	})
	expect(scenarioCommandArguments(command)).toEqual({ rosterIndex: 3 })
  })

  it("uses generic party positions for scenario-targeted character arguments", () => {
    const command: CommandDefinition = {
      id: "party.character.update", category: "Party", name: "Character Alteration", description: "Update a character.",
      executionModes: ["live_safe"], destructive: false,
      parameters: [{
        id: "character", name: "Character", kind: "character", required: true,
        optionsProvider: "party.active-characters",
        scenarioChoices: [
          { value: "all", label: "All active party members" },
          ...Array.from({ length: 6 }, (_, index) => ({ value: String(index + 1), label: `Party Position ${index + 1}` })),
        ],
        scenarioValueResolver: "active-party-position",
      }],
    }

    expect(scenarioStepFromInvocation({ commandId: command.id, arguments: { character: "0.QXZhdGFy" } }, command)).toEqual({
      type: "command", commandId: command.id, arguments: { character: "1" },
    })
    expect(scenarioStepFromInvocation({ commandId: command.id, arguments: { character: "all" } }, command)).toEqual({
      type: "command", commandId: command.id, arguments: { character: "all" },
    })
    expect(scenarioCommandArguments(command)).toBeUndefined()
    expect(scenarioStepDisplay({ type: "command", commandId: command.id, arguments: { character: "2" } }, [], [command], {
      "party.active-characters": [{ value: "1.SW9sbw", label: "Position 2 · Iolo" }],
    }).fullDetail).toBe('Character: Party Position 2 ("2")')
  })

  it("inserts setup steps before a catalog-defined terminal command", () => {
    const commands: CommandDefinition[] = [
      { id: "setup", category: "Test", name: "Setup", description: "", executionModes: ["live_safe"], destructive: false },
      { id: "combat.encounter.start", category: "Combat", name: "Start Encounter", description: "", executionModes: ["synchronized_live"], destructive: false, scenarioTerminal: true },
    ]
    const steps = [
      { type: "command", commandId: "setup" },
      { type: "command", commandId: "combat.encounter.start" },
    ] as ScenarioStep[]

    const insertion = insertScenarioStep(steps, { type: "command", commandId: "party.character.update", arguments: { currentHp: 90 } }, commands)

    expect(insertion.index).toBe(1)
    expect(insertion.steps.map((step) => step.type === "command" ? step.commandId : step.type)).toEqual([
      "setup", "party.character.update", "combat.encounter.start",
    ])
    expect(scenarioTerminalStepIndex(insertion.steps, commands)).toBe(2)

		const requestedFirst = insertScenarioStep(steps, { type: "note", note: "Prepare" }, commands, 0)
		expect(requestedFirst.index).toBe(0)
		expect(requestedFirst.steps[0]).toEqual({ type: "note", note: "Prepare" })

		const requestedPastTerminal = insertScenarioStep(steps, { type: "note", note: "Before combat" }, commands, 99)
		expect(requestedPastTerminal.index).toBe(1)
		expect(requestedPastTerminal.steps[1]).toEqual({ type: "note", note: "Before combat" })

		const terminalWithoutExistingTerminal = insertScenarioStep(
			[{ type: "command", commandId: "setup" }],
			{ type: "command", commandId: "combat.encounter.start" },
			commands,
			0,
		)
		expect(terminalWithoutExistingTerminal.index).toBe(1)
		expect(terminalWithoutExistingTerminal.steps[1]).toEqual({ type: "command", commandId: "combat.encounter.start" })
  })

  it("admits scripted-scene workflows and keeps the finale terminal", () => {
    const commands: CommandDefinition[] = [
      { id: "scenario.blackthorn-interrogation.start", category: "Scenarios", name: "Blackthorn", description: "", executionModes: ["workflow"], destructive: true },
      { id: "scenario.lord-british-finale.start", category: "Scenarios", name: "Finale", description: "", executionModes: ["workflow"], destructive: true, scenarioTerminal: true },
    ]

    expect(scenarioSafeCommands(commands).map((command) => command.id)).toEqual([
      "scenario.blackthorn-interrogation.start",
      "scenario.lord-british-finale.start",
    ])
    expect(scenarioTerminalStepIndex([
      { type: "command", commandId: "scenario.blackthorn-interrogation.start" },
      { type: "command", commandId: "scenario.lord-british-finale.start" },
    ], commands)).toBe(1)
  })

  it("explicitly limits the Phase 1 composer to safely editable parameter kinds", () => {
    expect(scenarioComposerSupportsParameter("string")).toBe(true)
    expect(scenarioComposerSupportsParameter("integer")).toBe(true)
    expect(scenarioComposerSupportsParameter("boolean")).toBe(true)
    expect(scenarioComposerSupportsParameter("integer_list")).toBe(false)
    expect(scenarioComposerSupportsParameter("monster")).toBe(false)
  })

  it("derives stable personal scenario IDs from names", () => {
    expect(scenarioIDFromName(" Waterfall Regression ")).toBe("waterfall-regression")
    expect(scenarioIDFromName("5 Minute Test")).toBe("scenario-5-minute-test")
    expect(scenarioIDFromName("---")).toBe("")
  })

  it("tracks the loaded revision and original definition snapshot independently", () => {
    const draft = hydrateScenarioComposer({
      reference: "project:typed-camping",
      revision: "a".repeat(64),
      definition: { schemaVersion: 2, id: "typed-camping", name: "Typed Camping", description: "Original", baseline: { type: "current" }, steps: [] },
    })

    expect(draft.loadedRevision).toBe("a".repeat(64))
    expect(scenarioDraftIsDirty(draft)).toBe(false)
    draft.definition.description = "Changed"
    expect(scenarioDraftIsDirty(draft)).toBe(true)
    expect(draft.originalDefinition.description).toBe("Original")
  })

  it("hydrates migrated definitions as explicitly savable dirty drafts", () => {
    const draft = hydrateScenarioComposer({
      reference: "project:legacy-character",
      revision: "b".repeat(64),
      definition: {
        schemaVersion: 2, id: "legacy-character", name: "Legacy Character", description: "Migrated in memory.",
        baseline: { type: "current" },
        steps: [{ type: "command", commandId: "party.character.update", arguments: { character: "1", strength: 20 } }],
      },
      migrations: [{
        path: "$.steps[0].arguments.character", code: "legacy_character_token", before: "0.QXZhdGFy", after: "1",
      }],
    })

    expect(draft.originalDefinition.steps[0]).toMatchObject({ arguments: { character: "0.QXZhdGFy" } })
    expect(draft.definition.steps[0]).toMatchObject({ arguments: { character: "1" } })
    expect(scenarioDraftHasPendingMigrations(draft)).toBe(true)
    expect(scenarioDraftIsDirty(draft)).toBe(true)
    draft.originalDefinition = JSON.parse(JSON.stringify(draft.definition))
    expect(scenarioDraftHasPendingMigrations(draft)).toBe(false)
    expect(scenarioDraftIsDirty(draft)).toBe(false)
  })

  it("restores removed runtime preconditions only in the original dirty snapshot", () => {
    const draft = hydrateScenarioComposer({
      reference: "project:legacy-world-review",
      revision: "c".repeat(64),
      definition: {
        schemaVersion: 2, id: "legacy-world-review", name: "Legacy World Review", description: "Migrated in memory.",
        baseline: { type: "current" },
        steps: [{ type: "command", commandId: "world.position.set", arguments: { location: "33" } }],
      },
      migrations: [
        {
          path: "$.steps[0].arguments.expectedLocationId", code: "runtime_world_precondition",
          before: "2", after: "resolved at runtime",
        },
        {
          path: "$.steps[0].arguments.expectedSourceFingerprint", code: "runtime_world_precondition",
          before: "{\"parameter\":\"source\"}", after: "resolved at runtime",
        },
      ],
    })

    expect(draft.originalDefinition.steps[0]).toMatchObject({
      arguments: { location: "33", expectedLocationId: 2, expectedSourceFingerprint: { parameter: "source" } },
    })
    expect(draft.definition.steps[0]).toEqual({
      type: "command", commandId: "world.position.set", arguments: { location: "33" },
    })
    expect(scenarioDraftHasPendingMigrations(draft)).toBe(true)
    expect(scenarioDraftIsDirty(draft)).toBe(true)
    draft.originalDefinition = JSON.parse(JSON.stringify(draft.definition))
    expect(scenarioDraftHasPendingMigrations(draft)).toBe(false)
  })

  it("filters commands with the exact scenario execution policy", () => {
    const command = (id: string, executionModes: ExecutionMode[]): CommandDefinition => ({
      id, category: "Test", name: id, description: id, executionModes, destructive: false,
    })
    const commands = [
      command("party.food.set", ["live_safe"]),
		{ ...command("redux.preference.set", ["live_safe"]), scenarioExcluded: true },
      command("mixed.command", ["live_safe", "query"]),
      command("vfx.flash", ["presentation"]),
      command("scenario.definition.create", ["coordinated_alteration"]),
      command("scenario.starvation.trigger", ["synchronized_live"]),
    ]

    expect(scenarioSafeCommands(commands).map((candidate) => candidate.id)).toEqual([
      "party.food.set",
		"mixed.command",
      "scenario.starvation.trigger",
    ])
  })

  it("initializes only catalog defaults and validates launch values without coercing unset values", () => {
    const command: CommandDefinition = {
      id: "test.configure", category: "Test", name: "Configure", description: "Configure state.",
      executionModes: ["live_safe"], destructive: false,
      parameters: [
        { id: "requiredName", name: "Name", kind: "string", required: true },
        { id: "count", name: "Count", kind: "integer", required: false, default: 0, minimum: 0, maximum: 4 },
        { id: "enabled", name: "Enabled", kind: "boolean", required: false, default: false },
      ],
    }

    expect(scenarioCommandArguments(command)).toEqual({ count: 0, enabled: false })
    expect(scenarioLaunchValueIssues(command.parameters ?? [], {})).toEqual({ requiredName: "A value is required." })
    expect(scenarioLaunchValueIssues(command.parameters ?? [], { requiredName: "Avatar", count: 5 })).toEqual({ count: "Must be at most 4." })
  })

  it("only offers launch parameters whose kinds and constraints narrow the command argument", () => {
    const target = { id: "hours", name: "Hours", kind: "integer" as const, required: true, minimum: 1, maximum: 24 }
    const candidates = [
      { id: "safe", name: "Safe", kind: "integer" as const, required: true, minimum: 1, maximum: 12 },
      { id: "optional", name: "Optional", kind: "integer" as const, required: false, minimum: 1, maximum: 12 },
      { id: "wide", name: "Wide", kind: "integer" as const, required: true, minimum: 0, maximum: 24 },
      { id: "text", name: "Text", kind: "string" as const, required: true },
    ]

    expect(compatibleScenarioParameters(target, candidates).map((parameter) => parameter.id)).toEqual(["safe"])
  })

  it("parses and displays integer-list literals without losing an invalid editing value", () => {
    const parameter = { id: "indices", name: "Indices", kind: "integer_list" as const, required: true, minimum: 0, maximum: 8 }
    expect(scenarioLiteralValue(parameter, "1, 3,8")).toEqual([1, 3, 8])
    expect(scenarioLiteralDisplay([1, 3, 8])).toBe("1,3,8")
    expect(scenarioLiteralValue(parameter, "1,nope")).toBe("1,nope")
  })

  it("summarizes typed literals, catalog choices, parameter references, and unknown commands", () => {
    const command: CommandDefinition = {
      id: "test.configure", category: "Test", name: "Configure State", description: "Configure state.",
      executionModes: ["live_safe"], destructive: false,
      parameters: [
        { id: "mode", name: "Mode", kind: "enum", required: true, choices: [{ value: "safe", label: "Safe" }] },
        { id: "target", name: "Target", kind: "character", required: true, optionsProvider: "party" },
        { id: "enabled", name: "Enabled", kind: "boolean", required: true },
        { id: "count", name: "Count", kind: "integer", required: true },
      ],
    }
    const display = scenarioStepDisplay({
      type: "command",
      commandId: command.id,
      arguments: { mode: "safe", target: "avatar", enabled: true, count: { parameter: "amount" }, legacy: [1, 2] },
    }, [{ id: "amount", name: "Amount", kind: "integer", required: true }], [command], {
      party: [{ value: "avatar", label: "Avatar" }],
    })

    expect(display.commandName).toBe("Configure State")
    expect(display.fullDetail).toBe('Mode: Safe ("safe") · Target: Avatar ("avatar") · Enabled: TRUE · Count: $Amount [amount] · legacy: [1,2]')
    expect(display.unknownCommand).toBe(false)
    expect(scenarioStepDisplay({ type: "command", commandId: "missing.command", arguments: { value: "kept" } }, [], [], {}).unknownCommand).toBe(true)
  })
})
