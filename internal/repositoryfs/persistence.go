package repositoryfs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

// PublishTemporaryFile writes, flushes, and closes a temporary file before
// passing it to publish. The temporary path is removed on every return path.
// Callers remain responsible for syncing directory metadata after publication.
func PublishTemporaryFile(root, pattern string, data []byte, mode os.FileMode, publish func(string) error) (err error) {
	temporary, err := os.CreateTemp(root, pattern)
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		if cleanupErr := os.Remove(temporaryPath); cleanupErr != nil && !errors.Is(cleanupErr, os.ErrNotExist) {
			if err != nil {
				err = errors.Join(err, fmt.Errorf("remove temporary file: %w", cleanupErr))
			} else {
				logging.Warnf(logging.CategorySystem, "Published file but could not remove temporary %s: %v", temporaryPath, cleanupErr)
			}
		}
	}()

	if err = temporary.Chmod(mode); err == nil {
		_, err = temporary.Write(data)
	}
	if err == nil {
		err = temporary.Sync()
	}
	closeErr := temporary.Close()
	if closeErr != nil {
		closeErr = fmt.Errorf("close temporary file: %w", closeErr)
	}
	if err != nil {
		return errors.Join(fmt.Errorf("write temporary file: %w", err), closeErr)
	}
	if closeErr != nil {
		return closeErr
	}
	if err = publish(temporaryPath); err != nil {
		return err
	}

	return nil
}

// ContentRevision returns the lowercase SHA-256 revision of data.
func ContentRevision(data []byte) string {
	digest := sha256.Sum256(data)

	return hex.EncodeToString(digest[:])
}

// DecodeStrictJSON decodes exactly one JSON document and rejects unknown fields.
func DecodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err == nil {
		return errors.New("trailing JSON value")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("trailing JSON: %w", err)
	}

	return nil
}
