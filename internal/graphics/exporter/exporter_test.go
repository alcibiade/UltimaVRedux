package exporter

import (
	"image"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/graphics/legacy"
	"github.com/stretchr/testify/assert"
)

func TestMakeContactSheetWrapsEntries(t *testing.T) {
	entries := []legacy.Entry{
		{Image: image.NewPaletted(image.Rect(0, 0, 800, 10), legacy.EGAPalette)},
		{Image: image.NewPaletted(image.Rect(0, 0, 400, 20), legacy.EGAPalette)},
	}

	sheet := makeContactSheet(entries)

	assert.Equal(t, 816, sheet.Bounds().Dx())
	assert.Equal(t, 54, sheet.Bounds().Dy())
}

func TestSanitizeName(t *testing.T) {
	assert.Equal(t, "lord-british", sanitizeName("Lord British"))
}
