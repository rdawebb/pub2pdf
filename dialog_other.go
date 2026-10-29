//go:build !windows

package main

// showDialog is Windows-only; elsewhere the summary has already been printed
func showDialog(title, text string, problem bool) {}
