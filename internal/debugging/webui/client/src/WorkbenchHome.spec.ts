import { createApp, type App as VueApp } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import WorkbenchHome from "./WorkbenchHome.vue"
import type { NavigationSurface } from "./command_navigation"

let app: VueApp<Element> | undefined
let root: HTMLDivElement

const recent: NavigationSurface = {
  id: "codex:codex.consumable.catalog",
  workspace: "codex",
  workspaceLabel: "Codex",
  label: "Consumables Codex",
  description: "Original and current Redux behavior.",
  commandId: "codex.consumable.catalog",
  commands: [],
}

beforeEach(() => {
  root = document.createElement("div")
  document.body.append(root)
})

afterEach(() => {
  app?.unmount()
  app = undefined
  root.remove()
})

describe("Workbench Home", () => {
  it("opens recent surfaces and the navigation palette without executing commands", () => {
    const openPalette = vi.fn()
    const openSurface = vi.fn()
    mount({ recents: [recent], connected: true, onOpenPalette: openPalette, onOpenSurface: openSurface })

    click("[data-home-go-to]")
    click('[data-home-recent="codex:codex.consumable.catalog"]')

    expect(openPalette).toHaveBeenCalledOnce()
    expect(openSurface).toHaveBeenCalledOnce()
    expect(openSurface).toHaveBeenCalledWith(recent)
  })

  it("summarizes the retained scenario run and opens Latest Run", () => {
    const openScenarios = vi.fn()
    const reviewAgain = vi.fn()
    mount({
      recents: [],
      connected: false,
      runStatus: {
        run: {
          runId: "run-guard-test",
          reference: "project:guard-test",
          definitionId: "guard-test",
          definitionRevision: "a".repeat(64),
          baseline: { type: "current" },
          launch: { parameterValues: { seed: 77 }, baselineApplied: true },
          state: "succeeded",
          startedAt: "2026-08-07T14:31:42Z",
          finishedAt: "2026-08-07T14:32:08Z",
          currentProcess: false,
          nextStep: 6,
          totalSteps: 6,
          initialSceneGeneration: 40,
          sceneGeneration: 42,
          steps: [],
        }, runs: [], capacity: 20,
      },
      onOpenScenarios: openScenarios,
      onReviewAgain: reviewAgain,
    })

    expect(root.querySelector(".home-latest-run")?.textContent).toContain("LAST RECEIVED")
    expect(root.querySelector(".home-latest-run")?.textContent).toContain("project:guard-test")
    expect(root.querySelector(".home-latest-run")?.textContent).toContain("SUCCEEDED")
    expect(root.querySelector(".home-latest-run")?.textContent).toContain("6 / 6")
    click('[data-home-run="run-guard-test"]')
    expect(openScenarios).toHaveBeenCalledWith("latest-run", "run-guard-test")
    click('[data-home-play="project:guard-test"]')
    expect(reviewAgain).toHaveBeenCalledWith(expect.objectContaining({ runId: "run-guard-test", launch: { parameterValues: { seed: 77 }, baselineApplied: true } }))
  })

  it("offers useful empty-state destinations", () => {
    const openScenarios = vi.fn()
    mount({ recents: [], connected: true, runStatus: { run: null, runs: [], capacity: 20 }, onOpenScenarios: openScenarios })

    expect(root.querySelector(".home-recents")?.textContent).toContain("NO RECENT SURFACES")
    expect(root.querySelector(".home-latest-run")?.textContent).toContain("NO DURABLE RUNS YET")
    click("[data-home-open-scenarios]")
    expect(openScenarios).toHaveBeenCalledWith("library")
  })
})

function mount(props: Record<string, unknown>): void {
  app = createApp(WorkbenchHome, props)
  app.mount(root)
}

function click(selector: string): void {
  const element = root.querySelector<HTMLElement>(selector)
  if (!element) throw new Error(`missing test element: ${selector}`)
  element.dispatchEvent(new MouseEvent("click", { bubbles: true }))
}
