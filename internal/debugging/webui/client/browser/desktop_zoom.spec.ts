import { expect, type Locator, type Page } from "@playwright/test"
import { test } from "./desktop_zoom"
import { mockWorkbench } from "./mock_workbench"

async function geometry(page: Page) {
  return page.evaluate(() => ({
    innerWidth, innerHeight, outerWidth, outerHeight, devicePixelRatio,
    visualViewportScale: visualViewport!.scale,
  }))
}

// Native Tab traversal must scroll each control into view, not a forced focus
// or DOM click that could pass despite clipping or an inaccessible tab order.
async function tabTo(page: Page, control: Locator) {
  for (let step = 0; step < 120; step++) {
    if (await control.evaluate((element) => element === document.activeElement)) break
    await page.keyboard.press("Tab")
  }
  await expect(control).toBeFocused()
  await expect(control).toBeInViewport({ ratio: 1 })
  await control.click({ trial: true })
  return control.boundingBox()
}

for (const dock of ["bottom", "right"]) {
  test(`W05 actual desktop 200% zoom with ${dock} history`, async ({ page, setBrowserZoom }, testInfo) => {
    const errors: string[] = []
    page.on("pageerror", (error) => errors.push(error.message))
    await page.addInitScript((dock) => localStorage.setItem("u5r-debug-workbench-history-dock", dock), dock)
    const wire = await mockWorkbench(page)
    await page.goto("/")
    await expect(page.locator(".connection")).toHaveText("READY")
    await page.locator('[data-workspace="scenarios"]').click()
    await page.locator('[data-scenario-action="new-personal"]').click()
    const draftName = "A long active scenario name that must wrap without hiding Tools or history actions"
    await page.locator('[data-scenario-field="name"]').fill(draftName)
    wire.send({ version: 1, id: "zoom-history", type: "command.completed", payload: {
      sessionId: "session-1", requestId: "zoom-history", commandId: "inventory.provision.set", sceneGeneration: 7,
      success: true, effectiveExecutionMode: "live_safe", summary: "Long operation summary retains diagnostics and completed effects without hiding capture actions at actual desktop browser zoom.",
      invocation: { commandId: "inventory.provision.set", arguments: { provision: "gold", targetQuantity: 555 } },
    } })

    expect(await setBrowserZoom(1)).toBe(1)
    const before = await geometry(page)
    const cdp = await page.context().newCDPSession(page)
    const windowBefore = (await cdp.send("Browser.getWindowForTarget")).bounds
    expect(await setBrowserZoom(2)).toBe(2)
    await expect.poll(async () => (await geometry(page)).devicePixelRatio).toBe(before.devicePixelRatio * 2)
    const zoomed = await geometry(page)
    expect(zoomed.innerWidth).toBeCloseTo(before.innerWidth / 2, 0)
    expect(Math.abs(zoomed.innerHeight - before.innerHeight / 2)).toBeLessThanOrEqual(1)
    expect([zoomed.outerWidth, zoomed.outerHeight]).toEqual([before.outerWidth, before.outerHeight])
    expect(zoomed.visualViewportScale).toBe(1)
    expect((await cdp.send("Browser.getWindowForTarget")).bounds).toEqual(windowBefore)
    await expect(page.locator(".workbench-shell")).toHaveClass(`workbench-shell history-dock-${dock}`)

    const controls: Record<string, Awaited<ReturnType<typeof tabTo>>> = {}
    controls.tools = await tabTo(page, page.locator('[data-workspace="tools"]'))
    await page.keyboard.press("Enter")
    await expect(page.locator('[data-workspace="tools"]')).toHaveClass("selected")
    await tabTo(page, page.locator('[data-workspace="game"]'))
    await page.keyboard.press("Enter")

    for (const id of ["calendar.status", "inventory.locker.status"]) {
      // Catalog entries use roving tabindex; select by pointer, then exercise
      // native Tab scrolling through the instrument's editable controls.
      await page.locator(`[data-command-select="${id}"]`).click()
      const calendar = id === "calendar.status"
      const input = calendar ? page.getByRole("spinbutton", { name: "Hour", exact: true }) : page.getByRole("spinbutton", { name: "gold target quantity", exact: true })
      controls[id] = await tabTo(page, input)
      await page.keyboard.press("ControlOrMeta+A")
      await page.keyboard.type(calendar ? "12" : "555")
      await expect(input).toHaveValue(calendar ? "12" : "555")
      const action = page.getByRole("button", { name: calendar ? "SET EXACT TIME" : "STAGE CHANGE", exact: true })
      controls[`${id}:action`] = await tabTo(page, action)
      await expect(action).toBeEnabled()
      const overflowing = await page.locator(".instrument-bank, .workspace-tabs, .history-panel").evaluateAll((elements) => elements.filter((element) => element.scrollWidth > element.clientWidth + 1).map((element) => `${element.className}: ${element.clientWidth}/${element.scrollWidth}`))
      expect(overflowing).toEqual([])
      await page.screenshot({ path: testInfo.outputPath(`${calendar ? "calendar" : "inventory"}-200-percent.png`) })
      if (!calendar) {
        await page.keyboard.press("Enter")
        await expect(page.locator("[data-locker-review-apply]")).toBeVisible()
      }
    }

    controls.capture = await tabTo(page, page.locator('[data-add-scenario="zoom-history"]'))
    await page.keyboard.press("Enter")
    await expect(page.locator('[data-add-scenario="zoom-history"]')).toHaveAttribute("data-state", "completed")
    await expect(page.locator("[data-active-scenario]")).toContainText("1 STEPS")
    controls.dock = await tabTo(page, page.locator('[data-history-action="dock"]'))
    controls.collapse = await tabTo(page, page.locator('[data-history-action="collapse"]'))
    await page.screenshot({ path: testInfo.outputPath("history-200-percent.png") })
    await page.keyboard.press("Enter")
    await expect(page.locator(".history-panel")).toHaveClass(/collapsed/)
    await page.keyboard.press("Enter")
    await expect(page.locator(".history-panel")).not.toHaveClass(/collapsed/)
    expect(await geometry(page)).toEqual(zoomed)
    expect((await cdp.send("Browser.getWindowForTarget")).bounds).toEqual(windowBefore)
    expect(wire.executions("inventory.provision.set")).toHaveLength(0)
    expect(wire.executions("calendar.time.set")).toHaveLength(0)
    expect(errors).toEqual([])
    await testInfo.attach("desktop-zoom-measurements", {
      body: JSON.stringify({ before, zoomed, windowBefore, controls }, null, 2), contentType: "application/json",
    })
  })
}
