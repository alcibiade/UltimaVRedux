# ADR-004: Integration Testing with Real Game Data

**Status**: Accepted
**Date**: 2025-08-25

## Context

Ultima 5 Redux interprets proprietary Ultima V saves, maps, references, and conversation files. Synthetic fixtures are appropriate for isolated rules, but they cannot reliably represent all byte layouts, cross-file relationships, map content, or conversation scripts found in the original data. Integration defects can therefore remain invisible in unit tests even when each component passes independently.

The complete original installation remains externally provisioned; retained
bundled assets and fixtures have their own notices and do not substitute for that
installation. Core gameplay communicates with presentation systems through
`SystemCallbacks`, so integration tests need to observe those contracts without
starting the real UI.

## Decision

Use real Ultima V game data for integration tests whose behavior depends on original file content or fully initialized cross-system state. Keep pure rules covered by asset-independent unit tests. Make real-data tests deterministic through controlled game-state randomness and time, and observe presentation contracts through test callbacks.

Real-data tests skip by default when assets are unavailable. Environments that explicitly promise the assets may require their presence and fail instead of skip.

## Rationale

- Real files expose parsing, indexing, map-layout, and cross-reference defects that representative mocks can miss.
- Testing against authentic content provides stronger evidence of compatibility with the original game.
- Asset-independent unit tests preserve fast feedback and remain runnable by all contributors.
- Deterministic seeds and clocks make failures reproducible despite using complex game state.
- Mocked `SystemCallbacks` validate user-visible effects while preserving the core/presentation package boundary.
- Optional local assets respect distribution constraints without forcing the entire suite to become environment-dependent.

## Consequences

- Integration coverage depends on developers or CI environments provisioning legally obtained game data.
- A skipped test does not prove real-data compatibility, so provisioned validation environments should fail when expected assets are missing.
- Helpers for loading authentic state and installing callbacks require maintenance as formats and callback contracts evolve.
- Real-data tests are slower and broader than unit tests, so assertions should focus on stable behavior rather than incidental data details.

Operational guidance and fixture recipes belong in [Testing](../TESTING.md),
not in this decision record.
