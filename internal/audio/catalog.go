package audio

import (
	"fmt"
	"io/fs"
	"slices"

	"github.com/bradhannah/berad-sdk/modules/berad-audio/backend/ebitengine"
	sdkcatalog "github.com/bradhannah/berad-sdk/tools/berad-audio/catalog"
)

// Catalog schema and review states retain the SDK inventory contract.
const (
	CatalogUnreviewed   = sdkcatalog.CatalogUnreviewed
	CatalogPresent      = sdkcatalog.CatalogPresent
	CatalogMissing      = sdkcatalog.CatalogMissing
	CatalogIncorrect    = sdkcatalog.CatalogIncorrect
	CatalogPlaceholder  = sdkcatalog.CatalogPlaceholder
	catalogNotScanned   = "not_scanned"
	catalogDeclaredOnly = "declared_only"
	catalogNoReference  = "no_static_reference_found"
	catalogSFX          = "sfx"
	catalogMusic        = "music"
	catalogAmbience     = "ambience"
)

// ErrAudioCatalog identifies invalid inventory or game enrichment.
var ErrAudioCatalog = sdkcatalog.ErrAudioCatalog

// Catalog is Redux's game-enriched view of the SDK inventory. Game source evidence,
// proposed moments and review notes remain here; measurement/rendering are shared.
type Catalog sdkcatalog.Catalog

// CatalogMoment describes an enriched semantic cue or context.
type CatalogMoment = sdkcatalog.CatalogMoment

// CatalogEvidence records game-owned static source references.
type CatalogEvidence = sdkcatalog.CatalogEvidence

// BuildCatalog enriches the shared inventory with Redux gameplay metadata.
func BuildCatalog(measure bool) (*Catalog, error) {
	manifest, err := soundFiles.ReadFile(libraryManifestPath)
	if err != nil {
		return nil, fmt.Errorf("read embedded catalog manifest: %w", err)
	}
	files, err := fs.Sub(soundFiles, "assets")
	if err != nil {
		return nil, fmt.Errorf("mount catalog assets: %w", err)
	}
	base, err := sdkcatalog.Build(manifest, files, sdkcatalog.Options{
		Decode: ebitengine.Decode, Measure: measure, PathPrefix: "assets/", CueCategories: nil})
	if err != nil {
		return nil, fmt.Errorf("build SDK inventory: %w", err)
	}
	catalog := (*Catalog)(base)
	catalog.Moments = nil
	catalog.addEffectMoments()
	catalog.addMusicMoments()
	catalog.addAmbientMoments()
	catalog.Moments = append(catalog.Moments, proposedCatalogMoments()...)
	applyCatalogNotes(catalog)
	base.Relink()
	slices.SortFunc(catalog.Moments, func(a, b CatalogMoment) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}

		return 0
	})
	if err := catalog.Validate(); err != nil {
		return nil, err
	}

	return catalog, nil
}

func newCatalogMoment(id, name, category string) CatalogMoment {
	var moment CatalogMoment
	moment.ID, moment.Name, moment.Category = id, name, category
	moment.Availability = "missing_definition"
	moment.AssignmentReview, moment.ListeningReview = CatalogUnreviewed, CatalogUnreviewed
	moment.TriggerCoverage = catalogNotScanned
	moment.Evidence, moment.SourceSymbols = []CatalogEvidence{}, []string{}
	moment.SpriteIndices, moment.RelatedIssues = []int{}, []int{}
	moment.Playback, moment.Overlap = "not configured", "not configured"

	return moment
}

func (catalog *Catalog) assignCue(moment *CatalogMoment, cueID string) {
	library := productionLibrary()
	entry := library.Cues[cueID]
	moment.SearchTerms = append(slices.Clone(entry.SearchTerms),
		library.Taxonomy.ProfileSearchTerms[library.Taxonomy.cueProfile(cueID)]...)
	profile := library.Taxonomy.Profiles[library.Taxonomy.cueProfile(cueID)]
	moment.Category = cueCategory(cueID)
	moment.Family, moment.Phase = profile.TargetAcousticFamily, profile.Phase
	moment.Outcome, moment.MixRole = profile.Outcome, profile.MixRole
	moment.Emitters, moment.Contexts = slices.Clone(profile.Emitters), slices.Clone(profile.Contexts)
	moment.CueID, moment.AssetID = cueID, entry.PrimaryRecording()
	for _, id := range entry.Recordings() {
		family := library.Taxonomy.recordingFamily(id)
		if !slices.Contains(moment.AssignedFamilies, family) {
			moment.AssignedFamilies = append(moment.AssignedFamilies, family)
		}
	}
	if entry.Selection == SelectionCycle {
		moment.AssetIDs, moment.Selection = entry.Recordings(), entry.Selection
	}
	moment.HowToHear, moment.ConfiguredGain = entry.HowToHear, &entry.VolumeModifier
	moment.ConfiguredFile = library.recordingPath(entry.PrimaryRecording())
	if window := library.Recordings[entry.PrimaryRecording()].PlaybackWindow; window != nil {
		selection := *window
		moment.Window = &selection
	}
	moment.Availability = CatalogPresent
	if moment.ConfiguredFile == "" {
		moment.Availability, moment.AssignmentReview = CatalogMissing, CatalogPlaceholder
	}
}

// Validate checks game enrichment and SDK graph integrity.
func (catalog *Catalog) Validate() error {
	for effect := range soundEffectMetadataMap() {
		if _, exists := catalogEffectNames()[effect]; !exists {
			return fmt.Errorf("%w: effect %d has no catalog identity", ErrAudioCatalog, effect)
		}
	}
	if err := (*sdkcatalog.Catalog)(catalog).Validate(); err != nil {
		return fmt.Errorf("validate SDK inventory: %w", err)
	}

	return nil
}

// GenerationGuide renders the shared escaped report with game instructions.
func (catalog *Catalog) GenerationGuide() string {
	return (*sdkcatalog.Catalog)(catalog).GenerationGuide(sdkcatalog.ReportOptions{
		ManifestLabel:  LibraryManifestRelativePath,
		AssetRootLabel: "assets/", ReloadNotice: "Rebuild the game to use saved changes."})
}
