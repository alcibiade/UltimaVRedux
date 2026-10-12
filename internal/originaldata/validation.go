package originaldata

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/files"
	"github.com/bradhannah/UltimaVRedux/internal/originaldata/dungeonarchive"
	"github.com/bradhannah/UltimaVRedux/internal/originaldata/graphicsarchive"
	"github.com/bradhannah/UltimaVRedux/internal/originaldata/introarchive"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	originalDataOvlSize     = 48_464
	originalInitGamSize     = 4_192
	originalInitObjectsSize = 256
	maximumReferenceBytes   = 4 << 20
)

var (
	errDataOvlSize      = errors.New("DATA.OVL has an unexpected size")
	errRequiredFileSize = errors.New("required original-data file has an unexpected size")
)

// ValidateInstallation verifies an exact data directory without creating GPU
// resources or active gameplay state.
func ValidateInstallation(ctx context.Context, directory string) error {
	if strings.TrimSpace(directory) == "" {
		return config.ErrDataFilePathUnset
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("cancelling installation validation: %w", err)
	}
	dataPath := filepath.Join(directory, files.DATA_OVL)
	info, err := os.Stat(dataPath)
	if err != nil {
		return fmt.Errorf("required %s is unavailable: %w", files.DATA_OVL, err)
	}
	if info.Size() != originalDataOvlSize {
		return fmt.Errorf("%w: %s has size %d, expected %d", errDataOvlSize, files.DATA_OVL, info.Size(), originalDataOvlSize)
	}
	if err := validateRequiredCatalogue(ctx, directory); err != nil {
		return err
	}
	if err := validateRequiredGraphics(ctx, directory); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("cancelling installation validation: %w", err)
	}
	configuration, err := config.NewUltimaVConfigurationWithPath(directory)
	if err != nil {
		return fmt.Errorf("loading original data: %w", err)
	}
	if _, err := references.NewGameReferences(configuration); err != nil {
		return fmt.Errorf("validating original game references: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("cancelling installation validation: %w", err)
	}

	return nil
}

//nolint:mnd // Canonical original file lengths are part of the installation format.
func validateRequiredCatalogue(ctx context.Context, directory string) error {
	exactFiles := []struct {
		name string
		size int
	}{
		{name: files.INIT_GAM, size: originalInitGamSize},
		{name: files.INIT_OOL, size: originalInitObjectsSize},
		{name: files.BRIT_DAT, size: 52_480},
		{name: files.UNDER_DAT, size: 65_536},
		{name: files.CASTLE_DAT, size: 16_384},
		{name: files.TOWNE_DAT, size: 16_384},
		{name: files.DWELLING_DAT, size: 16_384},
		{name: files.KEEP_DAT, size: 16_384},
		{name: files.DUNGEON_DAT, size: 4_096},
		{name: files.CASTLE_NPC, size: 4_608},
		{name: files.TOWNE_NPC, size: 4_608},
		{name: files.DWELLING_NPC, size: 4_608},
		{name: files.KEEP_NPC, size: 4_608},
		{name: files.BRIT_CBT, size: 5_632},
		{name: files.DUNGEON_CBT, size: 39_424},
		{name: files.MISCMAPS_DAT, size: 1_871},
		{name: "QUESTION.DAT", size: 7_746},
		{name: "STORY.DAT", size: 11_679},
	}
	for _, required := range exactFiles {
		if err := validateExactFile(ctx, directory, required.name, required.size); err != nil {
			return err
		}
	}
	boundedFiles := []string{
		files.CASTLE_TLK, files.TOWNE_TLK, files.DWELLING_TLK, files.KEEP_TLK,
		files.LOOK2_DAT, files.SIGNS_DAT, files.SHOPPE_DAT, files.KarmaDAT,
		"END.DAT", "ENDMSG.DAT",
	}
	for _, name := range boundedFiles {
		if err := validateBoundedFile(ctx, directory, name, maximumReferenceBytes); err != nil {
			return err
		}
	}

	return nil
}

func validateRequiredGraphics(ctx context.Context, directory string) error {
	if _, err := introarchive.LoadAndValidateContext(ctx, directory); err != nil {
		return fmt.Errorf("validating original title data: %w", err)
	}
	if _, err := dungeonarchive.LoadAndValidate(ctx, directory); err != nil {
		return fmt.Errorf("validating original dungeon data: %w", err)
	}
	extraGraphics := []string{
		"CREATE.16", "PROPORT.PCS", "TEXT.16",
		"STORY1.16", "STORY2.16", "STORY3.16", "STORY4.16", "STORY5.16", "STORY6.16",
		"END1.16", "END2.16", "ENDSC.16",
	}
	for _, name := range extraGraphics {
		if _, err := graphicsarchive.Load(ctx, directory, name); err != nil {
			return fmt.Errorf("validating required graphics %s: %w", name, err)
		}
	}

	return nil
}

func validateBoundedFile(ctx context.Context, directory, name string, maximumSize int64) (err error) {
	path := filepath.Join(directory, name)
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required %s is unavailable: %w", name, err)
	}
	if info.Size() <= 0 || info.Size() > maximumSize {
		return fmt.Errorf("%w: %s has unsupported size %d", errRequiredFileSize, name, info.Size())
	}
	//nolint:gosec // The player selects directory; name comes from the fixed required-file catalogue.
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening required %s: %w", name, err)
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	reader := newValidationReader(ctx, file, name)
	read, err := io.Copy(io.Discard, io.LimitReader(reader, maximumSize+1))
	if err != nil {
		return fmt.Errorf("reading required %s: %w", name, err)
	}
	if read != info.Size() || read > maximumSize {
		return fmt.Errorf("%w: %s changed while being read", errRequiredFileSize, name)
	}

	return nil
}

func validateExactFile(ctx context.Context, directory, name string, expectedSize int) (err error) {
	path := filepath.Join(directory, name)
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required %s is unavailable: %w", name, err)
	}
	if info.Size() != int64(expectedSize) {
		return fmt.Errorf("%w: %s has size %d, expected %d", errRequiredFileSize, name, info.Size(), expectedSize)
	}
	//nolint:gosec // The player selects directory; name is a fixed required filename.
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening required %s: %w", name, err)
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	reader := newValidationReader(ctx, file, name)
	read, err := io.Copy(io.Discard, io.LimitReader(reader, int64(expectedSize)+1))
	if err != nil {
		return fmt.Errorf("reading required %s: %w", name, err)
	}
	if read != int64(expectedSize) {
		return fmt.Errorf(
			"%w: %s changed while being read: got %d bytes, expected %d",
			errRequiredFileSize, name, read, expectedSize,
		)
	}

	return nil
}

func newValidationReader(ctx context.Context, file io.Reader, name string) validationReaderFunc {
	return func(buffer []byte) (int, error) {
		if err := ctx.Err(); err != nil {
			return 0, fmt.Errorf("reading required %s: %w", name, err)
		}
		read, readErr := file.Read(buffer)
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return read, io.EOF
			}

			return read, fmt.Errorf("reading required %s: %w", name, readErr)
		}

		return read, nil
	}
}

type validationReaderFunc func([]byte) (int, error)

func (read validationReaderFunc) Read(buffer []byte) (int, error) {
	return read(buffer)
}
