// Package reduxadapter stages the game-owned bridge before the published module
// cutover. It is compiled/tested by the root audio-module gate. The generic module
// cannot import this package or interpret any Redux routing/JSON itself.
package reduxadapter

import (
	"fmt"
	"io/fs"

	gameaudio "github.com/bradhannah/UltimaVRedux/internal/audio"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	audio "github.com/bradhannah/berad-sdk/modules/berad-audio"
)

// Ambient joins the Redux source planner to an exclusively owned generic lane.
type Ambient struct {
	planner *gameaudio.AmbientPlanner
	lane    *audio.AmbientController
}

// NewAmbient uses the caller's JSON-loaded Redux project and explicitly rooted
// filesystem. Both libraries freeze their own data during this staging proof.
func NewAmbient(project *gameaudio.Library, files fs.FS,
	maxBytes int64, options audio.AmbientOptions,
) (*Ambient, error) {
	planner, err := gameaudio.NewAmbientPlanner(project)
	if err != nil {
		return nil, fmt.Errorf("construct Redux ambient planner: %w", err)
	}
	definition, err := PlaybackDefinition(project.Document())
	if err != nil {
		return nil, err
	}
	library, err := audio.NewLibrary(definition, files, maxBytes)
	if err != nil {
		return nil, fmt.Errorf("freeze Redux playback projection: %w", err)
	}
	lane, err := audio.NewAmbientController(library, options)
	if err != nil {
		return nil, fmt.Errorf("construct ambient lane: %w", err)
	}

	return &Ambient{planner: planner, lane: lane}, nil
}

// PlaybackDefinition projects playback properties only. JSON taxonomy, routing,
// names, provenance and credits remain in the caller's Redux document.
func PlaybackDefinition(document gameaudio.LibraryDocument) (audio.Definition, error) {
	definition, err := gameaudio.PlaybackDefinition(document)
	if err != nil {
		return audio.Definition{}, fmt.Errorf("project Redux playback: %w", err)
	}

	return definition, nil
}

// Reconcile translates the game's already-discovered world snapshot.
func (adapter *Ambient) Reconcile(listener references.Position,
	tiles []gameaudio.TileWithPosition, dynamic []gameaudio.AmbientSoundSource,
) error {
	plans, err := adapter.planner.Sources(listener, tiles, dynamic)
	if err != nil {
		return fmt.Errorf("project Redux ambient sources: %w", err)
	}
	sources := make([]audio.AmbientSource, 0, len(plans))
	for _, plan := range plans {
		sources = append(sources, audio.AmbientSource{ID: audio.SourceID(plan.ID), Cue: audio.CueID(plan.CueID),
			Gain: plan.Gain, Loop: plan.Loop, FadeIn: plan.FadeIn, FadeOut: plan.FadeOut})
	}

	return ambientError("reconcile", adapter.lane.Reconcile(sources))
}

// Update services playback independently of world discovery.
func (adapter *Ambient) Update() error { return ambientError("update", adapter.lane.Update()) }

// SetVolume changes the category preference without reprojecting JSON gains.
func (adapter *Ambient) SetVolume(gain float64) error {
	return ambientError("set volume", adapter.lane.SetVolume(gain))
}

// Snapshot exposes actual source and voice identities to game presentation policy.
func (adapter *Ambient) Snapshot() audio.AmbientSnapshot { return adapter.lane.Snapshot() }

// Close ends this lane, never the host device.
func (adapter *Ambient) Close() error { return ambientError("close", adapter.lane.Close()) }

func ambientError(operation string, err error) error {
	if err != nil {
		return fmt.Errorf("redux ambient %s: %w", operation, err)
	}

	return nil
}
