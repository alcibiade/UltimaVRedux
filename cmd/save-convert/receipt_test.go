package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/stretchr/testify/require"
)

func TestReceiptQuotesPathAndReportsPublishedIdentity(t *testing.T) {
	path := "a directory/a \"quoted\" name\n.json"
	envelope := save_snapshot.Envelope{
		Format: save_snapshot.EnvelopeFormat, EnvelopeVersion: save_snapshot.CurrentEnvelopeVersion,
		Metadata: save_snapshot.Metadata{PlaythroughID: testPlaythroughID},
		Snapshot: save_snapshot.SaveSnapshot{Version: save_snapshot.CurrentVersion},
	}
	var output bytes.Buffer
	require.NoError(t, writeReceipt(&output, path, envelope, 3))
	require.Equal(t, fmt.Sprintf(
		"Converted save: %q\nPlaythrough ID: %s\nFormat: %s (envelope %d, snapshot %d)\nWarnings: 3\n",
		path, testPlaythroughID, save_snapshot.EnvelopeFormat, save_snapshot.CurrentEnvelopeVersion, save_snapshot.CurrentVersion,
	), output.String())
	err := writeReceipt(failingReceiptWriter{}, path, envelope, 3)
	require.ErrorIs(t, err, io.ErrClosedPipe)
	require.ErrorContains(t, err, "output was published")
}

func TestImportNamesMatchPlayerSlotNames(t *testing.T) {
	require.Equal(t, "Imported original save", importName(""))
	for index, slot := range player_save.AllSlotIDs() {
		require.Equal(t, fmt.Sprintf("Slot %d", index+1), importName(string(slot)))
	}
}

type failingReceiptWriter struct{}

func (failingReceiptWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
