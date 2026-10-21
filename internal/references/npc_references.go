package references

import (
	"fmt"
	"io"
	"os"

	"github.com/bradhannah/UltimaVRedux/internal/config"
)

const (
	npcsPerTown     = 32
	townsPerNPCFile = 8

	startingNpcTypeOffset         = sizeOfNPCSchedule * npcsPerTown
	startingNpcDialogNumberOffset = startingNpcTypeOffset + npcsPerTown

	singleTownSize = (sizeOfNPCSchedule * npcsPerTown) + (npcsPerTown * 2)
)

// NPCReferences holds every NPC slot of the original game, npcsPerTown slots for each location.
type NPCReferences struct {
	Npcs []NPCReference `json:"npcs" yaml:"npcs"`
}

// NewNPCReferences loads NPC definitions from the configured original data files.
func NewNPCReferences(config *config.UltimaVConfiguration) (*NPCReferences, error) {
	allNpcs := &NPCReferences{}

	npcFiles := config.GetAllNpcFilePaths()

	for i, filePath := range npcFiles {
		npcs, err := getNPCsFromFile(filePath, i*townsPerNPCFile)
		if err != nil {
			return nil, fmt.Errorf("loading NPC references from %s: %w", filePath, err)
		}
		allNpcs.Npcs = append(allNpcs.Npcs, npcs...)
	}

	return allNpcs, nil
}

func getNPCsFromFile(path string, locationOffset int) ([]NPCReference, error) {
	npcRaw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading NPC data: %w", err)
	}
	const expectedSize = singleTownSize * townsPerNPCFile
	if len(npcRaw) < expectedSize {
		return nil, fmt.Errorf("reading NPC data: expected at least %d bytes, got %d: %w", expectedSize, len(npcRaw), io.ErrUnexpectedEOF)
	}

	npcs := make([]NPCReference, 0)
	for townIndex := range townsPerNPCFile {
		townOffset := singleTownSize * townIndex
		townRawData := npcRaw[townOffset : townOffset+singleTownSize]
		for npcIndex := range npcsPerTown {
			npc := NPCReference{}
			npc.Location = Location(locationOffset + townIndex + 1)
			npc.Schedule = CreateNPCSchedule(townRawData[npcIndex*sizeOfNPCSchedule : (npcIndex*sizeOfNPCSchedule)+sizeOfNPCSchedule])
			npc.NpcType = NPCType(townRawData[startingNpcTypeOffset+npcIndex])
			npc.DialogNumber = townRawData[startingNpcDialogNumberOffset+npcIndex]

			npcs = append(npcs, npc)
		}
	}

	return npcs, nil
}

func (n *NPCReferences) getNPCIndexesByLocation(location Location) (int, int) {
	adjLocationIndex := int(location) - 1

	return adjLocationIndex * npcsPerTown, adjLocationIndex*npcsPerTown + npcsPerTown
}

// GetNPCReferencesByLocation returns the NPC slots of one location.
func (n *NPCReferences) GetNPCReferencesByLocation(location Location) *[]NPCReference {
	startIndex, endIndex := n.getNPCIndexesByLocation(location)
	npcs := n.Npcs[startIndex:endIndex]

	return &npcs
}
