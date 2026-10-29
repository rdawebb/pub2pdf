package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// A document that hangs LibreOffice would otherwise hang the whole run, and a
// headless soffice has no window to cancel from
const libreOfficeTimeout = 5 * time.Minute

// findLibreOffice returns the launcher to run, looking on PATH first so an
// explicitly installed or portable copy wins over the default location
func findLibreOffice() (string, error) {
	for _, name := range sofficeCommandNames {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	for _, candidate := range sofficeCandidatePaths() {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("LibreOffice not found")
}

func exportWithLibreOffice(soffice, profile, pubPath, pdfPath string) error {
	// LibreOffice writes <name>.pdf into --outdir, which is exactly pdfPath
	before, statErr := os.Stat(pdfPath)

	ctx, cancel := context.WithTimeout(context.Background(), libreOfficeTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, soffice,
		"-env:UserInstallation="+fileURL(profile),
		"--headless",
		"--norestore",
		"--convert-to", "pdf",
		"--outdir", filepath.Dir(pdfPath),
		pubPath,
	)
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return fmt.Errorf("LibreOffice gave up after %s", libreOfficeTimeout)
	}
	if err != nil {
		return fmt.Errorf("LibreOffice failed: %v%s", err, indentOutput(output))
	}

	// soffice reports success even when it has written nothing, so trust the
	// file rather than the exit code
	after, err := os.Stat(pdfPath)
	if err != nil {
		return fmt.Errorf("LibreOffice produced no file%s", indentOutput(output))
	}
	if statErr == nil && after.ModTime().Equal(before.ModTime()) && after.Size() == before.Size() {
		return fmt.Errorf("LibreOffice left the existing PDF untouched%s", indentOutput(output))
	}
	return nil
}

// indentOutput folds soffice's chatter onto the end of an error message, or
// contributes nothing when it had nothing to say
func indentOutput(output []byte) string {
	if fields := strings.Fields(string(output)); len(fields) > 0 {
		return ": " + strings.Join(fields, " ")
	}
	return ""
}

// fileURL builds the file:// URL LibreOffice wants for -env:UserInstallation,
// including on Windows, where the path is C:\... and the URL is not
func fileURL(path string) string {
	slashed := filepath.ToSlash(path)
	if !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed
	}
	u := url.URL{Scheme: "file", Path: slashed}
	return u.String()
}
