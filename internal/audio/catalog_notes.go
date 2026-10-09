package audio

type catalogNote struct {
	incorrect bool
	note      string
	brief     string
}

//nolint:lll // Keep human-authored recording briefs readable as complete prose in the annotation table.
func catalogMomentNotes() map[string]catalogNote {
	const woodSourceNote = "Assigned source is walking over dead leaves, not documented wooden flooring."

	return map[string]catalogNote{
		"sfx/SoundPushObject": {true, "Confirmed inappropriate healing substitution; player reports it is unpleasant.",
			"One short furniture/object scrape ending in a soft settling thud. No magic chime, voice, music, or second action. Start with an isolated sub-second candidate; inspect sustained repeated pushes. Material variants can follow the first accepted cue."},
		"sfx/SoundUnlock": {true, "Physical unlocking currently uses a healing recording.",
			"One close, restrained mechanical key-and-latch release. No door swing, magic, voice, music, or repeated lock sequence. Short tail; comparable prominence to the existing door cue."},
		"sfx/SoundGlassBreak": {true, "Glass breaking currently uses a healing recording.",
			"One brief brittle glass crack with a small falling-shard tail. No explosion, voice, music, or multiple break events. Avoid piercing high-frequency loudness; test in the actual break context."},
		"sfx/SoundCannonFire": {true, "Burning-cardboard recording is assigned to a cannon report.",
			"One compact cannon discharge with short outdoor decay. No battle music, shouting, repeated fire, or long explosion bed. Audition against ordinary feedback rather than maximizing impact."},
		"sfx/SoundBroadside":      {false, "Earthquake recording reused; verify that this suits the actual broadside event.", ""},
		"sfx/SoundTrapTrigger":    {false, "Secondary healing recording substitution needs contextual replacement review.", "One abrupt mechanical trap release; keep trigger distinct from subsequent damage/fall feedback."},
		"sfx/SoundStepOnTrap":     {false, "Healing substitution needs contextual replacement review.", "One short pressure-trigger click or snap; no footsteps sequence or healing chime."},
		"sfx/SoundMoongateTravel": {false, "Healing substitution; distinguish travel from sustained gate hum.", "One restrained magical passage/whoosh with short tail; no music or spoken incantation."},
		"sfx/SoundWalkMetal1": {false, "The multi-footfall source is split into metadata clips for an ordered grate cycle. Each request replaces the previous metal voice; source measurements describe the full file. Timbre and clip boundaries still need listening acceptance.",
			"One isolated boot contact on a metal grate, with a short restrained metallic resonance. No walking sequence, mechanical hum, music, or extra clanks. Prepare a few consistently leveled alternatives for later acceptance."},
		"sfx/SoundWalkWood1": {false, woodSourceNote, ""},
		"sfx/SoundWalkWood2": {false, woodSourceNote, ""},
		"sfx/SoundWalkWood3": {false, woodSourceNote, ""},
		"sfx/SoundMagicLock": {false, "Prepared recording slot for magical locking. Playback requests exist; the missing recording is now an intentional no-op.",
			"One brief magical seal settling over a door latch. Related but distinguishable from magical unlocking; no music, voice, door slam, or looping hum."},
		"sfx/SoundMagicUnlock": {false, "Prepared recording slot for magical unlocking. Playback requests exist; the missing recording is now an intentional no-op.",
			"One brief magical seal dissolving with a restrained latch release. Related to magical locking; no speech, music, or long celebratory tail."},
		"sfx/SoundWhiteNoiseDamage": {false, "Referenced by combat cones and other effects; now has a prepared recording slot. Review shared contexts before designing one universal damage sound.", ""},
		"sfx/SoundSpellFailure": {false, "Failure cues already have callers; the prepared recording slot can be filled without editing gameplay code.",
			"One quiet magical fizzle that quickly loses energy. Clear failure without a harsh buzzer, speech, or musical sting. Keep short and tolerable on repeated failures."},
		"sfx/SoundCreateFood": {false, "Conjuration cue; verify current cast request while retaining shared magic-family identity.",
			"One small conjuring shimmer resolving into a soft material arrival. No voice, music, eating noises, or excessive sparkle tail."},
		"sfx/SoundLocate": {false, "Informational magic should be less prominent than a major event.",
			"One subtle clear magical orientation ping. No radar sequence, voice, music, or sustained tone."},
		"sfx/SoundResurrect": {false, "Major restoration needs distinct scale without drowning the environment.",
			"One restrained rising restoration swell ending in a warm release. Related to healing; no choir, speech, music, or repeated pulses. Agree sequence duration before final production."},
		"ambient/AmbientOcean": {false, "Ocean is currently mapped to wind; consider water-specific texture after listening.",
			"A steady open-water wave/wash bed without distinct timed crashes, boats, voice, or music. Seamless loop candidate; avoid excessive bass or hiss."},
		"ambient/AmbientHum":  {false, "Shared by fields and dynamic moongates; inspect each context before changing shared calibration.", ""},
		"ambient/AmbientBard": {false, "Local music suppresses score; contextual listening remains required.", ""},
	}
}

func applyCatalogNotes(catalog *Catalog) {
	notes := catalogMomentNotes()
	baselineIDs := catalogNoteRecordingIDs()
	for i := range catalog.Moments {
		moment := &catalog.Moments[i]
		if expected, bound := baselineIDs[moment.ID]; bound && moment.AssetID != expected {
			continue // A newly assigned candidate must not inherit an old recording's diagnosis.
		}
		if note, exists := notes[moment.ID]; exists {
			moment.Note = note.note
			if note.incorrect {
				moment.AssignmentReview = CatalogIncorrect
			}
			if note.brief != "" {
				moment.GenerationBrief = note.brief + " Draft brief; recording still requires contextual acceptance."
			}
		}
	}
}

func catalogNoteRecordingIDs() map[string]string {
	const healingRecording = "healing_01"

	return map[string]string{
		"sfx/SoundPushObject": healingRecording, "sfx/SoundUnlock": healingRecording, "sfx/SoundGlassBreak": healingRecording,
		"sfx/SoundCannonFire": "cardboard_burning_01", "sfx/SoundBroadside": "earthquake_01",
		"sfx/SoundTrapTrigger": "healing_02", "sfx/SoundStepOnTrap": healingRecording,
		"sfx/SoundMoongateTravel": healingRecording,
		"sfx/SoundWalkMetal1":     "stone_steps_01", "sfx/SoundWalkWood1": "footstep_leaves_01",
		"sfx/SoundWalkWood2": "footstep_leaves_02", "sfx/SoundWalkWood3": "footstep_leaves_03",
		"ambient/AmbientOcean": "wind_01",
	}
}

//nolint:lll // Context-review proposals are prose data, not executable policy.
func proposedCatalogMoments() []CatalogMoment {
	proposals := []struct{ id, name, category, note string }{
		{"ambience/indoor-room", "Indoor room ambience", catalogAmbience,
			"Coverage candidate: intentional quiet may be correct. Do not add a generic loop to every interior by default."},
		{"ambience/night-outdoors", "Night outdoor ambience", catalogAmbience,
			"Coverage candidate: current forest source mapping has no time-specific recording selection. Agree eligibility and desired texture first."},
		{"sfx/spell-projectile-impact", "Spell projectile and impact families", catalogSFX,
			"Coverage candidate: audit fire/energy projectile workflows and generic magic/damage reuse. Do not assume all projectiles need cast, travel, and impact sounds."},
		{"sfx/combat-weapon-impact", "Weapon contact and miss families", catalogSFX,
			"Coverage candidate: audit existing combat feedback and event boundaries before choosing material/weapon variants."},
	}
	moments := make([]CatalogMoment, 0, len(proposals))
	for _, proposal := range proposals {
		moment := newCatalogMoment("proposed/"+proposal.id, proposal.name, proposal.category)
		moment.Note = proposal.note
		moment.TriggerCoverage = "proposed_not_integrated"
		moment.Availability = "not_assigned"
		moment.Family = proposal.category
		moment.MixRole = "requires context review"
		moment.RelatedIssues = []int{574}
		moments = append(moments, moment)
	}

	return moments
}
