# Ultima V Redux Brand Guidelines

## Quick Reference

- **Identity:** A faithful, modern reimplementation of *Ultima V: Warriors of Destiny*.
- **Visual character:** EGA-authentic, high-contrast, deliberate, and information-dense.
- **Voice:** Direct, historically grounded, pragmatic, and respectful of the original game.
- **Primary accent:** EGA yellow (`#FFFF55`).
- **Supporting accents:** EGA light blue (`#5555FF`), green (`#55FF55`), and red (`#FF5555`).
- **Primary type style:** Readable monospace for tools and diagnostics; established in-game fonts for game presentation.

## Visual Identity

### Colour

Use the project's semantic colour tokens and EGA palette rather than introducing unrelated colours.

| Role | Colour | Hex | Usage |
|---|---|---|---|
| Primary accent | EGA Yellow | `#FFFF55` | Titles, focus, prompts, primary highlights |
| Interactive accent | EGA Light Blue | `#5555FF` | Borders, active navigation, informational controls |
| Positive state | EGA Light Green | `#55FF55` | Enabled, healthy, successful, safe |
| Negative state | EGA Light Red | `#FF5555` | Disabled, damaged, failed, dangerous |
| Primary text | EGA White | `#FFFFFF` | Main readable content |
| Secondary text | EGA Light Gray | `#AAAAAA` | Supporting labels and descriptions |
| Background | EGA Black | `#000000` | Game and tool foundations |

Maintain WCAG AA contrast for browser tools. In game UI, preserve established EGA presentation and use semantic colours from `pkg/color`.

### Typography

- Use the established Ultima fonts for in-game presentation.
- Use a compact monospace stack for the Debug Workbench and technical tools.
- Prefer uppercase labels sparingly for instrument headings, statuses, and compact controls.
- Keep body text sentence-cased and concise.

### Interface Character

- Prefer strong borders, explicit state labels, restrained animation, and dense but readable layouts.
- Enabled and disabled states must differ by text, colour, and structure; never rely on colour alone.
- Avoid generic dashboard cards, soft gradients, excessive rounding, glass effects, or ornamental shadows.
- Preserve the existing visual language when extending an established screen or instrument.

## Voice And Tone

### Traits

- **Faithful:** Distinguish original Ultima V behavior from Redux quality-of-life changes.
- **Direct:** State outcomes, constraints, and errors without filler.
- **Pragmatic:** Prefer actionable language and concrete game terminology.
- **Respectful:** Treat the original game, its creators, and its community with care.

### Context

| Context | Tone |
|---|---|
| Player-facing game text | Period-appropriate where established by original dialogue |
| Debug and developer tools | Precise, concise, operational |
| Documentation | Clear, evidence-based, implementation-aware |
| Errors | Specific about what failed and what remains safe |

Do not imitate archaic speech in technical tooling. Do not claim fidelity without original-game evidence.

## Imagery And Assets

- Use original-compatible pixel-art proportions and nearest-neighbour presentation for game assets.
- Do not smooth, stretch, recolour, or add modern effects to original assets without an explicit design reason.
- Keep copyrighted original game data outside the repository.
- Name new project-owned assets by feature and purpose using the repository's existing conventions.

## Brand Usage

- Refer to the project as **Ultima V Redux**.
- Refer to the 1988 game as **Ultima V: Warriors of Destiny** or **the original Ultima V**.
- Do not imply ownership of the Ultima trademark or original copyrighted assets.
- Use this document as the source of truth for project marketing and design-tool context; implementation-specific UI rules remain authoritative in `docs/ui/` and `pkg/color`.
