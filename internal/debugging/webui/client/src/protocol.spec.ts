import { describe, expect, it } from "vitest"

import type { CommandResult } from "./protocol"

describe("CommandResult protocol", () => {
  it("preserves a normalized typed invocation without transport ownership fields", () => {
    const result = JSON.parse(`{
      "sessionId":"session-1",
      "requestId":"request-1",
      "commandId":"scenario.apply",
      "sceneGeneration":7,
      "invocation":{
        "commandId":"scenario.apply",
        "arguments":{"count":2,"enabled":true,"mode":"all","members":[1,3]}
      },
      "success":true
    }`) as CommandResult

    expect(result.invocation).toEqual({
      commandId: "scenario.apply",
      arguments: { count: 2, enabled: true, mode: "all", members: [1, 3] },
    })
    expect(Object.keys(result.invocation ?? {})).toEqual(["commandId", "arguments"])
  })

  it("allows pre-normalization failures to omit the invocation", () => {
    const result: CommandResult = {
      sessionId: "session-1",
      requestId: "request-1",
      commandId: "unknown.command",
      sceneGeneration: 7,
      success: false,
      errorCode: "invalid_request",
    }

    expect(result.invocation).toBeUndefined()
  })
})
