# Game Facts

Technical descriptions of original game behavior, compatibility formats, and
Redux handling. Rules use stable identifiers so reports and implementation can
refer to a specific condition rather than an entire subsystem.

This is a documentation pilot. The application has not yet been staged here;
implementation/test paths are logical locators for the reviewed development
implementation, not links to files already available in this checkout. No public
release or cross-version gameplay acceptance is implied.

| Topic | Scope |
| --- | --- |
| [Karma](KARMA.md) | Party karma, moral events, dialogue, recovery, and reports. |
| [Legacy saves](SAVED_GAM_STRUCTURE.md) | Saved allocation, roster, inventory, effects, and object-list ownership. |
| [Combat records](BRIT_CBT_FORMAT.md) | Battlefield storage, formations, selection, and dungeon-only metadata. |
| [Equipment data](EQUIPMENT_ORIGINAL_DATA.md) | Canonical equipment identities and semantic reference ratings. |
| [Equipment mechanics](EQUIPMENT_STATS.md) | Ready, attacks, commitment, special equipment, and pricing. |
| [Inventory](INVENTORY_SYSTEM.md) | Ownership categories, capacities, and resource commitment. |
| [Graphics formats](GRAPHICS_FORMATS.md) | Archive layouts, decoder assumptions, fonts, and signature paths. |
| [Signs and fonts](SIGNS_AND_FONTS.md) | Text/control semantics and symbolic displays. |
| [Tiles](TILES.md) | Contextual identity, movement, occupancy, animation, and presentation. |
| [Conventions](CONVENTIONS.md) | Rule IDs, statuses, pseudocode, and groups. |

Report suspected errors through the [issue-first contribution process](../../CONTRIBUTING.md).
Include the rule ID and the circumstances that distinguish the behaviors.
