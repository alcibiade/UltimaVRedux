package extractor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/require"
)

func TestOriginalArbuthnotStrandedAcknowledgment(t *testing.T) {
	dir := originaldata.Directory(t)
	tlk, err := os.ReadFile(filepath.Join(dir, "DWELLING.TLK"))
	require.NoError(t, err)
	ovl, err := os.ReadFile(filepath.Join(dir, "DATA.OVL"))
	require.NoError(t, err)
	dictionary, err := parseWordTable(ovl[compressedWordsOffset : compressedWordsOffset+compressedWordsLength])
	require.NoError(t, err)
	instructions, err := parseExecutable(rawField{data: tlk[0x1f0d:0x1f27]}, dictionary, false)
	require.NoError(t, err)
	var speech strings.Builder
	var asksName bool
	for _, instruction := range instructions {
		speech.WriteString(instruction.text)
		if instruction.kind == instructionWord {
			speech.WriteByte(' ')
		}
		if instruction.kind == instructionAction && instruction.action == actionAskName {
			asksName = true
		}
	}
	require.Equal(t, "'Tisindeedanhonor!", strings.ReplaceAll(speech.String(), " ", ""))
	require.True(t, asksName)
	t.Logf("Stranded original acknowledgment: %q", speech.String())
}
