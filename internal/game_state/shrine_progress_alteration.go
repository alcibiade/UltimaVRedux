package game_state

import (
	"errors"
	"fmt"
	"hash"
)

// ShrineProgressState is a valid combination of the original completed and active quest flags.
type ShrineProgressState string

const (
	// ShrineProgressNotAssigned represents cleared completed and active flags.
	ShrineProgressNotAssigned ShrineProgressState = "not_assigned"
	// ShrineProgressAssigned represents a cleared completed flag and set active flag.
	ShrineProgressAssigned ShrineProgressState = "assigned"
	// ShrineProgressReadyToComplete represents set completed and active flags.
	ShrineProgressReadyToComplete ShrineProgressState = "ready_to_complete"
	// ShrineProgressOffering represents a set completed flag and cleared active flag.
	ShrineProgressOffering ShrineProgressState = "offering"
)

var errInvalidShrineProgressAlteration = errors.New("invalid shrine progress alteration")

var (
	// ErrAllShrineQuestStateStale means shrine quest flags changed after review.
	ErrAllShrineQuestStateStale = errors.New("shrine quest state changed after review")
	// ErrAllShrineQuestStateInvalid means a bulk shrine normalization is unsupported or unsafe.
	ErrAllShrineQuestStateInvalid = errors.New("invalid all-shrine quest state alteration")
)

// AllShrineQuestState identifies an absolute normalized target for every shrine quest.
type AllShrineQuestState string

const (
	// AllShrineQuestNotAssigned clears both quest flags for every shrine.
	AllShrineQuestNotAssigned AllShrineQuestState = "not_assigned"
	// AllShrineQuestCompleted sets completed and clears active for every shrine.
	AllShrineQuestCompleted AllShrineQuestState = "completed"
)

// ShrineProgressSnapshot is the mutable progression state for one shrine.
type ShrineProgressSnapshot struct {
	Index      int
	QuestState ShrineProgressState
	Completed  bool
	Active     bool
	Ruined     bool
}

// ShrineProgressAlteration describes an atomic shrine-state edit without gameplay rewards.
type ShrineProgressAlteration struct {
	Index      int
	QuestState *ShrineProgressState
	Ruined     *bool
}

// ShrineProgressAlterationResult contains the state surrounding an applied shrine edit.
type ShrineProgressAlterationResult struct {
	Before ShrineProgressSnapshot
	After  ShrineProgressSnapshot
}

// AllShrineQuestStateReview is a detached eight-shrine review with a stale-state fingerprint.
type AllShrineQuestStateReview struct {
	Fingerprint string
	Shrines     []ShrineProgressSnapshot
}

// AllShrineQuestStateChange reports one shrine normalized by a bulk alteration.
type AllShrineQuestStateChange struct {
	Index  int
	Before ShrineProgressSnapshot
	After  ShrineProgressSnapshot
}

// AllShrineQuestStateResult reports exact changes from an atomic eight-shrine normalization.
type AllShrineQuestStateResult struct {
	BeforeFingerprint string
	AfterFingerprint  string
	Changes           []AllShrineQuestStateChange
}

// ShrineProgress returns normalized mutable state for one shrine.
func (gs *GameState) ShrineProgress(index int) ShrineProgressSnapshot {
	return ShrineProgressSnapshot{
		Index:      index,
		QuestState: gs.shrineProgressState(index),
		Completed:  gs.IsShrineQuestCompleted(index),
		Active:     gs.IsShrineQuestActive(index),
		Ruined:     gs.IsShrineRuined(index),
	}
}

// AlterShrineProgress atomically updates shrine flags without replaying shrine rewards or presentation.
func (gs *GameState) AlterShrineProgress(input ShrineProgressAlteration) (ShrineProgressAlterationResult, error) {
	if input.Index < 0 || input.Index >= TotalShrines {
		return ShrineProgressAlterationResult{}, fmt.Errorf(
			"%w: shrine index %d is outside 0-%d",
			errInvalidShrineProgressAlteration,
			input.Index,
			TotalShrines-1,
		)
	}
	if input.QuestState == nil && input.Ruined == nil {
		return ShrineProgressAlterationResult{}, fmt.Errorf("%w: no changes supplied", errInvalidShrineProgressAlteration)
	}
	if input.QuestState != nil && !validShrineProgressState(*input.QuestState) {
		return ShrineProgressAlterationResult{}, fmt.Errorf(
			"%w: unknown quest state %q",
			errInvalidShrineProgressAlteration,
			*input.QuestState,
		)
	}

	before := gs.ShrineProgress(input.Index)
	if input.QuestState != nil {
		completed, active := shrineProgressFlags(*input.QuestState)
		gs.SetShrineQuestCompleted(input.Index, completed)
		gs.SetShrineQuestActive(input.Index, active)
	}
	if input.Ruined != nil {
		gs.SetShrineRuined(input.Index, *input.Ruined)
	}

	return ShrineProgressAlterationResult{
		Before: before,
		After:  gs.ShrineProgress(input.Index),
	}, nil
}

// ReviewAllShrineQuestState returns all eight raw quest-flag combinations for destructive review.
func (gs *GameState) ReviewAllShrineQuestState() AllShrineQuestStateReview {
	shrines := make([]ShrineProgressSnapshot, 0, TotalShrines)
	for index := range TotalShrines {
		shrines = append(shrines, gs.ShrineProgress(index))
	}

	return AllShrineQuestStateReview{
		Fingerprint: allShrineQuestStateFingerprint(shrines),
		Shrines:     shrines,
	}
}

// SetAllShrineQuestState validates review state and atomically normalizes all eight shrine quests.
func (gs *GameState) SetAllShrineQuestState(
	target AllShrineQuestState,
	expectedFingerprint string,
) (AllShrineQuestStateResult, error) {
	completed, err := allShrineQuestCompletion(target)
	if err != nil {
		return AllShrineQuestStateResult{}, err
	}
	if err := gs.validateReviewedMutationContext("all-shrine quest alteration"); err != nil {
		return AllShrineQuestStateResult{}, fmt.Errorf("%w: %w", ErrAllShrineQuestStateInvalid, err)
	}
	review := gs.ReviewAllShrineQuestState()
	if expectedFingerprint == "" || expectedFingerprint != review.Fingerprint {
		return AllShrineQuestStateResult{}, ErrAllShrineQuestStateStale
	}

	changes := make([]AllShrineQuestStateChange, 0, TotalShrines)
	for index, before := range review.Shrines {
		if before.Completed == completed && !before.Active {
			continue
		}
		gs.SetShrineQuestCompleted(index, completed)
		gs.SetShrineQuestActive(index, false)
		changes = append(changes, AllShrineQuestStateChange{
			Index: index, Before: before, After: gs.ShrineProgress(index),
		})
	}
	after := gs.ReviewAllShrineQuestState()

	return AllShrineQuestStateResult{
		BeforeFingerprint: review.Fingerprint,
		AfterFingerprint:  after.Fingerprint,
		Changes:           changes,
	}, nil
}

func allShrineQuestCompletion(target AllShrineQuestState) (bool, error) {
	switch target {
	case AllShrineQuestNotAssigned:
		return false, nil
	case AllShrineQuestCompleted:
		return true, nil
	default:
		return false, fmt.Errorf("%w: unknown target %q", ErrAllShrineQuestStateInvalid, target)
	}
}

func allShrineQuestStateFingerprint(shrines []ShrineProgressSnapshot) string {
	return reviewedMutationFingerprint(func(digest hash.Hash) {
		for _, shrine := range shrines {
			writeFingerprintf(digest, "%d:%t:%t;", shrine.Index, shrine.Completed, shrine.Active)
		}
	})
}

func (gs *GameState) shrineProgressState(index int) ShrineProgressState {
	if !gs.IsShrineQuestCompleted(index) {
		if gs.IsShrineQuestActive(index) {
			return ShrineProgressAssigned
		}

		return ShrineProgressNotAssigned
	}
	if gs.IsShrineQuestActive(index) {
		return ShrineProgressReadyToComplete
	}

	return ShrineProgressOffering
}

func validShrineProgressState(state ShrineProgressState) bool {
	return state == ShrineProgressNotAssigned || state == ShrineProgressAssigned ||
		state == ShrineProgressReadyToComplete || state == ShrineProgressOffering
}

func shrineProgressFlags(state ShrineProgressState) (bool, bool) {
	switch state {
	case ShrineProgressNotAssigned:
		return false, false
	case ShrineProgressAssigned:
		return false, true
	case ShrineProgressReadyToComplete:
		return true, true
	case ShrineProgressOffering:
		return true, false
	default:
		panic("validated shrine progress state has no flag mapping")
	}
}
