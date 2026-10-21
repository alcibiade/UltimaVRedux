package game_state

import (
	"errors"
	"fmt"
	"sort"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

type dungeonDiscoveryKind string

const (
	dungeonDiscoveryHiddenDoor dungeonDiscoveryKind = "hiddenDoor"
	dungeonDiscoveryHiddenPit  dungeonDiscoveryKind = "hiddenPit"
)

type dungeonDiscoveryKey struct {
	location references.Location
	floor    references.FloorNumber
	position references.Position
}

type dungeonDiscovery struct {
	dungeonDiscoveryKey
	kind dungeonDiscoveryKind
}

func (g *GameState) persistentDungeonDiscoveriesEnabled() bool {
	return g.GameConfig != nil && g.GameConfig.GameplayConfiguration != nil &&
		g.GameConfig.GameplayConfiguration.PersistHiddenDungeonDiscoveries
}

func (g *GameState) recordDungeonDiscovery(
	kind dungeonDiscoveryKind,
	floor references.FloorNumber,
	position references.Position,
) {
	if !g.persistentDungeonDiscoveriesEnabled() {
		return
	}
	location := g.MapState.PlayerLocation.Location
	if !g.isImmutableDungeonDiscoverySource(location, kind, floor, position) {
		return
	}
	if g.dungeonDiscoveries == nil {
		g.dungeonDiscoveries = make(map[dungeonDiscoveryKey]dungeonDiscoveryKind)
	}
	key := dungeonDiscoveryKey{
		location: location,
		floor:    floor,
		position: position,
	}
	if previous, exists := g.dungeonDiscoveries[key]; exists && previous != kind {
		panic("dungeon discovery kind changed for one coordinate")
	}
	g.dungeonDiscoveries[key] = kind
}

func (g *GameState) isImmutableDungeonDiscoverySource(
	location references.Location,
	kind dungeonDiscoveryKind,
	floor references.FloorNumber,
	position references.Position,
) bool {
	if g.GameReferences == nil || g.GameReferences.LocationReferences == nil {
		return false
	}
	reference := g.GameReferences.LocationReferences.GetLocationReference(location)
	if reference == nil {
		return false
	}
	source, ok := reference.GetDungeonCell(floor, position)
	if !ok {
		return false
	}

	switch kind {
	case dungeonDiscoveryHiddenDoor:
		return source.Kind() == references.DungeonHiddenDoor
	case dungeonDiscoveryHiddenPit:
		return isDungeonHiddenPit(source)
	default:
		panic("unsupported dungeon discovery kind")
	}
}

func (g *GameState) sortedDungeonDiscoveries() []dungeonDiscovery {
	discoveries := make([]dungeonDiscovery, 0, len(g.dungeonDiscoveries))
	for key, kind := range g.dungeonDiscoveries {
		discoveries = append(discoveries, dungeonDiscovery{dungeonDiscoveryKey: key, kind: kind})
	}
	sort.Slice(discoveries, func(i, j int) bool {
		left, right := discoveries[i], discoveries[j]
		if left.location != right.location {
			return left.location < right.location
		}
		if left.floor != right.floor {
			return left.floor < right.floor
		}
		if left.position.Y != right.position.Y {
			return left.position.Y < right.position.Y
		}
		if left.position.X != right.position.X {
			return left.position.X < right.position.X
		}

		return left.kind < right.kind
	})

	return discoveries
}

func (g *GameState) applyPersistentDungeonDiscoveries(
	location references.Location,
	cells *references.DungeonCells,
) error {
	if !g.persistentDungeonDiscoveriesEnabled() {
		return nil
	}
	immutableCells := *cells
	for _, discovery := range g.sortedDungeonDiscoveries() {
		if discovery.location != location {
			continue
		}
		immutableSource, ok := immutableCells.Cell(discovery.floor, discovery.position)
		if !ok {
			return errors.New("persistent dungeon discovery is out of bounds")
		}
		source, ok := cells.Cell(discovery.floor, discovery.position)
		if !ok {
			return errors.New("persistent dungeon discovery is out of bounds")
		}
		switch discovery.kind {
		case dungeonDiscoveryHiddenDoor:
			if immutableSource.Kind() != references.DungeonHiddenDoor ||
				source.Kind() != references.DungeonHiddenDoor {
				return errors.New("persistent hidden-door discovery does not match source cell")
			}
			cells.SetCell(
				discovery.floor,
				discovery.position,
				references.DungeonCell(0xe0|(source.Raw()&0x08)),
			)
		case dungeonDiscoveryHiddenPit:
			if !isDungeonHiddenPit(immutableSource) || !isDungeonHiddenPit(source) {
				return errors.New("persistent hidden-pit discovery does not match source cell")
			}
			cells.SetCell(
				discovery.floor,
				discovery.position,
				references.DungeonCell(source.Raw()&0xf8),
			)
			if discovery.floor < references.DungeonFloorCount-1 {
				below, _ := cells.Cell(discovery.floor+1, discovery.position)
				// REDUX DEVIATION: Original visits discard map mutations on re-entry. Replay
				// applies Search semantics to pristine 0x61 sources and fall semantics to
				// pristine 0x69 sources while retaining the evolving chained-pit state.
				// See ref:DNG-0014.
				if immutableSource.Raw() == 0x61 || below.Raw() < 0x90 {
					cells.SetCell(
						discovery.floor+1,
						discovery.position,
						references.DungeonCell(below.Raw()|0x08),
					)
				}
			}
		default:
			return fmt.Errorf("unsupported persistent dungeon discovery kind %q", discovery.kind)
		}
	}

	return nil
}
