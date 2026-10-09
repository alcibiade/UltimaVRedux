package conversation

import (
	"testing"

	conversationpatchdata "github.com/bradhannah/UltimaVRedux/data/conversation-patches"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/catalog"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/extractor"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/require"
)

func TestOriginalLighthouseUnpatchedEvidence(t *testing.T) {
	imported, err := extractor.Import(originaldata.Directory(t))
	require.NoError(t, err)
	kenneth := originalSemanticDefinition(t, imported, "original.dwelling.014")
	engine, err := NewSemanticConversationEngine(kenneth, &semanticTestCallbacks{avatarName: "Avatar"}, false)
	require.NoError(t, err)
	engine.Start(7)
	engine.ProcessInput("COMP")
	require.Contains(t, engine.ProcessInput("Y").Output, "Open the Book of Lore")
	engine.ProcessInput("")
	engine.ProcessInput("")
	engine.ProcessInput("")
	engine.ProcessInput("DCB")
	require.Contains(t, engine.ProcessInput("987").Output, "678 987 8767653")
	arbuthnot := originalSemanticDefinition(t, imported, "original.dwelling.015")
	require.Empty(t, arbuthnot.Questions["question_03"].Answers)
	engine, err = NewSemanticConversationEngine(arbuthnot, &semanticTestCallbacks{avatarName: "Avatar"}, false)
	require.NoError(t, err)
	engine.Start(7)
	engine.ProcessInput("AVAT")
	engine.ProcessInput("Y")
	require.Contains(t, engine.ProcessInput("YES").Output, "Oh.")
	patched, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	repaired := originalCatalogDefinition(t, patched, semantic.OriginalSourceDwelling, 15)
	require.Equal(t, arbuthnot.Topics, repaired.Topics, "existing ordered topic routes and wording are preserved")
	require.Equal(t, arbuthnot.Name, repaired.Name)
	require.Equal(t, arbuthnot.Description, repaired.Description)
	require.Equal(t, arbuthnot.Greeting, repaired.Greeting)
	require.Equal(t, arbuthnot.Job, repaired.Job)
	require.Equal(t, arbuthnot.Farewell, repaired.Farewell)
}

func TestOriginalKennethTeachingBranches(t *testing.T) {
	catalogue, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	definition := originalCatalogDefinition(t, catalogue, semantic.OriginalSourceDwelling, 14)
	for _, topic := range []string{"COUR", "COMP", "STONES", "MUSIC", "HARPSICHORD"} {
		for _, knows := range []string{"Y", "YES", "N", "NO"} {
			t.Run(topic+"/"+knows, func(t *testing.T) {
				callbacks := &semanticTestCallbacks{avatarName: "Avatar"}
				engine, engineErr := NewSemanticConversationEngine(definition, callbacks, false)
				require.NoError(t, engineErr)
				engine.Start(7)
				require.Contains(t, engine.ProcessInput("JOB").Output, "MUSIC")
				for range 2 {
					engine.ProcessInput(topic)
					engine.ProcessInput("Y")
					before := len(callbacks.discoveries)
					response := engine.ProcessInput(knows)
					require.True(t, response.WaitingForKeypress)
					require.Len(t, callbacks.discoveries, before)
					response = engine.ProcessInput("")
					require.NoError(t, response.Error)
					require.Contains(t, response.Output, "immediately north")
					require.Contains(t, response.Output, "numpad")
					require.Contains(t, response.Output, "6 7 8 9 8 7 8 7 6 7 6 5 3")
					require.Equal(t, promptInterest, response.InputPrompt)
					require.Equal(t, semantic.KnowledgeJournalStonesMelody, callbacks.discoveries[before])
				}
			})
		}
	}
}

func TestOriginalArbuthnotRepairedNameAndStory(t *testing.T) {
	catalogue, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	definition := originalCatalogDefinition(t, catalogue, semantic.OriginalSourceDwelling, 15)
	for _, affirmative := range []string{"Y", "YES"} {
		for _, name := range []string{"Avatar", "Iolo", "Nobody", "Ava"} {
			t.Run(affirmative+"/"+name, func(t *testing.T) {
				callbacks := &semanticTestCallbacks{avatarName: "Avatar"}
				engine, engineErr := NewSemanticConversationEngine(definition, callbacks, false)
				require.NoError(t, engineErr)
				engine.Start(7)
				engine.ProcessInput("AVAT")
				engine.ProcessInput(affirmative)
				response := engine.ProcessInput(affirmative)
				require.Contains(t, response.Output, "honor")
				require.True(t, engine.waitingForName)
				response = engine.ProcessInput(name)
				if name == "Nobody" || name == "Ava" {
					require.False(t, callbacks.hasMet)
					require.NotContains(t, response.Output, "coin")
					require.Equal(t, promptInterest, response.InputPrompt)
				} else {
					require.Equal(t, []int{7}, callbacks.metActorIDs, "recognition uses the actor slot, not dialog 15")
					require.Contains(t, response.Output, "I minted a magical coin!")
					require.True(t, response.WaitingForKeypress)
					response = engine.ProcessInput("")
					require.Contains(t, response.Output, "It was in the symbol of the Codex!")
					require.True(t, response.WaitingForKeypress)
					response = engine.ProcessInput("")
					require.Contains(t, response.Output, "commisioned secretly by persons unknown even to me")
					require.Contains(t, response.Output, "call back the one true Avatar!")
					require.Equal(t, promptInterest, response.InputPrompt)
					engine.ProcessInput("AVAT")
					engine.ProcessInput(affirmative)
					response = engine.ProcessInput(affirmative)
					require.False(t, engine.waitingForName, "an already recognized visitor need not reintroduce themselves")
					require.Contains(t, response.Output, "I minted a magical coin!")
					engine.ProcessInput("")
					response = engine.ProcessInput("")
					require.Equal(t, promptInterest, response.InputPrompt)
					require.Equal(t, []int{7}, callbacks.metActorIDs)
				}
				require.Contains(t, engine.ProcessInput("COIN").Output, "wanted man")
				require.Contains(t, engine.ProcessInput("MAGI").Output, "Codex")
				require.Contains(t, engine.ProcessInput("SYMB").Output, "commisioned secretly")
				require.Empty(t, callbacks.discoveries, "Arbuthnot never grants automatic Chronicle lore")
			})
		}
	}
	for _, answer := range []string{"N", "NO", "perhaps"} {
		engine, engineErr := NewSemanticConversationEngine(definition, &semanticTestCallbacks{avatarName: "Avatar"}, false)
		require.NoError(t, engineErr)
		engine.Start(7)
		engine.ProcessInput("AVAT")
		require.Contains(t, engine.ProcessInput(answer).Output, "I see.")
		engine.ProcessInput("AVAT")
		engine.ProcessInput("Y")
		require.Contains(t, engine.ProcessInput(answer).Output, "Oh.")
		require.False(t, engine.waitingForName)
	}
}
