package extractor

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseExecutableClassifiesAllSupportedControls(t *testing.T) {
	t.Parallel()

	dictionary := map[byte]string{0x01: "word"}
	data := append(upper("A"), 0x01, opAvatarName, opPause, opJoinParty)
	data = append(data,
		opTakeGold, 0xb1, 0xb2, 0xb3,
		opGiveItem, 0x41,
		opIncludeNext,
		opAskName, opKarmaUp, opKarmaDown, opCallGuards,
		opIfKnowsName, firstQuestionID,
		opNewline, opRunic,
	)
	data = append(data, upper("R")...)
	data = append(data,
		opRunic, opWaitForKey, firstQuestionID, opQuoteBoundary,
		opIfKarma, 50, firstQuestionID, opReturnTopics, opEnd,
	)

	instructions, err := parseExecutable(testField(data), dictionary, true)
	require.NoError(t, err)
	require.Len(t, instructions, 23)
	assert.Equal(t, instructionWord, instructions[1].kind)
	assert.Equal(t, 123, instructions[5].value)
	assert.Equal(t, "food", instructions[6].text)
	assert.Equal(t, instructionInclude, instructions[7].kind)
	assert.Equal(t, instructionIfKnowsName, instructions[12].kind)
	assert.Equal(t, instructionAsk, instructions[18].kind)
	assert.Equal(t, instructionIfKarma, instructions[20].kind)
	assert.Equal(t, actionReturnTopics, instructions[21].action)
	assert.Equal(t, actionEnd, instructions[22].action)
}

func TestParseExecutableRejectsMalformedControls(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		data    []byte
		include bool
		message string
	}{
		{name: "low control", data: []byte{0x08}, include: true, message: "unsupported executable byte 0x08"},
		{name: "take gold short", data: []byte{opTakeGold, 0xb1}, include: true, message: "requires exactly three"},
		{name: "take gold digit", data: []byte{opTakeGold, 0xb1, 0xaf, 0xb3}, include: true, message: "not an ASCII digit"},
		{name: "take gold zero", data: []byte{opTakeGold, 0xb0, 0xb0, 0xb0}, include: true, message: "amount 000"},
		{name: "give item short", data: []byte{opGiveItem}, include: true, message: "requires one item byte"},
		{name: "give item unknown", data: []byte{opGiveItem, 0x40}, include: true, message: "no verified semantic item mapping"},
		{name: "include outside response", data: []byte{opIncludeNext}, message: "only supported inside"},
		{name: "knows target short", data: []byte{opIfKnowsName}, include: true, message: "requires one target"},
		{name: "knows target invalid", data: []byte{opIfKnowsName, 0x80}, include: true, message: "neither a question nor 0xff"},
		{name: "karma short", data: []byte{opIfKarma, 50}, include: true, message: "requires threshold and target"},
		{name: "karma target invalid", data: []byte{opIfKarma, 50, 0x80}, include: true, message: "is not a question"},
		{name: "literal closing brace", data: []byte{0xfd}, include: true, message: "cannot be represented"},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseExecutable(testField(testCase.data), nil, testCase.include)
			require.ErrorContains(t, err, testCase.message)
			assert.Contains(t, err.Error(), "SYNTH.TLK conversation 7 offset")
		})
	}
}

func TestInstructionsToBodyPreservesTextActionsBranchesAndFormatting(t *testing.T) {
	t.Parallel()

	data := append(upper("One"), opNewline)
	data = append(data, upper("line")...)
	data = append(data, opQuoteBoundary, opNewline, opNewline, opPause, opQuoteBoundary)
	data = append(data, upper("Rune ")...)
	data = append(data, opRunic)
	data = append(data, upper("text")...)
	data = append(data, opRunic, opIfKnowsName, opReturnTopics)
	data = append(data, upper("After")...)

	instructions, err := parseExecutable(testField(data), nil, false)
	require.NoError(t, err)
	body, err := instructionsToBody(instructions, semantic.ActionKindSay, testTargetResolver)
	require.NoError(t, err)
	require.Len(t, body, 6)
	first, ok := body[0].(*semantic.SayAction)
	require.True(t, ok)
	require.Len(t, first.Content, 3)
	assert.IsType(t, &semantic.LineBreakPart{}, first.Content[1])
	separator, ok := body[1].(*semantic.NarrateAction)
	require.True(t, ok)
	require.Len(t, separator.Content, 2)
	assert.IsType(t, &semantic.LineBreakPart{}, separator.Content[0])
	assert.IsType(t, &semantic.LineBreakPart{}, separator.Content[1])
	assert.IsType(t, &semantic.PauseAction{}, body[2])
	second, ok := body[3].(*semantic.SayAction)
	require.True(t, ok)
	assert.IsType(t, &semantic.RunicPart{}, second.Content[1])
	branch, ok := body[4].(*semantic.BranchAction)
	require.True(t, ok)
	assert.IsType(t, &semantic.NPCKnowsPartyNameCondition{}, branch.Condition)
	assert.IsType(t, &semantic.ReturnToTopicsAction{}, branch.Then[0])
	assert.IsType(t, &semantic.SayAction{}, body[5])
}

func TestInstructionsToBodyPreservesEverySourceNewline(t *testing.T) {
	t.Parallel()

	data := append(upper("Before"), opNewline, opNewline, opNewline)
	data = append(data, upper("After")...)
	instructions, err := parseExecutable(testField(data), nil, false)
	require.NoError(t, err)
	body, err := instructionsToBody(instructions, semantic.ActionKindSay, testTargetResolver)
	require.NoError(t, err)
	require.Len(t, body, 1)
	speech, ok := body[0].(*semantic.SayAction)
	require.True(t, ok)
	require.Len(t, speech.Content, 5)
	assert.IsType(t, &semantic.TextPart{}, speech.Content[0])
	assert.IsType(t, &semantic.LineBreakPart{}, speech.Content[1])
	assert.IsType(t, &semantic.LineBreakPart{}, speech.Content[2])
	assert.IsType(t, &semantic.LineBreakPart{}, speech.Content[3])
	assert.IsType(t, &semantic.TextPart{}, speech.Content[4])
}

func TestInstructionsToBodyPreservesFalseFallthroughAndTerminalRules(t *testing.T) {
	t.Parallel()

	instructions, err := parseExecutable(testField(append([]byte{opIfKarma, 25, firstQuestionID}, upper("Fallback")...)), nil, false)
	require.NoError(t, err)
	body, err := instructionsToBody(instructions, semantic.ActionKindSay, testTargetResolver)
	require.NoError(t, err)
	require.Len(t, body, 2)
	branch, ok := body[0].(*semantic.BranchAction)
	require.True(t, ok)
	assert.Equal(t, 25, branch.Condition.(*semantic.KarmaAtLeastCondition).Minimum)
	assert.IsType(t, &semantic.AskAction{}, branch.Then[0])
	assert.IsType(t, &semantic.SayAction{}, body[1])

	terminal, err := parseExecutable(testField(append([]byte{opEnd}, upper("leak")...)), nil, false)
	require.NoError(t, err)
	_, err = instructionsToBody(terminal, semantic.ActionKindSay, testTargetResolver)
	require.ErrorContains(t, err, "bytes follow a terminal semantic action")

	join, err := parseExecutable(testField([]byte{opJoinParty}), nil, false)
	require.NoError(t, err)
	joinBody, err := instructionsToBody(join, semantic.ActionKindSay, testTargetResolver)
	require.NoError(t, err)
	require.Len(t, joinBody, 1)
	assert.IsType(t, &semantic.JoinPartyAction{}, joinBody[0])
	assert.True(t, bodyIsTerminal(joinBody))
}

func TestResponseAliasesFlattenPointerRelativeIncludes(t *testing.T) {
	t.Parallel()

	where := sourceLocation{filename: "SYNTH.TLK", conversationID: 1}
	standard := []rawPair{
		{key: rawField{data: upper("one"), where: where}, response: rawField{data: []byte{opIncludeNext}, where: where}},
		{key: rawField{data: upper("two"), where: where}, response: rawField{data: upper("Answer"), where: where}},
	}
	groups, auditEntries, err := convertResponseGroups(standard, nil, testTargetResolver)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Empty(t, auditEntries)
	assert.Equal(t, []string{"one", "two"}, groups[0].keywords)

}

func TestResponseAliasesRejectMissingTargetsAndControlValuedKeys(t *testing.T) {
	t.Parallel()

	where := sourceLocation{filename: "SYNTH.TLK", conversationID: 1}
	_, err := flattenResponses([]rawPair{{
		key: rawField{data: upper("one"), where: where}, response: rawField{data: []byte{opIncludeNext}, where: where},
	}}, nil, []bool{true})
	require.ErrorContains(t, err, "has no following response")

	groups, auditEntries, err := convertResponseGroups([]rawPair{
		{key: rawField{data: upper("one"), where: where}, response: rawField{data: []byte{opPause}, where: where}},
		{key: rawField{data: []byte{opIncludeNext}, where: where}, response: rawField{data: upper("Answer"), where: where}},
	}, nil, testTargetResolver)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Len(t, auditEntries, 1)
	assert.Equal(t, AuditReasonUnreachableMatchKey, auditEntries[0].Reason)
}

func TestConvertResponseGroupsPreservesOrphanedQuestionTransfer(t *testing.T) {
	t.Parallel()

	where := sourceLocation{filename: "TOWNE.TLK", conversationID: 23}
	marker := rawField{data: []byte{questionMarker}, offset: 0x36c6, where: where}
	pairs := []rawPair{{
		key:                      rawField{data: upper("ask"), offset: 0x3674, where: where},
		response:                 rawField{data: []byte{firstQuestionID}, offset: 0x36c7, where: where},
		unrenderableOutputMarker: &marker,
	}}

	groups, auditEntries, err := convertResponseGroups(pairs, nil, testTargetResolver)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, []string{"ask"}, groups[0].keywords)
	require.Len(t, groups[0].body, 1)
	ask, ok := groups[0].body[0].(*semantic.AskAction)
	require.True(t, ok)
	assert.Equal(t, "question_01", ask.Question)
	require.Len(t, auditEntries, 1)
	assert.Equal(t, AuditReasonUnrenderableControlOutput, auditEntries[0].Reason)
	assert.Equal(t, 0x36c6, auditEntries[0].KeyOffset)
}

func TestDecodeMatchKeyRejectsExecutableAndCompressedBytes(t *testing.T) {
	t.Parallel()

	for _, data := range [][]byte{{opPause}, {0x01}, {0x9f}} {
		_, reachable, err := decodeMatchKey(testField(data))
		require.NoError(t, err)
		assert.False(t, reachable)
	}
	key, reachable, err := decodeMatchKey(testField(upper("MiXeD key")))
	require.NoError(t, err)
	assert.True(t, reachable)
	assert.Equal(t, "MiXeD key", key)
}

func TestSemanticItemMappingIsExhaustiveForVerifiedValues(t *testing.T) {
	t.Parallel()

	for id := byte(0); id < 48; id++ {
		item, err := semanticItem(id)
		require.NoError(t, err)
		assert.NotEmpty(t, item)
	}
	for id := byte(0x41); id <= 0x4b; id++ {
		item, err := semanticItem(id)
		require.NoError(t, err)
		assert.NotEmpty(t, item)
	}
	_, err := semanticItem(0x40)
	require.Error(t, err)
}

func TestConvertConversationSupportsFallbackOnlyAndTerminalPromptQuestions(t *testing.T) {
	t.Parallel()

	record := append(fixedFields("Name", "Description", "Greeting", "Job", "Bye"),
		questionMarker, firstQuestionID)
	record = appendField(record, upper("Question?"))
	record = appendField(record, upper("Fallback."))
	record = append(record, questionMarker, firstQuestionID+1)
	record = appendField(record, append(upper("Done."), opReturnTopics))
	record = append(record, questionMarker, lastQuestionID, recordEndByte)
	raw, err := parseRecord("SYNTH.TLK", 1, 0, record)
	require.NoError(t, err)

	conversation, auditEntries, err := convertConversation(raw, semantic.OriginalSourceCastle, nil)
	require.NoError(t, err)
	assert.Empty(t, auditEntries)
	require.NoError(t, semantic.Validate(conversation))
	assert.Empty(t, conversation.Questions["question_01"].Answers)
	assert.Empty(t, conversation.Questions["question_02"].Answers)
	assert.Empty(t, conversation.Questions["question_02"].Default)
	assert.IsType(t, &semantic.ReturnToTopicsAction{}, conversation.Questions["question_02"].Prompt[1])
}

func TestConvertConversationSupportsPromptOnlyJoinQuestion(t *testing.T) {
	t.Parallel()

	record := append(fixedFields("Name", "Description", "Greeting", "Job", "Bye"),
		questionMarker, firstQuestionID)
	record = appendField(record, append(upper("Join?"), opJoinParty))
	record = append(record, questionMarker, lastQuestionID, recordEndByte)
	raw, err := parseRecord("SYNTH.TLK", 1, 0, record)
	require.NoError(t, err)

	conversation, auditEntries, err := convertConversation(raw, semantic.OriginalSourceCastle, nil)
	require.NoError(t, err)
	assert.Empty(t, auditEntries)
	require.NoError(t, semantic.Validate(conversation))
	question := conversation.Questions["question_01"]
	assert.Empty(t, question.Answers)
	assert.Empty(t, question.Default)
	assert.IsType(t, &semantic.JoinPartyAction{}, question.Prompt[len(question.Prompt)-1])
}

func TestRemoveShadowedKeywordsPreservesFirstMatchOrder(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{})
	assert.Equal(t, []string{"True", "other"}, removeShadowedKeywords([]string{"True", "other"}, seen))
	assert.Equal(t, []string{"master"}, removeShadowedKeywords([]string{" true ", "master"}, seen))
}

func testField(data []byte) rawField {
	return rawField{
		data: data, offset: 0x100,
		where: sourceLocation{filename: "SYNTH.TLK", conversationID: 7},
	}
}

func testTargetResolver(entry instruction) (semantic.Body, error) {
	if entry.target == opReturnTopics {
		return semantic.Body{&semantic.ReturnToTopicsAction{}}, nil
	}

	return semantic.Body{&semantic.AskAction{Question: questionName(entry.target)}}, nil
}
