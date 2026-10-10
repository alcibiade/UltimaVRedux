# Ultima V Redux

Ultima V Redux is a Go/Ebitengine recreation of Ultima V. This checkout contains
the application source; no source release or tag has been published. Describe
repository availability and native platform acceptance honestly.

Read [README.md](README.md), [installation guidance](docs/INSTALL.md),
[contribution policy](CONTRIBUTING.md), and the [documentation catalog](docs/README.md).

## Repository boundaries

- `cmd/ultimav/`: application, Ebitengine integration, and presentation.
- `internal/game_state/`: authoritative gameplay state and actions.
- `internal/party_state/`, `internal/map_state/`, `internal/ai/`: core systems.
- `internal/references/`: game data constants and reference types.
- `internal/ui/`: reusable presentation widgets.
- `docs/ADR/`: accepted architecture decisions. `docs/gamefacts/` defines stable
  public behavior IDs; executable code and tests establish current Redux behavior.
- `bin/`: local build output.

## Implementation invariants

- Keep core gameplay independent of Ebitengine, input, and UI packages. Prefer
  gameplay mutations in `GameState` and presentation/external effects through
  `SystemCallbacks`. Preserve the established `GameScene`/`GameState` relationship.
- Validate required callbacks at initialization, then use them directly. Do not
  hide missing mandatory wiring behind nil checks.
- Gameplay timing and randomness use `GameState` clocks and RNG sources. Cosmetic
  wall time must not change gameplay outcomes. Tests use fixed seeds and mocks.
- Movement and collision account for terrain and object occupancy. Jimmy consumes
  one key only after a failed physical unlock attempt.
- Mark verified intentional differences with `REDUX DEVIATION:`, a canonical
  Game Facts ID, original rule, Redux outcome, and rationale. Unknown behavior and
  unfixed fidelity bugs are not intentional deviations.
- Use `text.Output` for game text; direct `DrawText` methods are internal. Separate
  control keys from hints and labels from values using semantic colours. Preserve
  selected-row contrast and explain availability in text as well as colour.
- Feedback events are presentation-only: they never own input or mutate gameplay.
  Workflows own orchestration and cleanup.
- The player-facing Ebitengine UI is keyboard-only. Do not add mouse, wheel,
  click, or touch input. The browser Debug Workbench is exempt.
- Workbench wire changes update Go contracts and TypeScript protocol/forms
  together. Edit frontend source rather than generated `webui/dist`, then run
  `make web-build`.
- Return wrapped errors for recoverable input/configuration/save/I/O failures;
  panic for programming errors and impossible states. Use `log.Fatal` only at the
  application boundary and the centralized logging package for diagnostics.
- Improve architecture only where directly relevant. Profile before optimizing;
  reuse buffers in measured hot paths rather than adding speculative systems.

See [architecture](docs/ARCHITECTURE_GUIDELINES.md),
[coding conventions](docs/CODING_CONVENTIONS.md), and
[error handling](docs/ERROR_HANDLING.md) for detailed contracts.

## Working and validation

- Follow the issue-first contribution policy. Preserve unrelated work, investigate
  unfamiliar code, and use dedicated branches. Humans merge PRs.
- Run focused tests during development and `make validate` after final edits.
  Run `make quality-gate` before PR creation or merge when required. Validation
  stays local and non-networked; provision tools separately as described in
  [Development Guide](docs/DEVELOPMENT_GUIDE.md) and [Testing](docs/TESTING.md).
- Maintain zero findings with the repository-pinned linter and meaningful
  regression coverage. Do not describe skipped data-dependent tests as acceptance.
- Keep docs self-contained and preserve durable contracts, explicit uncertainty,
  and stable rule references. Catalog every retained `docs/**/*.md` exactly once
  with its semantic domain owner and authority classification.
- Preserve attribution and licenses for bundled assets and fixtures. Players
  supply the complete original installation externally; do not add original
  copyrighted source or a complete game-data installation.
- Do not stage, commit, push, create PRs, or alter GitHub state without explicit
  authorization. Never commit or push directly from `main`.
