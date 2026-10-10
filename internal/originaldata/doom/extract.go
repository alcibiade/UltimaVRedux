// Package doom extracts deterministic semantic evidence for Doom and the finale
// from original Ultima V data files.
package doom

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/bradhannah/UltimaVRedux/internal/graphics/legacy"
)

const (
	dungeonMinimumSize    = 0x1000
	dungeonCBTMinimumSize = 0x9a00
	miscMapsMinimumSize   = 1871
	doomBlockOffset       = 0x0e00
	dungeonDimension      = 8
	dungeonRoomSize       = 0x160
	doomRoomBaseIndex     = 96
	roomIDMask            = 0x0f
	rescueMapOffset       = 528
	rescueMapSize         = 176
	combatRowSize         = 32
	combatMapSize         = 11
	entityCount           = 16
	triggerCount          = 8
	partySize             = 6
	inMirrorFamily        = 0x3c
	mirrorTerrain         = 0x9d
	mirrorObjectRow       = 1
	absorptionRow         = 2
	endingShapeEntryCount = 3
	textShapeEntryCount   = 6
	proportEntryCount     = 91
	proportGlyphHeight    = 8
)

var (
	errRequiredAssetMissing       = errors.New("required asset is missing")
	errStructuralMinimum          = errors.New("structural minimum not met")
	errMalformedEntityCoordinates = errors.New("malformed active entity coordinates")
	errMirrorCandidateCount       = errors.New("final mirror candidate count is invalid")
	errMirrorTerrainMismatch      = errors.New("final mirror terrain does not match source mechanism")
	errGraphicsSpecUnavailable    = errors.New("legacy graphics specification is unavailable")
	errGraphicsEntryCount         = errors.New("graphics archive entry count is incompatible")
	errGraphicsEntryUnusable      = errors.New("required graphics entry is unusable")
	errTextRecordOffset           = errors.New("ending text record offset is out of range")
	errTextRecordTermination      = errors.New("ending text record is not terminated")
	errRescueMirrorMismatch       = errors.New("rescue map mirror does not match source data")
	errRescuePadding              = errors.New("rescue map row padding is nonzero")
	errMalformedPartyCoordinates  = errors.New("malformed party entry coordinates")
)

// Files contains the original file bytes required by Extract.
type Files struct {
	DungeonDAT  []byte
	DungeonCBT  []byte
	MiscMapsDAT []byte
	End1        []byte
	End2        []byte
	EndScreen   []byte
	Text        []byte
	Proport     []byte
	EndDAT      []byte
	EndMsgDAT   []byte
}

// Report is a JSON-ready semantic snapshot of Doom and finale data.
type Report struct {
	SchemaVersion int             `json:"schemaVersion"`
	Doom          DungeonEvidence `json:"doom"`
	RescueMap     RescueReport    `json:"rescueMap"`
	Ending        EndingReport    `json:"ending"`
}

// DungeonEvidence records all packed Doom room thresholds and their referenced rooms.
type DungeonEvidence struct {
	DATBlockOffset int             `json:"datBlockOffset"`
	Thresholds     []Threshold     `json:"thresholds"`
	Rooms          []RoomReport    `json:"rooms"`
	FinalMirror    FinalMirrorRoom `json:"finalMirrorRoom"`
}

// Threshold identifies one A* or F* packed Doom cell.
type Threshold struct {
	RawValue        byte `json:"rawValue"`
	Floor           int  `json:"floor"`
	X               int  `json:"x"`
	Y               int  `json:"y"`
	DATOffset       int  `json:"datOffset"`
	RoomID          int  `json:"roomId"`
	CBTRecordIndex  int  `json:"cbtRecordIndex"`
	CBTRecordOffset int  `json:"cbtRecordOffset"`
}

// RoomReport contains semantic metadata from one referenced DUNGEON.CBT record.
type RoomReport struct {
	RoomID          int              `json:"roomId"`
	RecordIndex     int              `json:"recordIndex"`
	RecordOffset    int              `json:"recordOffset"`
	RecordSHA256    string           `json:"recordSha256"`
	EntryFormations []EntryFormation `json:"entryFormations"`
	Entities        []FixedEntity    `json:"fixedEntities"`
	Triggers        []TriggerSlot    `json:"triggerSlots"`
}

// EntryFormation is one source-defined party entry row.
type EntryFormation struct {
	Direction string       `json:"direction"`
	Row       int          `json:"row"`
	Positions []Coordinate `json:"positions"`
}

// FixedEntity is one nonzero row-5 descriptor and its row-6/7 position.
type FixedEntity struct {
	Slot       int        `json:"slot"`
	Descriptor byte       `json:"descriptor"`
	Position   Coordinate `json:"position"`
}

// TriggerSlot preserves all source bytes for one ordered checktrap slot.
type TriggerSlot struct {
	Slot            int             `json:"slot"`
	ReplacementTile byte            `json:"replacementTile"`
	Trigger         Coordinate      `json:"trigger"`
	Disabled        bool            `json:"disabled"`
	Destinations    []TriggerTarget `json:"destinations"`
}

// TriggerTarget records a replacement coordinate and whether checktrap accepts it.
type TriggerTarget struct {
	Position Coordinate `json:"position"`
	InBounds bool       `json:"inBounds"`
}

// Coordinate is a semantic map coordinate.
type Coordinate struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// FinalMirrorRoom identifies the unique mirror route into the finale; ref:DNG-0015.
type FinalMirrorRoom struct {
	RoomID             int              `json:"roomId"`
	RecordIndex        int              `json:"recordIndex"`
	RecordOffset       int              `json:"recordOffset"`
	MirrorObject       Coordinate       `json:"mirrorObject"`
	MirrorTerrainValue byte             `json:"mirrorTerrainValue"`
	AbsorptionSquare   Coordinate       `json:"absorptionSquare"`
	EntryFormations    []EntryFormation `json:"entryFormations"`
}

// RescueReport combines map evidence with the original finale coordinates; ref:DNG-0015.
type RescueReport struct {
	MapOffset           int          `json:"mapOffset"`
	MapSize             int          `json:"mapSize"`
	MapSHA256           string       `json:"mapSha256"`
	Mirror              TileEvidence `json:"mirror"`
	LordBritishStart    Coordinate   `json:"lordBritishStart"`
	LordBritishEnd      Coordinate   `json:"lordBritishEnd"`
	PartyStart          Coordinate   `json:"partyStart"`
	PartyDestinations   []Coordinate `json:"partyDestinations"`
	BoxMoongatePosition Coordinate   `json:"boxMoongatePosition"`
}

// TileEvidence records one coordinate and its source data value.
type TileEvidence struct {
	Position Coordinate `json:"position"`
	Value    byte       `json:"value"`
}

// EndingReport records non-prose finale file evidence.
type EndingReport struct {
	Graphics []GraphicsAsset `json:"graphics"`
	TextData []TextDataAsset `json:"textData"`
}

// FileMetadata records stable identity without embedding original data.
type FileMetadata struct {
	Name   string `json:"name"`
	Size   int    `json:"size"`
	SHA256 string `json:"sha256"`
}

// GraphicsAsset records file identity and decoded archive dimensions.
type GraphicsAsset struct {
	File       FileMetadata    `json:"file"`
	EntryCount int             `json:"entryCount"`
	Entries    []GraphicsEntry `json:"requiredEntries"`
}

// GraphicsEntry is non-pixel metadata for one decoded archive entry.
type GraphicsEntry struct {
	Name    string `json:"name"`
	Index   int    `json:"index"`
	Missing bool   `json:"missing"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

// TextDataAsset records source-known record starts without original prose.
type TextDataAsset struct {
	File          FileMetadata `json:"file"`
	RecordOffsets []int        `json:"recordOffsets"`
}

// Extract validates and extracts a deterministic semantic report.
func Extract(files Files) (Report, error) {
	if err := validateFiles(files); err != nil {
		return Report{}, err
	}

	thresholds, roomIDs := scanThresholds(files.DungeonDAT)
	rooms := make([]RoomReport, 0, len(roomIDs))
	records := make(map[int][]byte, len(roomIDs))
	for _, roomID := range roomIDs {
		recordIndex := doomRoomBaseIndex + roomID
		recordOffset := recordIndex * dungeonRoomSize
		record := files.DungeonCBT[recordOffset : recordOffset+dungeonRoomSize]
		room, err := extractRoom(roomID, recordIndex, recordOffset, record)
		if err != nil {
			return Report{}, fmt.Errorf("extracting Doom room %d: %w", roomID, err)
		}
		rooms = append(rooms, room)
		records[roomID] = record
	}

	finalMirror, err := findFinalMirror(rooms, records)
	if err != nil {
		return Report{}, err
	}
	rescueMap, err := extractRescueMap(files.MiscMapsDAT)
	if err != nil {
		return Report{}, err
	}
	graphics, err := extractGraphics(files)
	if err != nil {
		return Report{}, err
	}
	endOffsets := endRecordOffsets()
	if err := validateTextRecords("END.DAT", files.EndDAT, endOffsets); err != nil {
		return Report{}, err
	}
	endMsgOffsets := endMessageRecordOffsets()
	if err := validateTextRecords("ENDMSG.DAT", files.EndMsgDAT, endMsgOffsets); err != nil {
		return Report{}, err
	}

	return Report{
		SchemaVersion: 1,
		Doom: DungeonEvidence{
			DATBlockOffset: doomBlockOffset,
			Thresholds:     thresholds,
			Rooms:          rooms,
			FinalMirror:    finalMirror,
		},
		RescueMap: rescueMap,
		Ending: EndingReport{
			Graphics: graphics,
			TextData: []TextDataAsset{
				{File: metadata("END.DAT", files.EndDAT), RecordOffsets: endOffsets},
				{
					File:          metadata("ENDMSG.DAT", files.EndMsgDAT),
					RecordOffsets: endMsgOffsets,
				},
			},
		},
	}, nil
}

// CanonicalJSON encodes the exhaustive local report printed by the default CLI.
func CanonicalJSON(report Report) ([]byte, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encoding Doom report: %w", err)
	}

	return append(data, '\n'), nil
}

func validateFiles(files Files) error {
	required := []struct {
		name string
		data []byte
	}{
		{"DUNGEON.DAT", files.DungeonDAT},
		{"DUNGEON.CBT", files.DungeonCBT},
		{"MISCMAPS.DAT", files.MiscMapsDAT},
		{"END1.16", files.End1},
		{"END2.16", files.End2},
		{"ENDSC.16", files.EndScreen},
		{"TEXT.16", files.Text},
		{"PROPORT.PCS", files.Proport},
		{"END.DAT", files.EndDAT},
		{"ENDMSG.DAT", files.EndMsgDAT},
	}
	for _, asset := range required {
		if len(asset.data) == 0 {
			return fmt.Errorf("%w: %s", errRequiredAssetMissing, asset.name)
		}
	}

	minimums := []struct {
		name string
		data []byte
		min  int
	}{
		{"DUNGEON.DAT", files.DungeonDAT, dungeonMinimumSize},
		{"DUNGEON.CBT", files.DungeonCBT, dungeonCBTMinimumSize},
		{"MISCMAPS.DAT", files.MiscMapsDAT, miscMapsMinimumSize},
	}
	for _, asset := range minimums {
		if len(asset.data) < asset.min {
			return fmt.Errorf(
				"%w: %s is %d bytes, need at least %d",
				errStructuralMinimum, asset.name, len(asset.data), asset.min,
			)
		}
	}

	return nil
}

func scanThresholds(data []byte) ([]Threshold, []int) {
	thresholds := make([]Threshold, 0)
	seen := make(map[int]struct{})
	roomIDs := make([]int, 0)
	for floor := range dungeonDimension {
		for row := range dungeonDimension {
			for column := range dungeonDimension {
				offset := doomBlockOffset + floor*64 + row*dungeonDimension + column
				raw := data[offset]
				if raw&0xf0 != 0xa0 && raw&0xf0 != 0xf0 {
					continue
				}
				roomID := int(raw & roomIDMask)
				recordIndex := doomRoomBaseIndex + roomID
				thresholds = append(thresholds, Threshold{
					RawValue: raw, Floor: floor, X: column, Y: row, DATOffset: offset,
					RoomID: roomID, CBTRecordIndex: recordIndex, CBTRecordOffset: recordIndex * dungeonRoomSize,
				})
				if _, ok := seen[roomID]; !ok {
					seen[roomID] = struct{}{}
					roomIDs = append(roomIDs, roomID)
				}
			}
		}
	}
	sort.Ints(roomIDs)

	return thresholds, roomIDs
}

func extractRoom(roomID, recordIndex, recordOffset int, record []byte) (RoomReport, error) {
	formations, err := extractEntryFormations(record)
	if err != nil {
		return RoomReport{}, err
	}

	entities := make([]FixedEntity, 0)
	for slot := range entityCount {
		descriptor := record[5*combatRowSize+11+slot]
		if descriptor == 0 {
			continue
		}
		position := Coordinate{
			X: int(record[6*combatRowSize+11+slot]),
			Y: int(record[7*combatRowSize+11+slot]),
		}
		if !inCombatBounds(position) {
			return RoomReport{}, fmt.Errorf(
				"%w: active entity slot %d descriptor %#02x has out-of-range position (%d,%d)",
				errMalformedEntityCoordinates, slot, descriptor, position.X, position.Y,
			)
		}
		entities = append(entities, FixedEntity{Slot: slot, Descriptor: descriptor, Position: position})
	}

	triggers := make([]TriggerSlot, 0, triggerCount)
	for slot := range triggerCount {
		trigger := Coordinate{
			X: int(record[8*combatRowSize+11+slot]),
			Y: int(record[8*combatRowSize+19+slot]),
		}
		destinationRows := []int{9, 10}
		destinations := make([]TriggerTarget, 0, len(destinationRows))
		for _, row := range destinationRows {
			position := Coordinate{
				X: int(record[row*combatRowSize+11+slot]),
				Y: int(record[row*combatRowSize+19+slot]),
			}
			destinations = append(destinations, TriggerTarget{Position: position, InBounds: inCombatBounds(position)})
		}
		triggers = append(triggers, TriggerSlot{
			Slot: slot, ReplacementTile: record[11+slot], Trigger: trigger,
			Disabled: trigger.X == 0xff && trigger.Y == 0xff, Destinations: destinations,
		})
	}

	return RoomReport{
		RoomID: roomID, RecordIndex: recordIndex, RecordOffset: recordOffset,
		RecordSHA256: hash(record), EntryFormations: formations, Entities: entities, Triggers: triggers,
	}, nil
}

func extractEntryFormations(record []byte) ([]EntryFormation, error) {
	directions := []string{"west", "east", "north", "south"}
	formations := make([]EntryFormation, 0, len(directions))
	for direction, name := range directions {
		row := direction + 1
		positions := make([]Coordinate, 0, partySize)
		for partyMember := range partySize {
			position := Coordinate{
				X: int(record[row*combatRowSize+11+partyMember]),
				Y: int(record[row*combatRowSize+17+partyMember]),
			}
			if !inCombatBounds(position) {
				return nil, fmt.Errorf(
					"%w: row %d party slot %d has out-of-range position (%d,%d)",
					errMalformedPartyCoordinates, row, partyMember, position.X, position.Y,
				)
			}
			positions = append(positions, position)
		}
		formations = append(formations, EntryFormation{Direction: name, Row: row, Positions: positions})
	}

	return formations, nil
}

func findFinalMirror(rooms []RoomReport, records map[int][]byte) (FinalMirrorRoom, error) {
	type candidate struct {
		room   RoomReport
		entity FixedEntity
	}
	candidates := make([]candidate, 0, 1)
	for _, room := range rooms {
		for _, entity := range room.Entities {
			if entity.Descriptor&0xfc == inMirrorFamily && entity.Position.Y == mirrorObjectRow {
				candidates = append(candidates, candidate{room: room, entity: entity})
			}
		}
	}
	if len(candidates) != 1 {
		return FinalMirrorRoom{}, fmt.Errorf(
			"%w: found %d referenced IN_MIRROR row-1 candidates, want 1",
			errMirrorCandidateCount, len(candidates),
		)
	}

	match := candidates[0]
	record := records[match.room.RoomID]
	terrain := record[match.entity.Position.Y*combatRowSize+match.entity.Position.X]
	if terrain != mirrorTerrain {
		return FinalMirrorRoom{}, fmt.Errorf(
			"%w: room %d terrain at mirror object (%d,%d) is %#02x, want %#02x",
			errMirrorTerrainMismatch, match.room.RoomID,
			match.entity.Position.X, match.entity.Position.Y, terrain, mirrorTerrain,
		)
	}

	return FinalMirrorRoom{
		RoomID: match.room.RoomID, RecordIndex: match.room.RecordIndex, RecordOffset: match.room.RecordOffset,
		MirrorObject: match.entity.Position, MirrorTerrainValue: terrain,
		AbsorptionSquare: Coordinate{X: match.entity.Position.X, Y: absorptionRow},
		EntryFormations:  match.room.EntryFormations,
	}, nil
}

//nolint:mnd // Exact finale coordinates are the evidence being reported; ref:DNG-0015.
func extractRescueMap(data []byte) (RescueReport, error) {
	rescueMap := data[rescueMapOffset : rescueMapOffset+rescueMapSize]
	mirrorPosition := Coordinate{X: 5, Y: 9}
	mirrorValue := rescueMap[mirrorPosition.Y*16+mirrorPosition.X]
	if mirrorValue != mirrorTerrain {
		return RescueReport{}, fmt.Errorf(
			"%w: value at (%d,%d) is %#02x, want %#02x",
			errRescueMirrorMismatch, mirrorPosition.X, mirrorPosition.Y, mirrorValue, mirrorTerrain,
		)
	}
	for row := range combatMapSize {
		for column := combatMapSize; column < 16; column++ {
			if rescueMap[row*16+column] != 0 {
				return RescueReport{}, fmt.Errorf(
					"%w: row %d column %d is %#02x",
					errRescuePadding, row, column, rescueMap[row*16+column],
				)
			}
		}
	}

	return RescueReport{
		MapOffset: rescueMapOffset,
		MapSize:   rescueMapSize,
		MapSHA256: hash(rescueMap),
		Mirror: TileEvidence{
			Position: mirrorPosition,
			Value:    mirrorValue,
		},
		LordBritishStart: Coordinate{X: 5, Y: 8},
		LordBritishEnd:   Coordinate{X: 5, Y: 3},
		PartyStart:       Coordinate{X: 5, Y: 9},
		PartyDestinations: []Coordinate{
			{X: 5, Y: 5}, {X: 4, Y: 6}, {X: 6, Y: 6},
			{X: 3, Y: 7}, {X: 5, Y: 7}, {X: 7, Y: 7},
		},
		BoxMoongatePosition: Coordinate{X: 5, Y: 4},
	}, nil
}

type graphicsAssetSpec struct {
	name            string
	data            []byte
	expectedEntries int
	requiredEntries []int
	font            bool
}

func extractGraphics(files Files) ([]GraphicsAsset, error) {
	assets := []graphicsAssetSpec{
		{
			name: "END1.16", data: files.End1, expectedEntries: endingShapeEntryCount,
			requiredEntries: []int{0, 1, 2}, font: false,
		},
		{
			name: "END2.16", data: files.End2, expectedEntries: endingShapeEntryCount,
			requiredEntries: []int{0, 1, 2}, font: false,
		},
		{name: "ENDSC.16", data: files.EndScreen, expectedEntries: 1, requiredEntries: []int{0}, font: false},
		{
			name: "TEXT.16", data: files.Text, expectedEntries: textShapeEntryCount,
			requiredEntries: []int{0, 4, 5}, font: false,
		},
		{
			name: "PROPORT.PCS", data: files.Proport, expectedEntries: proportEntryCount,
			requiredEntries: []int{}, font: true,
		},
	}
	reports := make([]GraphicsAsset, 0, len(assets))
	for _, asset := range assets {
		report, err := extractGraphicsAsset(asset)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func extractGraphicsAsset(asset graphicsAssetSpec) (GraphicsAsset, error) {
	spec, ok := legacy.FindSpec(asset.name)
	if !ok {
		return GraphicsAsset{}, fmt.Errorf("decoding %s: %w", asset.name, errGraphicsSpecUnavailable)
	}
	archive, err := legacy.Decode(spec, asset.data)
	if err != nil {
		return GraphicsAsset{}, fmt.Errorf("decoding required asset %s: %w", asset.name, err)
	}
	if err := validateGraphicsArchive(asset, archive); err != nil {
		return GraphicsAsset{}, err
	}
	entries, err := requiredGraphicsEntries(asset, archive)
	if err != nil {
		return GraphicsAsset{}, err
	}

	return GraphicsAsset{
		File: metadata(asset.name, asset.data), EntryCount: len(archive.Entries), Entries: entries,
	}, nil
}

func validateGraphicsArchive(asset graphicsAssetSpec, archive legacy.Archive) error {
	if len(archive.Entries) != asset.expectedEntries {
		return fmt.Errorf(
			"%w: %s has %d entries, want %d",
			errGraphicsEntryCount, asset.name, len(archive.Entries), asset.expectedEntries,
		)
	}
	if asset.font {
		for _, entry := range archive.Entries {
			unexpectedZeroWidth := entry.Width == 0 && entry.Index != 0
			if entry.Missing || unexpectedZeroWidth || entry.Height != proportGlyphHeight ||
				(entry.Width > 0 && entry.Image == nil) {
				return graphicsEntryError(asset.name, entry)
			}
		}
	}

	return nil
}

func requiredGraphicsEntries(asset graphicsAssetSpec, archive legacy.Archive) ([]GraphicsEntry, error) {
	entries := make([]GraphicsEntry, 0, len(asset.requiredEntries))
	for _, entryIndex := range asset.requiredEntries {
		entry := archive.Entries[entryIndex]
		if entry.Missing || entry.Width <= 0 || entry.Height <= 0 || entry.Image == nil {
			return nil, graphicsEntryError(asset.name, entry)
		}
		entries = append(entries, GraphicsEntry{
			Name: entry.Name, Index: entry.Index, Missing: entry.Missing,
			Width: entry.Width, Height: entry.Height,
		})
	}

	return entries, nil
}

func graphicsEntryError(name string, entry legacy.Entry) error {
	return fmt.Errorf(
		"%w: %s entry %d has missing=%t dimensions=%dx%d",
		errGraphicsEntryUnusable, name, entry.Index, entry.Missing, entry.Width, entry.Height,
	)
}

func validateTextRecords(name string, data []byte, offsets []int) error {
	for index, offset := range offsets {
		if offset < 0 || offset >= len(data) {
			return fmt.Errorf(
				"%w: %s record %d starts at %d in %d-byte file",
				errTextRecordOffset, name, index, offset, len(data),
			)
		}
		end := len(data)
		if index+1 < len(offsets) {
			end = offsets[index+1]
		}
		if end <= offset || end > len(data) || bytes.IndexByte(data[offset:end], 0) < 0 {
			return fmt.Errorf(
				"%w: %s record %d range [%d,%d)",
				errTextRecordTermination, name, index, offset, end,
			)
		}
	}

	return nil
}

func endRecordOffsets() []int {
	return []int{0, 424, 956, 1530, 2280, 2932}
}

func endMessageRecordOffsets() []int {
	return []int{0, 33, 73, 171, 211, 296, 359, 457, 529, 587, 725}
}

func metadata(name string, data []byte) FileMetadata {
	return FileMetadata{Name: name, Size: len(data), SHA256: hash(data)}
}

func hash(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

func inCombatBounds(position Coordinate) bool {
	return position.X >= 0 && position.X < combatMapSize && position.Y >= 0 && position.Y < combatMapSize
}
