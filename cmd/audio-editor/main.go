// Command audio-editor edits the production audio manifest with live terminal auditioning.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
	"github.com/bradhannah/UltimaVRedux/internal/audio/editorui"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

func main() {
	logging.SetGlobalEnabled(true)
	logging.SetCategoryEnabled(logging.CategoryAudio, true)
	logging.SetStdoutEnabled(true)
	if err := run(os.Args[1:]); err != nil {
		logging.Error(logging.CategoryAudio, "Audio editor: %v", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("audio-editor", flag.ContinueOnError)
	path := flags.String("library", audio.LibraryManifestRelativePath, "Production library.json to edit")
	check := flags.Bool("check", false, "Validate the disk-backed library without opening audio or writing files")
	probe := flags.String("probe", "", "Play one recording ID briefly and verify standalone playback progress")
	if err := flags.Parse(arguments); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return fmt.Errorf("parse editor options: %w", err)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("%w: unexpected positional arguments", audio.ErrAudioLibrary)
	}
	editor, err := audio.OpenLibraryEditor(*path)
	if err != nil {
		return fmt.Errorf("open editor library: %w", err)
	}
	if *check {
		if err := editor.Validate(); err != nil {
			return fmt.Errorf("validate editor library: %w", err)
		}
		logging.Info(logging.CategoryAudio, "Valid library: %d recordings, %d cues",
			len(editor.RecordingIDs()), len(editor.CueIDs()))

		return nil
	}
	if *probe != "" {
		return probeRecording(editor, *probe)
	}
	logging.SetStdoutEnabled(false)
	defer logging.SetStdoutEnabled(true)

	if err := editorui.Run(editor); err != nil {
		return fmt.Errorf("run terminal editor: %w", err)
	}

	return nil
}

func probeRecording(editor *audio.LibraryEditor, recordingID string) error {
	path, err := editor.RecordingPath(recordingID)
	if err != nil {
		return fmt.Errorf("resolve probe recording: %w", err)
	}
	clip, err := audio.LoadRecording(path)
	if err != nil {
		return fmt.Errorf("load probe recording: %w", err)
	}
	voice := new(audio.Audition)
	defer func() {
		if err := voice.Close(); err != nil {
			logging.Warn(logging.CategoryAudio, "Close probe recording: %v", err)
		}
	}()
	const probeGain = 0.2
	if err := voice.Play(clip, nil, false, probeGain); err != nil {
		return fmt.Errorf("play probe recording: %w", err)
	}
	const probeDuration = 2 * time.Second
	time.Sleep(probeDuration)
	if voice.PositionMs() <= 0 {
		return fmt.Errorf("%w: audio device did not advance playback", audio.ErrAudioLibrary)
	}
	logging.Info(logging.CategoryAudio, "Standalone playback advanced to %d ms (%s)", voice.PositionMs(), recordingID)

	return nil
}
