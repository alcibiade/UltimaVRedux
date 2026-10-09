package catalog

import (
	"strings"
	"testing"

	conversationpatchdata "github.com/bradhannah/UltimaVRedux/data/conversation-patches"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOriginalCatalogIntegration(t *testing.T) {
	dataDir := originaldata.Directory(t)
	catalog, err := LoadOriginal(dataDir, conversationpatchdata.Files)
	require.NoError(t, err)
	assert.Equal(t, 135, catalog.Count())

	balinor, exists := catalog.Original(semantic.OriginalSourceKeep, 10)
	require.True(t, exists)
	assert.Equal(t, "original.keep.010", balinor.ID())
	definition, err := balinor.Definition()
	require.NoError(t, err)
	assert.Equal(t, 10, definition.Origin.(*semantic.OriginalOrigin).Dialog) //nolint:forcetypeassert // Catalog validates this union.
}

func TestOriginalCatalogKnowledgePatches(t *testing.T) {
	dataDir := originaldata.Directory(t)
	catalog, err := LoadOriginal(dataDir, conversationpatchdata.Files)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		source    semantic.OriginalSource
		dialog    int
		knowledge []semantic.KnowledgeID
		body      func(*testing.T, *semantic.Conversation) semantic.Body
	}{
		{
			name: "Malifora Word of Power", source: semantic.OriginalSourceTowne, dialog: 2,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeDungeonDeceitWordOfPower},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalTopicBody(t, definition, "word")
			},
		},
		{
			name: "Annon Word of Power", source: semantic.OriginalSourceTowne, dialog: 12,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeDungeonDespiseWordOfPower},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalAnswerBody(t, definition, "question_05", "y")
			},
		},
		{
			name: "Goeth spoken Word of Power", source: semantic.OriginalSourceTowne, dialog: 15,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeDungeonDestardWordOfPower},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalAnswerBody(t, definition, "question_01", "drat")
			},
		},
		{
			name: "Felespar Word of Power", source: semantic.OriginalSourceTowne, dialog: 22,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeDungeonWrongWordOfPower},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalAnswerBody(t, definition, "question_05", "dawn")
			},
		},
		{
			name: "Fiona Word of Power", source: semantic.OriginalSourceTowne, dialog: 26,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeDungeonCovetousWordOfPower},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalAnswerBody(t, definition, "question_08", "word")
			},
		},
		{
			name: "Sindar Word of Power", source: semantic.OriginalSourceTowne, dialog: 32,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeDungeonShameWordOfPower},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalTopicBody(t, definition, "word")
			},
		},
		{
			name: "Hassad Word of Power", source: semantic.OriginalSourceCastle, dialog: 17,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeDungeonHythlothWordOfPower},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalAnswerBody(t, definition, "question_04", "y")
			},
		},
		{
			name: "Shalineth before oath", source: semantic.OriginalSourceKeep, dialog: 12,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeJournalDomainFalsehood},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalTopicBody(t, definition, "merc")
			},
		},
		{
			name: "Michael holy place", source: semantic.OriginalSourceKeep, dialog: 16,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeJournalDomainHatred},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalTopicBody(t, definition, "thre")
			},
		},
		{
			name: "Michael referral", source: semantic.OriginalSourceKeep, dialog: 16,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeJournalDomainHatred},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalTopicBody(t, definition, "shad")
			},
		},
		{
			name: "Sir Sean job", source: semantic.OriginalSourceKeep, dialog: 26,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeJournalFlameTruth},
			body:      func(_ *testing.T, definition *semantic.Conversation) semantic.Body { return definition.Job },
		},
		{
			name: "Barbra job", source: semantic.OriginalSourceKeep, dialog: 19,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeJournalFlameLove},
			body:      func(_ *testing.T, definition *semantic.Conversation) semantic.Body { return definition.Job },
		},
		{
			name: "Barbra keeper", source: semantic.OriginalSourceKeep, dialog: 19,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeJournalFlameLove},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalTopicBody(t, definition, "keep")
			},
		},
		{
			name: "Gardner job", source: semantic.OriginalSourceKeep, dialog: 25,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeJournalFlameCourage},
			body:      func(_ *testing.T, definition *semantic.Conversation) semantic.Body { return definition.Job },
		},
		{
			name: "Sutek complete method", source: semantic.OriginalSourceDwelling, dialog: 10,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeJournalBanishmentMethod},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalAnswerBody(t, definition, "question_02", "y")
			},
		},
		{
			name: "Temme word topic", source: semantic.OriginalSourceKeep, dialog: 5,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeSpellAnYlem},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalTopicBody(t, definition, "word")
			},
		},
		{
			name: "Lord Stuart affirmative formula", source: semantic.OriginalSourceTowne, dialog: 5,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeSpellInXenMani},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalAnswerBody(t, definition, "question_02", "y")
			},
		},
		{
			name: "Grendel affirmative formula", source: semantic.OriginalSourceDwelling, dialog: 12,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeSpellRelXenBet},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalAnswerBody(t, definition, "question_01", "y")
			},
		},
		{
			name: "Sin'Vraal job", source: semantic.OriginalSourceDwelling, dialog: 11,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeShadowlordAstarothName},
			body:      func(_ *testing.T, definition *semantic.Conversation) semantic.Body { return definition.Job },
		},
		{
			name: "Sin'Vraal Astaroth confirmation", source: semantic.OriginalSourceDwelling, dialog: 11,
			knowledge: []semantic.KnowledgeID{
				semantic.KnowledgeShadowlordAstarothName,
				semantic.KnowledgeShadowlordAstarothAssociation,
			},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalTopicBody(t, definition, "asta")
			},
		},
		{
			name: "Shalineth confirmation", source: semantic.OriginalSourceKeep, dialog: 12,
			knowledge: []semantic.KnowledgeID{
				semantic.KnowledgeShadowlordFaulineiName,
				semantic.KnowledgeShadowlordFaulineiAssociation,
			},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalAnswerBody(t, definition, "question_01", "y")
			},
		},
		{
			name: "Malone names Nosfentor", source: semantic.OriginalSourceKeep, dialog: 22,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeShadowlordNosfentorName},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalTopicBody(t, definition, "evil")
			},
		},
		{
			name: "Malone confirms Nosfentor", source: semantic.OriginalSourceKeep, dialog: 22,
			knowledge: []semantic.KnowledgeID{
				semantic.KnowledgeShadowlordNosfentorName,
				semantic.KnowledgeShadowlordNosfentorAssociation,
			},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalTopicBody(t, definition, "nosf")
			},
		},
		{
			name: "Malifora mantra", source: semantic.OriginalSourceTowne, dialog: 2,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeShrineHonestyMantra},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalTopicBody(t, definition, "mant")
			},
		},
		{
			name: "Greyson mantra", source: semantic.OriginalSourceTowne, dialog: 6,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeShrineCompassionMantra},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalAnswerBody(t, definition, "question_02", "comp")
			},
		},
		{
			name: "Thorne mantra", source: semantic.OriginalSourceTowne, dialog: 16,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeShrineValourMantra},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalAnswerBody(t, definition, "question_02", "brit")
			},
		},
		{
			name: "Chamfort mantra", source: semantic.OriginalSourceTowne, dialog: 17,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeShrineJusticeMantra},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalAnswerBody(t, definition, "question_04", "jere")
			},
		},
		{
			name: "Chamfort password", source: semantic.OriginalSourceTowne, dialog: 17,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeJournalResistancePassword},
			body: func(_ *testing.T, definition *semantic.Conversation) semantic.Body {
				return definition.Questions["question_03"].Prompt
			},
		},
		{
			name: "Rew mantra", source: semantic.OriginalSourceTowne, dialog: 27,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeShrineSacrificeMantra},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalAnswerBody(t, definition, "question_02", "y")
			},
		},
		{
			name: "Gruman mantra", source: semantic.OriginalSourceTowne, dialog: 34,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeShrineHonorMantra},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				question, exists := definition.Questions["question_04"]
				require.True(t, exists)

				return question.Prompt
			},
		},
		{
			name: "Flain password", source: semantic.OriginalSourceTowne, dialog: 38,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeJournalOppressionPassword},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalAnswerBody(t, definition, "question_04", "mali")
			},
		},
		{
			name: "Kindor mantra", source: semantic.OriginalSourceTowne, dialog: 39,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeShrineSpiritualityMantra},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalAnswerBody(t, definition, "question_01", "y")
			},
		},
		{
			name: "Wartow mantra", source: semantic.OriginalSourceTowne, dialog: 48,
			knowledge: []semantic.KnowledgeID{semantic.KnowledgeShrineHumilityMantra},
			body: func(t *testing.T, definition *semantic.Conversation) semantic.Body {
				return originalAnswerBody(t, definition, "question_06", "y")
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			program, exists := catalog.Original(testCase.source, testCase.dialog)
			require.True(t, exists)
			assert.True(t, program.Patched())
			definition, definitionErr := program.Definition()
			require.NoError(t, definitionErr)
			body := testCase.body(t, definition)
			assertKnowledgeDiscoveriesFollowSay(t, body, testCase.knowledge)
		})
	}
}

func assertKnowledgeDiscoveriesFollowSay(
	t *testing.T,
	body semantic.Body,
	expected []semantic.KnowledgeID,
) {
	t.Helper()
	actual := make([]semantic.KnowledgeID, 0, len(expected))
	firstDiscovery := -1
	for index, action := range body {
		discovery, ok := action.(*semantic.DiscoverKnowledgeAction)
		if !ok {
			continue
		}
		if firstDiscovery == -1 {
			firstDiscovery = index
		}
		actual = append(actual, discovery.Knowledge)
	}
	require.Equal(t, expected, actual)
	require.Greater(t, firstDiscovery, 0)
	assert.IsType(t, &semantic.SayAction{}, body[firstDiscovery-1])
	for offset, knowledge := range expected {
		discovery, ok := body[firstDiscovery+offset].(*semantic.DiscoverKnowledgeAction)
		require.True(t, ok)
		assert.Equal(t, knowledge, discovery.Knowledge)
	}
}

func originalTopicBody(t *testing.T, definition *semantic.Conversation, keyword string) semantic.Body {
	t.Helper()
	for _, topic := range definition.Topics {
		for _, candidate := range topic.Keywords {
			if strings.EqualFold(candidate, keyword) {
				return topic.Response
			}
		}
	}
	t.Fatalf("topic %q not found", keyword)

	return nil
}

func originalAnswerBody(
	t *testing.T,
	definition *semantic.Conversation,
	questionName string,
	keyword string,
) semantic.Body {
	t.Helper()
	question, exists := definition.Questions[questionName]
	require.True(t, exists)
	for _, answer := range question.Answers {
		for _, candidate := range answer.Keywords {
			if strings.EqualFold(candidate, keyword) {
				return answer.Response
			}
		}
	}
	t.Fatalf("answer %q not found in %s", keyword, questionName)

	return nil
}
