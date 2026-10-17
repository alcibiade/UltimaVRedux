package save_snapshot

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
)

// Formats must be registered AND asserted. JSON Schema maxLength measures
// runes, not bytes; standard keywords also cannot compare visit revisions.
func registerJournalSchemaFormats(compiler *jsonschema.Compiler, version int) {
	compiler.RegisterFormat(&jsonschema.Format{Name: "ultima5-world-journal", Validate: func(value any) error {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		var snapshot SaveSnapshot
		if version == 24 {
			var previous saveSnapshotV24
			if err = decodeRequiredJSON(data, &previous); err != nil {
				return err
			}
			snapshot = previous.validationView()
		} else if version == 23 {
			var previous saveSnapshotV23
			if err = decodeRequiredJSON(data, &previous); err != nil {
				return err
			}
			snapshot = previous.validationView()
		} else if version == 22 {
			var previous saveSnapshotV22
			if err = decodeRequiredJSON(data, &previous); err != nil {
				return err
			}
			snapshot = previous.validationView()
		} else if version == 21 {
			var previous saveSnapshotV21
			if err = decodeRequiredJSON(data, &previous); err != nil {
				return err
			}
			snapshot = previous.validationView()
		} else if version == 20 {
			var previous saveSnapshotV20
			if err = decodeRequiredJSON(data, &previous); err != nil {
				return err
			}
			snapshot = previous.validationView()
		} else {
			if err = decodeRequiredJSON(data, &snapshot); err != nil {
				return err
			}
		}
		var issues []Issue
		add := func(path, code, message string) {
			issues = append(issues, Issue{Path: path, Code: code, Message: message})
		}
		validateWorldJournal(snapshot, add)
		validateChestAssessments(snapshot, add)
		validateOpenDoor(snapshot, add)
		if version >= 24 {
			validatePersonalStateV24(snapshot, add)
		}
		if version >= 25 {
			if err = snapshot.Excerpts.Validate(); err != nil {
				add("excerpts", "invalid", err.Error())
			}
		}
		if len(issues) != 0 {
			return &ValidationError{Issues: issues}
		}
		return nil
	}})
	compiler.RegisterFormat(&jsonschema.Format{Name: "ultima5-journal-conversation", Validate: func(value any) error {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		var conversation journal.Conversation
		if err = decodeRequiredJSON(data, &conversation); err != nil {
			return err
		}
		return conversation.Validate()
	}})
	compiler.RegisterFormat(&jsonschema.Format{Name: "ultima5-journal-history", Validate: func(value any) error {
		history, ok := value.(string)
		if !ok {
			return nil // The schema's type assertion owns non-string rejection.
		}
		return journal.ValidateHistory(history)
	}})
	compiler.RegisterFormat(&jsonschema.Format{Name: "ultima5-journal-presentation", Validate: func(value any) error {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		var presentation journal.Presentation
		if err = decodeRequiredJSON(data, &presentation); err != nil {
			return err
		}
		return journal.ValidatePresentation(presentation)
	}})
	compiler.RegisterFormat(&jsonschema.Format{Name: "ultima5-journal-events", Validate: func(value any) error {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		var events []QuestJournalEventSnapshot
		if err = decodeRequiredJSON(data, &events); err != nil {
			return err
		}
		var issues []Issue
		// Life-state consistency belongs to the complete snapshot validator.
		validateQuestJournal(events, [3]bool{}, version, func(path, code, message string) {
			issues = append(issues, Issue{Path: path, Code: code, Message: message})
		})
		if len(issues) != 0 {
			return &ValidationError{Issues: issues}
		}
		return nil
	}})
}

func removeJournalV16Fields(snapshot map[string]any) {
	removeV20Fields(snapshot)
	delete(snapshot, "lastConversation")
	delete(snapshot, "dungeonJournal")
	delete(snapshot, "presentation")
	for _, value := range snapshot["redux"].(map[string]any)["questJournal"].([]any) {
		delete(value.(map[string]any), "sourceName")
	}
}

func versionFifteenEnvelopeData(t *testing.T, snapshot SaveSnapshot, mutate func(map[string]any)) []byte {
	t.Helper()
	document := currentEnvelopeDocument(t, snapshot)
	body := document["snapshot"].(map[string]any)
	body["version"] = 15
	removeJournalV16Fields(body)
	if mutate != nil {
		mutate(body)
	}
	data, err := json.Marshal(document)
	require.NoError(t, err)
	return data
}

func TestJournalV16HistoryByteAndCanonicalSchemaParity(t *testing.T) {
	t.Parallel()
	compiled, _ := compileCurrentNativeSaveSchema(t)
	for _, test := range []struct {
		name, history string
		valid         bool
	}{
		{name: "empty", valid: true},
		{name: "spaces preserved", history: "  A  \n\nB ", valid: true},
		{name: "repeated entries", history: "A\nA", valid: true},
		{name: "line boundary", history: strings.Repeat("x", 4096), valid: true},
		{name: "multibyte boundary", history: strings.Repeat("\u00e9", 2048), valid: true},
		{name: "multibyte overflow", history: strings.Repeat("\u00e9", 2049)},
		{name: "line overflow", history: strings.Repeat("x", 4097)},
		{name: "entry boundary", history: strings.Repeat("A\n", 63) + "A", valid: true},
		{name: "entry overflow", history: strings.Repeat("A\n", 64) + "A"},
		{name: "aggregate boundary", history: strings.Repeat(strings.Repeat("x", 4095)+"\n", 15) + strings.Repeat("x", 4096), valid: true},
		{name: "aggregate overflow", history: strings.Repeat(strings.Repeat("x", 4096)+"\n", 15) + strings.Repeat("x", 4096)},
		{name: "leading separator", history: "\nA"},
		{name: "trailing separator", history: "A\n"},
		{name: "doubled separator", history: "A\n\n\nB"},
		{name: "blank spaces", history: "A\n \nB"},
		{name: "blank unicode", history: "\u00a0"},
		{name: "tab", history: "A\tB"},
		{name: "control", history: "A\x00B"},
		{name: "line separator", history: "A\u2028B"},
		{name: "paragraph separator", history: "A\u2029B"},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validSnapshot()
			snapshot.Presentation.History = test.history
			_, encodeErr := EncodeCanonical(validEnvelope(snapshot))
			data, err := json.Marshal(validEnvelope(snapshot))
			require.NoError(t, err)
			decoded, decodeErr := Decode(data)
			schemaErr := compiled.Validate(decodeJSONSchemaInstance(t, data))
			if test.valid {
				require.NoError(t, encodeErr)
				require.NoError(t, decodeErr)
				require.NoError(t, schemaErr)
				assert.Equal(t, snapshot.Presentation, decoded.Snapshot.Presentation)
			} else {
				assert.ErrorIs(t, encodeErr, ErrInvalidSnapshot)
				assert.Error(t, decodeErr)
				assert.Error(t, schemaErr)
			}
		})
	}
	snapshot := validSnapshot()
	snapshot.Presentation.History = "\xff"
	_, err := EncodeCanonical(validEnvelope(snapshot))
	assert.ErrorIs(t, err, ErrInvalidSnapshot)
}

func TestJournalV16PresentationRequiredMembersAndRevisions(t *testing.T) {
	t.Parallel()
	compiled, _ := compileCurrentNativeSaveSchema(t)
	base := currentEnvelopeDocument(t, validSnapshot())
	for _, path := range [][]string{
		{"presentation"}, {"presentation", "history"}, {"presentation", "visits"},
		{"presentation", "visits", "revision"}, {"presentation", "visits", "changes"}, {"presentation", "visits", "seen"},
	} {
		for _, value := range []any{nil, true, 1, "wrong", []any{}} {
			t.Run(strings.Join(path, ".")+fmt.Sprint(value), func(t *testing.T) {
				instance := cloneJSONDocument(t, base)
				object := instance["snapshot"].(map[string]any)
				for _, member := range path[:len(path)-1] {
					object = object[member].(map[string]any)
				}
				member := path[len(path)-1]
				delete(object, member)
				assertJournalWireRejected(t, compiled, instance)
				object[member] = value
				// These are valid scalar/empty values for their own wire types.
				if (member == "history" && value == "wrong") || (member == "revision" && value == 1) {
					return
				}
				if _, array := value.([]any); array && member == "changes" {
					return
				}
				assertJournalWireRejected(t, compiled, instance)
			})
		}
	}
	for _, visits := range []journal.Visits{
		{Changes: nil},
		{Changes: []journal.Change{}, Seen: [3]uint64{1}},
		{Revision: 1, Changes: []journal.Change{{Tab: 0, Entry: 0, Revision: 1}}},
		{Revision: 1, Changes: []journal.Change{{Tab: 0, Entry: journal.MaxChronicleEntries + 1, Revision: 1}}},
		{Revision: 1, Changes: []journal.Change{{Tab: 1, Entry: 8, Revision: 1}}},
		{Revision: 1, Changes: []journal.Change{{Tab: 2, Entry: 3, Revision: 1}}},
		{Revision: 1, Changes: []journal.Change{{Tab: 3, Entry: 0, Revision: 1}}},
		{Revision: 1, Changes: []journal.Change{{Tab: 0, Entry: 1, Revision: 0}}},
		{Revision: 1, Changes: []journal.Change{{Tab: 0, Entry: 1, Revision: 2}}},
		{Revision: 1, Changes: []journal.Change{{Tab: 1, Entry: 0, Revision: 1}, {Tab: 0, Entry: 1, Revision: 1}}},
		{Revision: 1, Changes: []journal.Change{{Tab: 0, Entry: 1, Revision: 1}, {Tab: 0, Entry: 1, Revision: 1}}},
	} {
		snapshot := validSnapshot()
		snapshot.Presentation.Visits = visits
		_, err := EncodeCanonical(validEnvelope(snapshot))
		assert.ErrorIs(t, err, ErrInvalidSnapshot)
		data, err := json.Marshal(validEnvelope(snapshot))
		require.NoError(t, err)
		_, err = Decode(data)
		assert.Error(t, err)
		assert.Error(t, compiled.Validate(decodeJSONSchemaInstance(t, data)))
	}
	snapshot := validSnapshot()
	snapshot.Presentation.Visits.Revision = math.MaxUint64
	snapshot.Presentation.Visits.Seen = [3]uint64{math.MaxUint64, 0, 1}
	for tab, count := range []int{journal.MaxChronicleEntries, 8, 3} {
		for entry := range count {
			index := entry
			if tab == 0 {
				index++
			}
			snapshot.Presentation.Visits.Changes = append(snapshot.Presentation.Visits.Changes, journal.Change{Tab: tab, Entry: index, Revision: math.MaxUint64})
		}
	}
	require.Len(t, snapshot.Presentation.Visits.Changes, journal.MaxChanges)
	data, err := EncodeCanonical(validEnvelope(snapshot))
	require.NoError(t, err)
	require.NoError(t, compiled.Validate(decodeJSONSchemaInstance(t, data)))
	decoded, err := Decode(data)
	require.NoError(t, err)
	assert.Equal(t, snapshot, decoded.Snapshot)
	snapshot.Presentation.Visits.Changes = append(snapshot.Presentation.Visits.Changes, snapshot.Presentation.Visits.Changes[0])
	_, err = EncodeCanonical(validEnvelope(snapshot))
	assert.ErrorIs(t, err, ErrInvalidSnapshot)
	data, err = json.Marshal(validEnvelope(snapshot))
	require.NoError(t, err)
	_, err = Decode(data)
	assert.ErrorIs(t, err, ErrInvalidSnapshot)
	assert.Error(t, compiled.Validate(decodeJSONSchemaInstance(t, data)))
}

func assertJournalWireRejected(t *testing.T, schema *jsonschema.Schema, instance map[string]any) {
	t.Helper()
	data, err := json.Marshal(instance)
	require.NoError(t, err)
	_, err = Decode(data)
	assert.Error(t, err)
	assert.Error(t, schema.Validate(decodeJSONSchemaInstance(t, data)))
}

func TestJournalV16HistoricalPromotionIsQuietAndHonest(t *testing.T) {
	t.Parallel()
	for version, encode := range map[int]func(*testing.T, SaveSnapshot, func(map[string]any)) []byte{
		12: versionTwelveEnvelopeData, 13: versionThirteenEnvelopeData, 14: versionFourteenEnvelopeData, 15: versionFifteenEnvelopeData,
	} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			source := validSnapshot()
			source.Redux.PlayerKnowledge = append(validLoreKnowledge(), validPlayerKnowledge()...)
			source.Progression.Dungeons = [DungeonCount]bool{true, true, true, true, true, true, true, true}
			source.Progression.Shadowlords.Alive[0] = false
			source.Redux.QuestJournal = []QuestJournalEventSnapshot{{Sequence: 1, Kind: QuestJournalEventShadowlordDefeated, Subject: QuestJournalSubjectFaulinei, Source: QuestJournalSourceMigration, Acknowledged: true}}
			decoded, err := Decode(encode(t, source, nil))
			require.NoError(t, err)
			assert.Equal(t, CurrentVersion, decoded.Snapshot.Version)
			assert.Equal(t, source.Redux.PlayerKnowledge, decoded.Snapshot.Redux.PlayerKnowledge, "preserve unread, source and date without inferred associations")
			assert.Equal(t, source.Redux.QuestJournal[0], decoded.Snapshot.Redux.QuestJournal[0])
			assert.Equal(t, journal.EmptyPresentation(), decoded.Snapshot.Presentation, "old projected facts start seen with no transcript")
			require.Len(t, decoded.Snapshot.Redux.QuestJournal, 1+len(validLoreKnowledge()), "dungeon progress does not fabricate Word of Power lore or events")
			for index, lore := range validLoreKnowledge() {
				event := decoded.Snapshot.Redux.QuestJournal[index+1]
				assert.Equal(t, QuestJournalEventKnowledgeLearned, event.Kind)
				assert.Equal(t, lore.Category+"."+lore.Key, event.Subject)
				assert.Equal(t, lore.DiscoveredOn, event.OccurredOn)
				assert.Equal(t, lore.Source.Kind, event.Source)
				assert.Equal(t, lore.Source.DisplayName, event.SourceName)
				assert.True(t, event.Acknowledged)
			}
			canonical, err := EncodeCanonical(decoded)
			require.NoError(t, err)
			reloaded, err := Decode(canonical)
			require.NoError(t, err)
			decoded.legacyExcerpts = false
			assert.Equal(t, decoded, reloaded, "do not duplicate backfill on current reload")
		})
	}
}

func TestJournalV16FrozenVersionsRejectNewSemantics(t *testing.T) {
	t.Parallel()
	for version, encode := range map[int]func(*testing.T, SaveSnapshot, func(map[string]any)) []byte{
		12: versionTwelveEnvelopeData, 13: versionThirteenEnvelopeData, 14: versionFourteenEnvelopeData, 15: versionFifteenEnvelopeData,
	} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			compiled, _ := compileNativeSaveSchema(t, version)
			for name, mutate := range map[string]func(map[string]any){
				"presentation": func(body map[string]any) { body["presentation"] = journal.EmptyPresentation() },
				"new category": func(body map[string]any) {
					body["redux"].(map[string]any)["playerKnowledge"].([]any)[0].(map[string]any)["category"] = "journal"
					body["redux"].(map[string]any)["playerKnowledge"].([]any)[0].(map[string]any)["key"] = "domain.falsehood"
				},
				"new key": func(body map[string]any) {
					body["redux"].(map[string]any)["playerKnowledge"].([]any)[0].(map[string]any)["key"] = "domain.falsehood"
				},
				"observation": func(body map[string]any) {
					body["redux"].(map[string]any)["playerKnowledge"].([]any)[0].(map[string]any)["source"].(map[string]any)["kind"] = "observation"
				},
				"revelation": func(body map[string]any) {
					body["redux"].(map[string]any)["playerKnowledge"].([]any)[0].(map[string]any)["source"].(map[string]any)["kind"] = "revelation"
				},
				"new event kind": func(body map[string]any) {
					body["redux"].(map[string]any)["questJournal"].([]any)[0].(map[string]any)["kind"] = "knowledge-learned"
				},
				"codex event kind": func(body map[string]any) {
					body["redux"].(map[string]any)["questJournal"].([]any)[0].(map[string]any)["kind"] = "codex-answer-learned"
				},
				"new event source": func(body map[string]any) {
					body["redux"].(map[string]any)["questJournal"].([]any)[0].(map[string]any)["source"] = "dialogue"
				},
				"sourceName": func(body map[string]any) {
					body["redux"].(map[string]any)["questJournal"].([]any)[0].(map[string]any)["sourceName"] = ""
				},
			} {
				t.Run(name, func(t *testing.T) {
					source := validSnapshot()
					source.Redux.PlayerKnowledge = validLoreKnowledge()
					source.Progression.Shadowlords.Alive[0] = false
					source.Redux.QuestJournal = []QuestJournalEventSnapshot{{Sequence: 1, Kind: QuestJournalEventShadowlordDefeated, Subject: QuestJournalSubjectFaulinei, Source: QuestJournalSourceMigration, Acknowledged: true}}
					data := encode(t, source, mutate)
					_, err := Decode(data)
					assert.Error(t, err)
					assert.Error(t, compiled.Validate(archivedEnvelopeOneSchemaInstance(t, data)))
				})
			}
		})
	}
}

func TestJournalV16PreservesV15CadenceAndCanonicalRepositoryRevision(t *testing.T) {
	t.Parallel()
	source := validSnapshot()
	source.Effects.DurationType, source.Effects.DurationTurns = 'Q', 7
	source.Effects.QuicknessWorldActorSkip, source.Effects.TownMountedActorSkip = true, true
	source.Effects.TownQuicknessActorSkip, source.Effects.OutdoorMountedActorSkip = true, true
	data := versionFifteenEnvelopeData(t, source, nil)
	decoded, err := Decode(data)
	require.NoError(t, err)
	assert.Equal(t, source, decoded.Snapshot, "all v15 cadence bits retain their meaning")
	_, err = DecodeUnsupportedVersion(data)
	assert.ErrorIs(t, err, ErrInvalidEnvelope)
	repository, roots := newTestRepository(t)
	require.NoError(t, os.MkdirAll(roots[0], 0o755))
	path := filepath.Join(roots[0], "journal-v15.json")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	loaded, err := repository.Load("project:journal-v15")
	require.NoError(t, err)
	canonical, err := EncodeCanonical(loaded.Envelope)
	require.NoError(t, err)
	assert.Equal(t, repositoryfs.ContentRevision(canonical), loaded.Revision)
	assert.NotEqual(t, repositoryfs.ContentRevision(data), loaded.Revision)
	retained, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, retained)
}

func TestJournalV16SchemaRetainsUnchangedDefinitions(t *testing.T) {
	t.Parallel()
	_, previous := compileNativeSaveSchemaFile(t, "native-save-e2-s15.schema.json")
	_, current := compileNativeSaveSchemaFile(t, "native-save-e2-s16.schema.json")
	before := previous.(map[string]any)["$defs"].(map[string]any)
	after := current.(map[string]any)["$defs"].(map[string]any)
	for name, definition := range before {
		if strings.HasPrefix(name, "bundled") && name != "bundled303" && name != "bundled304" || name == "continuation" {
			assert.Equal(t, definition, after[name], name)
		}
	}
}

func TestJournalV16LoreSourceUnicodeWhitespaceSchemaParity(t *testing.T) {
	t.Parallel()
	compiled, _ := compileCurrentNativeSaveSchema(t)
	names := []string{"", "Speaker", "\u001c", "\u180e", "\u200b", "\ufeff"}
	whitespace := ""
	for char := rune(0); char <= unicode.MaxRune; char++ {
		if unicode.IsSpace(char) {
			space := string(char)
			whitespace += space
			names = append(names, space, space+"Speaker"+space)
		}
	}
	names = append(names, whitespace, whitespace+"Speaker"+whitespace)
	for _, lore := range []struct{ category, key string }{
		{KnowledgeCategoryDungeon, DungeonKnowledgeDoomWordOfPower},
		{KnowledgeCategoryJournal, JournalKnowledgeFlameTruth},
		{KnowledgeCategoryShadowlord, ShadowlordKnowledgeFaulineiName},
		{KnowledgeCategoryShrine, ShrineKnowledgeHonestyMantra},
	} {
		for _, source := range []string{KnowledgeSourceDialogue, KnowledgeSourceObservation, KnowledgeSourceRevelation} {
			for _, name := range names {
				t.Run(fmt.Sprintf("%s/%s/%q", lore.category, source, name), func(t *testing.T) {
					snapshot := validSnapshot()
					snapshot.Redux.PlayerKnowledge = []PlayerKnowledgeSnapshot{{
						Category: lore.category, Key: lore.key,
						Source:       KnowledgeSourceSnapshot{Kind: source, DisplayName: name},
						DiscoveredOn: &BritannianDateSnapshot{Year: 139, Month: 4, Day: 5},
					}}
					_, encodeErr := EncodeCanonical(validEnvelope(snapshot))
					data, err := json.Marshal(validEnvelope(snapshot))
					require.NoError(t, err)
					decoded, decodeErr := Decode(data)
					schemaErr := compiled.Validate(decodeJSONSchemaInstance(t, data))
					if strings.TrimSpace(name) == "" {
						assert.ErrorIs(t, encodeErr, ErrInvalidSnapshot)
						assert.ErrorIs(t, decodeErr, ErrInvalidSnapshot)
						assert.Error(t, schemaErr)
					} else {
						require.NoError(t, encodeErr)
						require.NoError(t, decodeErr)
						require.NoError(t, schemaErr)
						assert.Equal(t, name, decoded.Snapshot.Redux.PlayerKnowledge[0].Source.DisplayName)
					}
				})
			}
		}
	}
}

func TestJournalV16CurrentLoreAndEventCatalogParity(t *testing.T) {
	t.Parallel()
	compiled, _ := compileCurrentNativeSaveSchema(t)
	keys := map[string][]string{
		KnowledgeCategoryDungeon:    {"deceit.word_of_power", "despise.word_of_power", "destard.word_of_power", "wrong.word_of_power", "covetous.word_of_power", "shame.word_of_power", "hythloth.word_of_power", "doom.word_of_power"},
		KnowledgeCategoryJournal:    {"domain.falsehood", "domain.hatred", "domain.cowardice", "flame.truth", "flame.love", "flame.courage", "banishment.method", "resistance.password", "stones.melody"},
		KnowledgeCategoryShadowlord: {"faulinei.name", "faulinei.association", "astaroth.name", "astaroth.association", "nosfentor.name", "nosfentor.association"},
		KnowledgeCategoryShrine:     {"honesty.mantra", "compassion.mantra", "valour.mantra", "justice.mantra", "sacrifice.mantra", "honor.mantra", "spirituality.mantra", "humility.mantra"},
	}
	date := &BritannianDateSnapshot{Year: 139, Month: 4, Day: 5}
	allEvents := validSnapshot()
	for _, category := range []string{KnowledgeCategoryJournal, KnowledgeCategoryShadowlord, KnowledgeCategoryShrine, KnowledgeCategoryDungeon} {
		for _, key := range keys[category] {
			for _, source := range []string{KnowledgeSourceDialogue, KnowledgeSourceObservation, KnowledgeSourceRevelation} {
				snapshot := validSnapshot()
				snapshot.Redux.PlayerKnowledge = []PlayerKnowledgeSnapshot{{Category: category, Key: key, Source: KnowledgeSourceSnapshot{Kind: source, DisplayName: "Honest provenance"}, DiscoveredOn: date}}
				snapshot.Redux.QuestJournal = []QuestJournalEventSnapshot{{Sequence: 1, Kind: QuestJournalEventKnowledgeLearned, Subject: category + "." + key, OccurredOn: date, Source: source, SourceName: "Honest provenance"}}
				data, err := EncodeCanonical(validEnvelope(snapshot))
				require.NoError(t, err)
				require.NoError(t, compiled.Validate(decodeJSONSchemaInstance(t, data)))
				decoded, err := Decode(data)
				require.NoError(t, err)
				assert.Equal(t, snapshot, decoded.Snapshot, "standalone facts require no invented associations or companion facts")
			}
			allEvents.Redux.QuestJournal = append(allEvents.Redux.QuestJournal, QuestJournalEventSnapshot{Sequence: uint64(len(allEvents.Redux.QuestJournal) + 1), Kind: QuestJournalEventKnowledgeLearned, Subject: category + "." + key, OccurredOn: date, Source: QuestJournalSourceDialogue, SourceName: "Speaker"})
		}
	}
	for _, virtue := range []string{"honesty", "compassion", "valour", "justice", "sacrifice", "honor", "spirituality", "humility"} {
		allEvents.Redux.QuestJournal = append(allEvents.Redux.QuestJournal, QuestJournalEventSnapshot{Sequence: uint64(len(allEvents.Redux.QuestJournal) + 1), Kind: QuestJournalEventCodexAnswerLearned, Subject: virtue, OccurredOn: date, Source: QuestJournalSourceRevelation, SourceName: "Codex"})
	}
	for _, lord := range []string{QuestJournalSubjectFaulinei, QuestJournalSubjectAstaroth, QuestJournalSubjectNosfentor} {
		allEvents.Redux.QuestJournal = append(allEvents.Redux.QuestJournal, QuestJournalEventSnapshot{Sequence: uint64(len(allEvents.Redux.QuestJournal) + 1), Kind: QuestJournalEventShadowlordDefeated, Subject: lord, OccurredOn: date, Source: QuestJournalSourceGameplay})
	}
	allEvents.Progression.Shadowlords.Alive = [3]bool{}
	require.Len(t, allEvents.Redux.QuestJournal, 42)
	data, err := EncodeCanonical(validEnvelope(allEvents))
	require.NoError(t, err)
	require.NoError(t, compiled.Validate(decodeJSONSchemaInstance(t, data)))
	decoded, err := Decode(data)
	require.NoError(t, err)
	assert.Equal(t, allEvents, decoded.Snapshot)
	for name, mutate := range map[string]func(*SaveSnapshot){
		"duplicate pair":      func(s *SaveSnapshot) { s.Redux.QuestJournal[1].Subject = s.Redux.QuestJournal[0].Subject },
		"out of sequence":     func(s *SaveSnapshot) { s.Redux.QuestJournal[0].Sequence = 2 },
		"too many":            func(s *SaveSnapshot) { s.Redux.QuestJournal = append(s.Redux.QuestJournal, s.Redux.QuestJournal[0]) },
		"spell event":         func(s *SaveSnapshot) { s.Redux.QuestJournal[0].Subject = "spell.an-ylem" },
		"unknown journal key": func(s *SaveSnapshot) { s.Redux.QuestJournal[0].Subject = "journal.domain.truth" },
		"unknown dungeon":     func(s *SaveSnapshot) { s.Redux.QuestJournal[21].Subject = "dungeon.abyss.word_of_power" },
		"bare dungeon key":    func(s *SaveSnapshot) { s.Redux.QuestJournal[21].Subject = "doom.word_of_power" },
		"bare knowledge key":  func(s *SaveSnapshot) { s.Redux.QuestJournal[0].Subject = "flame.truth" },
		"codex spelling":      func(s *SaveSnapshot) { s.Redux.QuestJournal[29].Subject = "valor" },
		"undated discovery":   func(s *SaveSnapshot) { s.Redux.QuestJournal[0].OccurredOn = nil },
		"unnamed discovery":   func(s *SaveSnapshot) { s.Redux.QuestJournal[0].SourceName = "  " },
		"gameplay discovery":  func(s *SaveSnapshot) { s.Redux.QuestJournal[0].Source = QuestJournalSourceGameplay },
		"migration discovery": func(s *SaveSnapshot) { s.Redux.QuestJournal[0].Source = QuestJournalSourceMigration },
		"named defeat":        func(s *SaveSnapshot) { s.Redux.QuestJournal[39].SourceName = "Speaker" },
		"dialogue defeat":     func(s *SaveSnapshot) { s.Redux.QuestJournal[39].Source = QuestJournalSourceDialogue },
	} {
		t.Run(name, func(t *testing.T) {
			snapshot := allEvents
			snapshot.Redux.QuestJournal = append([]QuestJournalEventSnapshot{}, allEvents.Redux.QuestJournal...)
			mutate(&snapshot)
			_, err := EncodeCanonical(validEnvelope(snapshot))
			assert.ErrorIs(t, err, ErrInvalidSnapshot)
			data, err := json.Marshal(validEnvelope(snapshot))
			require.NoError(t, err)
			_, err = Decode(data)
			assert.Error(t, err)
			assert.Error(t, compiled.Validate(decodeJSONSchemaInstance(t, data)))
		})
	}
}
