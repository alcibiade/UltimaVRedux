package catalog

import (
	"testing"
	"testing/fstest"

	"github.com/bradhannah/UltimaVRedux/internal/conversation/extractor"
	conversationpatch "github.com/bradhannah/UltimaVRedux/internal/conversation/patch"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPatchesValidatesPathAndMetadata(t *testing.T) {
	base := []byte("base\n")
	edited := []byte("edited\n")
	patchData, err := conversationpatch.Generate(
		"keep/010.yaml",
		semantic.SchemaVersion,
		extractor.ExtractorVersion,
		base,
		edited,
	)
	require.NoError(t, err)

	loaded, err := loadPatches(fstest.MapFS{
		"keep/010.patch": &fstest.MapFile{Data: patchData},
		".gitkeep":       &fstest.MapFile{},
	})
	require.NoError(t, err)
	assert.Equal(t, patchData, loaded["keep/010.yaml"])

	_, err = loadPatches(fstest.MapFS{"wrong.patch": &fstest.MapFile{Data: patchData}})
	require.ErrorContains(t, err, "does not match target")
}

func TestProgramDefinitionIsDetached(t *testing.T) {
	definition := &semantic.Conversation{
		Schema: semantic.SchemaVersion, Protocol: semantic.ProtocolUltima5,
		ID: "redux.detached", Origin: &semantic.ReduxOrigin{},
		Name:        semantic.Body{&semantic.SayAction{Content: semantic.RichText{&semantic.TextPart{Text: "Mara"}}}},
		Description: semantic.Body{&semantic.NarrateAction{Content: semantic.RichText{&semantic.TextPart{Text: "a keeper"}}}},
		Greeting:    semantic.Body{}, Job: semantic.Body{}, Farewell: semantic.Body{}, Topics: []semantic.Topic{},
		Questions: map[string]semantic.Question{},
	}
	data, err := semantic.Marshal(definition)
	require.NoError(t, err)
	program := &Program{id: definition.ID, data: data}

	first, err := program.Definition()
	require.NoError(t, err)
	first.ID = "changed"
	second, err := program.Definition()
	require.NoError(t, err)
	assert.Equal(t, "redux.detached", second.ID)
	canonical := program.CanonicalYAML()
	canonical[0] = 'X'
	assert.Equal(t, data, program.CanonicalYAML())
	assert.False(t, program.patched)
}

func TestNewIndexesOriginalDefinitions(t *testing.T) {
	definition := testConversationDefinition("original.towne.001", &semantic.OriginalOrigin{
		Source: semantic.OriginalSourceTowne,
		Dialog: 1,
	})

	catalog, err := New(definition)
	require.NoError(t, err)
	program, exists := catalog.Original(semantic.OriginalSourceTowne, 1)
	require.True(t, exists)
	assert.Equal(t, definition.ID, program.ID())
	assert.False(t, program.patched)
}

func TestOriginalKeysAreOrderedDetachedAndSourceSpecific(t *testing.T) {
	catalog, err := New(
		testConversationDefinition("original.towne.017", &semantic.OriginalOrigin{Source: semantic.OriginalSourceTowne, Dialog: 17}),
		testConversationDefinition("original.castle.001", &semantic.OriginalOrigin{Source: semantic.OriginalSourceCastle, Dialog: 1}),
		testConversationDefinition("redux.custom", &semantic.ReduxOrigin{}),
		testConversationDefinition("original.towne.002", &semantic.OriginalOrigin{Source: semantic.OriginalSourceTowne, Dialog: 2}),
	)
	require.NoError(t, err)
	want := []OriginalKey{{Source: semantic.OriginalSourceTowne, Dialog: 2}, {Source: semantic.OriginalSourceTowne, Dialog: 17}}
	keys := catalog.OriginalKeys(semantic.OriginalSourceTowne)
	assert.Equal(t, want, keys)
	keys[0].Dialog = 99
	assert.Equal(t, want, catalog.OriginalKeys(semantic.OriginalSourceTowne))
	assert.Empty(t, catalog.OriginalKeys(semantic.OriginalSourceKeep))
	assert.Empty(t, (*Catalog)(nil).OriginalKeys(semantic.OriginalSourceTowne))
}

func TestAddOriginalRetainsPatchedCanonicalDocument(t *testing.T) {
	definition := testConversationDefinition("original.keep.010", &semantic.OriginalOrigin{
		Source: semantic.OriginalSourceKeep,
		Dialog: 10,
	})
	base, err := semantic.Marshal(definition)
	require.NoError(t, err)
	definition.Name = semantic.Body{&semantic.SayAction{Content: semantic.RichText{&semantic.TextPart{Text: "Patched Mara"}}}}
	edited, err := semantic.Marshal(definition)
	require.NoError(t, err)
	patchData, err := conversationpatch.Generate(
		"keep/010.yaml", semantic.SchemaVersion, extractor.ExtractorVersion, base, edited,
	)
	require.NoError(t, err)
	catalog := &Catalog{originals: make(map[OriginalKey]*Program), byID: make(map[string]*Program)}

	err = catalog.addOriginal(extractor.ImportedConversation{
		Path: "keep/010.yaml", Data: base, Conversation: definition,
	}, patchData)
	require.NoError(t, err)
	program, exists := catalog.Original(semantic.OriginalSourceKeep, 10)
	require.True(t, exists)
	assert.True(t, program.patched)
	assert.Equal(t, edited, program.CanonicalYAML())
	loaded, err := program.Definition()
	require.NoError(t, err)
	assert.Equal(t, "Patched Mara", loaded.Name[0].(*semantic.SayAction).Content[0].(*semantic.TextPart).Text)
}

func testConversationDefinition(id string, origin semantic.Origin) *semantic.Conversation {
	return &semantic.Conversation{
		Schema: semantic.SchemaVersion, Protocol: semantic.ProtocolUltima5, ID: id, Origin: origin,
		Name:        semantic.Body{&semantic.SayAction{Content: semantic.RichText{&semantic.TextPart{Text: "Mara"}}}},
		Description: semantic.Body{&semantic.NarrateAction{Content: semantic.RichText{&semantic.TextPart{Text: "a keeper"}}}},
		Greeting:    semantic.Body{}, Job: semantic.Body{}, Farewell: semantic.Body{}, Topics: []semantic.Topic{},
		Questions: map[string]semantic.Question{},
	}
}
