# Game Facts conventions

## Rules and references

A rule is declared by a heading such as `### KARMA-0011`, followed by a descriptive
title. Its stable anchor is `#karma-0011`; title or file changes do not repurpose
the identifier. IDs are unique and never recycled.

Each pilot section has `Kind` (`rule` or `group`) and `Uses` (comma-separated IDs
or `none`). A rule additionally states `Original` and `Redux` assessments. Other
prose explains scope, conditions, algorithms, examples, and limitations.

Original assessments are **Established**, **Incomplete**, or **Conflicting**,
always within the stated edition and input scope. Redux assessments are
**Matches**, **Intentional deviation**, **Known discrepancy**, **Unverified**, or
**Not implemented**. A scoped match is not a claim about every malformed input,
unsupported release, or unexamined caller.

An intentional deviation explains what the original does, how Redux differs, and
why that choice was made. An unknown or accidental mismatch is not a deviation.

Implementation comments may use `ref:KARMA-0011` or the established
`REDUX DEVIATION: KARMA-0014` prefix, alongside a useful explanation. Public rules
must be understandable on their own. Original data filenames, format offsets,
semantic field names, and public observations are appropriate reference material.

## Pseudocode

State units, valid inputs, inclusive random bounds, integer rounding, saturation
or wrapping, and relevant operation order. Show gates, early exits, resource
consumption, and failure effects. Descriptive pseudocode explains behavior rather
than reproducing incidental implementation structure.

In the Karma pilot, `min` and `max` have their ordinary mathematical meaning,
`floor` rounds downward, and arithmetic is wide enough for the stated valid inputs
unless wrapping is explicitly specified. A range written `0..29 inclusive`
includes both endpoints. Effects at different steps must not be silently combined
when intermediate state matters.

## Groups and shared rules

A group supplies navigation and a brief purpose, not another copy of its members.
The pilot allows one group layer; members must be rules. `Uses` dependencies are
acyclic. Prefer specific leaf IDs in implementation comments and issue reports.
Ordinary related-topic links need not imply dependency expansion.

The public Markdown is the maintained explanation. Indexes and assembled views
are derived from it. Keep numeric tables in one canonical location and refer to
them rather than maintaining competing copies.
