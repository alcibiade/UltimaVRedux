package audio

import (
	"fmt"
	"slices"
	"strings"
)

// CueClassification describes an intended gameplay moment, not the sound of
// the currently assigned recording. Several existing assignments are stand-ins.
type CueClassification struct {
	TargetAcousticFamily string   `json:"targetAcousticFamily"`
	Phase                string   `json:"phase"`
	Outcome              string   `json:"outcome"`
	MixRole              string   `json:"mixRole"`
	Emitters             []string `json:"emitters"`
	Contexts             []string `json:"contexts"`
}

// Taxonomy groups identities under controlled, reusable classifications.
// The groups are authored in library.json; they do not affect playback routing.
type Taxonomy struct {
	RecordingFamilies  map[string][]string          `json:"recordingFamilies"`
	CueProfiles        map[string][]string          `json:"cueProfiles"`
	Profiles           map[string]CueClassification `json:"profiles"`
	ProfileSearchTerms map[string][]string          `json:"profileSearchTerms,omitempty"`
}

var acousticFamilies = []string{ //nolint:gochecknoglobals // Closed JSON vocabulary.
	catalogAmbience, "consumable", "creature", "economy", familyFire, "footstep", "impact",
	familyInstrument, "inventory", "magic", "mechanical_interaction", "movement", familyNature,
	familyScore, "special_event", "status", "unreviewed",
}

const audioContextCombat = "combat"

func (taxonomy *Taxonomy) recordingFamily(id string) string {
	for family, ids := range taxonomy.RecordingFamilies {
		if slices.Contains(ids, id) {
			return family
		}
	}

	return ""
}

func (taxonomy *Taxonomy) cueProfile(id string) string {
	for profile, ids := range taxonomy.CueProfiles {
		if slices.Contains(ids, id) {
			return profile
		}
	}

	return ""
}

func (taxonomy *Taxonomy) validateStructure(library *audioLibrary) error {
	if taxonomy.RecordingFamilies == nil || taxonomy.CueProfiles == nil || taxonomy.Profiles == nil {
		return fmt.Errorf("%w: schema v2 requires recordingFamilies, cueProfiles, and profiles", ErrAudioLibrary)
	}
	if err := taxonomy.validateProfileTerms(); err != nil {
		return err
	}
	for name, profile := range taxonomy.Profiles {
		if name == "" || profile.TargetAcousticFamily == "" || profile.Phase == "" ||
			profile.Outcome == "" || profile.MixRole == "" ||
			!validOpenTags(profile.Emitters) || !validOpenTags(profile.Contexts) {
			return fmt.Errorf("%w: incomplete classification profile %q", ErrAudioLibrary, name)
		}
	}
	if err := taxonomy.validateRecordingFamilies(library); err != nil {
		return err
	}

	return taxonomy.validateCueProfiles(library)
}

func validOpenTags(tags []string) bool {
	if len(tags) == 0 {
		return false
	}
	for index, tag := range tags {
		if strings.TrimSpace(tag) == "" || strings.TrimSpace(tag) != tag || (index > 0 && tags[index-1] >= tag) {
			return false
		}
	}

	return true
}

func (taxonomy *Taxonomy) validateProfiles() error {
	if err := taxonomy.validateProfileTerms(); err != nil {
		return err
	}
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

func (taxonomy *Taxonomy) validateProfileTerms() error {
	for profile, terms := range taxonomy.ProfileSearchTerms {
		if _, exists := taxonomy.Profiles[profile]; !exists {
			return fmt.Errorf("%w: search terms reference unknown profile %s", ErrAudioLibrary, profile)
		}
		if err := validateSearchTerms(terms); err != nil {
			return fmt.Errorf("profile %s: %w", profile, err)
		}
	}

	return nil
}

func (taxonomy *Taxonomy) validateRecordingFamilies(library *audioLibrary) error {
	seenRecordings := make(map[string]bool, len(library.Recordings))
	for family, ids := range taxonomy.RecordingFamilies {
		if family == "" || len(ids) == 0 {
			return fmt.Errorf("%w: invalid recording family %q", ErrAudioLibrary, family)
		}
		for _, id := range ids {
			if _, exists := library.Recordings[id]; !exists || seenRecordings[id] {
				return fmt.Errorf("%w: unknown or duplicate classified recording %q", ErrAudioLibrary, id)
			}
			seenRecordings[id] = true
		}
	}
	if len(seenRecordings) != len(library.Recordings) {
		return fmt.Errorf("%w: every recording requires one acoustic family", ErrAudioLibrary)
	}

	return nil
}

func (taxonomy *Taxonomy) validateCueProfiles(library *audioLibrary) error {
	seenCues := make(map[string]bool, len(library.Cues))
	for profile, ids := range taxonomy.CueProfiles {
		if _, exists := taxonomy.Profiles[profile]; !exists || len(ids) == 0 {
			return fmt.Errorf("%w: unknown or empty cue profile %q", ErrAudioLibrary, profile)
		}
		for _, id := range ids {
			if _, exists := library.Cues[id]; !exists || seenCues[id] {
				return fmt.Errorf("%w: unknown or duplicate classified cue %q", ErrAudioLibrary, id)
			}
			seenCues[id] = true
		}
	}
	if len(seenCues) != len(library.Cues) {
		return fmt.Errorf("%w: every cue requires one semantic profile", ErrAudioLibrary)
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
