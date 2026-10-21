package references

import (
	"fmt"
	"io"

	"github.com/bradhannah/UltimaVRedux/internal/config"
)

const (
	startDockXOffset = 0x4D86
	startDockYOffset = 0x4D8A
)

type DockReference struct {
	Location Location `json:"location" yaml:"location"`
	Position Position `json:"position" yaml:"position"`
}

type DockReferences struct {
	Docks []DockReference `json:"docks" yaml:"docks"`
}

func GetListOfAllLocationsWithDocks() []Location {
	return []Location{Jhelom, Minoc, EastBritanny, BuccaneersDen}
}

// NewDocks loads dock coordinates from the configured DATA.OVL bytes.
func NewDocks(gameConfig *config.UltimaVConfiguration) (*DockReferences, error) {
	requiredLength := startDockYOffset + len(GetListOfAllLocationsWithDocks())
	if len(gameConfig.RawDataOvl) < requiredLength {
		return nil, fmt.Errorf("reading dock coordinates: expected at least %d DATA.OVL bytes, got %d: %w",
			requiredLength, len(gameConfig.RawDataOvl), io.ErrUnexpectedEOF)
	}

	dockRefs := &DockReferences{}
	dockLocations := GetListOfAllLocationsWithDocks()

	for i, loc := range dockLocations {
		dock := DockReference{
			Location: loc,
			Position: Position{
				X: Coordinate(gameConfig.RawDataOvl[startDockXOffset+i]),
				Y: Coordinate(gameConfig.RawDataOvl[startDockYOffset+i]),
			},
		}
		dockRefs.Docks = append(dockRefs.Docks, dock)
	}

	return dockRefs, nil
}

// GetDockPositionChecked returns a dock position and reports whether the
// location is present, avoiding ambiguity with the valid zero position.
func (d *DockReferences) GetDockPositionChecked(dockLocation Location) (Position, bool) {
	if d == nil {
		return Position{}, false
	}

	for _, dock := range d.Docks {
		if dock.Location == dockLocation {
			return dock.Position, true
		}
	}

	return Position{}, false
}
