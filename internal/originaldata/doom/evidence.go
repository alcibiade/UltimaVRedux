package doom

import (
	"encoding/json"
	"fmt"
)

// CompatibilityEvidence is the compact canonical installed-data fingerprint.
// It intentionally excludes exhaustive room metadata, text offsets, and font glyph dimensions.
type CompatibilityEvidence struct {
	SchemaVersion int                         `json:"schemaVersion"`
	Doom          CompatibilityDoom           `json:"doom"`
	RescueMap     RescueReport                `json:"rescueMap"`
	Ending        EndingCompatibilityEvidence `json:"ending"`
}

// CompatibilityDoom contains only placement, room identity, and final mirror evidence.
type CompatibilityDoom struct {
	DATBlockOffset   int                 `json:"datBlockOffset"`
	Thresholds       []Threshold         `json:"thresholds"`
	RoomFingerprints []RoomFingerprint   `json:"roomFingerprints"`
	FinalMirror      FinalMirrorEvidence `json:"finalMirror"`
}

// RoomFingerprint identifies one referenced room without embedding its semantic contents.
type RoomFingerprint struct {
	RoomID       int    `json:"roomId"`
	RecordIndex  int    `json:"recordIndex"`
	RecordOffset int    `json:"recordOffset"`
	RecordSHA256 string `json:"recordSha256"`
}

// FinalMirrorEvidence contains the unique source-mechanism coordinates and entry rows.
type FinalMirrorEvidence struct {
	RoomID             int              `json:"roomId"`
	MirrorObject       Coordinate       `json:"mirrorObject"`
	MirrorTerrainValue byte             `json:"mirrorTerrainValue"`
	AbsorptionSquare   Coordinate       `json:"absorptionSquare"`
	EntryFormations    []EntryFormation `json:"entryFormations"`
}

// EndingCompatibilityEvidence fingerprints ending files without text offsets or glyph dimensions.
type EndingCompatibilityEvidence struct {
	Graphics []GraphicsAsset `json:"graphics"`
	TextData []FileMetadata  `json:"textData"`
}

// CompactEvidence projects an exhaustive report into the tracked compatibility format.
func CompactEvidence(report Report) CompatibilityEvidence {
	rooms := make([]RoomFingerprint, 0, len(report.Doom.Rooms))
	for _, room := range report.Doom.Rooms {
		rooms = append(rooms, RoomFingerprint{
			RoomID: room.RoomID, RecordIndex: room.RecordIndex,
			RecordOffset: room.RecordOffset, RecordSHA256: room.RecordSHA256,
		})
	}
	textData := make([]FileMetadata, 0, len(report.Ending.TextData))
	for _, asset := range report.Ending.TextData {
		textData = append(textData, asset.File)
	}

	return CompatibilityEvidence{
		SchemaVersion: report.SchemaVersion,
		Doom: CompatibilityDoom{
			DATBlockOffset:   report.Doom.DATBlockOffset,
			Thresholds:       report.Doom.Thresholds,
			RoomFingerprints: rooms,
			FinalMirror: FinalMirrorEvidence{
				RoomID:             report.Doom.FinalMirror.RoomID,
				MirrorObject:       report.Doom.FinalMirror.MirrorObject,
				MirrorTerrainValue: report.Doom.FinalMirror.MirrorTerrainValue,
				AbsorptionSquare:   report.Doom.FinalMirror.AbsorptionSquare,
				EntryFormations:    report.Doom.FinalMirror.EntryFormations,
			},
		},
		RescueMap: report.RescueMap,
		Ending:    EndingCompatibilityEvidence{Graphics: report.Ending.Graphics, TextData: textData},
	}
}

// CanonicalEvidenceJSON encodes the compact tracked compatibility format.
func CanonicalEvidenceJSON(report Report) ([]byte, error) {
	data, err := json.MarshalIndent(CompactEvidence(report), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encoding compact Doom evidence: %w", err)
	}

	return append(data, '\n'), nil
}
