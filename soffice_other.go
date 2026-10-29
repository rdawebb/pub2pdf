//go:build !windows

package main

// sofficeCommandNames is what the LibreOffice launcher is called on PATH
var sofficeCommandNames = []string{"soffice", "libreoffice"}

// sofficeCandidatePaths covers the usual macOS and Linux install locations for
// the times LibreOffice is not on PATH
func sofficeCandidatePaths() []string {
	return []string{
		"/Applications/LibreOffice.app/Contents/MacOS/soffice",
		"/usr/bin/soffice",
		"/usr/local/bin/soffice",
		"/opt/libreoffice/program/soffice",
		"/snap/bin/libreoffice",
	}
}
