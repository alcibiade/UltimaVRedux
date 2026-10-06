# Game Facts

Technical descriptions of original game behavior, compatibility formats, and
Redux handling. Rules use stable identifiers so reports and implementation can
refer to a specific condition rather than an entire subsystem.

This is a documentation conversion candidate. The application has not yet been staged here;
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
| [Time costs](TIME_COSTS.md) | Calendar modifiers, context ownership, actor cadence, slow steps, and special loops. |
| [Provisions](PROVISIONS_CONSUMPTION.md) | Poison/meal ordering, observed hours, fuel, and key/gem commitment. |
| [Camping and rest](CAMPING_MECHANICS.md) | Guard/ambush ordering, duration, rewards, beds, and paid inn rest. |
| [Astronomy](ASTRONOMY.md) | Sky inheritance, instrument gates, planet tracks, stars/comets, and flexible navigation. |
| [Moonstones](MOONSTONE_LOCATIONS.md) | Placements, phases, gate stages/routes, and earned destination knowledge. |
| [Lighting](LIGHTING.md) | Original sight/illumination and Redux tint, daylight, and opening contracts. |
| [Conventions](CONVENTIONS.md) | Rule IDs, statuses, pseudocode, and groups. |

Report suspected errors through the [issue-first contribution process](../../CONTRIBUTING.md).
Include the rule ID and the circumstances that distinguish the behaviors.
