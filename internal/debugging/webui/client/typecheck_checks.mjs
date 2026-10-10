import assert from "node:assert/strict"
import { test } from "node:test"
import { verifyPartition } from "./typecheck.mjs"

test("new Vue and TypeScript roots cannot disappear from serial checking", () => {
  const roots = ["src/new.vue", "src/new.spec.ts", "browser/new.spec.ts"]
  const partitions = [["app", [roots[0]]], ["tests", [roots[1]]], ["browser", [roots[2]]]]
  assert.doesNotThrow(() => verifyPartition(roots, partitions))
  for (let index = 0; index < partitions.length; index++) {
    assert.throws(() => verifyPartition(roots, partitions.filter((_, i) => i !== index)), /Missing typecheck roots/)
  }
})

test("overlapping, unexpected and empty checking partitions fail closed", () => {
  assert.throws(() => verifyPartition(["a"], [["one", ["a"]], ["two", ["a"]]]), /Duplicate or unexpected/)
  assert.throws(() => verifyPartition(["a"], [["one", ["b"]]]), /Duplicate or unexpected/)
  assert.throws(() => verifyPartition(["a"], [["one", []]]), /Empty typecheck partition/)
})
