import { describe, expect, it } from "vitest"

import { readFileSync } from "node:fs"

const styles = readFileSync("src/style.css", "utf8")

describe("party instrument responsive layout", () => {
  it("stacks roster and loadout layouts and keeps narrow controls touch-sized", () => {
    expect(styles).toMatch(/@media \(max-width: 1000px\)[\s\S]*\.loadout-slots \{ grid-template-columns: 1fr; \}/)
    expect(styles).toMatch(/@media \(max-width: 1000px\)[\s\S]*\.party-roster-layout \{ grid-template-columns: 1fr; \}/)
    expect(styles).toMatch(/@media \(max-width: 760px\)[\s\S]*\.loadout-slot select, \.loadout-review-button, \.loadout-review button \{ min-height: 44px; \}/)
    expect(styles).toMatch(/@media \(max-width: 760px\)[\s\S]*\.party-roster select, \.party-roster input, \.party-roster button \{ min-height: 44px; \}/)
  })
})
