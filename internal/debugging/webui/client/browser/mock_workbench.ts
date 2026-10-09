import type { Page, WebSocketRoute } from "@playwright/test"
import initialCatalog from "../src/initial_catalog.fixture.json" with { type: "json" }
import { sessionReady, successfulQueryCompletion, calendarStatusCompletion, scenarioStatusCompletion, inventoryLockerCompletion } from "../src/app_test_fixtures"
import type { ScenarioRunStatus } from "../src/scenarios"

export interface Request { id: string; type: string; payload: { commandId: string; arguments: Record<string, unknown> } }

export async function mockWorkbench(page: Page, runs: ScenarioRunStatus[] = [], options: { parameterless?: boolean; goldPlan?: boolean; pendingMigration?: boolean; deferCommands?: string[] } = {}) {
  const requests: Request[] = []
  const delayed = new Map<string, () => void>()
  let handshakes = 0
  let socket: WebSocketRoute
  const revision = "a".repeat(64)
  const snapshot = {
    reference: "personal:baseline", source: "personal", id: "baseline", name: "Before Dungeon Entry", description: "Synthetic checkpoint",
    revision, createdAt: "2026-08-06T12:00:00Z", gameVersion: "test", compatibility: { compatible: true },
    eligibility: { canCapture: true, canLoad: true, canDelete: true },
  }
  const definitions = ["current", "init", "snapshot"].map((type) => ({
    schemaVersion: 2, id: `probe-${type}`, name: `Probe ${type}`, description: "Synthetic required launch parameter regression",
    baseline: { type, ...(type === "snapshot" ? { reference: snapshot.reference } : {}) },
    parameters: options.parameterless || options.goldPlan ? [] : [
      { id: "seed", name: "Seed", kind: "integer", required: true, minimum: 0, maximum: 2147483647 },
      { id: "enabled", name: "Enabled", kind: "boolean", required: true },
      { id: "label", name: "Label", kind: "string", required: true },
    ],
    steps: options.goldPlan
      ? [{ type: "command", commandId: "inventory.provision.set", arguments: { provision: "gold", targetQuantity: 0 } }]
      : [{ type: "command", commandId: "scenario.starvation.trigger", arguments: { seed: options.parameterless ? 7 : { parameter: "seed" } } }],
  }))
  const rollback = { ...snapshot, reference: "rollback:run-before", source: "rollback", id: "run-before", name: "Automatic Rollback" }
  const send = (message: object) => socket.send(JSON.stringify(message))
  // Intercept only the test page's wire. The source App, handshake and production
  // endpoint authentication are unmodified; no game service or game assets run.
  await page.routeWebSocket("**/ws*", (route) => {
    socket = route
    route.onMessage((raw) => {
      const request = JSON.parse(String(raw)) as Request
      if (request.type === "session.hello") { send(sessionReady("session-1", ++handshakes > 1)); return }
      if (request.type === "catalog.request") {
        send({ version: 1, id: request.id, type: "catalog.snapshot", payload: { sceneGeneration: 7, optionsRevision: 9, commands: initialCatalog, optionSets: [] } })
        return
      }
      if (request.type !== "command.execute") return
      requests.push(request)
      const { commandId, arguments: args } = request.payload
      const complete = (data: object) => {
        const deliver = () => send(successfulQueryCompletion(request.id, commandId, `${commandId} synthetic result.`, data))
        if (options.deferCommands?.includes(commandId)) delayed.set(request.id, deliver)
        else deliver()
      }
      if (commandId === "calendar.status") send(calendarStatusCompletion(request.id))
      else if (commandId === "scenario.status") send(scenarioStatusCompletion(request.id))
      else if (commandId === "inventory.locker.status") send(inventoryLockerCompletion(request.id, "locker-1", 100))
      else if (commandId === "combat.status") complete({ active: false, monsters: [], party: [], forecast: [], monsterTypes: [] })
      else if (commandId === "scenario.definition.list") complete({ definitions: definitions.map((definition) => ({ ...definition, reference: `personal:${definition.id}`, source: "personal", revision })) })
      else if (commandId === "scenario.definition.get") {
        const definition = definitions.find((definition) => `personal:${definition.id}` === args.reference)!
        complete({ reference: args.reference, source: "personal", revision,
          definition: options.pendingMigration ? { ...definition, steps: [{ type: "command", commandId: "party.character.update", arguments: { character: "1" } }] } : definition,
          migrations: options.pendingMigration ? [{ path: "$.steps[0].arguments.character", code: "legacy_character_token", before: "0.QXZhdGFy", after: "1" }] : [],
        })
      } else if (commandId === "scenario.definition.validate") complete({ valid: true, issues: [] })
      else if (commandId === "scenario.run.status") complete({ run: runs[0] ?? null, runs, capacity: 20 })
      else if (commandId === "scenario.run.cancel") {
        const run = runs.find((run) => run.runId === args.runId)!
        run.state = "cancelled"
        run.finishedAt = "2026-08-07T10:01:00Z"
        send({ version: 1, id: request.id, type: "command.completed", payload: {
          sessionId: "session-1", requestId: request.id, commandId, sceneGeneration: 7, effectiveExecutionMode: "live_safe", success: true,
          summary: "No remaining steps will be dispatched. Completed effects and production workflows are retained.", data: { run, runs, capacity: 20 },
        } })
      } else if (commandId === "save.snapshot.list") complete({ snapshots: [snapshot, rollback], captureEligibility: { canCapture: true } })
      else if (commandId === "save.snapshot.get") complete({ snapshot: { summary: args.reference === rollback.reference ? rollback : snapshot, locationName: "Britain", location: 2, floor: 0, x: 1, y: 1, year: 138, month: 7, day: 12, hour: 16, minute: 15, partyMembers: 1, moonPhaseIndex: 1, moonPhase: "Crescent Waxing" } })
      else if (commandId === "environment.wind.status") complete({ direction: "north", enabled: true })
      else if (commandId.startsWith("codex.")) complete([])
      else if (commandId === "debug.options.status") complete({ freeMove: false })
      // Mutations remain captured and pending, never executed against a game.
    })
  })
  return {
    requests, send, executions: (commandId: string) => requests.filter((request) => request.payload.commandId === commandId),
    release: (requestID: string) => {
      const deliver = delayed.get(requestID)
      if (!deliver) throw new Error(`No delayed response for ${requestID}`)
      delayed.delete(requestID)
      deliver()
    },
    disconnect: () => socket.close({ code: 1012, reason: "Synthetic reconnect during rollback review" }),
  }
}
