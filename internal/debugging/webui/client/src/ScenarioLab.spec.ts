import { createApp, nextTick, ref, type App as VueApp } from "vue"
import { afterEach, beforeEach, describe, expect, it } from "vitest"

import ScenarioLab from "./ScenarioLab.vue"
import type { OptionSets } from "./command_forms"
import type { CommandDefinition } from "./protocol"
import type { SaveSnapshotSummary } from "./saves"
import { hydrateScenarioComposer, newScenarioComposerDraft, type ScenarioComposerDraft, type ScenarioRunStatus, type ScenarioTask } from "./scenarios"

let app: VueApp<Element> | undefined
let root: HTMLElement

const editorCommands: CommandDefinition[] = [
  {
    id: "test.configure", category: "Test", name: "Configure State", description: "Configure test state.", executionModes: ["live_safe"], destructive: false,
    parameters: [
      { id: "name", name: "Name", description: "Required label.", kind: "string", required: true },
      { id: "count", name: "Count", kind: "integer", required: false, default: 2, minimum: 1, maximum: 8 },
      { id: "enabled", name: "Enabled", kind: "boolean", required: false, default: false },
      { id: "mode", name: "Mode", kind: "enum", required: true, default: "safe", choices: [{ value: "safe", label: "Safe" }, { value: "fast", label: "Fast" }] },
    ],
  },
  { id: "test.other", category: "Test", name: "Other Command", description: "Replacement.", executionModes: ["workflow"], destructive: false, parameters: [{ id: "count", name: "Count", kind: "string", required: true }] },
  { id: "test.query", category: "Test", name: "Query", description: "Not safe.", executionModes: ["query"], destructive: false },
  { id: "test.presentation", category: "Test", name: "Presentation", description: "Not safe.", executionModes: ["presentation"], destructive: false },
  { id: "scenario.definition.create", category: "Scenarios", name: "Create Scenario", description: "Management.", executionModes: ["coordinated_alteration"], destructive: false },
]

beforeEach(() => {
  root = document.createElement("div")
  document.body.append(root)
})

afterEach(() => {
  app?.unmount()
  app = undefined
  root.remove()
})

describe("Scenarios workspace", () => {
  it("defaults to Library, groups sources, filters definitions, and opens Details", async () => {
    mountScenarioLab(newScenarioComposerDraft(), "library", undefined, {
      definitions: [
        { reference: "project:shared", source: "project", id: "shared", name: "Shared Setup", description: "Team baseline", baseline: { type: "init" }, revision: "a".repeat(64), steps: [] },
        { reference: "personal:local", source: "personal", id: "local", name: "Local Setup", description: "Private iteration", baseline: { type: "current" }, revision: "b".repeat(64), steps: [] },
      ],
    })

    expect(root.querySelector('[data-scenario-task="library"]')?.getAttribute("aria-current")).toBe("page")
    expect(root.querySelector('[data-scenario-source="project"]')?.textContent).toContain("Project / Shared")
    expect(root.querySelector('[data-scenario-source="personal"]')?.textContent).toContain("Personal / Local")
    setInput('[data-scenario-search]', "private")
    await nextTick()
    expect(root.querySelector('[data-scenario-reference="project:shared"]')).toBeNull()
    expect(root.querySelector('[data-scenario-reference="personal:local"]')).not.toBeNull()
    expect(root.querySelector('[data-scenario-play="personal:local"]')?.textContent).toContain("RUN")
    expect(root.querySelector('[data-scenario-play="personal:local"]')?.getAttribute("aria-label")).toBe("Run Local Setup")

    click('[data-scenario-reference="personal:local"]')
    expect(root.querySelector('[data-scenario-screen="library"]')).not.toBeNull()
  })

  it("expands catalog-aware Library summaries and retains full long argument values", async () => {
    const longName = "A very long scenario argument value that must remain available even when its visible summary is truncated for compact browsing"
    mountScenarioLab(newScenarioComposerDraft(), "library", undefined, {
      definitions: [{
        reference: "personal:inline", source: "personal", id: "inline", name: "Inline", description: "Summary", baseline: { type: "current" }, revision: "a".repeat(64),
        parameters: [{ id: "amount", name: "Amount", kind: "integer", required: true }],
        steps: [
          { type: "command", commandId: "test.configure", arguments: { name: longName, count: { parameter: "amount" }, enabled: false, mode: "safe" } },
          { type: "note", note: "Verify the configured state before continuing." },
        ],
      }],
    }, [], editorCommands)

    click('[data-scenario-expand="personal:inline"]')
    await nextTick()

    const summary = root.querySelector('[data-scenario-summary-step="1"]')
    expect(summary?.textContent).toContain("Configure State")
    expect(summary?.textContent).toContain("Count: $Amount [amount]")
    expect(summary?.textContent).toContain('Mode: Safe ("safe")')
    const detail = summary?.querySelector("small")
    expect(detail?.textContent).toContain("...")
    expect(detail?.getAttribute("title")).toContain(longName)
    expect(summary?.classList.contains("scenario-summary-step-command")).toBe(true)
    expect(summary?.getAttribute("data-summary-step-kind")).toBe("command")
    expect(root.querySelector('[data-scenario-summary-step="2"]')?.classList.contains("scenario-summary-step-note")).toBe(true)
  })

  it("keeps ordered editor steps functional and preserves the draft across navigation", async () => {
    const draft = newScenarioComposerDraft()
    draft.definition.steps = [
      { type: "command", commandId: "movement.free-move.set", arguments: { enabled: true } },
      { type: "command", commandId: "scenario.starvation.trigger", arguments: { seed: 7 } },
    ]
    const mounted = mountScenarioLab(draft, "editor")

    expect(root.querySelector('[data-scenario-step="0"]')?.classList.contains("scenario-step-command")).toBe(true)
    expect(root.querySelector('[data-scenario-step="0"]')?.getAttribute("data-step-kind")).toBe("command")

    click('[data-move-step-up="1"]')
    await nextTick()
    expect(root.querySelector('[data-scenario-step="0"]')?.textContent).toContain("scenario.starvation.trigger")
    click('[data-scenario-task="library"]')
    await nextTick()
    expect(root.querySelector('[data-scenario-screen="library"]')).not.toBeNull()
    expect(mounted.composer.value.definition.steps[0]).toMatchObject({ commandId: "scenario.starvation.trigger" })
  })

  it("requires a stronger review before updating a shared project revision", async () => {
    const draft = hydrateScenarioComposer({
      reference: "project:shared",
      revision: "c".repeat(64),
      definition: { schemaVersion: 2, id: "shared", name: "Shared", description: "Shared scenario", baseline: { type: "current" }, steps: [{ type: "note", note: "review" }] },
    })
    draft.definition.name = "Shared Updated"
    mountScenarioLab(draft, "editor")

    click('[data-scenario-action="save"]')
    await nextTick()
    expect(root.querySelector('[data-scenario-review="project-write"]')?.textContent).toContain("REVIEW SHARED PROJECT UPDATE")
    expect(root.querySelector('[data-scenario-review="project-write"]')?.textContent).toContain("project:shared")
  })

  it("reviews pending character-token migrations and disables duplicate until save", () => {
    const draft = hydrateScenarioComposer({
      reference: "personal:legacy-character",
      revision: "e".repeat(64),
      definition: {
        schemaVersion: 2, id: "legacy-character", name: "Legacy Character", description: "Migrated in memory.",
        baseline: { type: "current" },
        steps: [{ type: "command", commandId: "party.character.update", arguments: { character: "1" } }],
      },
      migrations: [{
        path: "$.steps[0].arguments.character", code: "legacy_character_token", before: "0.QXZhdGFy", after: "1",
      }],
    })
    mountScenarioLab(draft, "details")

    const warning = root.querySelector("[data-scenario-migration-review]")
    expect(warning?.textContent).toContain("0.QXZhdGFy to 1")
    expect(warning?.textContent).toContain("whoever occupies that party position")
    expect((root.querySelector('[data-scenario-action="duplicate"]') as HTMLButtonElement).disabled).toBe(true)
    expect((root.querySelector('[data-scenario-action="review-run"]') as HTMLButtonElement).disabled).toBe(true)
    expect(root.querySelector('[data-scenario-screen="details"]')?.textContent).toContain("SAVE FIRST")
  })

  it("presents a migrated definition as an explicitly savable editor draft", () => {
    const draft = hydrateScenarioComposer({
      reference: "personal:legacy-editor",
      revision: "f".repeat(64),
      definition: {
        schemaVersion: 2, id: "legacy-editor", name: "Legacy Editor", description: "Review before saving.",
        baseline: { type: "current" },
        steps: [{ type: "command", commandId: "party.character.update", arguments: { character: "2" } }],
      },
      migrations: [{
        path: "$.steps[0].arguments.character", code: "legacy_character_token", before: "1.SW9sbw", after: "2",
      }],
    })
    mountScenarioLab(draft, "editor")

    expect(root.querySelector("[data-scenario-migration-review]")?.textContent).toContain("Review and save this revision explicitly")
    expect(root.querySelector(".scenario-composer-actions")?.textContent).toContain("UNSAVED CHANGES")
    expect((root.querySelector('[data-scenario-action="save"]') as HTMLButtonElement).disabled).toBe(false)
  })

  it("renders validation issues beside the editor summary with exact paths", () => {
    const draft = newScenarioComposerDraft()
    draft.definition.steps = [{ type: "note", note: "invalid until command is added" }]
    mountScenarioLab(draft, "editor", undefined, undefined, [{ path: "$.steps[0].commandId", code: "command.required", message: "command ID is required" }])

    const issues = root.querySelector('[data-scenario-validation-issues]')
    expect(issues?.textContent).toContain("$.steps[0].commandId")
    expect(issues?.textContent).toContain("command ID is required")
    expect(root.querySelector('[data-scenario-step="0"]')?.classList.contains("scenario-step-note")).toBe(true)
    expect(root.querySelector('[data-scenario-step="0"]')?.classList.contains("has-issues")).toBe(true)
  })

  it("renders actual command IDs, errors, and state changes in Latest Run", () => {
    mountScenarioLab(newScenarioComposerDraft(), "latest-run", {
      run: {
        runId: "run-failure", definitionRevision: "a".repeat(64), startedAt: "2026-08-07T10:00:00Z", finishedAt: "2026-08-07T10:01:00Z", currentProcess: true,
        reference: "project:failure", definitionId: "failure", baseline: { type: "current" }, state: "failed",
        nextStep: 1, totalSteps: 1, initialSceneGeneration: 4, sceneGeneration: 4,
        errorCode: "blocked", errorMessage: "Command was blocked.",
        steps: [{ number: 1, type: "command", state: "failed", result: {
          sessionId: "scenario:project:failure", requestId: "step-1", commandId: "movement.free-move.set",
          sceneGeneration: 4, success: false, errorCode: "blocked", errorMessage: "Command was blocked.",
          changes: [{ path: "debug.freeMove", before: false, after: true }],
        } }],
      }, runs: [], capacity: 20,
    })

    const run = root.querySelector('.scenario-run-detail[data-run-state="failed"]')
    expect(run?.textContent).toContain("movement.free-move.set")
    expect(run?.textContent).toContain("blocked :: Command was blocked.")
    expect(run?.textContent).toContain("debug.freeMove")
    expect(run?.textContent).toContain("RUN MUTATIONS REMAIN IN THE ACTIVE GAME STATE")
    expect(run?.querySelector(".scenario-run-step > strong")?.textContent).toBe("movement.free-move.set")
    expect(run?.querySelector(".scenario-run-step > em")?.textContent).toBe("FAILED")
    expect(run?.querySelector(".scenario-run-step")?.classList.contains("state-failed")).toBe(true)
    expect(run?.querySelector(".scenario-run-step")?.getAttribute("data-step-state")).toBe("failed")
    expect(run?.querySelector(".scenario-run-changes li")?.classList.contains("scenario-run-step")).toBe(false)
  })

  it("marks the current pending run step for highlighted shading", () => {
    mountScenarioLab(newScenarioComposerDraft(), "latest-run", {
      run: {
        runId: "run-running", definitionRevision: "b".repeat(64), startedAt: "2026-08-07T10:00:00Z", currentProcess: true,
        reference: "personal:running", definitionId: "running", baseline: { type: "current" }, state: "running",
        nextStep: 1, totalSteps: 2, initialSceneGeneration: 4, sceneGeneration: 4,
        steps: [
          { number: 1, type: "command", commandId: "environment.wind.set", state: "pending" },
          { number: 2, type: "command", commandId: "movement.free-move.set", state: "pending" },
        ],
      }, runs: [], capacity: 20,
    })

    expect(root.querySelector('[data-run-step="1"]')?.classList.contains("is-current")).toBe(true)
    expect(root.querySelector('[data-run-step="1"]')?.classList.contains("state-pending")).toBe(true)
    expect(root.querySelector('[data-run-step="1"] strong')?.textContent).toBe("environment.wind.set")
    expect(root.querySelector('[data-run-step="2"] strong')?.textContent).toBe("movement.free-move.set")
    expect(root.querySelector('[data-run-step="2"]')?.classList.contains("is-current")).toBe(false)
  })

  it("selects durable runs by unique run ID without executing them", async () => {
    const newest = scenarioRun("run-new", "project:repeat", "succeeded", "2026-08-07T11:00:00Z")
    const older = scenarioRun("run-old", "project:repeat", "failed", "2026-08-07T10:00:00Z")
    const mounted = mountScenarioLab(newScenarioComposerDraft(), "latest-run", {
      run: newest, runs: [newest, older], capacity: 20,
    }, undefined, [], [], {}, undefined, [], "run-old")

    expect(root.querySelector('[data-scenario-run="run-old"]')?.getAttribute("aria-current")).toBe("true")
    expect(root.querySelector("[data-selected-run]")?.getAttribute("data-selected-run")).toBe("run-old")
    click('[data-scenario-run="run-new"]')
    await nextTick()
    expect(mounted.selectedRunID.value).toBe("run-new")
    expect(root.querySelector("[data-selected-run]")?.getAttribute("data-selected-run")).toBe("run-new")
  })

  it("adds only scenario-safe catalog commands and initializes defaults without inventing required values", async () => {
    const mounted = mountScenarioLab(newScenarioComposerDraft(), "editor", undefined, undefined, [], editorCommands)
    const choices = Array.from(root.querySelectorAll<HTMLOptionElement>("[data-scenario-command-choice] option")).map((option) => option.value)
    expect(choices).toContain("test.configure")
    expect(choices).not.toContain("test.query")
    expect(choices).not.toContain("test.presentation")
    expect(choices).not.toContain("scenario.definition.create")

    setSelect("[data-scenario-command-choice]", "test.configure")
    await nextTick()
		expect(mounted.composer.value.definition.steps).toHaveLength(0)
		expect((root.querySelector('[data-scenario-action="add-command"]') as HTMLButtonElement).disabled).toBe(true)
		setInput('[aria-label="Literal value for name"]', "staged")
		await nextTick()
    click('[data-scenario-action="add-command"]')
    await nextTick()

    const step = mounted.composer.value.definition.steps[0]
		expect(step).toEqual({ type: "command", commandId: "test.configure", arguments: { name: "staged", count: 2, enabled: false, mode: "safe" } })
    expect(root.querySelector('[data-scenario-argument="name"]')?.textContent).toContain("REQUIRED")
		expect((root.querySelector('[aria-label="Literal value for name"]') as HTMLInputElement).value).toBe("staged")
		expect(root.querySelector('[data-scenario-argument="name"] .scenario-literal-preview')?.textContent).toContain("STORED ASstaged")
    expect(root.querySelector('[data-scenario-argument="count"] .scenario-literal-preview')?.textContent).toContain("STORED AS2")
    expect(root.querySelector(".scenario-composer")?.textContent).not.toContain("CURRENT:")
		expect(root.querySelector('[data-scenario-argument-issue="name"]')).toBeNull()
  })

	it("searches catalog metadata and inserts a fully staged command at the chosen position", async () => {
		const draft = newScenarioComposerDraft()
		draft.definition.steps = [
			{ type: "note", note: "First checkpoint" },
			{ type: "note", note: "Last checkpoint" },
		]
		const mounted = mountScenarioLab(draft, "editor", undefined, undefined, [], editorCommands)

		setInput("[data-scenario-command-search]", "replacement")
		await nextTick()
		const choices = Array.from(root.querySelectorAll<HTMLOptionElement>("[data-scenario-command-choice] option")).map((option) => option.value)
		expect(choices).toContain("test.other")
		expect(choices).not.toContain("test.configure")
		expect(root.querySelector('optgroup[label^="Test"]')).not.toBeNull()

		setSelect("[data-scenario-command-choice]", "test.other")
		await nextTick()
		expect(root.querySelector("[data-scenario-command-staging]")?.textContent).toContain("Replacement.")
		setInput('[aria-label="Literal value for count"]', "between")
		setSelect("[data-scenario-insertion-position]", "1")
		await nextTick()
		click('[data-scenario-action="add-command"]')
		await nextTick()

		expect(mounted.composer.value.definition.steps).toEqual([
			{ type: "note", note: "First checkpoint" },
			{ type: "command", commandId: "test.other", arguments: { count: "between" } },
			{ type: "note", note: "Last checkpoint" },
		])
		expect(document.activeElement).toBe(root.querySelector('[data-scenario-step="1"]'))
	})

  it("does not require runtime-managed arguments in durable scenario steps", () => {
	const draft = newScenarioComposerDraft()
	draft.definition.steps = [{
		type: "command",
		commandId: "party.status.set",
		arguments: { rosterIndices: [0, 1], status: "good" },
	}]
	const commands: CommandDefinition[] = [{
		id: "party.status.set",
		category: "Party",
		name: "Set Party Status",
		description: "Set status for reviewed active members.",
		executionModes: ["synchronized_live"],
		destructive: true,
		parameters: [
			{ id: "expectedFingerprint", name: "Reviewed Roster", kind: "string", required: true, scenarioRuntime: true },
			{ id: "rosterIndices", name: "Roster Indices", kind: "integer_list", required: true },
			{ id: "status", name: "Status", kind: "enum", required: true, choices: [{ value: "good", label: "Good" }] },
		],
	}]

	mountScenarioLab(draft, "editor", undefined, undefined, [], commands)

	expect(root.querySelector('[data-scenario-argument="expectedFingerprint"]')).toBeNull()
	expect(root.querySelector('[data-scenario-step-issues="0"]')).toBeNull()
	expect(root.querySelector('[data-scenario-step="0"]')?.classList.contains("has-issues")).toBe(false)
  })

  it("requires explicit generic targets before character and loadout steps enter the definition", async () => {
    const command: CommandDefinition = {
      id: "party.character.update", category: "Party", name: "Character Alteration", description: "Update a character.",
      executionModes: ["live_safe"], destructive: false,
      parameters: [{
        id: "character", name: "Character", kind: "character", required: true,
        optionsProvider: "party.active-characters",
        scenarioChoices: Array.from({ length: 6 }, (_, index) => ({ value: String(index + 1), label: `Party Position ${index + 1}` })),
        scenarioValueResolver: "active-party-position",
      }],
    }
	const loadoutCommand: CommandDefinition = { ...command, id: "party.equipment.loadout.update", name: "Character Loadout" }
    const mounted = mountScenarioLab(newScenarioComposerDraft(), "editor", undefined, undefined, [], [command, loadoutCommand], {
      "party.active-characters": [{ value: "0.QXZhdGFy", label: "Position 1 · Avatar · Avatar" }],
    })

    setSelect("[data-scenario-command-choice]", command.id)
    await nextTick()
	expect(mounted.composer.value.definition.steps).toHaveLength(0)
	expect((root.querySelector('[data-scenario-action="add-command"]') as HTMLButtonElement).disabled).toBe(true)
	expect(root.querySelector("[data-scenario-command-staging]")?.textContent).toContain("COMPLETE REQUIRED ARGUMENTS")
	setSelect('[aria-label="Literal value for character"]', "2")
	await nextTick()
    click('[data-scenario-action="add-command"]')
    await nextTick()
    const select = root.querySelector<HTMLSelectElement>('[aria-label="Literal value for character"]')
    expect(Array.from(select?.options ?? []).map((option) => option.textContent)).toEqual([
      "UNSET", "Party Position 1", "Party Position 2", "Party Position 3", "Party Position 4", "Party Position 5", "Party Position 6",
    ])
    expect(mounted.composer.value.definition.steps[0]).toEqual({
      type: "command", commandId: command.id, arguments: { character: "2" },
    })
	expect(root.querySelector('[data-scenario-argument="character"]')?.textContent).toContain("resolves it against the active party")
    expect(JSON.stringify(mounted.composer.value.definition)).not.toContain("QXZhdGFy")

	setSelect("[data-scenario-command-choice]", loadoutCommand.id)
	await nextTick()
	expect((root.querySelector('[data-scenario-action="add-command"]') as HTMLButtonElement).disabled).toBe(true)
	expect(mounted.composer.value.definition.steps).toHaveLength(1)
  })

  it("preserves literals across launch-parameter mapping and migrates rename/removal safely", async () => {
    const draft = newScenarioComposerDraft()
    draft.definition.parameters = [{ id: "limit", name: "Limit", kind: "integer", required: true, minimum: 1, maximum: 8 }]
    draft.definition.steps = [{ type: "command", commandId: "test.configure", arguments: { count: 7 } }]
    const mounted = mountScenarioLab(draft, "editor", undefined, undefined, [], editorCommands)

    setSelect('[aria-label="Source for count"]', "limit")
    await nextTick()
    expect(mounted.composer.value.definition.steps[0]).toMatchObject({ arguments: { count: { parameter: "limit" } } })
    setInput('[data-scenario-parameter="0"] [aria-label="Parameter ID"]', "renamed")
    await nextTick()
    expect(mounted.composer.value.definition.steps[0]).toMatchObject({ arguments: { count: { parameter: "renamed" } } })
    setSelect('[aria-label="Source for count"]', "literal")
    await nextTick()
    expect(mounted.composer.value.definition.steps[0]).toMatchObject({ arguments: { count: 7 } })

    setSelect('[aria-label="Source for count"]', "renamed")
    click('[data-remove-parameter="0"]')
    await nextTick()
    expect(mounted.composer.value.definition.steps[0]).toMatchObject({ arguments: { count: 7 } })
  })

	it("keeps literal backups attached to a terminal step when a note is inserted before it", async () => {
		const draft = newScenarioComposerDraft()
		draft.definition.parameters = [{ id: "limit", name: "Limit", kind: "integer", required: true, minimum: 1, maximum: 8 }]
		draft.definition.steps = [{ type: "command", commandId: "test.final", arguments: { count: 7 } }]
		const commands: CommandDefinition[] = [{
			id: "test.final", category: "Test", name: "Final Command", description: "Terminal test.",
			executionModes: ["workflow"], destructive: true, scenarioTerminal: true,
			parameters: [{ id: "count", name: "Count", kind: "integer", required: true, minimum: 1, maximum: 8 }],
		}]
		const mounted = mountScenarioLab(draft, "editor", undefined, undefined, [], commands)

		setSelect('[aria-label="Source for count"]', "limit")
		click('[data-scenario-action="add-note"]')
		await nextTick()
		expect(mounted.composer.value.definition.steps[0]).toEqual({ type: "note", note: "Describe this scenario checkpoint." })
		const terminalStep = mounted.composer.value.definition.steps[1]
		mounted.composer.value = {
			...mounted.composer.value,
			definition: {
				...mounted.composer.value.definition,
				steps: [mounted.composer.value.definition.steps[0], { type: "note", note: "Externally captured step" }, terminalStep],
			},
		}
		await nextTick()
		setSelect('[aria-label="Source for count"]', "literal")
		await nextTick()

		expect(mounted.composer.value.definition.steps[2]).toMatchObject({ arguments: { count: 7 } })
	})

	it("forces a newly staged terminal command after all existing setup steps", async () => {
		const draft = newScenarioComposerDraft()
		draft.definition.steps = [{ type: "note", note: "Setup" }]
		const terminal: CommandDefinition = {
			id: "test.final", category: "Test", name: "Final Command", description: "Terminal test.",
			executionModes: ["workflow"], destructive: true, scenarioTerminal: true,
		}
		const mounted = mountScenarioLab(draft, "editor", undefined, undefined, [], [terminal])

		setSelect("[data-scenario-command-choice]", terminal.id)
		await nextTick()
		expect(root.querySelector("[data-scenario-insertion-position]")?.textContent).toContain("FINAL")
		click('[data-scenario-action="add-command"]')
		await nextTick()

		expect(mounted.composer.value.definition.steps).toEqual([
			{ type: "note", note: "Setup" },
			{ type: "command", commandId: terminal.id },
		])
	})

	it("resets the insertion cursor when another scenario is loaded", async () => {
		const draft = newScenarioComposerDraft()
		draft.definition.steps = [{ type: "note", note: "Old one" }, { type: "note", note: "Old two" }]
		const mounted = mountScenarioLab(draft, "editor", undefined, undefined, [], editorCommands)
		setSelect("[data-scenario-insertion-position]", "0")

		mounted.composer.value = hydrateScenarioComposer({
			reference: "personal:replacement",
			revision: "a".repeat(64),
			definition: {
				schemaVersion: 2, id: "replacement", name: "Replacement", description: "New draft.", baseline: { type: "current" },
				steps: [{ type: "note", note: "One" }, { type: "note", note: "Two" }, { type: "note", note: "Three" }],
			},
		})
		await nextTick()

		expect((root.querySelector("[data-scenario-insertion-position]") as HTMLSelectElement).value).toBe("3")
	})

  it("reviews incompatible command replacement and renders validation issues on the affected step", async () => {
    const draft = newScenarioComposerDraft()
    draft.definition.steps = [{ type: "command", commandId: "test.configure", arguments: { count: 4 } }]
    const mounted = mountScenarioLab(draft, "editor", undefined, undefined, [{ path: "$.steps[0].arguments.name", code: "argument.required", message: "name is required" }], editorCommands)

    expect(root.querySelector('[data-scenario-step-issues="0"]')?.textContent).toContain("name is required")
    setSelect('[aria-label="Replace command for step 1"]', "test.other")
    await nextTick()
    expect(root.querySelector("[data-scenario-replacement-review]")?.textContent).toContain("count")
    click('[data-scenario-action="confirm-replace-command"]')
    await nextTick()
    expect(mounted.composer.value.definition.steps[0]).toEqual({ type: "command", commandId: "test.other" })
  })

  it("deletes unset optional launch values and blocks running invalid required or bounded values", async () => {
    const draft = hydrateScenarioComposer({
      reference: "personal:launch", revision: "d".repeat(64),
      definition: {
        schemaVersion: 2, id: "launch", name: "Launch", description: "Launch values", baseline: { type: "current" },
        parameters: [
          { id: "requiredCount", name: "Required Count", kind: "integer", required: true, minimum: 1, maximum: 3 },
          { id: "optionalFlag", name: "Optional Flag", kind: "boolean", required: false },
        ],
        steps: [{ type: "command", commandId: "test.configure", arguments: { name: "launch", count: { parameter: "requiredCount" }, enabled: false, mode: "safe" } }],
      },
    })
    draft.launchValues = { optionalFlag: false }
    const mounted = mountScenarioLab(draft, "run-review", undefined, undefined, [], editorCommands, {}, {
      combatActive: false, dialogActive: false, gameplayWorkflowActive: false, scriptedSceneActive: false, gameplayTerminal: false, finaleActive: false, allPartyDead: false,
    })

    expect((root.querySelector('[data-scenario-action="confirm-run"]') as HTMLButtonElement).disabled).toBe(true)
    expect(root.querySelector('[data-scenario-summary-step="1"]')?.textContent).toContain("Count: $Required Count [requiredCount]")
    expect(root.querySelector('[data-scenario-launch-issue="requiredCount"]')?.textContent).toContain("required")
    setSelect("#scenario-launch-optionalFlag", "")
    await nextTick()
    setInput("#scenario-launch-requiredCount", "2")
    await nextTick()
    expect(mounted.composer.value.launchValues).toEqual({ requiredCount: 2 })
    expect((root.querySelector('[data-scenario-action="confirm-run"]') as HTMLButtonElement).disabled).toBe(false)
    setInput("#scenario-launch-requiredCount", "4")
    await nextTick()
    expect((root.querySelector('[data-scenario-action="confirm-run"]') as HTMLButtonElement).disabled).toBe(true)
  })

  it("blocks quick setups after a terminal finale", () => {
    const data = {
      partyFingerprint: "terminal-party",
      wipeReview: { fingerprint: "terminal-wipe", members: [] },
      locationId: 250, locationName: "Finale", mapType: "scripted", floor: 0,
      date: "1-1-142", time: "12:00", food: 100,
      combatActive: false, dialogActive: false, gameplayWorkflowActive: false,
      scriptedSceneActive: false, gameplayTerminal: true, finaleActive: true, allPartyDead: false,
      party: [],
    }

    mountScenarioLab(newScenarioComposerDraft(), "quick-setups", undefined, undefined, [], [], {}, data)

    expect(root.querySelector(".scenario-context")?.textContent).toContain("TERMINAL FINALE")
    expect(root.querySelector<HTMLButtonElement>(".hole-up-card button")?.disabled).toBe(true)
  })

  it("offers schema-v2 named baselines with project portability filtering", async () => {
    const draft = newScenarioComposerDraft()
    draft.source = "project"
    const snapshots: SaveSnapshotSummary[] = [
      { reference: "project:shared", source: "project", id: "shared", name: "Shared", createdAt: "2026-08-06T00:00:00Z", gameVersion: "test", revision: "a".repeat(64), compatibility: { compatible: true }, eligibility: { canCapture: true, canLoad: true, canDelete: true } },
      { reference: "personal:local", source: "personal", id: "local", name: "Local", createdAt: "2026-08-06T00:00:00Z", gameVersion: "test", revision: "b".repeat(64), compatibility: { compatible: true }, eligibility: { canCapture: true, canLoad: true, canDelete: true } },
    ]
    const mounted = mountScenarioLab(draft, "editor", undefined, undefined, [], editorCommands, {}, undefined, snapshots)

    setSelect('[data-scenario-field="baseline"]', "snapshot")
    await nextTick()
    const options = Array.from(root.querySelectorAll<HTMLOptionElement>('[data-scenario-field="baseline-reference"] option')).map((option) => option.value)
    expect(options).toContain("project:shared")
    expect(options).not.toContain("personal:local")
    expect(mounted.composer.value.definition.baseline).toEqual({ type: "snapshot", reference: "project:shared" })
  })
})

function mountScenarioLab(composerValue: ScenarioComposerDraft, taskValue: ScenarioTask, runStatus?: object, definitionList?: object, validationIssues: object[] = [], commands: CommandDefinition[] = [], optionSets: OptionSets = {}, data?: object, snapshots: SaveSnapshotSummary[] = [], selectedRunValue = "") {
  const composer = ref(composerValue)
  const selectedRunID = ref(selectedRunValue)
  app = createApp({
    components: { ScenarioLab },
    setup() {
      const activeTask = ref(taskValue)
      return { activeTask, commands, composer, data, definitionList, optionSets, runStatus, selectedRunID, snapshots, validationIssues }
    },
    template: `<ScenarioLab
      :composer="composer" :data="data" :definition-list="definitionList" :run-status="runStatus" :selected-run-id="selectedRunID" :active-task="activeTask"
      :validation-issues="validationIssues" :pending-status="false"
      :pending-hole-up="false" :pending-kill-party="false" :pending-death-sequence="false"
      :pending-definition-list="false" :pending-definition-get="false" :pending-definition-save="false"
      :pending-run-start="false"
      :commands="commands" :option-sets="optionSets" :snapshots="snapshots"
      @update-composer="composer = $event" @update-active-task="activeTask = $event" @select-run="selectedRunID = $event"
    />`,
  })
  app.mount(root)
  return { composer, selectedRunID }
}

function scenarioRun(runId: string, reference: string, state: "succeeded" | "failed", finishedAt: string): ScenarioRunStatus {
  return {
    runId, reference, definitionId: "repeat", definitionRevision: "a".repeat(64), baseline: { type: "current" }, state,
    startedAt: "2026-08-07T09:59:00Z", finishedAt, currentProcess: false,
    nextStep: state === "succeeded" ? 2 : 1, totalSteps: 1, initialSceneGeneration: 1, sceneGeneration: 1,
    steps: [{ number: 1, type: "command", commandId: "environment.wind.set", state }],
    ...(state === "failed" ? { errorCode: "failed", errorMessage: "Failed." } : {}),
  }
}

function click(selector: string): void {
  const element = root.querySelector<HTMLElement>(selector)
  if (!element) throw new Error(`missing ${selector}`)
  element.click()
}

function setInput(selector: string, value: string): void {
  const input = root.querySelector<HTMLInputElement>(selector)
  if (!input) throw new Error(`missing ${selector}`)
  input.value = value
  input.dispatchEvent(new Event("input", { bubbles: true }))
}

function setSelect(selector: string, value: string): void {
  const select = root.querySelector<HTMLSelectElement>(selector)
  if (!select) throw new Error(`missing ${selector}`)
  select.value = value
  select.dispatchEvent(new Event("change", { bubbles: true }))
}
