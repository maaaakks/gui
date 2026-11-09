//go:build windows

package main

import (
	"strings"
	"syscall"
	"unsafe"
)

var storyUser32 = syscall.NewLazyDLL("user32.dll")
var storyForeground = storyUser32.NewProc("GetForegroundWindow")
var storyWindowText = storyUser32.NewProc("GetWindowTextW")
var storyKeyState = storyUser32.NewProc("GetAsyncKeyState")
var storyXInput = syscall.NewLazyDLL("xinput1_4.dll").NewProc("XInputGetState")

type storyGamepad struct {
	buttons                            uint16
	leftTrigger, rightTrigger          byte
	thumbLX, thumbLY, thumbRX, thumbRY int16
}
type storyController struct {
	packet  uint32
	gamepad storyGamepad
}

// Only inspect the Flow key while Rocket League is the foreground window.
// v5 currently omits UseItem from human LastInput packets, so the default
// keyboard R / controller left-stick button needs a local input fallback.
func storyFlowButtonDown() bool {
	hwnd, _, _ := storyForeground.Call()
	if hwnd == 0 {
		return false
	}
	title := make([]uint16, 256)
	storyWindowText.Call(hwnd, uintptr(unsafe.Pointer(&title[0])), uintptr(len(title)))
	if !strings.HasPrefix(syscall.UTF16ToString(title), "Rocket League") {
		return false
	}
	key, _, _ := storyKeyState.Call(uintptr('R'))
	if key&0x8000 != 0 {
		return true
	}
	if storyXInput.Find() != nil {
		return false
	}
	for i := 0; i < 4; i++ {
		var state storyController
		result, _, _ := storyXInput.Call(uintptr(i), uintptr(unsafe.Pointer(&state)))
		if result == 0 && state.gamepad.buttons&0x0040 != 0 {
			return true
		}
	}
	return false
}
