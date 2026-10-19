# Moongate System

Moongates are integrated across moonstone state, nighttime rendering, travel
workflows, lighting, and audio. Moonstone coordinates remain documented in
[MOONSTONE_LOCATIONS.md](../gamefacts/MOONSTONE_LOCATIONS.md).

## State And Destination Logic

`MoonstoneSystem` owns the eight moonstone records. A stone can create a gate
when its current map, floor, and coordinate identify the source and it has been
placed in the world, either buried or exposed. Discovery is not a prerequisite
and Search does not remove source eligibility. Collecting it moves its map to
`0xFF`, rejecting both source and destination travel.

All eligible moonstones can be source gates while the shared stage is above
zero. Trammel selects the normal destination before noon and Felucca selects it
from noon onward. `GetMoongateTravelRoute` returns a tagged route: from 00:00
through 00:09 it selects Shrine of Spirituality index 6 without coordinates;
otherwise it carries the phase-selected moonstone destination.

For ordinary moonstone routes, destination review fails if the phase-selected destination moonstone is in inventory. Stepping on the source still runs the activation workflow and plays the travel sound, but leaves the player in place. The 00:00-00:09 Spirituality route does not require that destination stone; ordinary destination availability checks resume at 00:10. Failed activation and the Spirituality handoff contract the shared stage to zero, and no activation charges an intrinsic travel minute. Successful moonstone-to-moonstone Redux travel deliberately keeps the source gate steady under the Avatar dissolve and establishes the destination gate at full stage before revealing the map and Avatar. The original instead contracts the source to zero before travel, then resumes destination growth from zero. This presentation deviation avoids an empty gate during the dissolve; ordinary owning movement and shrine costs remain separate. See [MOON-0005](../gamefacts/MOONSTONE_LOCATIONS.md#moon-0005).

`GameState.moongateStage` is the single runtime authority from 0 (hidden) to 16
(full). At hour 20 it grows one stage at a time over three elapsed seconds; at
hour 5 it shrinks similarly. Time Stop freezes progression without canceling
the transition. Loads initialize active hours 20:00-04:59 at 16 and other hours
at 0. The stage is intentionally not serialized, and the active transition
workflow makes snapshot capture ineligible.

## Rendering

`MoongateVisualSystem` maintains presentation resources for buried or exposed,
uncollected moonstones near the player. It never owns stage or gameplay
eligibility. Core source checks use `GetActiveMoongatePositions` and
`HasActiveMoongateAtPosition` to match the current map and floor as well as the
coordinate.

Current presentation limitation: `MoongateVisualSystem.Update` reads the raw
moonstone records and filters state and distance, but not map or floor identity.
A stone from another location whose stored coordinates fall within range can
therefore contribute a renderer, light position, or ambient source until the
adapter is changed to use the authoritative active-position query. Gameplay
entry still uses the full identity check.

- Day/night changes use one authoritative `moongate_transition_workflow` in `GameState`.
- Gates expose exactly the stage's number of sprite rows; stages 1-16 are visible and enterable.
- Entering proximity during an active hour creates a renderer at the current shared stage.
- Midday work is skipped and moonstones farther than 30 wrapped tiles are culled.
- `drawMoongates` draws original tile 220 after the map and before item overlays and UI.
- Search is blocked while a source gate is active. An already exposed stone remains a loose item under its gate and Get can collect it. Its item overlay uses 60% opacity while the gate is active, keeping both visible without changing occupancy or gate activity.
- Stable and transitioning gates use the existing base-sprite `ScaleAlpha` pulse
  (0.60–1.10), plus `sprites.MoongateGlowOverlay`. The latter now uses the accurately
  named `WithAdditiveAlphaPulse`: its alpha cycles 0.30–0.60, while its added RGB
  contribution remains constant under `BlendLighter`. Appearance is preserved.
- Visible renderer positions are supplied to lighting and ambient-audio collection.

Berad Graphics supplies the silhouette and additive draw mechanisms; gate stages,
row clipping, transforms, pulse policy and travel sequencing remain game-owned.
Application-lifetime moongate artwork can outlive its originating sprite sheet's
scene. Scene disposal must not deallocate those read-only borrowed frames/masks.

## Travel Workflow

`GameScene.checkMoongateEntry` is the bridge from movement to travel:

1. Confirm that the player occupies an active source gate and no teleport is active.
2. Call `ReviewMoongateTravel` before reserving a workflow.
3. Start high-priority `moongate_teleport` with input blocked.
4. Apply travel feedback using the captured route without advancing the calendar. Failed activation and Spirituality contract all gates; a successful moonstone route preserves the source stage. Stage progression is suspended while teleportation is active.
5. For a moonstone route, append `BuildMoongateTeleportationSequence`, dissolve the Avatar over the source gate, fade the map, apply the authoritative `TransitionToLocation` while covered, and fully materialize the destination gate before map fade-in and Avatar dissolve-in.
6. For Spirituality, clear the source renderer and use `HandoffTo` so source cleanup finishes before the input-owning `shrine_meditation` workflow starts for index 6.
7. Clear dissolve and fade state and process pending provisions in source cleanup.

The coordinate teleportation sequence is six seconds of event-queue animation.
It uses `ThenSequence` rather than starting a nested workflow, so the travel
workflow retains input ownership through map mutation and cleanup. Spirituality
never enters that sequence: `TransitionToShrineMap` snapshots the full source
`PlayerLocation`, and `FinishShrineVisit` restores it after the shrine workflow
and applies the shrine time cost.

Normal gameplay and reviewed Debug Workbench travel share this queue builder.
Workbench status exposes either a coordinate-bearing moonstone route or a
coordinate-free Spirituality route; its stale fingerprint includes the route
kind so a 00:09 review cannot be submitted as ordinary travel after 00:10.

## Audio

Audio is implemented in two paths:

- Visible gates contribute positional `AmbientHum` sources. Volume follows growth and shrink progress and uses the audio manager's distance falloff.
- Travel plays `SoundMoongateTravel` when activation begins and again during materialization. The teleport sequence also uses the unlock sound at dissolve-out.

These sources are collected from active renderers during the normal ambient-audio update; there is no separate unimplemented moongate audio subsystem.

## Key Files

- `internal/game_state/moonstone_system.go`
- `internal/game_state/moonstone_location.go`
- `internal/game_state/moongate_system_integration.go`
- `internal/game_state/moongate_events.go`
- `internal/game_state/teleportation_sequence.go`
- `cmd/ultimav/moongate_renderer.go`
- `cmd/ultimav/gamescene_moongate.go`
- `cmd/ultimav/gamescene_shrine.go`
- `cmd/ultimav/gamescene_teleportation.go`
- `cmd/ultimav/gamescene_update.go`

Focused coverage is in `moongate_stage_test.go`, `moongate_functions_test.go`,
`moongate_entry_integration_test.go`, and `moongate_phase_logic_test.go`, plus
`cmd/ultimav/moongate_renderer_test.go`.
Search/exposure, loose-item arrival, pickup, and source/destination eligibility
regressions are in `internal/game_state/moonstone_integration_test.go`.
End-to-end Spirituality handoff and restoration coverage is in
`cmd/ultimav/gamescene_moongate_spirituality_test.go`.
