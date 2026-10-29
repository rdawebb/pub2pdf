//go:build windows

package main

import "golang.org/x/sys/windows"

// showDialog shows a standard message box. TOPMOST and SETFOREGROUND are
// needed because a double-clicked program has no window to own the box, so it
// could open behind whatever the user was looking at; x/sys loads user32 from
// System32 (see kernel32 in console_windows.go)
func showDialog(title, text string, problem bool) {
	icon := uint32(windows.MB_ICONINFORMATION)
	if problem {
		icon = windows.MB_ICONWARNING
	}

	textPtr, err := windows.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	titlePtr, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return
	}

	windows.MessageBox(0, textPtr, titlePtr,
		windows.MB_OK|icon|windows.MB_SETFOREGROUND|windows.MB_TOPMOST)
}
