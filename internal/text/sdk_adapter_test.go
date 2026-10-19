package text

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutputRejectsNonprogressingColumnsAndKeepsEmptyPanels(t *testing.T) {
	require.Panics(t, func() { NewOutput(nil, 0, 1, 0) })
	require.Panics(t, func() { NewOutput(nil, 0, 1, -1) })
	output := NewOutput(nil, 0, 0, 20)
	output.AddText("no visible rows")
	require.Empty(t, output.lines)
}

func TestWrappedPresentationCopyDoesNotMutateLogicalHistory(t *testing.T) {
	output := NewOutput(nil, 0, 4, 20)
	output.RetainLogicalHistory(10, 100)
	output.AddText("source")
	copy := output.WrappedCopy(10)
	copy.AddText("copy only")
	require.Len(t, output.history.entries, 1)
	require.Equal(t, "source", output.history.entries[0].raw)
	require.Nil(t, copy.history)
}
