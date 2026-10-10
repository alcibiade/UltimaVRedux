package game_state

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestActionLookSmallMapDirectionalSignPieceDescribesTerminalTile(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(12, 7).
		WithSystemCallbacks().
		Build()

	require.True(t, state.ActionLookSmallMap(references.Down))
	assert.False(t, callbacks.SignDialogShown)
	assert.True(t, callbacks.MessagesContain("Thou dost see the sign of a provisioner"))
}

func TestActionLookSmallMapBuildsYewWantedPosterFromParty(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Yew).
		WithPlayerAt(wantedPosterX, wantedPosterY+1).
		WithSystemCallbacks().
		Build()

	names := []string{"Avatar", "Shamino", "Iolo"}
	require.GreaterOrEqual(t, state.PartyState.GetActivePartyCount(), len(names))
	for index, name := range names {
		clear(state.PartyState.GetCharacter(index).Name[:])
		copy(state.PartyState.GetCharacter(index).Name[:], name)
	}

	require.True(t, state.ActionLookSmallMap(references.Up))
	require.True(t, callbacks.SignDialogShown)
	require.NotNil(t, callbacks.SignDialogSign)

	sign := callbacks.SignDialogSign
	assert.Equal(t, references.Yew, sign.Location)
	assert.Equal(t, wantedPosterX, sign.X)
	assert.Equal(t, wantedPosterY, sign.Y)
	assertWantedPosterLines(t, sign.RawText, names)

	staticSign := state.GameReferences.SignReferences.GetSignAt(
		references.Yew,
		int(references.MainFloor),
		wantedPosterX,
		wantedPosterY,
	)
	require.NotNil(t, staticSign)
	assert.False(t, bytes.Equal(staticSign.RawText, sign.RawText), "generated poster must bypass static SIGNS.DAT text")
}

func TestBuildWantedPosterTextUsesFirstThreeNamesAndMixedFonts(t *testing.T) {
	for _, names := range [][]string{
		{"One"},
		{"One", "Two"},
		{"One", "Two", "Three"},
		{"One", "Two", "Three", "Ignored", "AlsoIgnored", "StillIgnored"},
	} {
		rawText := buildWantedPosterText(names)
		visibleNames := names[:min(len(names), wantedPosterNameRows)]
		assertWantedPosterLines(t, rawText, visibleNames)
		assert.NotContains(t, decodeSignBytes(rawText), "Ignored")
	}

	rawText := buildWantedPosterText([]string{"One", "Two", "Three"})

	lines := bytes.Split(rawText, []byte{'\n'})
	for _, lineIndex := range []int{0, 8} {
		for _, value := range lines[lineIndex] {
			assert.True(t, references.IsRunicChar(value))
		}
	}
	for lineIndex := 1; lineIndex < 8; lineIndex++ {
		assert.True(t, references.IsRunicChar(lines[lineIndex][0]))
		assert.True(t, references.IsRunicChar(lines[lineIndex][len(lines[lineIndex])-1]))
		for _, value := range lines[lineIndex][1 : len(lines[lineIndex])-1] {
			assert.True(t, references.IsNormalChar(value))
		}
	}
}

func assertWantedPosterLines(t *testing.T, rawText []byte, names []string) {
	t.Helper()
	decodedLines := bytes.Split([]byte(decodeSignBytes(rawText)), []byte{'\n'})
	require.Len(t, decodedLines, 9)
	assert.Equal(t, "abbbbbbbbbbbbbc", string(decodedLines[0]))
	assert.Equal(t, "g   Wanted:   g", string(decodedLines[1]))
	assert.Equal(t, "g             g", string(decodedLines[2]))
	for row := 0; row < wantedPosterNameRows; row++ {
		name := ""
		if row < len(names) {
			name = names[row]
		}
		padding := wantedPosterInteriorWidth - len(name)
		expected := "g" + string(bytes.Repeat([]byte{' '}, padding/2)) + name +
			string(bytes.Repeat([]byte{' '}, padding-padding/2)) + "g"
		assert.Equal(t, expected, string(decodedLines[row+3]))
	}
	assert.Equal(t, "g             g", string(decodedLines[6]))
	assert.Equal(t, "gDead or Aliveg", string(decodedLines[7]))
	assert.Equal(t, "deeeeeeeeeeeeef", string(decodedLines[8]))
}

func decodeSignBytes(rawText []byte) string {
	decoded := make([]byte, len(rawText))
	for index, value := range rawText {
		decoded[index] = value & references.SignCharIndexMask
	}

	return string(decoded)
}
