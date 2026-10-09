package references

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKarmaReportsDecodeAndSelectAllBands(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "KARMA.DAT")
	require.NoError(t, os.WriteFile(path, syntheticKarmaData(), 0o600))
	reports, err := NewKarmaReports(path)
	require.NoError(t, err)

	tests := []struct {
		karma         int
		deathReport   string
		campingReport string
	}{
		{karma: 0, deathReport: "report-1", campingReport: "report-1"},
		{karma: 19, deathReport: "report-1", campingReport: "report-1"},
		{karma: 20, deathReport: "report-2", campingReport: "report-2"},
		{karma: 39, deathReport: "report-2", campingReport: "report-2"},
		{karma: 40, deathReport: "report-3", campingReport: "report-3"},
		{karma: 59, deathReport: "report-3", campingReport: "report-3"},
		{karma: 60, deathReport: "report-4", campingReport: "report-4"},
		{karma: 79, deathReport: "report-4", campingReport: "report-4"},
		{karma: 80, deathReport: "report-5", campingReport: "report-6"},
		{karma: 99, deathReport: "report-5", campingReport: "report-6"},
	}
	for _, test := range tests {
		death, deathErr := reports.Select(test.karma, KarmaReportDeath)
		require.NoError(t, deathErr)
		assert.Equal(t, test.deathReport, strings.TrimSpace(death))

		camping, campingErr := reports.Select(test.karma, KarmaReportCamping)
		require.NoError(t, campingErr)
		assert.Equal(t, test.campingReport, strings.TrimSpace(camping))
	}
}

func TestKarmaReportsRejectInvalidData(t *testing.T) {
	t.Parallel()

	t.Run("missing file", func(t *testing.T) {
		_, err := NewKarmaReports(filepath.Join(t.TempDir(), "KARMA.DAT"))
		require.Error(t, err)
		assert.ErrorContains(t, err, "reading required KARMA.DAT")
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("truncated", func(t *testing.T) {
		_, err := loadSyntheticKarmaReports(t, syntheticKarmaData()[:671])
		assert.ErrorContains(t, err, "parsing KARMA.DAT")
		assert.ErrorIs(t, err, errCorruptKarmaReports)
		assert.ErrorContains(t, err, "offset 671 is outside")
	})

	t.Run("missing terminator", func(t *testing.T) {
		data := syntheticKarmaData()
		data[131] = 'x'
		_, err := loadSyntheticKarmaReports(t, data)
		assert.ErrorIs(t, err, errCorruptKarmaReports)
		assert.ErrorContains(t, err, "no null terminator")
	})

	t.Run("empty record", func(t *testing.T) {
		data := syntheticKarmaData()
		data[269] = 0
		_, err := loadSyntheticKarmaReports(t, data)
		assert.ErrorIs(t, err, errCorruptKarmaReports)
		assert.ErrorContains(t, err, "report 3 is empty")
	})

	t.Run("shifted boundary", func(t *testing.T) {
		data := syntheticKarmaData()
		data[130] = 0
		_, err := loadSyntheticKarmaReports(t, data)
		assert.ErrorIs(t, err, errCorruptKarmaReports)
		assert.ErrorContains(t, err, "expected record boundary")
	})
}

func TestKarmaReportsRejectInvalidSelection(t *testing.T) {
	reports, err := parseKarmaReports(syntheticKarmaData())
	require.NoError(t, err)

	_, err = reports.Select(-1, KarmaReportDeath)
	assert.ErrorContains(t, err, "outside 0-99")
	_, err = reports.Select(100, KarmaReportCamping)
	assert.ErrorContains(t, err, "outside 0-99")
	_, err = reports.Select(50, KarmaReportContext(99))
	assert.ErrorContains(t, err, "unknown karma report context")
}

func syntheticKarmaData() []byte {
	offsets := [...]int{0, 132, 269, 410, 546, 671}
	const finalRecordSize = 32
	data := make([]byte, offsets[len(offsets)-1]+finalRecordSize)
	for index, offset := range offsets {
		limit := len(data)
		if index+1 < len(offsets) {
			limit = offsets[index+1]
		}
		label := []byte("report-" + string(rune('1'+index)))
		copy(data[offset:], label)
		for position := offset + len(label); position < limit-1; position++ {
			data[position] = ' '
		}
		data[limit-1] = 0
	}

	return data
}

func loadSyntheticKarmaReports(t *testing.T, data []byte) (*KarmaReports, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "KARMA.DAT")
	require.NoError(t, os.WriteFile(path, data, 0o600))

	return NewKarmaReports(path)
}
