package audio

import (
	"slices"
	"strings"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

const contextGainSemantics = "linear cue gain; combines with recording, category, position and fade; not final output"
const catalogMagicFamily = "magic"

func (catalog *Catalog) addEffectMoments() {
	for effect, symbol := range catalogEffectNames() {
		moment := newCatalogMoment("sfx/"+symbol, symbol, catalogSFX)
		moment.SourceSymbols = []string{symbol}
		moment.RelatedIssues = []int{580}
		moment.Window = soundPlaybackWindow(effect)
		if metadata, exists := GetSoundEffectMetadata(effect); exists {
			moment.Name = metadata.Name
			moment.GainSemantics = contextGainSemantics
			moment.Playback = "request-authoritative one-shot or loop; source references require lifecycle review"
			moment.Overlap = "independent voices; eight-SFX limit evicts oldest finite voice, protects owned loops"
			catalog.assignCue(&moment, "sfx/"+symbol)
			moment.GenerationBrief = familyGenerationBrief(moment.Family, symbol)
			if moment.Family == catalogMagicFamily {
				moment.RelatedIssues = []int{582}
			}
		}
		if effect == game_state.SoundWalkMetal1 {
			moment.Playback = "one footfall per successful step; recording may select an excerpt"
			moment.Overlap = "new metal footfall replaces previous metal voice"
			moment.RelatedIssues = []int{569}
		}
		catalog.Moments = append(catalog.Moments, moment)
	}
}

func (catalog *Catalog) addMusicMoments() {
	for id, metadata := range musicTrackMetadataMap {
		moment := catalog.musicMoment("music/track/"+id, metadata.Name, id)
		moment.SourceSymbols = []string{"PlayMusicForContext", "PlayBackgroundMusic"}
		moment.Note = "Library track: selection availability is not proof a particular gameplay context requests it."
		catalog.Moments = append(catalog.Moments, moment)
	}
	for context, track := range musicContextTracks() {
		moment := catalog.musicMoment("music/context/"+context, context, track)
		moment.SourceSymbols = []string{"PlayMusicForContext"}
		moment.Note = "Named selection context; shared dispatch evidence does not prove this context is invoked."
		if context == audioContextCombat {
			moment.SourceSymbols = []string{"SetSceneContext"}
			moment.Note = "Tactical encounter ownership bypasses exploration quiet intervals; one finite score per encounter."
		}
		catalog.Moments = append(catalog.Moments, moment)
	}
	for location, track := range locationMusicMappings() {
		moment := catalog.musicMoment("music/location/"+location.String(), location.String(), track)
		moment.HowToHear = "Enter " + location.String() + " with music enabled; verify the active scene uses location music."
		moment.SourceSymbols = []string{"UpdateLocationMusic", "updateMusicForCurrentLocation"}
		moment.Note = "Derived location mapping; source evidence is shared dispatch, not a walkthrough of this location."
		catalog.Moments = append(catalog.Moments, moment)
	}
	for _, world := range []string{"overworld", "underworld"} {
		cue, _ := resolveAudioAssignment(productionLibrary().Routing.Music, "world/"+world)
		moment := catalog.musicMoment("music/world/"+world, world, strings.TrimPrefix(cue, "music/track/"))
		moment.SourceSymbols = []string{"SetSceneLocation"}
		catalog.Moments = append(catalog.Moments, moment)
	}
	if cue := productionLibrary().Routing.Naval["sailing"]; cue != nil {
		moment := catalog.musicMoment("music/naval/sailing", "Sailing", strings.TrimPrefix(*cue, "music/track/"))
		moment.SourceSymbols = []string{"SetSceneSoundscape"}
		moment.Playback = "once per sail-hoist; fade on furling; combat takes precedence"
		catalog.Moments = append(catalog.Moments, moment)
	}
}

func (catalog *Catalog) musicMoment(id, name, track string) CatalogMoment {
	moment := newCatalogMoment(id, name, catalogMusic)
	moment.Playback = "finite track; exploration uses quiet intervals, subdued introduction and stable quieter repeats"
	moment.Overlap = "one score voice; local music and presentation owners suppress score; resume preserves position"
	moment.GainSemantics = contextGainSemantics
	moment.RelatedIssues = []int{579, 581}
	if track == "" {
		moment.Availability = "silent"
		moment.Playback = "authored silence"
		moment.Note = "The effective music assignment is silence, not an absent recording."

		return moment
	}
	catalog.assignCue(&moment, "music/track/"+track)

	return moment
}

func (catalog *Catalog) addAmbientMoments() {
	definitions := defaultAmbientSoundDefinitions()
	for soundType, symbol := range catalogAmbientNames() {
		moment := newCatalogMoment("ambient/"+symbol, symbol, catalogAmbience)
		catalog.assignCue(&moment, "ambient/"+symbol)
		moment.Name = productionLibrary().Cues[moment.CueID].Name
		moment.SourceSymbols = []string{symbol}
		moment.GenerationBrief = familyGenerationBrief(catalogAmbience, symbol)
		moment.RelatedIssues = []int{581}
		if definition, exists := definitions[soundType]; exists {
			moment.Playback = "infinite decoded loop; fade on entering/leaving range"
			moment.Overlap = "strongest static source per cue after tile distance/level overrides; dynamic sources coexist"
			moment.Note = "Range is the sound's default; individual tile families can override range and relative level."
			moment.MaxDistanceTiles = &definition.MaxDistance
			moment.FadeInMs = int64(definition.FadeInTime / time.Millisecond)
			moment.FadeOutMs = int64(definition.FadeOutTime / time.Millisecond)
			moment.GainSemantics = contextGainSemantics
			for sprite, mappedType := range ambientSpriteMappings() {
				if mappedType == soundType {
					moment.SpriteIndices = append(moment.SpriteIndices, int(sprite))
				}
			}
			slices.Sort(moment.SpriteIndices)
			moment.SourceSymbols = append(moment.SourceSymbols, "updateAmbientSounds")
		} else {
			moment.TriggerCoverage = catalogDeclaredOnly
			moment.Note = "Declared ambient type has no production definition; do not generate until an intended use is agreed."
		}
		catalog.Moments = append(catalog.Moments, moment)
	}
	for target, name := range DungeonTargets() {
		moment := newCatalogMoment("ambient/dungeon/"+target, name+" background", catalogAmbience)
		cue, _ := resolveAudioAssignment(productionLibrary().Routing.Dungeon, target)
		moment.SourceSymbols = []string{"DungeonAmbientSource"}
		moment.Playback = "non-spatial persistent loop; retained through dungeon room transitions"
		if cue == "" {
			moment.Availability = "silent"
			moment.Note = "Dungeon background is explicitly or implicitly silent."
		} else {
			catalog.assignCue(&moment, cue)
		}
		catalog.Moments = append(catalog.Moments, moment)
	}
	for _, vessel := range []string{NavalFrigate, NavalSkiff} {
		moment := newCatalogMoment("ambient/naval/"+vessel, NavalTargets()[vessel], catalogAmbience)
		moment.SourceSymbols = []string{"NavalAmbientSource"}
		moment.Playback = "persistent non-spatial loop while aboard"
		if cue := productionLibrary().Routing.Naval[vessel]; cue != nil {
			catalog.assignCue(&moment, *cue)
		} else {
			moment.Availability, moment.Note = "silent", "No onboard ambient sound assigned yet."
		}
		catalog.Moments = append(catalog.Moments, moment)
	}
}

func familyGenerationBrief(family, name string) string {
	if strings.Contains(name, "Bard") || strings.Contains(name, "Harpsichord") {
		return name + ": Isolated period-instrument performance or note; " +
			"agree duration and looping with the owning workflow. No voices or unrelated instruments. Draft brief."
	}
	var brief string
	switch family {
	case "footstep":
		brief = "One isolated footfall, not a walking sequence. Short natural tail; no voice/music. " +
			"Prepare comparable variants."
	case catalogMagicFamily:
		brief = "One restrained magical cue, recognizable within its family; no speech or background music. " +
			"Agree cast/impact/loop role before generation."
	case catalogAmbience:
		brief = "Continuous environmental texture, no music/voice or prominent repeated foreground event. " +
			"Prepare a seamless loop; agree source and density first."
	case "travel":
		brief = "Agree one movement stroke versus sustained travel bed first. " +
			"Keep frequent cues restrained; no speech/music or unrelated impacts."
	default:
		brief = "One isolated context-appropriate event, no speech/music or unrelated action sequence. " +
			"Agree duration and tail against the actual workflow."
	}

	return name + ": " + brief + " Draft brief; audition before accepting."
}
