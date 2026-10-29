//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// NewLazySystemDLL, not syscall.NewLazyDLL: it loads only from System32; the
// default search order starts with the exe's own folder, and this exe gets
// dropped into arbitrary folders, so a stray kernel32.dll there would win
var kernel32 = windows.NewLazySystemDLL("kernel32.dll")

// consoleProcessCount reports how many processes are attached to this console:
// 1 is a double-click (the window closes when we exit), 2+ is a shell that
// keeps the output, 0 is no console at all (a -H windowsgui build)
func consoleProcessCount() int {
	getConsoleProcessList := kernel32.NewProc("GetConsoleProcessList")

	var pids [2]uint32
	count, _, _ := getConsoleProcessList.Call(
		uintptr(unsafe.Pointer(&pids[0])),
		uintptr(len(pids)),
	)
	return int(count)
}
