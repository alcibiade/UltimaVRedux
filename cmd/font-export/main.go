// Package main exports Ultima V bitmap fonts (RUNES.CH and IBM.CH) to PNG sprite sheets.
// This is a one-time development tool - the resulting PNGs are embedded at compile time.
//
// Font file format:
//   - 1024 bytes total (128 characters × 8 bytes per character)
//   - Each character is 8×8 pixels, stored as 8 rows of 1 byte each
//   - Each bit represents one pixel (1 = foreground, 0 = background)
//   - Bit 7 (MSB) is the leftmost pixel, bit 0 (LSB) is the rightmost
//
// Output format:
//   - 128×64 pixel PNG (16 characters wide × 8 rows high)
//   - Each character occupies an 8×8 cell
//   - White foreground (255,255,255,255), transparent background (0,0,0,0)
package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

const (
	charsPerRow  = 16   // Characters per row in output sprite sheet
	charRows     = 8    // Rows of characters in output sprite sheet
	charWidth    = 8    // Width of each character in pixels
	charHeight   = 8    // Height of each character in pixels
	bytesPerChar = 8    // Bytes per character in source file
	totalChars   = 128  // Total characters in font
	fontFileSize = 1024 // Expected font file size (128 * 8)
	sheetWidth   = charsPerRow * charWidth
	sheetHeight  = charRows * charHeight
	msbBitPos    = 7 // Most significant bit position (leftmost pixel in row)
)

var errFontExportUsage = errors.New("usage: font-export -runes <RUNES.CH path> -ibm <IBM.CH path> [-output <dir>]")

var (
	foreground = color.RGBA{R: 255, G: 255, B: 255, A: 255} // White
	background = color.RGBA{R: 0, G: 0, B: 0, A: 0}         // Transparent
)

func main() {
	if err := run(); err != nil {
		if _, writeErr := fmt.Fprintln(os.Stderr, err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}

func run() error {
	var (
		runesPath string
		ibmPath   string
		outputDir string
	)

	flag.StringVar(&runesPath, "runes", "", "Path to RUNES.CH file")
	flag.StringVar(&ibmPath, "ibm", "", "Path to IBM.CH file")
	flag.StringVar(&outputDir, "output", "internal/text/assets", "Output directory for PNG files")
	flag.Parse()

	if runesPath == "" || ibmPath == "" {
		return errFontExportUsage
	}

	// Export runic font
	runicOutput := filepath.Join(outputDir, "runic_font.png")
	if err := exportFont(runesPath, runicOutput); err != nil {
		return fmt.Errorf("exporting runic font: %w", err)
	}
	if _, err := fmt.Fprintf(os.Stdout, "Exported runic font to %s\n", runicOutput); err != nil {
		return fmt.Errorf("writing runic font summary: %w", err)
	}

	// Export IBM font
	ibmOutput := filepath.Join(outputDir, "ibm_font.png")
	if err := exportFont(ibmPath, ibmOutput); err != nil {
		return fmt.Errorf("exporting IBM font: %w", err)
	}
	if _, err := fmt.Fprintf(os.Stdout, "Exported IBM font to %s\n", ibmOutput); err != nil {
		return fmt.Errorf("writing IBM font summary: %w", err)
	}

	if _, err := fmt.Fprintln(os.Stdout, "\nFont export complete!"); err != nil {
		return fmt.Errorf("writing font export completion: %w", err)
	}

	return nil
}

// exportFont reads a .CH font file and exports it as a PNG sprite sheet.
func exportFont(inputPath, outputPath string) error {
	// Read the font file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading font file: %w", err)
	}

	if len(data) != fontFileSize {
		return fmt.Errorf("invalid font file size: expected %d bytes, got %d", fontFileSize, len(data))
	}

	// Create the output image
	img := image.NewRGBA(image.Rect(0, 0, sheetWidth, sheetHeight))

	// Process each character
	for charIndex := 0; charIndex < totalChars; charIndex++ {
		// Calculate position in sprite sheet
		sheetX := (charIndex % charsPerRow) * charWidth
		sheetY := (charIndex / charsPerRow) * charHeight

		// Get the character data (8 bytes)
		charOffset := charIndex * bytesPerChar

		// Process each row of the character
		for row := 0; row < charHeight; row++ {
			rowByte := data[charOffset+row]

			// Process each pixel in the row (bit 7 = leftmost, bit 0 = rightmost)
			for col := 0; col < charWidth; col++ {
				// Check if bit is set (MSB first)
				bitMask := byte(1 << (msbBitPos - col))
				pixelSet := (rowByte & bitMask) != 0

				pixelX := sheetX + col
				pixelY := sheetY + row

				if pixelSet {
					img.Set(pixelX, pixelY, foreground)
				} else {
					img.Set(pixelX, pixelY, background)
				}
			}
		}
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Write the PNG file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil {
			// Log but don't override the main error
			_, _ = fmt.Fprintf(os.Stderr, "Warning: error closing file: %v\n", closeErr) //nolint:forbidigo // CLI tool output
		}
	}()

	if err := png.Encode(outFile, img); err != nil {
		return fmt.Errorf("encoding PNG: %w", err)
	}

	return nil
}
