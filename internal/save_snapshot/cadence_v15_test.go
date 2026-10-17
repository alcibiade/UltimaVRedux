package save_snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
)

func removeCadenceV15Fields(snapshot map[string]any) {
	removeJournalV16Fields(snapshot)
	effects := snapshot["effects"].(map[string]any)
	delete(effects, "townMountedActorSkip")
	delete(effects, "townQuicknessActorSkip")
	delete(effects, "outdoorMountedActorSkip")
}

func versionFourteenEnvelopeData(t *testing.T, snapshot SaveSnapshot, mutate func(map[string]any)) []byte {
	t.Helper()
	document := currentEnvelopeDocument(t, snapshot)
	snapshotDocument := document["snapshot"].(map[string]any)
	snapshotDocument["version"] = 14
	removeCadenceV15Fields(snapshotDocument)
	if mutate != nil {
		mutate(snapshotDocument)
	}
	data, err := json.Marshal(document)
	require.NoError(t, err)

	return data
}

func TestCadenceV15HistoricalMigrationContextAndDefaults(t *testing.T) {
	for _, version := range []int{12, 13, 14} {
		for _, location := range []references.Location{references.BritanniaUnderworld, references.Britain, references.Deceit} {
			for _, active := range []bool{false, true} {
				for _, oldSkipNext := range []bool{false, true} {
					t.Run(fmt.Sprintf("v%d/%s/active=%v/skip=%v", version, location, active, oldSkipNext), func(t *testing.T) {
						snapshot := validSnapshot()
						snapshot.World.Position = PositionSnapshot{Location: int(location), X: 1, Y: 1}
						if location == references.Deceit {
							snapshot.Dungeon = &DungeonSnapshot{Location: int(location), Cells: make([]int, DungeonCellCount), QuicknessSkipNext: true}
						}
						if active {
							snapshot.Effects.DurationType, snapshot.Effects.DurationTurns = 'Q', 10
						}
						snapshot.Effects.QuicknessWorldActorSkip = oldSkipNext
						var data []byte
						switch version {
						case 12:
							data = versionTwelveEnvelopeData(t, snapshot, nil)
						case 13:
							data = versionThirteenEnvelopeData(t, snapshot, nil)
						case 14:
							data = versionFourteenEnvelopeData(t, snapshot, nil)
						}
						decoded, err := Decode(data)
						require.NoError(t, err)
						assert.Equal(t, CurrentVersion, decoded.Snapshot.Version)
						want := snapshot.Effects
						want.QuicknessWorldActorSkip = active && location == references.BritanniaUnderworld && !oldSkipNext
						want.TownQuicknessActorSkip = active && location == references.Britain && !oldSkipNext
						assert.Equal(t, want, decoded.Snapshot.Effects, "unknown riding and inactive-context phases default false")
						assert.Equal(t, snapshot.Random, decoded.Snapshot.Random, "promotion consumes no RNG")
						assert.Equal(t, snapshot.Dungeon, decoded.Snapshot.Dungeon, "dungeon phase is independent")
						canonical, err := EncodeCanonical(decoded)
						require.NoError(t, err)
						reloaded, err := Decode(canonical)
						require.NoError(t, err)
						decoded.legacyExcerpts = false // Decode-only provenance is consumed by current encoding.
						assert.Equal(t, decoded, reloaded, "v15 reload must not apply the inversion again")
					})
				}
			}
		}
	}
}

func TestCadenceV15RequiredBooleanContractMatchesStandaloneSchema(t *testing.T) {
	compiled, _ := compileNativeSaveSchemaFile(t, "native-save-e2-s15.schema.json")
	base := currentEnvelopeDocument(t, validSnapshot())
	base["snapshot"].(map[string]any)["version"] = 15
	removeJournalV16Fields(base["snapshot"].(map[string]any))
	for _, member := range []string{"quicknessWorldActorSkip", "townMountedActorSkip", "townQuicknessActorSkip", "outdoorMountedActorSkip"} {
		for _, test := range []struct {
			name    string
			missing bool
			value   any
		}{
			{name: "missing", missing: true}, {name: "null", value: nil},
			{name: "number", value: 0}, {name: "string", value: "false"},
			{name: "false", value: false}, {name: "true", value: true},
		} {
			t.Run(member+"/"+test.name, func(t *testing.T) {
				instance := cloneJSONDocument(t, base)
				effects := instance["snapshot"].(map[string]any)["effects"].(map[string]any)
				if test.missing {
					delete(effects, member)
				} else {
					effects[member] = test.value
				}
				data, err := json.Marshal(instance)
				require.NoError(t, err)
				_, decodeErr := Decode(data)
				schemaErr := compiled.Validate(decodeJSONSchemaInstance(t, data))
				if _, boolean := test.value.(bool); boolean {
					assert.NoError(t, decodeErr)
					assert.NoError(t, schemaErr)
				} else {
					assert.ErrorIs(t, decodeErr, ErrInvalidEnvelope)
					assert.Error(t, schemaErr)
				}
			})
		}
	}
}

func TestCadenceV15SchemaChangesOnlyVersionAndCadence(t *testing.T) {
	_, previous := compileNativeSaveSchema(t, 14)
	_, current := compileNativeSaveSchema(t, 15)
	before := previous.(map[string]any)
	after := current.(map[string]any)
	after["$id"], after["description"] = before["$id"], before["description"]
	defs := after["$defs"].(map[string]any)
	oldDefs := before["$defs"].(map[string]any)
	defs["saveSnapshot"].(map[string]any)["properties"].(map[string]any)["version"] = oldDefs["saveSnapshot"].(map[string]any)["properties"].(map[string]any)["version"]
	effects := defs["bundled067"].(map[string]any)
	properties := effects["properties"].(map[string]any)
	for _, member := range []string{"townMountedActorSkip", "townQuicknessActorSkip", "outdoorMountedActorSkip"} {
		delete(properties, member)
	}
	var required []any
	for _, member := range effects["required"].([]any) {
		if _, exists := properties[member.(string)]; exists {
			required = append(required, member)
		}
	}
	effects["required"] = required
	assert.Equal(t, before, after, "non-cadence v14 schema constraints must remain unchanged")
}

func TestCadenceV15FrozenV14RejectsNewFieldsAndPreservesExistingState(t *testing.T) {
	snapshot := validSnapshot()
	snapshot.Continuation.OpeningCampaignEncounterReduction = true
	snapshot.Continuation.CombatActionTimeCounter = 9
	snapshot.MapUnits[0].Kind = MapUnitKindChest
	snapshot.MapUnits[0].EnemyVariant = ""
	snapshot.MapUnits[0].EnemyCurrentHP = 0
	number := 158
	snapshot.MapUnits[0].SurfaceChestNumber = &number
	data := versionFourteenEnvelopeData(t, snapshot, nil)
	decoded, err := Decode(data)
	require.NoError(t, err)
	assert.Equal(t, snapshot, decoded.Snapshot)
	_, err = DecodeUnsupportedVersion(data)
	assert.ErrorIs(t, err, ErrInvalidEnvelope)
	for _, member := range []string{"townMountedActorSkip", "townQuicknessActorSkip", "outdoorMountedActorSkip"} {
		data = versionFourteenEnvelopeData(t, snapshot, func(document map[string]any) {
			document["effects"].(map[string]any)[member] = false
		})
		_, err = Decode(data)
		assert.ErrorIs(t, err, ErrInvalidEnvelope)
	}
}

func TestCadenceV15RepositoryPromotionRevisionIsCanonicalAndNonMutating(t *testing.T) {
	repository, roots := newTestRepository(t)
	snapshot := validSnapshot()
	snapshot.Effects.DurationType, snapshot.Effects.DurationTurns = 'Q', 10
	data := versionFourteenEnvelopeData(t, snapshot, nil)
	require.NoError(t, os.MkdirAll(roots[0], 0o755))
	path := filepath.Join(roots[0], "cadence-v14.json")
	require.NoError(t, os.WriteFile(path, data, 0o644))
	loaded, err := repository.Load("project:cadence-v14")
	require.NoError(t, err)
	require.True(t, loaded.Envelope.Snapshot.Effects.QuicknessWorldActorSkip)
	canonical, err := EncodeCanonical(loaded.Envelope)
	require.NoError(t, err)
	assert.Equal(t, repositoryfs.ContentRevision(canonical), loaded.Revision)
	assert.NotEqual(t, repositoryfs.ContentRevision(data), loaded.Revision)
	retained, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, retained)
	require.NoError(t, os.WriteFile(path, append([]byte("\n  "), data...), 0o644))
	loadedAgain, err := repository.Load("project:cadence-v14")
	require.NoError(t, err)
	assert.Equal(t, loaded.Revision, loadedAgain.Revision)
}
