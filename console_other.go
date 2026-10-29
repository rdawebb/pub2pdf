//go:build !windows

package main

// consoleProcessCount: outside Windows the program runs from a terminal that
// outlives it, so report a shared console and skip the pause and dialog
func consoleProcessCount() int { return 2 }
