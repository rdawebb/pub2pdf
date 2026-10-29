# pub2pdf

A single-file executable for Windows, macOS and Linux to convert Microsoft Publisher
files to PDF, due to Microsoft Publisher reaching end of life in October 2026.

This is based on `Convert-PubFileToPDF.ps1`, the not very user-friendly PowerShell
script that Microsoft recommends, but is much simpler to use.

Copy `pub2pdf.exe` (or equivalent for Mac/Linux) into a folder containing Publisher
files, and double-click it to create a `.pdf` copy of every `.pub` file in the folder.

Nothing needs to be installed to run it except a program that can read `.pub`,
no Python, no .NET, no Go runtime, no PowerShell execution-policy changes.

## Requirements

- Windows (64-bit) 10-11, macOS 12 Monterey or later, or Linux (kernel 3.2 or later)
- **Microsoft Publisher or LibreOffice installed.** `.pub` is a proprietary
  format, so an app that can read them is required. Publisher is used when installed,
  automated through COM, otherwise the tool falls back to LibreOffice, run headlessly.
  On macOS and Linux only LibreOffice is used.

If you are using this after October 2026 and you don't have a working version of Publisher,
the only option is to download and use [LibreOffice](https://www.libreoffice.org/).

### Which converter is used

Publisher is tried first and then LibreOffice if it is missing. Publisher renders
its own format faithfully and LibreOffice's `.pub` import (libmspub) is approximate,
so text frames, fonts and complex layouts can shift. The final summary says when the
fallback is used, so the PDFs can be checked.

LibreOffice is found on `PATH`, in the usual `Program Files` locations, or through
the install path the installer records in the registry. It works using a private,
temporary user profile, so conversion works even while LibreOffice is open on the
desktop.

`--libreoffice` forces it in preference to Publisher.

## Installation

### 1. Download the release (recommended)

Download `pub2pdf.exe` (or Mac/Linux equivalent) from the
[latest release](https://github.com/rdawebb/pub2pdf/releases/latest) and copy it
into the folder with your Publisher files. Nothing else is needed.

### 2. `go install`

With [Go](https://go.dev/dl/) 1.26 or later installed:

```sh
go install github.com/rdawebb/pub2pdf@latest
```

This puts `pub2pdf.exe` in `%USERPROFILE%\go\bin` on Windows, or `pub2pdf` in
`~/go/bin` on macOS and Linux. Copy it next to your Publisher files, or run it with
a folder path, as it converts the folder it sits in by default.

### 3. Build from source

Requires [Git](https://git-scm.com/) and [Go](https://go.dev/dl/) 1.26 or later.

From macOS or Linux:

```sh
git clone https://github.com/rdawebb/pub2pdf.git
cd pub2pdf
./build.sh            # for this machine
./build.sh windows    # or mac / linux
```

Set `GOARCH` to override the architecture, e.g. `GOARCH=amd64 ./build.sh mac` for
an Intel Mac, or `GOARCH=arm64 ./build.sh windows` for Windows on ARM.

On Windows, `./build.sh` works in Git Bash or WSL. From PowerShell, run the two
steps directly:

```powershell
git clone https://github.com/rdawebb/pub2pdf.git
cd pub2pdf
go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.7.0 -64 -manifest=pub2pdf.manifest -icon=icon.ico -o resource_windows_amd64.syso versioninfo.json
$env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -trimpath -o pub2pdf.exe .
```

Either way, this writes `pub2pdf.exe` (or `pub2pdf-mac` / `pub2pdf-linux`) into the
repo folder.

## First run on Windows

The exe has been submitted to Microsoft for verification, so it should download and
run with no issues, but occasionally the first launch may raise: **"Windows protected
your PC — Microsoft Defender SmartScreen prevented an unrecognised app from starting"**

In that case click **More info → Run anyway**. It should only be asked once per
copy of the file, if at all, but in most cases it downloads and works just fine and
this isn't a concern.

## Usage

| Command                         | Effect                                             |
| ------------------------------- | -------------------------------------------------- |
| double-click                    | convert `.pub` files sitting next to the exe       |
| `pub2pdf.exe -r`                | include subdirectories                             |
| `pub2pdf.exe --overwrite`       | replace existing PDFs instead of skipping          |
| `pub2pdf.exe "C:\Docs"`         | convert a different folder                         |
| `pub2pdf.exe "C:\Docs\one.pub"` | convert a single file                              |
| `pub2pdf.exe --libreoffice`     | use LibreOffice even if Publisher is installed     |
| `pub2pdf.exe --no-pause`        | don't wait for a keypress (for scripts/scheduling) |
| `pub2pdf.exe --no-dialog`       | don't show the summary pop-up                      |

Flags and the path may be given in any order. Exit code is `0` on success, `1`
if any file failed, `2` for a bad command line.

When double-clicked, a standard Windows message box reports how many files were
converted, skipped and failed (naming the failures), and the program exits when
it is dismissed. `--no-dialog` falls back to behaviour of holding the console window
open until Enter is pressed. When run from an existing terminal neither happens, and the
output stays in the shell and the program exits immediately.

## Differences from the PowerShell script

- **No `-Filter` argument.** The folder is implied by where the exe sits, which
  is what makes double-clicking work. It resolves its own location rather than
  the shell's working directory, so it behaves the same however it is launched.
- **An existing PDF is a skip, not an error,** and is counted separately. The
  original script counted it as a failure, so re-running a finished folder reported
  everything as errors. `--overwrite` forces re-export.
- **Publisher is not launched at all** if every document already has a PDF.
- **Publisher always shuts down.** In the original, a `Continue` after a failed
  open skipped `$doc.Close()`, and an early `exit 1` bypassed the `finally`
  block, and both could leave a stray `PUBLISHER.EXE` running. Here the document
  close and the app quit are deferred, so they run on every path.
- **Case-insensitive matching**, so `.PUB` files are picked up too.
- **LibreOffice fallback.** The script reported an error per file and stopped
  when Publisher was missing. This converts with LibreOffice instead if it is
  installed.
