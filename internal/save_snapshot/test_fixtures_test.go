package save_snapshot

import (
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func validEnvelope(snapshot SaveSnapshot) Envelope {
	return Envelope{Format: EnvelopeFormat, EnvelopeVersion: CurrentEnvelopeVersion, Metadata: Metadata{Name: "Native capture", Description: "Complete test capture.", CreatedAt: time.Date(2026, time.August, 6, 12, 30, 0, 0, time.UTC), GameVersion: "test"}, Snapshot: snapshot}
}

func validSnapshot() SaveSnapshot {
	position := validPosition()
	arrivedScheduleSlot := 0
	snapshot := SaveSnapshot{
		Version:      CurrentVersion,
		Presentation: journal.EmptyPresentation(),
		Party:        PartySnapshot{Karma: 75, MetNPCs: []NPCFlagSnapshot{{Location: 2, NPC: 3}}},
		Inventory:    InventorySnapshot{Food: 100, Gold: 25, Keys: 3, Gems: 1, Torches: 4},
		World:        WorldSnapshot{Position: position, LastLargeMapPosition: position, Calendar: CalendarSnapshot{Year: 100, Month: 4, Day: 12, Hour: 9, Minute: 30, Turns: 42}},
		Wind:         WindSnapshot{Direction: 1, UpdateCounter: 18, NextTickMs: 55},
		Random:       RandomSnapshot{Seed: 1, SourceCalls: 4},
		Continuation: ContinuationSnapshot{
			ActivePlayerIndex:  -1,
			BeggarKarmaCounter: 100,
		},
		Redux: ReduxSnapshot{
			DungeonDiscoveries: make([]DungeonDiscoverySnapshot, 0),
			PlayerKnowledge:    make([]PlayerKnowledgeSnapshot, 0),
			QuestJournal:       make([]QuestJournalEventSnapshot, 0),
		},
		MapUnits:     []MapUnitSnapshot{{Kind: "enemy", ReferenceID: 1, NPCNum: 2, Position: position, Visible: true, AIOverrideProvenance: string(references.AIOverrideProvenanceNone), ArrivedScheduleSlot: &arrivedScheduleSlot, EnemyVariant: "standard", EnemyCurrentHP: 10}},
		ItemStacks:   []ItemStackSnapshot{{Position: position, Items: []ItemSnapshot{{Type: 9, ID: 5, Quantity: 10}}}},
		MapMutations: []MapMutationSnapshot{{Position: position, Kind: MapMutationSetTile, Value: 42}},
	}
	snapshot.Inventory.Equipment[1] = 2
	snapshot.Inventory.Spells[3] = 4
	snapshot.Progression.Recipes[3] = RecipeSnapshot{Known: true, Mask: 6}
	snapshot.Party.Roster[0] = CharacterSnapshot{Occupied: true, Name: "Avatar", Class: 'A', Status: 'G', Gender: 0x0b, Level: 5, Experience: 100, Strength: 20, Dexterity: 18, Intelligence: 25, CurrentHP: 50, MaximumHP: 60, CurrentMP: 20, Weapon: 1, Armor: 1}
	snapshot.Progression.Moonstones[0] = MoonstoneSnapshot{X: 10, Y: 20, State: 0, IsFound: true}
	snapshot.Progression.Shrines[0] = ShrineSnapshot{Active: true, Completed: true}
	snapshot.Progression.Shadowlords.Alive = [ShadowlordCount]bool{true, true, true}
	snapshot.Progression.Shadowlords.LastCalled = -1
	snapshot.Progression.Shadowlords.AutomaticPresence = -1
	return snapshot
}

func validPosition() PositionSnapshot { return PositionSnapshot{Location: 0, X: 10, Y: 20, Floor: 0} }
