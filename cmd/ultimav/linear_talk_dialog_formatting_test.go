package main

import (
	"strings"
	"testing"

	conversationpatchdata "github.com/bradhannah/UltimaVRedux/data/conversation-patches"
	"github.com/bradhannah/UltimaVRedux/internal/conversation"
	conversationcatalog "github.com/bradhannah/UltimaVRedux/internal/conversation/catalog"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	ultimacolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOriginalBalinorProductionDialogRows(t *testing.T) {
	catalog, err := conversationcatalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	var roster [party_state.NRosterSlots]*party_state.PlayerCharacter
	avatar := &party_state.PlayerCharacter{PartyMap: party_state.PartyMapActive}
	copy(avatar.Name[:], "Avatar")
	roster[0] = avatar
	state := &game_state.GameState{ //nolint:exhaustruct // Production conversation fixture needs only catalog, location, party, and debug state.
		PartyState: *party_state.NewPartyStateFromRoster(roster),
		MapState: map_state.MapState{ //nolint:exhaustruct // Only current location participates in conversation resolution.
			PlayerLocation: references.PlayerLocation{Location: references.Stonegate},
		},
		GameReferences: &references.GameReferences{ConversationCatalog: catalog}, //nolint:exhaustruct // Focused catalog fixture.
	}
	keyboard := newPresentationTestKeyboard()
	scene := &GameScene{gameState: state, keyboard: keyboard} //nolint:exhaustruct // Focused production dialog fixture.
	npc := map_units.NewNPCFriendly(references.NPCReference{  //nolint:exhaustruct // Balinor identity fields only.
		Location: references.Stonegate, DialogNumber: 10, NpcType: references.NoStatedNpc,
	}, 1)

	dialog := NewLinearTalkDialog(scene, npc)
	require.NotNil(t, dialog)
	assert.True(t, dialog.waitingForKeypressContinue)
	assert.Equal(t, []string{
		"You see a leather-winged abomination",
		"",
		`"Foul mortal, thou hast disturbed my`,
		`peaceful rest!"`,
		"",
	}, rowsFromFirstContent(dialog.Output))

	expectedStages := [][]string{
		{`"My bound duty is to prevent intruders from`, `passing! And so I shall!"`, ""},
		{`"But first, thou dost seem an educated`, `person and so I offer thee an educated`, `solution to the problem at hand..."`, ""},
		{`"I suggest a battle of wits for thy passage.`, `If I win, I shall slay thee, if thou dost`, `win, I shall let thee pass to face the`, `perils within!"`, ""},
		{`"Dost thou accept?"`, "", "You respond:", ">"},
	}
	for _, expectedSuffix := range expectedStages {
		dialog.waitingForKeypressContinue = false
		dialog.currentResponse = dialog.engine.ProcessInput("")
		dialog.processCurrentResponse()
		rows := serializedOutputRows(dialog.Output)
		require.GreaterOrEqual(t, len(rows), len(expectedSuffix))
		assert.Equal(t, expectedSuffix, rows[len(rows)-len(expectedSuffix):])
	}
	assert.True(t, dialog.waitingForInput)
	assert.False(t, dialog.waitingForKeypressContinue)
}

func rowsFromFirstContent(output *text.Output) []string {
	rows := serializedOutputRows(output)
	for len(rows) > 0 && rows[0] == "" {
		rows = rows[1:]
	}

	return rows
}

func TestLinearTalkDialogKeyWaitDoesNotChangeConversationOutput(t *testing.T) {
	keyboard := newPresentationTestKeyboard()
	output := text.NewOutput(nil, 1, 8, 45)
	output.AddTextRaw("\"Before\"\n\n")
	dialog := &LinearTalkDialog{ //nolint:exhaustruct // Focused KeyWait presentation fixture.
		gameScene: &GameScene{ //nolint:exhaustruct // Focused KeyWait presentation fixture.
			gameState: &game_state.GameState{},
			keyboard:  keyboard,
		},
		Output: output,
		TextInput: widgets.NewTextInput(
			sprites.PercentBasedPlacement{},
			widgets.GetScaledLineHeight(talkFontSize),
			talkMaxCharsInput,
			nil,
			widgets.TextInputCallbacks{},
			keyboard,
		),
	}
	before := output.GetOutputStr(true)

	dialog.enterKeyWaitState()

	assert.True(t, dialog.waitingForKeypressContinue)
	assert.Equal(t, before, output.GetOutputStr(true))
}

func TestLinearTalkDialogResponseCanContinuePreviousOutput(t *testing.T) {
	output := text.NewOutput(nil, 1, 8, 45)
	output.AddTextRaw("\"Before ")
	dialog := &LinearTalkDialog{ //nolint:exhaustruct // Focused response-presentation fixture.
		Output: output,
		currentResponse: &conversation.ConversationResponse{ //nolint:exhaustruct // Protocol fixture sets relevant fields only.
			Output:                  "after.\"",
			ContinuesPreviousOutput: true,
		},
	}

	dialog.displayResponseOutput()

	assert.Equal(t, "\"Before after.\"", output.GetOutputStr(false))
}

func TestLinearTalkDialogPromptPresentationStylesRemainSelectable(t *testing.T) {
	testCases := []struct {
		name     string
		style    linearTalkPresentationStyle
		prompt   string
		expected []string
	}{
		{
			name: "joined", style: linearTalkPresentationJoined, prompt: "Your interest?",
			expected: []string{"Your interest?"},
		},
		{
			name: "continuous colon", style: linearTalkPresentationContinuousColon, prompt: "You respond:",
			expected: []string{"You respond-", ":"},
		},
		{
			name: "continuous marker", style: linearTalkPresentationContinuousMarker, prompt: "Your interest?",
			expected: []string{"Your interest?", ">"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			output := text.NewOutput(nil, 1, 4, 45)
			dialog := &LinearTalkDialog{Output: output} //nolint:exhaustruct // Focused prompt presentation fixture.

			dialog.addInputPromptForStyle(testCase.prompt, testCase.style)

			assert.Equal(t, testCase.expected, rowsFromFirstContent(output))
		})
	}
}

func TestLinearTalkDialogNormalResponseSpacing(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		response string
	}{
		{name: "name", input: "NAME", response: "\"My name is Chuckles\""},
		{name: "job", input: "JOB", response: "\"I am the court jester.\""},
		{name: "topic", input: "JEST", response: "\"A merry subject!\""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			output := text.NewOutput(nil, 1, 5, 90)
			prompt := []text.ColorSegment{{Text: "Your interest? ", Color: ultimacolor.EGALightGreen}}
			output.AddStyledPrompt(prompt, etext.AlignStart)
			output.AppendText(testCase.input)
			dialog := &LinearTalkDialog{ //nolint:exhaustruct // Focused response-presentation fixture.
				Output: output,
				currentResponse: &conversation.ConversationResponse{ //nolint:exhaustruct // Relevant protocol fields only.
					Output: "\n\n" + testCase.response + "\n\n", ContinuesPreviousOutput: true,
				},
			}

			dialog.displayResponseOutput()
			output.AddStyledPrompt(prompt, etext.AlignStart)

			assert.Equal(t, []string{
				"Your interest? " + testCase.input,
				"",
				testCase.response,
				"",
				"Your interest?",
			}, serializedOutputRows(output))
		})
	}
}

func TestLinearTalkDialogRunicResponseSpacing(t *testing.T) {
	output := text.NewOutput(nil, 1, 5, 90)
	prompt := []text.ColorSegment{{Text: "Your interest? ", Color: ultimacolor.EGALightGreen}}
	output.AddStyledPrompt(prompt, etext.AlignStart)
	output.AppendText("RUNE")
	dialog := &LinearTalkDialog{ //nolint:exhaustruct // Focused runic response-presentation fixture.
		Output: output,
		currentResponse: &conversation.ConversationResponse{ //nolint:exhaustruct // Relevant protocol fields only.
			Output: "\n\n\"VILIS\"\n\n",
			Segments: []conversation.TextSegment{
				{Text: "\n\n\"", IsRunic: false},
				{Text: "VILIS", IsRunic: true},
				{Text: "\"\n\n", IsRunic: false},
			},
			ContinuesPreviousOutput: true,
		},
	}

	dialog.displayResponseOutput()
	output.AddStyledPrompt(prompt, etext.AlignStart)

	assert.Equal(t, []string{
		"Your interest? RUNE",
		"",
		"\"VILIS\"",
		"",
		"Your interest?",
	}, serializedOutputRows(output))
}

func TestLinearTalkDialogDismissalSpacing(t *testing.T) {
	keyboard := newPresentationTestKeyboard()
	output := text.NewOutput(nil, 1, 5, 90)
	output.AddStyledPrompt(
		[]text.ColorSegment{{Text: "Your interest? ", Color: ultimacolor.EGALightGreen}},
		etext.AlignStart,
	)
	output.AppendText("BYE")
	dialog := &LinearTalkDialog{ //nolint:exhaustruct // Focused dismissal-presentation fixture.
		gameScene: &GameScene{ //nolint:exhaustruct // Focused dismissal-presentation fixture.
			gameState: &game_state.GameState{}, keyboard: keyboard,
		},
		Output: output,
		TextInput: widgets.NewTextInput(
			sprites.PercentBasedPlacement{}, widgets.GetScaledLineHeight(talkFontSize), talkMaxCharsInput,
			nil, widgets.TextInputCallbacks{}, keyboard,
		),
		currentResponse: &conversation.ConversationResponse{ //nolint:exhaustruct // Relevant protocol fields only.
			Output: "\n\n\"Farewell.\"\n", ContinuesPreviousOutput: true,
		},
	}

	dialog.displayResponseOutput()
	dialog.enterDismissalWaitState()

	assert.Equal(t, []string{
		"Your interest? BYE",
		"",
		"\"Farewell.\"",
	}, rowsFromFirstContent(output))
	assert.Contains(t, dialog.conversationFooterGuidance(600), "[Space] Close")
}

func TestLinearTalkDialogDismissalOrdersCallbackOutputBeforeHint(t *testing.T) {
	keyboard := newPresentationTestKeyboard()
	output := text.NewOutput(nil, 1, 6, 90)
	output.AddStyledPrompt(
		[]text.ColorSegment{{Text: "Your interest? ", Color: ultimacolor.EGALightGreen}},
		etext.AlignStart,
	)
	output.AppendText("JOIN")
	callbacks := &GameActionCallbacks{} //nolint:exhaustruct // Focused accumulated-output fixture.
	callbacks.ShowOutput("Chuckles has joined thee!")
	dialog := &LinearTalkDialog{ //nolint:exhaustruct // Focused dismissal-presentation fixture.
		gameScene: &GameScene{ //nolint:exhaustruct // Focused dismissal-presentation fixture.
			gameState: &game_state.GameState{}, keyboard: keyboard,
		},
		Output: output,
		TextInput: widgets.NewTextInput(
			sprites.PercentBasedPlacement{}, widgets.GetScaledLineHeight(talkFontSize), talkMaxCharsInput,
			nil, widgets.TextInputCallbacks{}, keyboard,
		),
		callbacks: callbacks,
		currentResponse: &conversation.ConversationResponse{ //nolint:exhaustruct // Relevant protocol fields only.
			Output: "\n\n\"I shall join thee.\"\n", ContinuesPreviousOutput: true,
		},
	}

	dialog.displayResponseOutput()
	dialog.enterDismissalWaitState()

	assert.Equal(t, []string{
		"Your interest? JOIN",
		"",
		"\"I shall join thee.\"",
		"Chuckles has joined thee!",
	}, rowsFromFirstContent(output))
	assert.Contains(t, dialog.conversationFooterGuidance(600), "[Space] Close")
	assert.Empty(t, callbacks.GetAccumulatedOutput())
}

func TestLinearTalkDialogCapacityRefusalOrdersAuthoredOutputBeforeCallback(t *testing.T) {
	keyboard := newPresentationTestKeyboard()
	output := text.NewOutput(nil, 1, 6, 90)
	callbacks := &GameActionCallbacks{} //nolint:exhaustruct // Focused accumulated-output fixture.
	callbacks.ShowOutput("\nThou hast no room for me in thy party!\n")
	dialog := &LinearTalkDialog{ //nolint:exhaustruct // Focused refusal-presentation fixture.
		gameScene: &GameScene{ //nolint:exhaustruct // Focused key-wait presentation fixture.
			gameState: &game_state.GameState{}, keyboard: keyboard,
		},
		Output: output,
		TextInput: widgets.NewTextInput(
			sprites.PercentBasedPlacement{}, widgets.GetScaledLineHeight(talkFontSize), talkMaxCharsInput,
			nil, widgets.TextInputCallbacks{}, keyboard,
		),
		callbacks: callbacks,
		currentResponse: &conversation.ConversationResponse{ //nolint:exhaustruct // Relevant protocol fields only.
			Output: "\"I shall join thee.\"", ContinuesPreviousOutput: true, WaitingForKeypress: true,
		},
	}

	dialog.processCurrentResponse()

	transcript := output.GetOutputStr(true)
	authored := strings.Index(transcript, "\"I shall join thee.\"")
	refusal := strings.Index(transcript, "Thou hast no room for me in thy party!")
	require.NotEqual(t, -1, authored)
	require.NotEqual(t, -1, refusal)
	assert.Less(t, authored, refusal)
	assert.Empty(t, callbacks.GetAccumulatedOutput())
	assert.True(t, dialog.waitingForKeypressContinue)
}

func serializedOutputRows(output *text.Output) []string {
	rows := strings.Split(output.GetOutputStr(true), " \n")
	for index := range rows {
		rows[index] = strings.TrimSuffix(rows[index], " ")
	}

	return rows
}
