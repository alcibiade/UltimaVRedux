package audio

import (
	"fmt"
	"slices"
	"strings"
)

var acousticFamilies = []string{ //nolint:gochecknoglobals // Closed JSON vocabulary.
	catalogAmbience, "consumable", "creature", "economy", familyFire, "footstep", "impact",
	familyInstrument, "inventory", "magic", "mechanical_interaction", "movement", familyNature,
	familyScore, "special_event", "status", "unreviewed",
}

const audioContextCombat = "combat"

func recordingFamily(taxonomy Taxonomy, id string) string {
	for family, ids := range taxonomy.RecordingFamilies {
		if slices.Contains(ids, id) {
			return family
		}
	}

	return ""
}

func cueProfile(taxonomy Taxonomy, id string) string {
	for profile, ids := range taxonomy.CueProfiles {
		if slices.Contains(ids, id) {
			return profile
		}
	}

	return ""
}

// validateProfiles applies Redux's closed classification vocabulary; Berad Audio
// has already checked each profile's structure and search terms.
func validateProfiles(taxonomy Taxonomy) error {
	for name, profile := range taxonomy.Profiles {
		if !slices.Contains(acousticFamilies, profile.TargetAcousticFamily) ||
			!slices.Contains([]string{"onset", "action", "impact", "sustain", "resolution", "state"}, profile.Phase) ||
			!slices.Contains([]string{"neutral", "success", "failure", "harm", "warning"}, profile.Outcome) ||
			!slices.Contains([]string{"foreground_feedback", "foley", "source_loop", "ambient_bed",
				"diegetic_music", "background_score", "stinger"}, profile.MixRole) ||
			!validTags(profile.Emitters, []string{"creature", "environment", "npc", "object", "party", "player", "system"}) ||
			!validTags(profile.Contexts, []string{audioContextCombat, "exploration", "interaction", "movement", "rest",
				"ritual", "spellcasting", "status", "travel"}) {
			return fmt.Errorf("%w: invalid classification profile %q", ErrAudioLibrary, name)
		}
	}

	return nil
}

func validTags(tags, vocabulary []string) bool {
	if len(tags) == 0 {
		return false
	}
	for i, tag := range tags {
		if !slices.Contains(vocabulary, tag) || strings.TrimSpace(tag) != tag ||
			(i > 0 && tags[i-1] >= tag) {
			return false
		}
	}

	return true
}

func cueCategory(cueID string) string {
	switch {
	case strings.HasPrefix(cueID, "sfx/"):
		return catalogSFX
	case strings.HasPrefix(cueID, "ambient/"):
		return catalogAmbience
	case strings.HasPrefix(cueID, "music/"):
		return catalogMusic
	default:
		return ""
	}
}
