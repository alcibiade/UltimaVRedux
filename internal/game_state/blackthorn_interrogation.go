package game_state

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/minimap"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	blackthornPrisonMapIndex = 0
	blackthornAnswerCount    = 4
	blackthornTortureMinutes = 2
)

var (
	errBlackthornInterrogationInvalid = errors.New("invalid Blackthorn interrogation")
	errBlackthornInterrogationStale   = errors.New("blackthorn interrogation state changed")
)

// Blackthorn source dialogue is shared by the authoritative action and UI workflow.
const (
	BlackthornSubduedMessage       = "Thou art subdued and blindfolded!"
	BlackthornDraggedMessage       = "Strong guards drag thee away!"
	BlackthornChainedMessage       = "Thou hast been chained and manacled!"
	BlackthornFootstepsMessage     = "Footsteps!"
	BlackthornQuestMessage         = "\"Wait!\"\n\n\"Since I myself seek Avatarhood as once did thee, mayhap thou couldst aid me in my Quest by answering a question.\""
	BlackthornTortureMessage       = "\"Make not the mistake of laughing at me, simple one!\""
	BlackthornLoneReleaseMessage   = "\"A child would catch thee in thy lies, foolish one! To the dungeon with thee!\""
	BlackthornLoneCorrectMessage   = "\"I sense Truth in thee.  This will be rewarded with thy life!\""
	BlackthornMercifulDeathMessage = "\"I thank thee, my friend! As a token of my esteem for thine honesty, I will grant thy companion a merciful death!\""
	BlackthornPendulumMessage      = "With a wave of Blackthorn's hand, the pendulum blade falls!"
	BlackthornUnfairMessage        = "\"None call me unfair! I have shown thee every kindness in the world and thou hast lied to me! Thy friend hath paid for thy treachery!\""
)

type blackthornInterrogationSession struct {
	review  BlackthornInterrogationReview
	attempt int
}

// BlackthornInterrogationReview captures the source-selected shrine and party state.
type BlackthornInterrogationReview struct {
	ShrineIndex  int
	VirtueName   string
	Mantra       string
	LivingCount  int
	NeedsAnswer  bool
	AvatarName   string
	AvatarFemale bool
}

// BlackthornAnswerOutcome reports one authoritative answer resolution.
type BlackthornAnswerOutcome struct {
	Attempt        int
	Correct        bool
	Resolved       bool
	StartedTorture bool
	ElapsedMinutes int
	MartyrName     string
	LoneLiving     bool
}

// BeginBlackthornInterrogation selects the first open shrine and installs the
// original prison mini-map when a question remains.
func (g *GameState) BeginBlackthornInterrogation() (BlackthornInterrogationReview, error) {
	return g.beginBlackthornInterrogation(nil)
}

func (g *GameState) beginBlackthornInterrogation(
	preparedMap *minimap.MiscMap,
) (BlackthornInterrogationReview, error) {
	if g.blackthornSession != nil || g.MapState.PlayerLocation.Location != references.PalaceOfBlackthorn {
		return BlackthornInterrogationReview{}, errBlackthornInterrogationInvalid
	}
	livingCount := g.PartyState.GetLivingPartyCount()
	if livingCount == 0 {
		return BlackthornInterrogationReview{ShrineIndex: -1}, nil
	}

	shrineIndex := -1
	for index := range TotalShrines {
		if !g.IsShrineRuined(index) {
			shrineIndex = index

			break
		}
	}
	avatar := g.PartyState.GetAvatar()
	review := BlackthornInterrogationReview{
		ShrineIndex: shrineIndex, LivingCount: livingCount, NeedsAnswer: shrineIndex >= 0,
		AvatarName: avatar.GetNameAsString(), AvatarFemale: avatar.Gender == party_state.Female,
	}
	if shrineIndex >= 0 {
		data := GetShrineData(shrineIndex)
		review.VirtueName, review.Mantra = data.VirtueName, data.Mantra
		if preparedMap == nil {
			decoded, err := minimap.LoadMiscMap(g.GameConfig.SavedConfigData.DataFilePath, blackthornPrisonMapIndex)
			if err != nil {
				return BlackthornInterrogationReview{}, fmt.Errorf("loading Blackthorn prison map: %w", err)
			}
			preparedMap = &decoded
		}
		g.transitionToBlackthornPrison(*preparedMap)
	}
	g.blackthornSession = &blackthornInterrogationSession{review: review}

	return review, nil
}

func (g *GameState) transitionToBlackthornPrison(decoded minimap.MiscMap) {
	var tiles [references.XCombatMapTiles][references.YCombatMapTiles]byte
	for x := range minimap.MiscMapSize {
		for y := range minimap.MiscMapSize {
			tiles[x][y] = decoded.Tiles[x][y]
		}
	}
	g.MapState.LayeredMaps.ResetAndCreateMiniMapFromData(
		&tiles,
		g.GameReferences.TileReferences,
		g.MapState.XTilesVisibleOnGameScreen,
		g.MapState.YTilesVisibleOnGameScreen,
	)
	g.MapState.PlayerLocation = references.PlayerLocation{
		Location: references.BlackthornPrison,
		Floor:    0,
		Position: references.Position{X: 5, Y: 5},
	}
	g.CurrentNPCAIController = nil
}

// BlackthornQuestion returns the exact source question for one answer attempt.
func BlackthornQuestion(attempt int, virtue string) string {
	switch attempt {
	case 0:
		return "What is the Mantra of the Mystic Shrine of " + virtue + "?\""
	case 1:
		return "Now tell me, what is the Mantra of " + virtue + "?\""
	case 2:
		return "Resistance is futile! Thou must yield the truth unto me! Tell me, what is the Mantra of " + virtue + "?\""
	case 3:
		return "My patience with thee has worn away! SPEAK UNTO ME THE MANTRA, NOW!"
	default:
		panic("BlackthornQuestion attempt out of range")
	}
}

// SubmitBlackthornAnswer applies one answer, including torture time and martyrdom.
func (g *GameState) SubmitBlackthornAnswer(
	review BlackthornInterrogationReview,
	attempt int,
	response string,
) (BlackthornAnswerOutcome, error) {
	session := g.blackthornSession
	if session == nil || !session.review.NeedsAnswer {
		return BlackthornAnswerOutcome{}, errBlackthornInterrogationInvalid
	}
	if session.review != review || session.attempt != attempt {
		return BlackthornAnswerOutcome{}, errBlackthornInterrogationStale
	}

	outcome := BlackthornAnswerOutcome{Attempt: attempt}
	outcome.Correct = strings.Contains(strings.ToLower(response), strings.ToLower(review.Mantra))
	if outcome.Correct {
		g.SetShrineRuined(review.ShrineIndex, true)
		g.AdjustPartyKarma(-5)
		outcome.Resolved = true
		if review.LivingCount > 1 {
			outcome.MartyrName = g.PartyState.MartyrSecondLiving().GetNameAsString()
		} else {
			outcome.LoneLiving = true
		}

		return outcome, nil
	}

	if review.LivingCount < 2 {
		outcome.Resolved = true
		outcome.LoneLiving = true

		return outcome, nil
	}
	if attempt == 0 {
		outcome.StartedTorture = true
	} else {
		outcome.ElapsedMinutes = g.advanceElapsedGameplayMinutes(blackthornTortureMinutes, true)
	}
	if attempt == blackthornAnswerCount-1 {
		outcome.Resolved = true
		outcome.MartyrName = g.PartyState.MartyrSecondLiving().GetNameAsString()
	}
	session.attempt++

	return outcome, nil
}

// FinishBlackthornInterrogation restores every original exit invariant. It is
// idempotent so workflow finalization can own cancellation cleanup safely.
func (g *GameState) FinishBlackthornInterrogation() {
	if g.blackthornSession == nil {
		return
	}
	g.blackthornSession = nil
	g.PartyState.Inventory.Provisions.Keys.Set(0)
	// Original Palace interrogation rules use wind display or assignment, which displays the current wind without changing it.
	// See ref:CRIME-0003.
	g.TransitionToLocation(
		references.PalaceOfBlackthorn,
		references.Basement,
		references.Position{X: 10, Y: 7},
		TransitionFullLocation,
		false,
	)
	g.ProcessPendingProvisionHours()
}
