# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-10-29

Initial release.

### Added

- Single-file Windows executable that converts every `.pub` file in its own folder
  to PDF when double-clicked, with no runtime or PowerShell setup required.
- macOS and Linux builds, which convert with LibreOffice only.
- Release downloads for every supported platform: Windows (`pub2pdf.exe`, and
  `pub2pdf-arm64.exe` for Windows on ARM), Mac (Apple silicon and Intel) and Linux
  (x86-64 and ARM); Mac and Linux downloads are `.tar.gz` archives, so the extracted
  program keeps its permission to run and works as is.
- README table of which download to pick, and how to allow the first run of the
  Mac program, which is not notarised by Apple.
- `build.sh` builds for the current machine or cross-compiles for `mac`, `linux` or
  `windows` from any host, with `GOARCH` to override the architecture. Windows builds
  (amd64 and arm64) embed version info, an application manifest and an icon.
- Conversion through Microsoft Publisher (COM automation), with a headless
  LibreOffice fallback when Publisher is not installed, or forced with `--libreoffice`.
- LibreOffice detection via `PATH`, the usual `Program Files` locations and the
  registry, using a private temporary profile so it works while LibreOffice is open.
- Command-line options: `-r` for subdirectories, `--overwrite`, a folder or single
  file path, `--no-pause` and `--no-dialog`.
- Summary message box of converted, skipped and failed files, with a note when the
  LibreOffice fallback was used, and exit codes `0`, `1` and `2`.
- Existing PDFs are skipped rather than counted as failures, `.pub` matching is
  case-insensitive, and Publisher is always shut down cleanly.

[Unreleased]: https://github.com/rdawebb/pub2pdf/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/rdawebb/pub2pdf/releases/tag/v0.1.0
