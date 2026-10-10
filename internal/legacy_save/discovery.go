package legacy_save

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bradhannah/UltimaVRedux/internal/originaldata"
)

// DiscoveredSave is a fully decoded and convertible original save pair found
// in one of the bounded installation locations.
type DiscoveredSave struct {
	SavedGamePath string
	Directory     string
	Source        originaldata.Source
	Label         string
	AvatarName    string
	Location      int
	Floor         int
}

// DiscoveryOptions controls the finite legacy-save search.
type DiscoveryOptions struct {
	Host                    originaldata.HostProfile
	ActiveDataDirectory     string
	PreviousImportDirectory string
}

// DiscoveryReport contains stable usable candidates and diagnostics for
// existing locations that did not contain a supported save pair.
type DiscoveryReport struct {
	Candidates []DiscoveredSave
	Details    []originaldata.Detail
	Cancelled  bool
}

// Discover searches only the finite original-data catalogue. Every candidate
// is decoded and converted before it is offered to the import UI.
func Discover(ctx context.Context, options DiscoveryOptions) DiscoveryReport {
	previews := make(map[string]DiscoveredSave)
	report := originaldata.Discover(ctx, originaldata.DiscoveryOptions{
		Host:                  options.Host,
		ConfiguredDirectory:   options.ActiveDataDirectory,
		PreviousDataDirectory: options.PreviousImportDirectory,
		AdditionalSeeds:       nil,
		Validate: func(ctx context.Context, directory string) error {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("cancelling legacy save validation: %w", err)
			}
			savedGamePath, err := ResolveSavedGamePathContext(ctx, directory, options.Host.HomeDirectory)
			if err != nil {
				return err
			}
			bundle, err := ReadBundleContext(ctx, savedGamePath)
			if err != nil {
				return err
			}
			conversion, err := Convert(bundle)
			if err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("cancelling legacy save validation: %w", err)
			}
			avatarName := conversion.Snapshot.Party.Roster[0].Name
			previews[filepath.Clean(directory)] = DiscoveredSave{
				SavedGamePath: savedGamePath,
				Directory:     filepath.Dir(savedGamePath),
				Source:        "",
				Label:         "",
				AvatarName:    avatarName,
				Location:      conversion.Snapshot.World.Position.Location,
				Floor:         conversion.Snapshot.World.Position.Floor,
			}

			return nil
		},
	})
	result := DiscoveryReport{Candidates: nil, Details: report.Details, Cancelled: report.Cancelled}
	for _, candidate := range report.Candidates {
		preview, ok := previews[filepath.Clean(candidate.Path)]
		if !ok {
			// Discovery accepted this exact path through the validator above.
			panic(fmt.Sprintf("legacy save discovery: missing preview for %q", candidate.Path))
		}
		preview.Source = candidate.Source
		preview.Label = legacyCandidateLabel(candidate.Source, candidate.Label)
		result.Candidates = append(result.Candidates, preview)
	}

	return result
}

func legacyCandidateLabel(source originaldata.Source, fallback string) string {
	switch source {
	case originaldata.SourceConfigured:
		return "Save in current game folder"
	case originaldata.SourcePrevious:
		return "Save in last import folder"
	case originaldata.SourceGOG:
		return "Save in GOG installation"
	case originaldata.SourceHeroic:
		return "Save in Heroic installation"
	case originaldata.SourcePortable:
		return "Save in local copy"
	case originaldata.SourceRecommended:
		return "Save in personal folder"
	case originaldata.SourceCommon:
		return "Save in common installation"
	default:
		return fallback
	}
}
