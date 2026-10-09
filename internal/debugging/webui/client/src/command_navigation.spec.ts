import { describe, expect, it } from "vitest"

import { rankNavigationSurfaces, recentNavigationSurfaces, updateNavigationRecents, type NavigationSurface } from "./command_navigation"
import type { CommandDefinition } from "./protocol"

const surfaces: NavigationSurface[] = [
  surface("game:shrine", "game", "Game", "Shrine Progress", "Live virtue progression", [command("shrine.status", "Shrine Status", ["meditation"])]),
  surface("codex:shrine", "codex", "Codex", "Shrine Reference", "Immutable virtue reference", [command("shrine.status", "Shrine Status", ["mantra"])]),
  surface("game:ships", "game", "Game", "Ships", "Vehicle controls", [command("vehicle.status", "Vehicle Status", ["frigate"]), command("ship.repair-full", "Repair Ship", ["hull"])]),
]

describe("command navigation", () => {
  it("ranks exact, prefix, then deterministic all-token matches", () => {
    expect(rankNavigationSurfaces(surfaces, "Ships").map(({ id, score }) => [id, score])).toEqual([["game:ships", 300]])
    expect(rankNavigationSurfaces(surfaces, "Shr").map(({ id, score }) => [id, score])).toEqual([
      ["codex:shrine", 200], ["game:shrine", 200],
    ])
    expect(rankNavigationSurfaces(surfaces, "repair hull").map(({ id }) => id)).toEqual(["game:ships"])
  })

  it("keeps workspace-qualified shrine surfaces distinct", () => {
    expect(rankNavigationSurfaces(surfaces, "Game Shrine Progress").map(({ id }) => id)).toEqual(["game:shrine"])
    expect(rankNavigationSurfaces(surfaces, "Codex Shrine Reference").map(({ id }) => id)).toEqual(["codex:shrine"])
  })

  it("aggregates companion intent into its owner surface", () => {
    expect(rankNavigationSurfaces(surfaces, "repair frigate").map(({ id }) => id)).toEqual(["game:ships"])
  })

  it("deduplicates and bounds local recents to eight existing surfaces", () => {
    let recents: string[] = []
    for (let index = 0; index < 10; index++) recents = updateNavigationRecents(recents, `surface-${index}`)
    recents = updateNavigationRecents(recents, "surface-5")
    expect(recents).toHaveLength(8)
    expect(recents[0]).toBe("surface-5")
    expect(recentNavigationSurfaces(surfaces, ["missing", "game:ships"]).map(({ id }) => id)).toEqual(["game:ships"])
  })
})

function command(id: string, name: string, searchTerms: string[]): CommandDefinition {
  return { id, name, category: "Test", description: name, executionModes: ["query"], destructive: false, searchTerms }
}

function surface(id: string, workspace: NavigationSurface["workspace"], workspaceLabel: string, label: string, description: string, commands: NavigationSurface["commands"]): NavigationSurface {
  return { id, workspace, workspaceLabel, label, description, commandId: commands[0]?.id ?? "", commands }
}
