//go:build windows

package clipboard

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const cfUnicodeText = 13

var (
	errNoText           = errors.New("clipboard does not contain text")
	errInvalidText      = errors.New("clipboard text data is invalid")
	errWindowsAPIFailed = errors.New("windows API call failed without an error code")
)

var (
	user32                = windows.NewLazySystemDLL("user32.dll")
	kernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procOpenClipboard     = user32.NewProc("OpenClipboard")
	procCloseClipboard    = user32.NewProc("CloseClipboard")
	procIsFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procGetClipboardData  = user32.NewProc("GetClipboardData")
	procGlobalLock        = kernel32.NewProc("GlobalLock")
	procGlobalUnlock      = kernel32.NewProc("GlobalUnlock")
	procGlobalSize        = kernel32.NewProc("GlobalSize")
)

// ReadText returns the clipboard's current CF_UNICODETEXT contents.
func ReadText() (text string, err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	opened, _, openErr := procOpenClipboard.Call(0)
	if opened == 0 {
		return "", fmt.Errorf("read clipboard text: OpenClipboard: %w", windowsCallError(openErr))
	}
	defer func() {
		closed, _, closeErr := procCloseClipboard.Call()
		if closed == 0 && err == nil {
			text = ""
			err = fmt.Errorf("read clipboard text: CloseClipboard: %w", windowsCallError(closeErr))
		}
	}()

	available, _, _ := procIsFormatAvailable.Call(cfUnicodeText)
	if available == 0 {
		return "", fmt.Errorf("read clipboard text: CF_UNICODETEXT: %w", errNoText)
	}

	handle, _, dataErr := procGetClipboardData.Call(cfUnicodeText)
	if handle == 0 {
		return "", fmt.Errorf("read clipboard text: GetClipboardData(CF_UNICODETEXT): %w", windowsCallError(dataErr))
	}

	data, _, lockErr := procGlobalLock.Call(handle)
	if data == 0 {
		return "", fmt.Errorf("read clipboard text: GlobalLock: %w", windowsCallError(lockErr))
	}
	defer func() {
		unlocked, _, unlockErr := procGlobalUnlock.Call(handle)
		if unlocked == 0 && !errors.Is(unlockErr, windows.ERROR_SUCCESS) && err == nil {
			text = ""
			err = fmt.Errorf("read clipboard text: GlobalUnlock: %w", unlockErr)
		}
	}()

	byteSize, _, sizeErr := procGlobalSize.Call(handle)
	if byteSize == 0 {
		return "", fmt.Errorf("read clipboard text: GlobalSize: %w", windowsCallError(sizeErr))
	}
	if byteSize%unsafe.Sizeof(uint16(0)) != 0 {
		return "", fmt.Errorf("read clipboard text: odd CF_UNICODETEXT byte length: %w", errInvalidText)
	}

	unitCount := byteSize / unsafe.Sizeof(uint16(0))
	maxInt := int(^uint(0) >> 1)
	if unitCount > uintptr(maxInt) {
		return "", fmt.Errorf("read clipboard text: CF_UNICODETEXT exceeds Go string capacity: %w", errInvalidText)
	}
	units := unsafe.Slice((*uint16)(unsafe.Pointer(data)), int(unitCount))
	terminator := -1
	for i, unit := range units {
		if unit == 0 {
			terminator = i
			break
		}
	}
	if terminator < 0 {
		return "", fmt.Errorf("read clipboard text: unterminated CF_UNICODETEXT: %w", errInvalidText)
	}

	return windows.UTF16ToString(units[:terminator]), nil
}

// Supported reports whether this build has native clipboard access.
func Supported() bool {
	return true
}

func windowsCallError(err error) error {
	if errors.Is(err, windows.ERROR_SUCCESS) {
		return errWindowsAPIFailed
	}
	return err
}
