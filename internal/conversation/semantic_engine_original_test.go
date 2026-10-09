package conversation

import (
	"strings"
	"testing"

	conversationpatchdata "github.com/bradhannah/UltimaVRedux/data/conversation-patches"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/catalog"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/extractor"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOriginalSemanticConversationsSmoke(t *testing.T) {
	dataDir := originaldata.Directory(t)
	originalCatalog, err := catalog.LoadOriginal(dataDir, conversationpatchdata.Files)
	require.NoError(t, err)
	require.Equal(t, 135, originalCatalog.Count())

	for _, source := range []semantic.OriginalSource{semantic.OriginalSourceCastle, semantic.OriginalSourceKeep, semantic.OriginalSourceTowne, semantic.OriginalSourceDwelling} {
		for _, key := range originalCatalog.OriginalKeys(source) {
			definition := originalCatalogDefinition(t, originalCatalog, key.Source, key.Dialog)
			t.Run(definition.ID, func(t *testing.T) {
				callbacks := &semanticTestCallbacks{avatarName: "Avatar", karma: 99}
				engine, engineErr := NewSemanticConversationEngine(definition, callbacks, false)
				require.NoError(t, engineErr)

				response := engine.Start(1)
				for step := 0; step < 100; step++ {
					assertSemanticResponseMirror(t, response)
					require.NoError(t, response.Error)
					if response.IsComplete || response.IsCompleteAwaitingDismissal {
						return
					}
					switch {
					case response.WaitingForKeypress:
						response = engine.ProcessInput("")
					case engine.waitingForName:
						response = engine.ProcessInput("Avatar")
					case response.IsLabelQuestion:
						response = engine.ProcessInput("n")
					case response.NeedsInput:
						response = engine.ProcessInput("bye")
					default:
						t.Fatalf("conversation stalled at step %d: %+v", step, response)
					}
				}
				t.Fatal("conversation did not reach a terminal response within 100 steps")
			})
		}
	}
}

func TestOriginalSemanticBalinorFormattingStages(t *testing.T) {
	dataDir := originaldata.Directory(t)
	imported, err := extractor.Import(dataDir)
	require.NoError(t, err)
	var definition *semantic.Conversation
	for _, document := range imported.Conversations {
		if document.Conversation.ID == "original.keep.010" {
			definition = document.Conversation

			break
		}
	}
	require.NotNil(t, definition)
	engine, err := NewSemanticConversationEngine(
		definition,
		&semanticTestCallbacks{avatarName: "Avatar"},
		false,
	)
	require.NoError(t, err)

	stages := []string{
		"You see a leather-winged abomination\n\n\"Foul mortal, thou hast disturbed my peaceful rest!\"\n\n",
		"\"My bound duty is to prevent intruders from passing! And so I shall!\"\n\n",
		"\"But first, thou dost seem an educated person and so I offer thee an educated solution to the problem at hand...\"\n\n",
		"\"I suggest a battle of wits for thy passage. If I win, I shall slay thee, if thou dost win, I shall let thee pass to face the perils within!\"\n\n",
		"\"Dost thou accept?\"\n\n",
	}
	response := engine.Start(1)
	for index, expected := range stages {
		assertSemanticResponseMirror(t, response)
		assert.Equal(t, expected, response.Output, "Balinor stage %d", index+1)
		if index < len(stages)-1 {
			assert.True(t, response.WaitingForKeypress, "Balinor stage %d", index+1)
			response = engine.ProcessInput("")
		}
	}
	assert.True(t, response.NeedsInput)
	assert.True(t, response.IsLabelQuestion)
	assert.Equal(t, promptRespond, response.InputPrompt)
}

func TestOriginalSemanticAnnonPreservesKarmaKeywordsAndQuestionAnswer(t *testing.T) {
	imported, err := extractor.Import(originaldata.Directory(t))
	require.NoError(t, err)
	definition := originalSemanticDefinition(t, imported, "original.towne.012")

	for _, testCase := range []struct {
		name         string
		karma        int
		wantText     string
		unwantedText string
	}{
		{name: "below threshold", karma: 74, wantText: "I trust not yet thy Virtue", unwantedText: "I know merely one of these Words"},
		{name: "at threshold", karma: 75, wantText: "I know merely one of these Words", unwantedText: "I trust not yet thy Virtue"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			engine, engineErr := NewSemanticConversationEngine(
				definition,
				&semanticTestCallbacks{avatarName: "Avatar", hasMet: true, karma: testCase.karma},
				false,
			)
			require.NoError(t, engineErr)
			start := engine.Start(1)
			require.True(t, start.NeedsInput)

			response := engine.ProcessInput("unlock")
			assert.Contains(t, response.Output, testCase.wantText)
			assert.NotContains(t, response.Output, testCase.unwantedText)
			if testCase.karma < 75 {
				return
			}
			require.True(t, response.NeedsInput)
			require.True(t, response.IsLabelQuestion)
			answer := engine.ProcessInput("yes")
			assert.Contains(t, answer.Output, "The Word is VILIS")
			assert.True(t, answer.WaitingForKeypress)
		})
	}

	for _, keyword := range []string{"unlock", "evil", "black", "hunt", "free"} {
		t.Run("keyword "+keyword, func(t *testing.T) {
			engine, engineErr := NewSemanticConversationEngine(
				definition,
				&semanticTestCallbacks{avatarName: "Avatar", hasMet: true, karma: 75},
				false,
			)
			require.NoError(t, engineErr)
			engine.Start(1)
			assert.Contains(t, engine.ProcessInput(keyword).Output, "I know merely one of these Words")
		})
	}
}

func TestOriginalPatchedSpellKnowledgeDiscoveries(t *testing.T) {
	originalCatalog, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)

	t.Run("Temme discovers An Ylem after the runic word", func(t *testing.T) {
		definition := originalCatalogDefinition(t, originalCatalog, semantic.OriginalSourceKeep, 5)
		callbacks := &semanticTestCallbacks{avatarName: "Avatar"}
		engine, engineErr := NewSemanticConversationEngine(definition, callbacks, false)
		require.NoError(t, engineErr)
		engine.Start(1)

		response := engine.ProcessInput("word")
		require.NoError(t, response.Error)
		assert.Contains(t, response.Output, "AN YLEM")
		assert.Equal(t, []semantic.KnowledgeID{semantic.KnowledgeSpellAnYlem}, callbacks.discoveries)
	})

	t.Run("Lord Stuart discovers In Xen Mani after the affirmative formula", func(t *testing.T) {
		definition := originalCatalogDefinition(t, originalCatalog, semantic.OriginalSourceTowne, 5)
		callbacks := &semanticTestCallbacks{avatarName: "Avatar"}
		engine, engineErr := NewSemanticConversationEngine(definition, callbacks, false)
		require.NoError(t, engineErr)
		engine.Start(1)

		wait := engine.ProcessInput("perf")
		require.True(t, wait.WaitingForKeypress)
		assert.Empty(t, callbacks.discoveries)
		question := engine.ProcessInput("")
		require.True(t, question.IsLabelQuestion)
		answer := engine.ProcessInput("y")
		require.NoError(t, answer.Error)
		assert.Contains(t, answer.Output, "IN XEN MANI")
		assert.Equal(t, []semantic.KnowledgeID{semantic.KnowledgeSpellInXenMani}, callbacks.discoveries)
	})

	t.Run("Grendel discovers Rel Xen Bet after the final formula paragraph", func(t *testing.T) {
		definition := originalCatalogDefinition(t, originalCatalog, semantic.OriginalSourceDwelling, 12)
		callbacks := &semanticTestCallbacks{avatarName: "Avatar"}
		engine, engineErr := NewSemanticConversationEngine(definition, callbacks, false)
		require.NoError(t, engineErr)
		namePrompt := engine.Start(1)
		require.True(t, namePrompt.IsLabelQuestion)
		engine.ProcessInput("Avatar")

		formulaPromptWait := engine.ProcessInput("less")
		require.True(t, formulaPromptWait.WaitingForKeypress)
		question := engine.ProcessInput("")
		require.True(t, question.IsLabelQuestion)
		formulaWait := engine.ProcessInput("y")
		require.True(t, formulaWait.WaitingForKeypress)
		assert.Empty(t, callbacks.discoveries)
		formula := engine.ProcessInput("")
		require.NoError(t, formula.Error)
		assert.Contains(t, formula.Output, "nightshade cap")
		assert.Equal(t, []semantic.KnowledgeID{semantic.KnowledgeSpellRelXenBet}, callbacks.discoveries)
	})
}

func TestOriginalPatchedLoreDiscoveryBranches(t *testing.T) {
	originalCatalog, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		source      semantic.OriginalSource
		dialog      int
		hasMet      bool
		karma       int
		inputs      []string
		reveal      string
		discoveries []semantic.KnowledgeID
	}{
		{
			name: "Sin'Vraal job names Astaroth", source: semantic.OriginalSourceDwelling, dialog: 11,
			inputs: []string{"job"}, reveal: "Astaroth",
			discoveries: []semantic.KnowledgeID{semantic.KnowledgeShadowlordAstarothName},
		},
		{
			name: "Sin'Vraal confirms Astaroth and Hatred", source: semantic.OriginalSourceDwelling, dialog: 11,
			inputs: []string{"asta"}, reveal: "Shadowlord of Hatred",
			discoveries: []semantic.KnowledgeID{
				semantic.KnowledgeShadowlordAstarothName,
				semantic.KnowledgeShadowlordAstarothAssociation,
			},
		},
		{
			name: "Shalineth reveals Faulinei through simple", source: semantic.OriginalSourceKeep, dialog: 12,
			inputs: []string{"", "Avatar", "simp", "", "y"}, reveal: "FAULINEI",
			discoveries: []semantic.KnowledgeID{
				semantic.KnowledgeJournalDomainFalsehood,
				semantic.KnowledgeShadowlordFaulineiName,
				semantic.KnowledgeShadowlordFaulineiAssociation,
			},
		},
		{
			name: "Shalineth reveals Faulinei through mercy", source: semantic.OriginalSourceKeep, dialog: 12,
			inputs: []string{"", "Avatar", "merc", "", "y"}, reveal: "FAULINEI",
			discoveries: []semantic.KnowledgeID{
				semantic.KnowledgeJournalDomainFalsehood,
				semantic.KnowledgeShadowlordFaulineiName,
				semantic.KnowledgeShadowlordFaulineiAssociation,
			},
		},
		{
			name: "Malone names Nosfentor", source: semantic.OriginalSourceKeep, dialog: 22,
			inputs: []string{"evil"}, reveal: "Even Nosfentor",
			discoveries: []semantic.KnowledgeID{semantic.KnowledgeShadowlordNosfentorName},
		},
		{
			name: "Malone confirms Nosfentor and Cowardice", source: semantic.OriginalSourceKeep, dialog: 22,
			inputs: []string{"nosf"}, reveal: "Shadowlord of Cowardice",
			discoveries: []semantic.KnowledgeID{
				semantic.KnowledgeShadowlordNosfentorName,
				semantic.KnowledgeShadowlordNosfentorAssociation,
			},
		},
		{
			name: "Malifora reveals AHM", source: semantic.OriginalSourceTowne, dialog: 2,
			inputs: []string{"mant"}, reveal: "AHM",
			discoveries: []semantic.KnowledgeID{semantic.KnowledgeShrineHonestyMantra},
		},
		{
			name: "Greyson reveals MU", source: semantic.OriginalSourceTowne, dialog: 6,
			inputs: []string{"mant", "brit", "comp"}, reveal: "MU",
			discoveries: []semantic.KnowledgeID{semantic.KnowledgeShrineCompassionMantra},
		},
		{
			name: "Thorne reveals RA", source: semantic.OriginalSourceTowne, dialog: 16,
			inputs: []string{"mant", "brit"}, reveal: "RA",
			discoveries: []semantic.KnowledgeID{semantic.KnowledgeShrineValourMantra},
		},
		{
			name: "Chamfort reveals BEH", source: semantic.OriginalSourceTowne, dialog: 17,
			inputs: []string{"mant", "jere"}, reveal: "BEH",
			discoveries: []semantic.KnowledgeID{semantic.KnowledgeShrineJusticeMantra},
		},
		{
			name: "Rew reveals CAH in the chant", source: semantic.OriginalSourceTowne, dialog: 27,
			inputs: []string{"week", "y", "", "y"}, reveal: "Cah",
			discoveries: []semantic.KnowledgeID{semantic.KnowledgeShrineSacrificeMantra},
		},
		{
			name: "Gruman reveals SUMM at fifty karma", source: semantic.OriginalSourceTowne, dialog: 34,
			hasMet: true, karma: 50, inputs: []string{"n", "mant", "y"}, reveal: "SUMM",
			discoveries: []semantic.KnowledgeID{semantic.KnowledgeShrineHonorMantra},
		},
		{
			name: "Kindor reveals OM", source: semantic.OriginalSourceTowne, dialog: 39,
			inputs: []string{"shri", "", "y", "", ""}, reveal: "OM",
			discoveries: []semantic.KnowledgeID{semantic.KnowledgeShrineSpiritualityMantra},
		},
		{
			name: "Wartow reveals LUM", source: semantic.OriginalSourceTowne, dialog: 48,
			inputs: []string{"humi", "brit", "n", "y"}, reveal: "LUM",
			discoveries: []semantic.KnowledgeID{semantic.KnowledgeShrineHumilityMantra},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			callbacks := &semanticTestCallbacks{
				avatarName: "Avatar", hasMet: testCase.hasMet, karma: testCase.karma,
			}
			if callbacks.karma == 0 {
				callbacks.karma = 99
			}
			engine, engineErr := NewSemanticConversationEngine(
				originalCatalogDefinition(t, originalCatalog, testCase.source, testCase.dialog),
				callbacks,
				false,
			)
			require.NoError(t, engineErr)
			response := engine.Start(1)
			require.NoError(t, response.Error)
			assert.Empty(t, callbacks.discoveries)
			for index, input := range testCase.inputs {
				response = engine.ProcessInput(input)
				require.NoError(t, response.Error, "input %d (%q)", index, input)
				if testCase.source == semantic.OriginalSourceKeep && testCase.dialog == 12 && index >= 2 && index < len(testCase.inputs)-1 {
					assert.Equal(t, []semantic.KnowledgeID{semantic.KnowledgeJournalDomainFalsehood}, callbacks.discoveries)
				} else if index < len(testCase.inputs)-1 {
					assert.Empty(t, callbacks.discoveries, "discovered before revealing input %d", index)
				}
			}
			assert.Contains(t, response.Output, testCase.reveal)
			assert.Equal(t, testCase.discoveries, callbacks.discoveries)
		})
	}
}

func TestOriginalPatchedLoreNonDiscoveryBranches(t *testing.T) {
	originalCatalog, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)

	testCases := []struct {
		name   string
		source semantic.OriginalSource
		dialog int
		hasMet bool
		karma  int
		inputs []string
	}{
		{name: "Sin'Vraal shard clue", source: semantic.OriginalSourceDwelling, dialog: 11, inputs: []string{"shar", ""}},
		{name: "Gardner generic flame", source: semantic.OriginalSourceKeep, dialog: 25, inputs: []string{"flam"}},
		{name: "Gardner shard vision", source: semantic.OriginalSourceKeep, dialog: 25, inputs: []string{"visi", "shar", "", "", ""}},
		{name: "Shalineth reserved name topic", source: semantic.OriginalSourceKeep, dialog: 12, inputs: []string{"", "Avatar", "name"}},
		{name: "Greyson wrong ruler", source: semantic.OriginalSourceTowne, dialog: 6, inputs: []string{"mant", "n"}},
		{name: "Greyson wrong mantra", source: semantic.OriginalSourceTowne, dialog: 6, inputs: []string{"mant", "brit", "n"}},
		{name: "Thorne wrong allegiance", source: semantic.OriginalSourceTowne, dialog: 16, inputs: []string{"mant", "n"}},
		{name: "Chamfort wrong source", source: semantic.OriginalSourceTowne, dialog: 17, inputs: []string{"mant", "n"}},
		{name: "Rew player supplies CAH", source: semantic.OriginalSourceTowne, dialog: 27, inputs: []string{"cah"}},
		{name: "Rew declines chant", source: semantic.OriginalSourceTowne, dialog: 27, inputs: []string{"week", "y", "", "n"}},
		{name: "Gruman below karma threshold", source: semantic.OriginalSourceTowne, dialog: 34, hasMet: true, karma: 49, inputs: []string{"n", "mant", "y"}},
		{name: "Gruman declines", source: semantic.OriginalSourceTowne, dialog: 34, hasMet: true, karma: 50, inputs: []string{"n", "mant", "n"}},
		{name: "Kindor declines", source: semantic.OriginalSourceTowne, dialog: 39, inputs: []string{"shri", "", "n"}},
		{name: "Wartow declines mantra", source: semantic.OriginalSourceTowne, dialog: 48, inputs: []string{"humi", "brit", "n", "n"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			callbacks := &semanticTestCallbacks{
				avatarName: "Avatar", hasMet: testCase.hasMet, karma: testCase.karma,
			}
			engine, engineErr := NewSemanticConversationEngine(
				originalCatalogDefinition(t, originalCatalog, testCase.source, testCase.dialog),
				callbacks,
				false,
			)
			require.NoError(t, engineErr)
			response := engine.Start(1)
			require.NoError(t, response.Error)
			for index, input := range testCase.inputs {
				response = engine.ProcessInput(input)
				require.NoError(t, response.Error, "input %d (%q)", index, input)
				assert.Empty(t, callbacks.discoveries)
			}
		})
	}
}

func originalCatalogDefinition(
	t *testing.T,
	originalCatalog *catalog.Catalog,
	source semantic.OriginalSource,
	dialog int,
) *semantic.Conversation {
	t.Helper()
	program, exists := originalCatalog.Original(source, dialog)
	require.True(t, exists)
	definition, err := program.Definition()
	require.NoError(t, err)

	return definition
}

func TestOriginalJournalDiscoveryResponses(t *testing.T) {
	originalCatalog, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	for _, test := range []struct {
		name      string
		dialog    int
		keywords  []string
		knowledge semantic.KnowledgeID
	}{
		{name: "Michael domain", dialog: 16, keywords: []string{"thre", "alon", "stan", "virt", "shad", "hate"}, knowledge: semantic.KnowledgeJournalDomainHatred},
		{name: "Sean Flame", dialog: 26, keywords: []string{"job", "work"}, knowledge: semantic.KnowledgeJournalFlameTruth},
		{name: "Barbra Flame", dialog: 19, keywords: []string{"job", "work", "keep"}, knowledge: semantic.KnowledgeJournalFlameLove},
		{name: "Gardner Flame", dialog: 25, keywords: []string{"job", "work"}, knowledge: semantic.KnowledgeJournalFlameCourage},
	} {
		for _, keyword := range test.keywords {
			t.Run(test.name+"/"+keyword, func(t *testing.T) {
				callbacks := &semanticTestCallbacks{avatarName: "Avatar", karma: 99}
				engine, engineErr := NewSemanticConversationEngine(originalCatalogDefinition(t, originalCatalog, semantic.OriginalSourceKeep, test.dialog), callbacks, false)
				require.NoError(t, engineErr)
				engine.Start(1)
				assert.Empty(t, callbacks.discoveries)
				response := engine.ProcessInput(keyword)
				require.NoError(t, response.Error)
				assert.Equal(t, []semantic.KnowledgeID{test.knowledge}, callbacks.discoveries)
			})
		}
	}
}

func TestOriginalWordOfPowerDiscoveryBranches(t *testing.T) {
	originalCatalog, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	for _, test := range []struct {
		name      string
		source    semantic.OriginalSource
		dialog    int
		inputs    []string
		word      string
		knowledge semantic.KnowledgeID
	}{
		{"Malifora", semantic.OriginalSourceTowne, 2, []string{"word"}, "FALLAX", semantic.KnowledgeDungeonDeceitWordOfPower},
		{"Annon", semantic.OriginalSourceTowne, 12, []string{"unlo", "y"}, "VILIS", semantic.KnowledgeDungeonDespiseWordOfPower},
		{"Goeth", semantic.OriginalSourceTowne, 15, []string{"drow", "dratsed"}, "AIPONI", semantic.KnowledgeDungeonDestardWordOfPower},
		{"Felespar", semantic.OriginalSourceTowne, 22, []string{"n", "word", "y", "", "y", "dawn"}, "MALUM", semantic.KnowledgeDungeonWrongWordOfPower},
		{"Fiona", semantic.OriginalSourceTowne, 26, []string{"", "n", "coun", "anno", "Avatar", "word"}, "AVIDUS", semantic.KnowledgeDungeonCovetousWordOfPower},
		{"Sindar", semantic.OriginalSourceTowne, 32, []string{"word"}, "INFAMA", semantic.KnowledgeDungeonShameWordOfPower},
		{"Hassad", semantic.OriginalSourceCastle, 17, []string{"y", "word", "dawn", "y"}, "IGNAVUS", semantic.KnowledgeDungeonHythlothWordOfPower},
	} {
		t.Run(test.name, func(t *testing.T) {
			callbacks := &deferredSemanticTestCallbacks{semanticTestCallbacks: semanticTestCallbacks{avatarName: "Avatar", hasMet: true, karma: 75}}
			engine, engineErr := NewDeferredSemanticConversationEngine(originalCatalogDefinition(t, originalCatalog, test.source, test.dialog), callbacks, false)
			require.NoError(t, engineErr)
			response := engine.Start(1)
			for _, input := range test.inputs {
				assert.Empty(t, callbacks.knowledge, "no knowledge before the revealing response")
				response = engine.ProcessInput(input)
				require.NoError(t, response.Error)
			}
			assert.Contains(t, response.Output, test.word)
			assert.Empty(t, callbacks.discoveries, "production discovers only after presentation")
			assert.Equal(t, []semantic.KnowledgeID{test.knowledge}, callbacks.knowledge)
			require.Len(t, callbacks.boundaries, 1)
			boundary := callbacks.boundaries[0]
			assert.Equal(t, response.Serial, boundary.Response)
			assert.GreaterOrEqual(t, boundary.SourceOffset, strings.Index(response.Output, test.word)+len(test.word))
		})
	}
}

func TestOriginalWordOfPowerNonDiscoveryBranches(t *testing.T) {
	originalCatalog, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	for _, test := range []struct {
		name   string
		source semantic.OriginalSource
		dialog int
		karma  int
		inputs []string
	}{
		{"Goeth ordinary word keyword", semantic.OriginalSourceTowne, 15, 99, []string{"word", "powe"}},
		{"Goeth incorrect answer", semantic.OriginalSourceTowne, 15, 99, []string{"drow", "destard"}},
		{"Annon below karma", semantic.OriginalSourceTowne, 12, 74, []string{"unlo"}},
		{"Annon refusal", semantic.OriginalSourceTowne, 12, 75, []string{"unlo", "n"}},
		{"Felespar incorrect password", semantic.OriginalSourceTowne, 22, 99, []string{"n", "word", "y", "", "y", "dusk"}},
		{"Fiona incorrect source", semantic.OriginalSourceTowne, 26, 99, []string{"", "n", "coun", "unknown"}},
		{"Hassad refusal", semantic.OriginalSourceCastle, 17, 99, []string{"y", "word", "dawn", "n"}},
		{"Hassad incorrect password", semantic.OriginalSourceCastle, 17, 99, []string{"y", "word", "dusk"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			callbacks := &semanticTestCallbacks{avatarName: "Avatar", hasMet: true, karma: test.karma}
			engine, engineErr := NewSemanticConversationEngine(originalCatalogDefinition(t, originalCatalog, test.source, test.dialog), callbacks, false)
			require.NoError(t, engineErr)
			engine.Start(1)
			for _, input := range test.inputs {
				response := engine.ProcessInput(input)
				require.NoError(t, response.Error)
				assert.Empty(t, callbacks.discoveries)
			}
		})
	}
}

func TestOriginalJournalMichaelThenSinVraalDoesNotBindJob(t *testing.T) {
	originalCatalog, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	callbacks := &semanticTestCallbacks{avatarName: "Avatar", karma: 99}
	michael, err := NewSemanticConversationEngine(originalCatalogDefinition(t, originalCatalog, semantic.OriginalSourceKeep, 16), callbacks, false)
	require.NoError(t, err)
	michael.Start(1)
	michael.ProcessInput("shad")
	sinvraal, err := NewSemanticConversationEngine(originalCatalogDefinition(t, originalCatalog, semantic.OriginalSourceDwelling, 11), callbacks, false)
	require.NoError(t, err)
	sinvraal.Start(1)
	sinvraal.ProcessInput("job")
	assert.Equal(t, []semantic.KnowledgeID{semantic.KnowledgeJournalDomainHatred, semantic.KnowledgeShadowlordAstarothName}, callbacks.discoveries)
	sinvraal.ProcessInput("shar")
	sinvraal.ProcessInput("")
	assert.NotContains(t, callbacks.discoveries, semantic.KnowledgeShadowlordAstarothAssociation)
	sinvraal.ProcessInput("asta")
	assert.Contains(t, callbacks.discoveries, semantic.KnowledgeShadowlordAstarothAssociation)
}

func TestOriginalJournalShalinethRefusalRetainsPreOathDomain(t *testing.T) {
	originalCatalog, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	callbacks := &semanticTestCallbacks{avatarName: "Avatar", karma: 99}
	engine, err := NewSemanticConversationEngine(originalCatalogDefinition(t, originalCatalog, semantic.OriginalSourceKeep, 12), callbacks, false)
	require.NoError(t, err)
	engine.Start(1)
	engine.ProcessInput("")
	engine.ProcessInput("Avatar")
	response := engine.ProcessInput("merc")
	require.True(t, response.WaitingForKeypress)
	assert.Contains(t, response.Output, "Shadowlord of Falsehood")
	assert.Equal(t, []semantic.KnowledgeID{semantic.KnowledgeJournalDomainFalsehood}, callbacks.discoveries)
	engine.ProcessInput("")
	engine.ProcessInput("n")
	assert.Equal(t, []semantic.KnowledgeID{semantic.KnowledgeJournalDomainFalsehood}, callbacks.discoveries)
}

func TestOriginalJournalSutekWaitsForCompleteMethod(t *testing.T) {
	originalCatalog, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	callbacks := &semanticTestCallbacks{avatarName: "Avatar", karma: 99}
	engine, err := NewSemanticConversationEngine(originalCatalogDefinition(t, originalCatalog, semantic.OriginalSourceDwelling, 10), callbacks, false)
	require.NoError(t, err)
	engine.Start(1)
	question := engine.ProcessInput("Avatar")
	require.True(t, question.IsLabelQuestion)
	response := engine.ProcessInput("y")
	for stage := 0; stage < 6; stage++ {
		require.True(t, response.WaitingForKeypress)
		assert.Empty(t, callbacks.discoveries, "incomplete method at stage %d", stage)
		response = engine.ProcessInput("")
	}
	require.NoError(t, response.Error)
	require.True(t, response.WaitingForKeypress)
	assert.Contains(t, response.Output, "Even as each respective Shadowlord")
	assert.Contains(t, response.Output, "Principle it opposes.")
	assert.Equal(t, []semantic.KnowledgeID{semantic.KnowledgeJournalBanishmentMethod}, callbacks.discoveries)
	engine.ProcessInput("")
	assert.Len(t, callbacks.discoveries, 1)
}

func originalSemanticDefinition(t *testing.T, imported *extractor.ImportResult, id string) *semantic.Conversation {
	t.Helper()
	for _, document := range imported.Conversations {
		if document.Conversation.ID == id {
			return document.Conversation
		}
	}
	t.Fatalf("original semantic conversation %q not found", id)

	return nil
}

func assertSemanticResponseMirror(t *testing.T, response *ConversationResponse) {
	t.Helper()
	require.NotNil(t, response)
	var output strings.Builder
	for _, segment := range response.Segments {
		output.WriteString(segment.Text)
	}
	assert.Equal(t, response.Output, output.String())
}
