# ADR-013: Shared RNG and keyboard mechanisms

## Context

Redux's deterministic random stream preserves exact save continuation, while its
keyboard adapter has accumulated distinct consumption, release and cadence rules.
These mechanisms are useful outside the game, but their game-policy wrappers are
not generic defaults. Extraction must preserve existing saves and input behavior.

## Decision

Use independently versioned `berad-rng` and `berad-input` modules from Berad SDK.
Both are engine-free and expose only production-backed mechanisms.

`internal/deterministic_random` is a type alias and constructor facade over the
SDK source, retaining existing borrower/provider identities. The SDK preserves
the established x/exp/rand PCG stream and raw draw counting, including rejection
sampling and shuffle order. Save seed/SourceCalls fields remain game-owned and
unchanged. Dice bounds, odds, seed selection and shared draw order remain Redux
policy. The unused floating-point convenience was not extracted.

`pkg/input.Keyboard` remains the Ebitengine/game adapter. It supplies concrete
key identities and live observations to a comparable-key SDK core. The SDK owns
consumption, stronger suppression, release barriers, captured edge identity,
binding filtering, millisecond cadence and owner-local repeat countdown. Redux
retains key meanings, modifier/chord precedence, original directional priority,
simulation compatibility, native text/IME and scene/workflow ownership.

This is an incremental replacement of the shared mechanisms, not a snapshot-only
rewrite. Four scene owners retain their release servicing and early-return order.
GameScene captures edges before ownership changes; ordinary queries remain live.
The input clock is independent of paused gameplay time. Strict legacy cadence,
focused-update list cadence and scene-owned Pass cadence remain distinct.

Mouse readiness means a future pointer adapter can invoke host operations alongside
keyboard input. No pointer handling, generalized action bus or widget framework is
introduced, and Redux remains keyboard-only.

## Consequences

- Reusable algorithm tests and independent consumers live with the SDK; game tests
  verify provider/save continuation and keyboard/focus/modal integration.
- Sources require host serialization. RNG has no hidden seed/time, and input has
  no native polling or implicit clock in the SDK.
- Input handling remains outside authoritative gameplay, even though its SDK
  implementation is engine-free; the dependency checker enforces that boundary.
- Existing compatibility setters and simulation paths remain only where live game
  callers need them. Verified unused helpers are retired rather than exported.
- Available SDK revisions must precede game pins. No checked-in local replacement
  is needed, and caches/examples cannot substitute for ordinary installation proof.

See the [time/RNG contract](../systems/TIME_SYSTEM.md#shared-rng-ownership) and
[input contract](../systems/INPUT_HANDLING.md#keyboard-mechanism-boundary).
