package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

const outputFileMode = 0o600

var (
	errOccupied = errors.New("destination is occupied; use --overwrite to authorize replacement")
	errStale    = errors.New("destination changed after inspection; inspect it before retrying")
)

type destination struct {
	path       string
	revision   string
	slot       player_save.SlotID
	repository *player_save.Repository
	inputs     []os.FileInfo
}

func inspectDestination(path string, slot player_save.SlotID, overwrite bool) (destination, error) {
	target := destination{path: path, revision: "", slot: slot, repository: nil, inputs: nil}
	if err := checkRegularDestination(path); err != nil {
		return target, err
	}
	var err error
	if slot != "" {
		target.repository, err = player_save.NewRepository(filepath.Dir(path), nil)
		if err != nil {
			return target, fmt.Errorf("open player save repository: %w", err)
		}
		target.revision, err = slotRevision(target.repository, slot)
	} else {
		target.revision, err = fileRevision(path)
	}
	if err != nil {
		return target, err
	}
	if target.revision != "" && !overwrite {
		return target, fmt.Errorf("%w: %s", errOccupied, path)
	}

	return target, nil
}

func checkRegularDestination(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect destination: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("destination must be a regular file: %w", repositoryfs.ErrNotRegular)
	}

	return nil
}

func fileRevision(path string) (string, error) {
	if err := checkRegularDestination(path); err != nil {
		return "", err
	}
	data, err := repositoryfs.ReadRegularFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read destination revision: %w", err)
	}

	return repositoryfs.ContentRevision(data), nil
}

func slotRevision(repository *player_save.Repository, slot player_save.SlotID) (string, error) {
	record, err := repository.Load(slot)
	if errors.Is(err, player_save.ErrSlotNotFound) {
		return "", nil
	}
	reclaimable := errors.Is(err, player_save.ErrIncompatible) ||
		errors.Is(err, save_snapshot.ErrInvalidEnvelope) || errors.Is(err, save_snapshot.ErrInvalidSnapshot)
	if err != nil && (!reclaimable || record.Revision == "") {
		return "", fmt.Errorf("inspect player save slot: %w", err)
	}

	return record.Revision, nil
}

func (target destination) publish(ctx context.Context, envelope save_snapshot.Envelope) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("cancel publication: %w", err)
	}
	if err := rejectInputAlias(target.path, target.inputs); err != nil {
		return err
	}
	if target.repository != nil {
		if err := checkRegularDestination(target.path); err != nil {
			return err
		}
		// The canonicalization step validates runtime compatibility explicitly;
		// Repository.Save validates the envelope and the reviewed slot revision.
		if _, err := target.repository.Save(target.slot, envelope, target.revision); err != nil {
			return fmt.Errorf("slot persistence failed; inspect destination before retrying (it may have changed): %w", err)
		}

		return nil
	}
	data, err := save_snapshot.EncodeCanonical(envelope)
	if err != nil {
		return fmt.Errorf("encode converted save: %w", err)
	}

	return publishFile(ctx, target.path, target.revision, data, repositoryfs.SyncDirectory, target.inputs...)
}

func publishFile(
	ctx context.Context, path, expected string, data []byte, syncDirectory func(string) error, inputs ...os.FileInfo,
) error {
	parent := filepath.Dir(path)
	unlock, err := repositoryfs.AcquireExistingDirectoryLock(ctx, parent)
	if err != nil {
		return fmt.Errorf("lock output directory: %w", err)
	}
	defer unlock()
	if err := rejectInputAlias(path, inputs); err != nil {
		return err
	}
	actual, err := fileRevision(path)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("%w: %s", errStale, path)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("cancel publication: %w", err)
	}
	publish := func(temporary string) error {
		if expected == "" {
			if err := os.Link(temporary, path); err != nil {
				return fmt.Errorf("create output without replacement: %w", err)
			}
		} else if err := os.Rename(temporary, path); err != nil {
			return fmt.Errorf("replace reviewed output: %w", err)
		}

		return nil
	}
	err = repositoryfs.PublishTemporaryFile(parent, ".save-convert-*.tmp", data, outputFileMode, publish)
	if err != nil {
		return fmt.Errorf("publish output: %w", err)
	}
	if err := syncDirectory(parent); err != nil {
		return fmt.Errorf("complete output may exist, but directory synchronization failed; inspect before retrying: %w", err)
	}

	return nil
}
