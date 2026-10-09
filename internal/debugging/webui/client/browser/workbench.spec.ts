import { expect, test, type Page } from "@playwright/test"
import { mockWorkbench } from "./mock_workbench"
import type { ScenarioRunStatus } from "../src/scenarios"
import { successfulQueryCompletion } from "../src/app_test_fixtures"

const pageErrors = new WeakMap<Page, string[]>()
test.beforeEach(({ page }) => {
  const errors: string[] = []
  pageErrors.set(page, errors)
  page.on("pageerror", (error) => errors.push(error.message))
})
test.afterEach(({ page }) => { expect(pageErrors.get(page)).toEqual([]) })

async function workspace(page: Page, id: string) { await page.locator(`[data-workspace="${id}"]`).click() }
async function instrument(page: Page, id: string) { await page.locator(`[data-command-select="${id}"]`).click() }

function historicalRun(): ScenarioRunStatus {
  return {
    runId: "actual-run", reference: "personal:probe-current", definitionId: "probe-current", definitionRevision: "a".repeat(64),
    baseline: { type: "current" }, launch: { parameterValues: { seed: 919, enabled: false, label: "Historical value" }, baselineApplied: true, rng: { seed: "18446744073709551615", sourceCalls: "9007199254740993" } },
    rollback: { reference: "rollback:run-before", revision: "a".repeat(64) }, state: "running", currentProcess: true,
    startedAt: "2026-08-07T10:00:00Z", initialSceneGeneration: 7, sceneGeneration: 7, nextStep: 2, totalSteps: 2,
    steps: [{ number: 1, type: "command", commandId: "inventory.provision.set", state: "succeeded" }, { number: 2, type: "command", commandId: "scenario.starvation.trigger", state: "pending" }],
  }
}

for (const dock of ["bottom", "right"]) {
  for (const width of [1440, 1280, 1024, 800, 640]) {
    test(`controls fit ${width}px with ${dock} history and active draft`, async ({ page }, testInfo) => {
      await page.setViewportSize({ width, height: 900 })
      await page.addInitScript((dock) => localStorage.setItem("u5r-debug-workbench-history-dock", dock), dock)
      const wire = await mockWorkbench(page)
      await page.goto("/")
      await workspace(page, "scenarios")
      await page.locator('[data-scenario-action="new-personal"]').click()
      await page.locator('[data-scenario-field="name"]').fill("A long active scenario name that should wrap without hiding Tools or history actions")
      wire.send({ version: 1, id: "layout-history", type: "command.completed", payload: {
        sessionId: "session-1", requestId: "layout-history", commandId: "inventory.provision.set", sceneGeneration: 7,
        success: true, effectiveExecutionMode: "live_safe", summary: "Long operation summary retains diagnostics and all completed effects without hiding capture actions.",
        invocation: { commandId: "inventory.provision.set", arguments: { provision: "gold", targetQuantity: 555 } },
      } })
      await workspace(page, "game")
      for (const id of ["calendar.status", "inventory.locker.status"]) {
        await instrument(page, id)
        await expect(page.locator(id === "calendar.status" ? ".calendar-readout" : "[data-locker-provision]")).toBeVisible()
        const overflowing = await page.locator(".instrument-bank, .workspace-tabs, .history-panel").evaluateAll((elements) => elements.filter((element) => element.scrollWidth > element.clientWidth + 1).map((element) => `${element.className}: ${element.clientWidth}/${element.scrollWidth}`))
        expect(overflowing).toEqual([])
        const clippedControls = await page.locator(".instrument button, .instrument input, .instrument select").evaluateAll((elements) => elements.filter((element) => {
          const rect = element.getBoundingClientRect()
          const bank = element.closest(".instrument-bank")!.getBoundingClientRect()
          return rect.width && (rect.left < bank.left - 1 || rect.right > bank.right + 1)
        }).map((element) => element.outerHTML.slice(0, 120)))
        expect(clippedControls).toEqual([])
      }
      await expect(page.locator('[data-workspace="tools"]')).toBeInViewport()
      if (width === 640) expect((await page.locator(".instrument-bank").boundingBox())!.y).toBeLessThan(650)
      if ((width === 1280 || width === 800 || width === 640) && dock === "right") await page.screenshot({ path: testInfo.outputPath("source-workbench.png"), fullPage: true })
    })
  }
}

for (const baseline of ["current", "init", "snapshot"]) {
  test(`parameterless ${baseline} shortcut never starts before Review confirmation`, async ({ page }) => {
    const wire = await mockWorkbench(page, [], { parameterless: true })
    await page.goto("/")
    await workspace(page, "scenarios")
    await page.locator(`[data-scenario-play="personal:probe-${baseline}"]`).click()
    await expect(page.locator('[data-scenario-screen="run-review"]')).toBeVisible()
    expect(wire.executions("scenario.run.start")).toHaveLength(0)
    await expect(page.locator('[data-scenario-action="confirm-run"]')).toBeEnabled()
    await workspace(page, "home")
    await workspace(page, "scenarios")
    await expect(page.locator('[data-scenario-screen="run-review"]')).toHaveCount(0)
    expect(wire.executions("scenario.run.start")).toHaveLength(0)
    await page.locator('[data-scenario-action="review-run"]').click()
    await page.locator('[data-scenario-action="confirm-run"]').click()
    await expect.poll(() => wire.executions("scenario.run.start").length).toBe(1)
  })

  test(`RUN reviews ${baseline} with required values and no pre-confirm Start`, async ({ page }) => {
    const wire = await mockWorkbench(page)
    await page.goto("/")
    await workspace(page, "scenarios")
    await page.locator(`[data-scenario-play="personal:probe-${baseline}"]`).click()
    await expect(page.locator('[data-scenario-screen="run-review"]')).toBeVisible()
    expect(wire.executions("scenario.run.start")).toHaveLength(0)
    await expect(page.locator('[data-scenario-action="confirm-run"]')).toBeDisabled()
    await page.locator("#scenario-launch-seed").fill("77")
    await page.locator("#scenario-launch-enabled").selectOption("false")
    await page.locator("#scenario-launch-label").fill("actual visible launch")
    await page.locator('[data-scenario-action="confirm-run"]').click()
    await expect.poll(() => wire.executions("scenario.run.start").length).toBe(1)
    expect(JSON.parse(String(wire.executions("scenario.run.start")[0].payload.arguments.parameterValues))).toEqual({ seed: 77, enabled: false, label: "actual visible launch" })
  })
}

test("inventory drafts survive detours, reviews disarm, blank is not zero", async ({ page }) => {
  const wire = await mockWorkbench(page)
  await page.goto("/")
  await workspace(page, "game")
  await instrument(page, "inventory.locker.status")
  const gold = page.getByRole("spinbutton", { name: "gold target quantity", exact: true })
  await gold.fill("555")
  await page.getByRole("button", { name: "STAGE CHANGE", exact: true }).click()
  await expect(page.locator("[data-locker-review-apply]")).toBeVisible()
  await workspace(page, "codex")
  await workspace(page, "home")
  await workspace(page, "game")
  await expect(gold).toHaveValue("555")
  await expect(page.locator("[data-locker-review-apply]")).toHaveCount(0)
  await gold.fill("")
  await page.getByRole("button", { name: "STAGE CHANGE", exact: true }).click()
  await expect(page.locator("[data-owner-error]")).toContainText("Blank is not zero")
  expect(wire.executions("inventory.provision.set")).toHaveLength(0)
})

test("Quick Setup edits and Escape disarm; overflow stays visible on owner", async ({ page }) => {
  const wire = await mockWorkbench(page)
  await page.goto("/")
  await workspace(page, "scenarios")
  await page.locator('[data-scenario-task="quick-setups"]').click()
  const direct = page.locator(".scenario-card").filter({ has: page.getByText("DIRECT STARVATION", { exact: true }) })
  await direct.locator("input").fill("111")
  await direct.locator("button").click()
  await expect(page.locator("[data-quick-review]")).toContainText("SEED 111")
  await direct.locator("input").fill("222")
  await expect(direct.locator("button")).toHaveText("REVIEW DAMAGE")
  await direct.locator("button").click()
  await page.keyboard.press("Escape")
  await expect(page.locator("[data-quick-review]")).toHaveCount(0)
  wire.send(successfulQueryCompletion("previous-setup", "scenario.starvation.trigger", "Previous setup succeeded.", {}))
  await expect(page.locator(".operation-banner.success")).toContainText("Previous setup succeeded")
  await direct.locator("input").fill("2147483648")
  await direct.locator("button").click()
  await expect(page.locator("[data-owner-error]")).toContainText("2147483647")
  await expect(page.locator(".operation-banner.success")).toHaveCount(0)
  expect(wire.executions("scenario.starvation.trigger")).toHaveLength(0)
})

test("parameter rename preserves node, caret and references; owned modal traps and restores focus", async ({ page }) => {
  await mockWorkbench(page)
  await page.goto("/")
  await workspace(page, "scenarios")
  await page.locator('[data-scenario-reference="personal:probe-current"]').click()
  await page.locator('[data-scenario-action="edit"]').click()
  const parameter = page.getByRole("textbox", { name: "Parameter ID", exact: true }).first()
  await parameter.focus()
  await parameter.evaluate((element) => { (element as HTMLInputElement).setSelectionRange(4, 4); element.setAttribute("data-original-node", "true") })
  await page.keyboard.type("-renamed")
  await expect(parameter).toBeFocused()
  await expect(parameter).toHaveValue("seed-renamed")
  await expect(parameter).toHaveAttribute("data-original-node", "true")
  expect(await parameter.evaluate((element) => (element as HTMLInputElement).selectionStart)).toBe(12)
  await expect(page.locator(".scenario-argument")).toContainText("seed-renamed")
  await parameter.fill("enabled")
  await expect(page.locator("[data-scenario-parameter-issue]")).toContainText("already in use")
  await page.locator('[data-scenario-action="editor-back"]').click()
  const launcher = page.locator('[data-scenario-action="delete"]')
  await launcher.click()
  const modal = page.getByRole("dialog")
  await expect(modal.getByRole("textbox")).toBeFocused()
  await page.keyboard.press("Shift+Tab")
  await expect(modal.getByRole("button", { name: "CANCEL", exact: true })).toBeFocused()
  await expect(page.locator(".workspace-tabs")).toHaveAttribute("inert", "")
  await page.keyboard.press("Escape")
  await expect(modal).toHaveCount(0)
  await expect(launcher).toBeFocused()
  await expect(page.locator(".workspace-tabs")).not.toHaveAttribute("inert", "")
})

test("Home Review Again uses actual history, not defaults, and still requires confirmation", async ({ page }) => {
  const wire = await mockWorkbench(page, [historicalRun()])
  await page.goto("/")
  await page.locator("[data-home-play]").click()
  await expect(page.locator('[data-scenario-screen="run-review"]')).toBeVisible()
  await expect(page.locator("#scenario-launch-seed")).toHaveValue("919")
  await expect(page.locator("#scenario-launch-label")).toHaveValue("Historical value")
  await expect(page.locator(".scenario-run-review")).toContainText("9007199254740993")
  expect(wire.executions("scenario.run.start")).toHaveLength(0)
  await page.locator('[data-scenario-action="confirm-run"]').click()
  await expect.poll(() => wire.executions("scenario.run.start").length).toBe(1)
  expect(JSON.parse(String(wire.executions("scenario.run.start")[0].payload.arguments.parameterValues))).toEqual(historicalRun().launch!.parameterValues)
})

test("old history is unsupported and changed definition revisions cannot masquerade as repeat", async ({ page }) => {
  const old = historicalRun()
  delete old.launch
  old.runId = "legacy-run"
  const changed = { ...historicalRun(), definitionRevision: "b".repeat(64) }
  const wire = await mockWorkbench(page, [old, changed])
  await page.goto("/")
  await expect(page.locator("[data-home-play]").first()).toBeDisabled()
  await expect(page.locator("[data-home-play]").first()).toContainText("UNKNOWN LAUNCH")
  await page.locator("[data-home-play]").nth(1).click()
  await expect(page.locator(".scenario-run-review")).toContainText("differs from current revision")
  await expect(page.locator('[data-scenario-action="confirm-run"]')).toBeDisabled()
  expect(wire.executions("scenario.run.start")).toHaveLength(0)
})

test("Cancel Remaining Steps is run-ID bound; rollback opens pinned Saves review only", async ({ page }) => {
  const wire = await mockWorkbench(page, [historicalRun()])
  await page.goto("/")
  await page.locator('[data-home-run="actual-run"]').click()
  await page.locator("[data-scenario-cancel-remaining]").click()
  await expect.poll(() => wire.executions("scenario.run.cancel").length).toBe(1)
  expect(wire.executions("scenario.run.cancel")[0].payload.arguments).toEqual({ runId: "actual-run" })
  await expect(page.locator("[data-run-step='1']")).toHaveAttribute("data-step-state", "succeeded")
  await expect(page.locator("[data-run-step='2']")).toHaveAttribute("data-step-state", "pending")
  await expect(page.locator("[data-selected-run='actual-run']")).toHaveAttribute("data-run-state", "cancelled")
  expect(wire.executions("save.snapshot.load")).toHaveLength(0)
  await page.locator("[data-scenario-review-rollback]").click()
  await expect(page.locator('[data-save-review="load"]')).toBeVisible()
  await expect(page.locator(".save-detail-header")).toContainText("rollback:run-before")
  expect(wire.executions("save.snapshot.load")).toHaveLength(0)
  await page.locator('[data-save-action="confirm-load"]').click()
  await expect.poll(() => wire.executions("save.snapshot.load").length).toBe(1)
  expect(wire.executions("save.snapshot.load")[0].payload.arguments).toEqual({ reference: "rollback:run-before", expectedRevision: "a".repeat(64) })
})

test("unsaved Gold 555 cannot Review the saved Gold 0 plan from Details", async ({ page }) => {
  const wire = await mockWorkbench(page, [], { goldPlan: true })
  await page.goto("/")
  await workspace(page, "scenarios")
  await page.locator('[data-scenario-reference="personal:probe-current"]').click()
  await page.locator('[data-scenario-action="edit"]').click()
  await page.getByRole("spinbutton", { name: "Literal value for targetQuantity", exact: true }).fill("555")
  await page.locator('[data-scenario-action="editor-back"]').click()
  await expect(page.locator('[data-scenario-screen="details"]')).toContainText("555")
  await expect(page.locator('[data-scenario-screen="details"]')).toContainText("SAVE FIRST")
  const review = page.locator('[data-scenario-action="review-run"]')
  await expect(review).toBeDisabled()
  // Bypass only the disabled DOM control to exercise App's independent boundary.
  await review.dispatchEvent("click")
  await expect(page.locator("[data-owner-error]")).toContainText("Only a saved definition revision can run")
  expect(wire.executions("scenario.definition.validate")).toHaveLength(0)
  expect(wire.executions("scenario.run.start")).toHaveLength(0)
})

test("a retained old Start control cannot launch after editing the saved plan", async ({ page }) => {
  const wire = await mockWorkbench(page, [], { goldPlan: true })
  await page.goto("/")
  await workspace(page, "scenarios")
  await page.locator('[data-scenario-play="personal:probe-current"]').click()
  const start = await page.locator('[data-scenario-action="confirm-run"]').elementHandle()
  await page.getByRole("button", { name: "BACK TO DETAILS", exact: true }).click()
  await page.locator('[data-scenario-action="edit"]').click()
  await page.getByRole("spinbutton", { name: "Literal value for targetQuantity", exact: true }).fill("555")
  await start!.evaluate((element) => element.dispatchEvent(new MouseEvent("click", { bubbles: true })))
  await expect(page.locator("[data-owner-error]")).toContainText("before starting a run")
  expect(wire.executions("scenario.run.start")).toHaveLength(0)
})

test("pending migrations block both shortcut Review and an old Start handler", async ({ page }) => {
  const options = { parameterless: true, pendingMigration: false }
  const wire = await mockWorkbench(page, [], options)
  await page.goto("/")
  await workspace(page, "scenarios")
  await page.locator('[data-scenario-play="personal:probe-current"]').click()
  const start = await page.locator('[data-scenario-action="confirm-run"]').elementHandle()
  await page.locator('[data-scenario-task="library"]').click()
  options.pendingMigration = true
  await page.locator('[data-scenario-play="personal:probe-init"]').click()
  await expect(page.locator('[data-scenario-screen="details"]')).toContainText("SAVE FIRST")
  await expect(page.locator('[data-scenario-action="review-run"]')).toBeDisabled()
  expect(wire.executions("scenario.definition.validate")).toHaveLength(1)
  await start!.evaluate((element) => element.dispatchEvent(new MouseEvent("click", { bubbles: true })))
  await expect(page.locator("[data-owner-error]")).toContainText("before starting a run")
  expect(wire.executions("scenario.run.start")).toHaveLength(0)
})

test("returning to the editor during delayed validation cannot rearm a stale run review", async ({ page }) => {
  const wire = await mockWorkbench(page, [], { goldPlan: true, deferCommands: ["scenario.definition.validate"] })
  await page.goto("/")
  await workspace(page, "scenarios")
  await page.locator('[data-scenario-reference="personal:probe-current"]').click()
  await page.locator('[data-scenario-action="review-run"]').click()
  await expect.poll(() => wire.executions("scenario.definition.validate").length).toBe(1)
  await page.locator('[data-scenario-action="edit"]').click()
  const quantity = page.getByRole("spinbutton", { name: "Literal value for targetQuantity", exact: true })
  await expect(quantity).toBeDisabled()
  wire.release(wire.executions("scenario.definition.validate")[0].id)
  await expect(quantity).toBeEnabled()
  await quantity.fill("555")
  await expect(page.locator('[data-scenario-screen="run-review"]')).toHaveCount(0)
  await page.locator('[data-scenario-action="editor-back"]').click()
  await expect(page.locator('[data-scenario-action="review-run"]')).toBeDisabled()
  expect(wire.executions("scenario.run.start")).toHaveLength(0)
})

for (const interruption of ["navigation", "disconnect", "pagehide", "selection"]) {
  test(`delayed rollback response cannot rearm after ${interruption}`, async ({ page }) => {
    const wire = await mockWorkbench(page, [historicalRun()], { deferCommands: ["save.snapshot.get"] })
    await page.goto("/")
    await page.locator('[data-home-run="actual-run"]').click()
    await page.locator("[data-scenario-review-rollback]").click()
    await expect.poll(() => wire.executions("save.snapshot.get").length).toBe(1)
    const oldRequest = wire.executions("save.snapshot.get")[0]
    if (interruption === "navigation") {
      await workspace(page, "home")
      await workspace(page, "saves")
    } else if (interruption === "disconnect") {
      wire.disconnect()
      await expect(page.locator(".connection")).toHaveText("DISCONNECTED")
      await expect(page.locator(".connection")).toHaveText("READY")
    } else if (interruption === "pagehide") {
      await page.evaluate(() => {
        Object.defineProperty(document, "hidden", { configurable: true, value: true })
        document.dispatchEvent(new Event("visibilitychange"))
        delete (document as unknown as Record<string, unknown>).hidden
        document.dispatchEvent(new Event("visibilitychange"))
      })
    } else {
      // Selecting the already-open workspace is still a new navigation intent.
      await workspace(page, "saves")
    }
    wire.release(oldRequest.id)
    await expect(page.locator(".save-detail-header")).toContainText("rollback:run-before")
    await expect(page.locator('[data-save-review="load"]')).toHaveCount(0)
    expect(wire.executions("save.snapshot.load")).toHaveLength(0)
    // A new explicit handoff must still work after the cancelled intent.
    await workspace(page, "home")
    await page.locator('[data-home-run="actual-run"]').click()
    await page.locator("[data-scenario-review-rollback]").click()
    await expect.poll(() => wire.executions("save.snapshot.get").length).toBe(2)
    wire.release(wire.executions("save.snapshot.get")[1].id)
    await expect(page.locator('[data-save-review="load"]')).toBeVisible()
    expect(wire.executions("save.snapshot.load")).toHaveLength(0)
  })
}

test("Capture Tab traversal includes insertion position and Escape restores invoking history action", async ({ page }) => {
  const wire = await mockWorkbench(page)
  await page.goto("/")
  wire.send({ ...successfulQueryCompletion("capturable", "inventory.provision.set", "Set gold.", {}), payload: {
    sessionId: "session-1", requestId: "capturable", commandId: "inventory.provision.set", sceneGeneration: 7,
    effectiveExecutionMode: "live_safe", success: true, summary: "Set gold.", invocation: { commandId: "inventory.provision.set", arguments: { provision: "gold", targetQuantity: 555 } },
  } })
  const launcher = page.locator('[data-add-scenario="capturable"]')
  await launcher.click()
  await page.locator('[data-scenario-target-reference="personal:probe-current"]').click()
  const insertion = page.locator("[data-scenario-target-insertion]")
  const visited: boolean[] = []
  for (let index = 0; index < 12; index++) {
    await page.keyboard.press("Tab")
    visited.push(await insertion.evaluate((element) => element === document.activeElement))
  }
  expect(visited).toContain(true)
  await page.keyboard.press("Escape")
  await expect(page.locator("[data-scenario-target-dialog]")).toHaveCount(0)
  await expect(launcher).toBeFocused()
})
