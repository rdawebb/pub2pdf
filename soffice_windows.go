//go:build windows

package main

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// soffice.com is the console front-end: it waits for the conversion and
// forwards its output; soffice.exe detaches, returning before the document is
// done, so it is only a fallback for odd installations
var sofficeCommandNames = []string{"soffice.com", "soffice.exe"}

// sofficeCandidatePaths lists where LibreOffice usually lands, since the
// installer doesn't add itself to PATH
func sofficeCandidatePaths() []string {
	var dirs []string
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "ProgramW6432"} {
		if base := os.Getenv(env); base != "" {
			dirs = append(dirs, filepath.Join(base, "LibreOffice"))
		}
	}
	// Per-user installs, which need no administrator to set up
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		dirs = append(dirs, filepath.Join(local, "Programs", "LibreOffice"))
	}

	var paths []string
	for _, dir := range dirs {
		for _, name := range sofficeCommandNames {
			paths = append(paths, filepath.Join(dir, "program", name))
		}
	}

	// A custom install directory is only discoverable from the registry, where
	// the installer records the launcher under App Paths
	if registered := registeredSofficePath(); registered != "" {
		paths = append(paths, registered)
		// The registry names soffice.exe; the .com sits beside it
		paths = append(paths, filepath.Join(filepath.Dir(registered), "soffice.com"))
	}
	return paths
}

func registeredSofficePath() string {
	const appPaths = `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\soffice.exe`
	for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		key, err := registry.OpenKey(root, appPaths, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		value, _, err := key.GetStringValue("")
		key.Close()
		if err == nil && value != "" {
			return value
		}
	}
	return ""
}
