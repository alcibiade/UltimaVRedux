package widgets

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

// Update processes keyboard input for text entry, command execution, and autocomplete
func (t *TextInput) Update() {
	if t.keyboard.IsModifierPressed(input.ModifierCtrl) {
		if t.keyboard.IsKeyPressedAndRegister(ebiten.KeyU) {
			t.output.Clear()
		}

		return
	}

	var boundKey *ebiten.Key
	key, keyStr := t.keyboard.GetAlphaNumericPressed()
	if key != nil {
		if !t.keyboard.TryToRegisterKeyPress(*key) {
			return
		}

		if t.autoReplaceMode {
			if !t.wouldNewCharacterBeValid(keyStr) {
				return
			}
		}

		if t.isPerfectMatchWithAutoFillFirstCharacter() {
			if t.autoReplaceMode {
				if !t.wouldNewCharacterBeValid(keyStr) {
					return
				}
				t.output.Clear()
			} else {
				return
			}
		}

		if t.freeText || t.inputValidationRegex != nil {
			keyStr = t.printableKey(*key, keyStr)
		}
		t.addCharacter(keyStr)

		if t.isPerfectMatchWithAutoFillFirstCharacter() {
			t.tryToAutoComplete(false, true)
		}

		return
	} else {
		boundKey = t.keyboard.GetBoundKeyPressed(&nonAlphaNumericBoundKeys)
		if boundKey == nil {
			t.keyboard.SetAllowKeyPressImmediately()

			return
		}
		if !t.keyboard.TryToRegisterKeyPress(*boundKey) {
			return
		}
	}

	switch *boundKey { //nolint:exhaustive
	case ebiten.KeyApostrophe, ebiten.KeyBackslash, ebiten.KeyBracketLeft,
		ebiten.KeyBracketRight, ebiten.KeyComma, ebiten.KeyEqual,
		ebiten.KeyGraveAccent, ebiten.KeyMinus, ebiten.KeyPeriod,
		ebiten.KeySemicolon, ebiten.KeySlash:
		if t.freeText || t.inputValidationRegex != nil {
			t.addCharacter(t.printableKey(*boundKey, ""))
		}
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		if t.freeText {
			if t.TextInputCallbacks.OnEnter != nil {
				t.TextInputCallbacks.OnEnter()
			}

			return
		}
		// Regex mode: Enter should be handled by InputBox, not here
		if t.textCommands == nil {
			return
		}

		command := t.output.GetOutputStr(false)
		matches := t.textCommands.GetAllPerfectMatches(command)

		commandExecuted := false

		if len(*matches) > 1 {
			// Defense-in-depth: registration-time ValidateNoAmbiguousCommands
			// should prevent this. If we get here, something is wrong.
			logging.Errorf(logging.CategorySystem,
				"Ambiguous debug command: %d perfect matches for %q (should never happen — check ValidateNoAmbiguousCommands)",
				len(*matches), command)
		} else if len(*matches) == 1 {
			firstMatch := (*matches)[0]
			if firstMatch.ExecuteCommand != nil {
				firstMatch.ExecuteCommand(command, &firstMatch)
				t.keyboard.SetLastKeyPressedNowPlusMs(keyPressDelay)
				commandExecuted = true
			}
		}

		if commandExecuted && t.TextInputCallbacks.OnCommandExecuted != nil {
			t.TextInputCallbacks.OnCommandExecuted(command)
		}

		if t.TextInputCallbacks.OnEnter != nil {
			t.TextInputCallbacks.OnEnter()
		}

		return
	case ebiten.KeySpace:
		if t.freeText || t.inputValidationRegex != nil {
			t.addCharacter(" ")
		} else {
			t.tryToAutoComplete(true, false)
		}
	case ebiten.KeyTab:
		if !t.freeText && t.inputValidationRegex == nil {
			t.tryToAutoComplete(true, true)
		}
	case ebiten.KeyBackspace, ebiten.KeyLeft:
		if t.isPerfectMatchWithAutoFillFirstCharacter() {
			t.output.Clear()
		} else {
			t.output.RemoveRightSideCharacter()
		}
	}

	return
}

func (t *TextInput) printableKey(key ebiten.Key, fallback string) string {
	shifted := t.keyboard.IsModifierPressed(input.ModifierShift)
	if key >= ebiten.Key0 && key <= ebiten.Key9 {
		if !shifted {
			return fallback
		}

		shiftedDigits := [...]string{")", "!", "@", "#", "$", "%", "^", "&", "*", "("}

		return shiftedDigits[key-ebiten.Key0]
	}

	punctuationKeys := [...]ebiten.Key{
		ebiten.KeyApostrophe, ebiten.KeyBackslash, ebiten.KeyBracketLeft,
		ebiten.KeyBracketRight, ebiten.KeyComma, ebiten.KeyEqual,
		ebiten.KeyGraveAccent, ebiten.KeyMinus, ebiten.KeyPeriod,
		ebiten.KeySemicolon, ebiten.KeySlash,
	}
	unshiftedValues := [...]string{"'", "\\", "[", "]", ",", "=", "`", "-", ".", ";", "/"}
	shiftedValues := [...]string{"\"", "|", "{", "}", "<", "+", "~", "_", ">", ":", "?"}
	for index, punctuationKey := range punctuationKeys {
		if key != punctuationKey {
			continue
		}
		if shifted {
			return shiftedValues[index]
		}

		return unshiftedValues[index]
	}

	return fallback
}

func (t *TextInput) addCharacter(keyStr string) {
	// Regex choice prompts have no grammar completion to trigger replacement.
	// Validate the replacement before clearing, so an invalid key retains the choice.
	if t.autoReplaceMode && t.inputValidationRegex != nil {
		if t.inputValidationRegex.MatchString(keyStr) {
			t.output.Clear()
			t.output.AppendText(keyStr)
		}

		return
	}
	if t.output.GetLengthOfCurrentRowStr() >= t.maxCharsPerLine {
		return
	}

	if t.inputValidationRegex != nil {
		newText := t.output.GetOutputStr(false) + keyStr
		if !t.inputValidationRegex.MatchString(newText) {
			return
		}
	}

	t.output.AppendText(keyStr)
}

func (t *TextInput) isPerfectMatchWithAutoFillFirstCharacter() bool {
	// Regex mode: textCommands is nil, no grammar matching
	if t.textCommands == nil {
		return false
	}

	perfectMatches := *t.textCommands.GetAllPerfectMatches(t.output.GetOutputStr(false))
	if len(perfectMatches) == 1 {
		if perfectMatches[0].Matches[0].ShouldAutofillWithFirstCharacter() {
			return true
		}
	}

	return false
}

func (t *TextInput) wouldNewCharacterBeValid(keyStr string) bool {
	if t.textCommands == nil {
		return true
	}

	matches := t.textCommands.HowManyCommandsMatch(keyStr)

	return matches > 0
}
