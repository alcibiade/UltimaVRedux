// Command audio-editor edits the production audio manifest with live terminal auditioning.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/bradhannah/berad-sdk/modules/berad-audio/authoring"
	"github.com/bradhannah/berad-sdk/tools/berad-audio/audition"
	editor "github.com/bradhannah/berad-sdk/tools/berad-audio/editor"
	"github.com/bradhannah/berad-sdk/tools/berad-audio/runtimeprobe"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
	"github.com/bradhannah/UltimaVRedux/internal/audio/tooling"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

func main() {
	logging.SetGlobalEnabled(true)
	logging.SetCategoryEnabled(logging.CategoryAudio, true)
	logging.SetStdoutEnabled(true)
	if err := run(os.Args[1:]); err != nil {
		logging.Errorf(logging.CategoryAudio, "Audio editor: %v", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("audio-editor", flag.ContinueOnError)
	path := flags.String("library", audio.LibraryManifestRelativePath, "Production library.json to edit")
	check := flags.Bool("check", false, "Validate the disk-backed library without opening audio or writing files")
	probe := flags.String("probe", "", "Play one recording ID briefly and verify standalone playback progress")
	runtimeProbe := flags.String("probe-runtime", "", "Run a bounded Ebitengine device probe for one cue ID")
	runtimeSoak := flags.String("probe-runtime-soak", "", "Run a muted one-minute Ebitengine lifecycle/heap probe")
	if err := flags.Parse(arguments); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return fmt.Errorf("parse editor options: %w", err)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("%w: unexpected positional arguments", audio.ErrAudioLibrary)
	}
	modes := 0
	for _, selected := range []bool{*check, *probe != "", *runtimeProbe != "", *runtimeSoak != ""} {
		if selected {
			modes++
		}
	}
	if modes > 1 {
		return fmt.Errorf("%w: choose one check or native probe mode", audio.ErrAudioLibrary)
	}
	document, err := tooling.OpenAuthoringEditor(*path)
	if err != nil {
		return fmt.Errorf("open editor library: %w", err)
	}
	if *check {
		if err := document.Validate(); err != nil {
			return fmt.Errorf("validate editor library: %w", err)
		}
		logging.Infof(logging.CategoryAudio, "Valid library: %d recordings, %d cues",
			len(document.RecordingIDs()), len(document.CueIDs()))

		return nil
	}
	if *probe != "" {
		return probeRecording(document, *probe)
	}
	if *runtimeProbe != "" {
		return runRuntimeProbe(document, *runtimeProbe, false)
	}
	if *runtimeSoak != "" {
		return runRuntimeProbe(document, *runtimeSoak, true)
	}
	logging.SetStdoutEnabled(false)
	defer logging.SetStdoutEnabled(true)

	if err := editor.Run(document, tooling.AudioEditorOptions()); err != nil {
		return fmt.Errorf("run terminal editor: %w", err)
	}

	return nil
}

func probeRecording(editor *authoring.LibraryEditor, recordingID string) error {
	path, err := editor.RecordingPath(recordingID)
	if err != nil {
		return fmt.Errorf("resolve probe recording: %w", err)
	}
	clip, err := editor.LoadRecording(context.Background(), path)
	if err != nil {
		return fmt.Errorf("load probe recording: %w", err)
	}
	voice := audition.NewVoice()
	defer func() {
		if err := voice.Close(); err != nil {
			logging.Warnf(logging.CategoryAudio, "Close probe recording: %v", err)
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
	logging.Infof(logging.CategoryAudio, "Standalone playback advanced to %d ms (%s)", voice.PositionMs(), recordingID)

	return nil
}

func runRuntimeProbe(editor *authoring.LibraryEditor, cue string, soak bool) error {
	const probeEncodedBudget = 512 << 20
	library, err := editor.RuntimeLibrary(probeEncodedBudget)
	if err != nil {
		return fmt.Errorf("freeze probe library: %w", err)
	}
	if err := runtimeprobe.Run(library, cue, soak, os.Stdout); err != nil {
		return fmt.Errorf("run SDK native probe: %w", err)
	}

	return nil
}
