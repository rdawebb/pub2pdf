// pub2pdf converts Microsoft Publisher (.pub) files to PDF
//
// Drop the executable into a folder and run it: every .pub beside it becomes a
// .pdf of the same name; reading the proprietary format takes Microsoft
// Publisher (via COM automation) or if not found, headless LibreOffice
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// Microsoft.Office.Interop.Publisher.PbFixedFormatType.pbFixedFormatTypePDF
const pbFixedFormatTypePDF = 2

// S_FALSE is returned by CoInitialize when the thread is already initialised,
// which is a success, not a failure
const sFalse = 0x00000001

type options struct {
	path        string
	recurse     bool
	overwrite   bool
	libreOffice bool
}

const usage = `Convert Microsoft Publisher .pub files to PDF.

Uses Microsoft Publisher when it is installed, otherwise LibreOffice.

  pub2pdf.exe                  convert .pub files in the executable's folder
  pub2pdf.exe -r               also search subdirectories
  pub2pdf.exe --overwrite      replace existing PDFs instead of skipping
  pub2pdf.exe "C:\some\folder" convert a different folder (or a single .pub)
  pub2pdf.exe --libreoffice    use LibreOffice even if Publisher is installed
  pub2pdf.exe --no-pause       don't wait for a keypress on exit
  pub2pdf.exe --no-dialog      don't show the summary pop-up when double-clicked
`

func parseArgs(args []string) (options, error) {
	var opts options
	for _, arg := range args {
		switch arg {
		case "-r", "--recurse":
			opts.recurse = true
		case "--overwrite":
			opts.overwrite = true
		case "--libreoffice", "--libre-office":
			opts.libreOffice = true
		case "--no-pause", "--no-dialog":
			// read by main through hasArg, which still works when parsing fails
		case "-h", "--help":
			return opts, flagHelp
		default:
			if strings.HasPrefix(arg, "-") {
				return opts, fmt.Errorf("unknown option %q", arg)
			}
			if opts.path != "" {
				return opts, errors.New("only one path may be given")
			}
			opts.path = arg
		}
	}
	return opts, nil
}

var flagHelp = errors.New("help requested")

// baseDirectory is the exe's folder, not the working directory, so a
// double-click converts the folder the exe sits in
func baseDirectory() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe), nil
}

func findPubFiles(target string, recurse bool) ([]string, error) {
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		if !strings.EqualFold(filepath.Ext(target), ".pub") {
			return nil, nil
		}
		return []string{target}, nil
	}

	var found []string
	// fs.WalkDir over os.DirFS rather than filepath.WalkDir: the latter Lstats
	// the root, so a target that is a symlink to a folder would yield nothing
	err = fs.WalkDir(os.DirFS(target), ".", func(rel string, d fs.DirEntry, err error) error {
		path := filepath.Join(target, filepath.FromSlash(rel))
		if err != nil {
			if rel == "." {
				return err
			}
			fmt.Printf("WARN  cannot read %s: %v\n", path, err)
			return nil
		}
		if d.IsDir() {
			if rel != "." && !recurse {
				return fs.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".pub") {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.Sort(found)
	return found, nil
}

type result struct {
	succeeded, failed, skipped int
	// failures names the documents that did not convert, for the summary
	failures []string
	// note explains a failure that stopped the whole run rather than one file
	note string
}

// summary is the human-readable outcome, shared by the console line and the
// dialog so the two can never disagree
func (r result) summary() string {
	msg := fmt.Sprintf("Converted %d file(s), %d error(s), %d skipped.",
		r.succeeded, r.failed, r.skipped)

	if r.note != "" {
		msg += "\n\n" + r.note
	}

	const listLimit = 10
	shown := min(len(r.failures), listLimit)
	if shown > 0 {
		msg += "\n\nFailed:\n  " + strings.Join(r.failures[:shown], "\n  ")
		if rest := len(r.failures) - shown; rest > 0 {
			msg += fmt.Sprintf("\n  ...and %d more", rest)
		}
	}
	return msg
}

// plan drops documents whose PDF already exists, before any COM work, so an
// already-converted folder never launches Publisher
func plan(files []string, overwrite bool) (todo []string, skipped int) {
	for _, pubPath := range files {
		if _, err := os.Stat(pdfPathFor(pubPath)); err == nil && !overwrite {
			fmt.Printf("SKIP  %s -> PDF already exists\n", filepath.Base(pubPath))
			skipped++
			continue
		}
		todo = append(todo, pubPath)
	}
	return todo, skipped
}

func pdfPathFor(pubPath string) string {
	return strings.TrimSuffix(pubPath, filepath.Ext(pubPath)) + ".pdf"
}

// exporter converts one document; Publisher and LibreOffice each supply one,
// so runAll's reporting and counting is shared between them
type exporter func(pubPath, pdfPath string) error

// convert prefers Publisher, which renders its own format faithfully, and
// falls back to LibreOffice
func convert(files []string, forceLibreOffice bool) result {
	if !forceLibreOffice {
		res, err := convertWithPublisher(files)
		if err == nil {
			return res
		}
		fmt.Printf("%v\nLooking for LibreOffice instead...\n", err)
	}

	soffice, err := findLibreOffice()
	if err != nil {
		msg := "Neither Microsoft Publisher nor LibreOffice is available.\n" +
			"Install one of them -- .pub files cannot be read without it."
		if forceLibreOffice {
			msg = "LibreOffice was not found."
		}
		fmt.Println("ERROR: " + msg)
		return result{failed: len(files), note: msg}
	}
	fmt.Printf("Using LibreOffice: %s\n\n", soffice)

	// The private profile directory is what makes this safe to run while the
	// user has LibreOffice open: without it a second soffice hands the job to
	// the running instance, which is not headless and may simply refuse
	profile, err := os.MkdirTemp("", "pub2pdf-libreoffice")
	if err != nil {
		msg := fmt.Sprintf("could not create a LibreOffice profile directory: %v", err)
		fmt.Printf("ERROR: %s\n", msg)
		return result{failed: len(files), note: msg}
	}
	defer os.RemoveAll(profile)

	res := runAll(files, func(pubPath, pdfPath string) error {
		return exportWithLibreOffice(soffice, profile, pubPath, pdfPath)
	})
	if !forceLibreOffice {
		res.note = "Microsoft Publisher was not found, so LibreOffice was used instead.\n" +
			"Its Publisher import is approximate -- check the layout of the PDFs."
	}
	return res
}

// convertWithPublisher returns an error only when Publisher can't be started,
// the cue to fall back to LibreOffice; per-document failures go in the result
func convertWithPublisher(files []string) (result, error) {
	// COM apartment state is per-thread, so the goroutine must stay put
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ole.CoInitialize(0); err != nil {
		if oleErr, ok := errors.AsType[*ole.OleError](err); !ok || oleErr.Code() != sFalse {
			return result{}, fmt.Errorf("could not initialise COM: %v", err)
		}
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("Publisher.Application")
	if err != nil {
		return result{}, errors.New("Microsoft Publisher is not installed or accessible.")
	}
	defer unknown.Release()

	app, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return result{}, fmt.Errorf("could not talk to Publisher: %v", err)
	}
	defer app.Release()
	defer oleutil.CallMethod(app, "Quit")

	return runAll(files, func(pubPath, pdfPath string) error {
		return exportOne(app, pubPath, pdfPath)
	}), nil
}

func runAll(files []string, export exporter) result {
	var res result
	for _, pubPath := range files {
		pdfPath := pdfPathFor(pubPath)
		name := filepath.Base(pubPath)

		if err := export(pubPath, pdfPath); err != nil {
			fmt.Printf("FAIL  %s: %v\n", name, err)
			res.failed++
			res.failures = append(res.failures, name)
			continue
		}

		fmt.Printf("OK    %s -> %s\n", name, filepath.Base(pdfPath))
		res.succeeded++
	}
	return res
}

// exportOne opens a single document and exports it, always closing the document
// afterwards so a failure cannot strand it open inside Publisher
func exportOne(app *ole.IDispatch, pubPath, pdfPath string) (err error) {
	opened, err := oleutil.CallMethod(app, "Open", pubPath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	doc := opened.ToIDispatch()
	if doc == nil {
		return errors.New("Publisher returned no document")
	}
	defer func() {
		if _, closeErr := oleutil.CallMethod(doc, "Close"); closeErr != nil && err == nil {
			err = fmt.Errorf("closing document: %w", closeErr)
		}
		doc.Release()
	}()

	if _, err = oleutil.CallMethod(doc, "ExportAsFixedFormat", pbFixedFormatTypePDF, pdfPath); err != nil {
		return fmt.Errorf("exporting: %w", err)
	}
	if _, statErr := os.Stat(pdfPath); statErr != nil {
		return errors.New("export produced no file")
	}
	return nil
}

// run returns the exit code alongside the text to show in the summary dialog
func run() (int, string) {
	opts, err := parseArgs(os.Args[1:])
	if errors.Is(err, flagHelp) {
		fmt.Print(usage)
		return 0, ""
	}
	if err != nil {
		fmt.Printf("ERROR: %v\n\n%s", err, usage)
		return 2, fmt.Sprintf("%v\n\n%s", err, usage)
	}

	fail := func(format string, args ...any) (int, string) {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("ERROR: %s\n", msg)
		return 1, msg
	}

	target := opts.path
	if target == "" {
		if target, err = baseDirectory(); err != nil {
			return fail("could not locate the program's folder: %v", err)
		}
	} else if target, err = filepath.Abs(target); err != nil {
		return fail("bad path: %v", err)
	}

	suffix := ""
	if opts.recurse {
		suffix = " (recursive)"
	}
	fmt.Printf("Scanning %s%s\n\n", target, suffix)

	files, err := findPubFiles(target, opts.recurse)
	if err != nil {
		return fail("%v", err)
	}
	if len(files) == 0 {
		fmt.Println("No .pub files found.")
		return 0, fmt.Sprintf("No .pub files found in\n%s", target)
	}

	todo, skipped := plan(files, opts.overwrite)

	var res result
	if len(todo) > 0 {
		res = convert(todo, opts.libreOffice)
	}
	res.skipped = skipped

	summary := res.summary()
	fmt.Printf("\n%s\n", summary)

	if res.failed > 0 {
		return 1, summary
	}
	return 0, summary
}

func main() {
	code, summary := run()

	// A shell sharing our console will keep the output on screen, so leave it
	// alone; otherwise the text vanishes with the window and the user needs
	// either the dialog or the pause to see what happened.
	if consoleProcessCount() < 2 {
		switch {
		case !hasArg("--no-dialog") && summary != "":
			showDialog("pub2pdf", summary, code != 0)
		case !hasArg("--no-pause") && consoleProcessCount() == 1:
			fmt.Print("\nPress Enter to close...")
			fmt.Scanln()
		}
	}
	os.Exit(code)
}

func hasArg(want string) bool {
	return slices.Contains(os.Args[1:], want)
}
