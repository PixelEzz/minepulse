package terminal

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Level indicates alert level for chips.
type Level string

const (
	LevelInfo Level = "INFO"
	LevelWarn Level = "WARN"
	LevelCrit Level = "CRIT"
	LevelOK   Level = "OK"
)

// ANSI Color codes
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	bgRed       = "\033[41m"
	bgGreen     = "\033[42m"
	bgYellow    = "\033[43m"
	bgBlue      = "\033[44m"
	fgWhite     = "\033[97m"
)

// UI handles terminal rendering and output styling.
type UI struct {
	Out       io.Writer
	Err       io.Writer
	JSONMode  bool
	Verbose   bool
	UseColors bool
}

// New creates a new UI instance.
func New(out, err io.Writer, jsonMode, verbose bool) *UI {
	useColor := !jsonMode && isTerminal(out)
	// Honor NO_COLOR environment variable (https://no-color.org)
	if os.Getenv("NO_COLOR") != "" {
		useColor = false
	}

	return &UI{
		Out:       out,
		Err:       err,
		JSONMode:  jsonMode,
		Verbose:   verbose,
		UseColors: useColor,
	}
}

// isTerminal checks if writer is an interactive terminal.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// Chip returns a colored badge/chip for a given level.
func (u *UI) Chip(level Level) string {
	if !u.UseColors {
		return fmt.Sprintf("[%s]", level)
	}

	switch level {
	case LevelOK:
		return fmt.Sprintf("%s%s OK %s", bgGreen, fgWhite, colorReset)
	case LevelWarn:
		return fmt.Sprintf("%s%s WARN %s", bgYellow, fgWhite, colorReset)
	case LevelCrit:
		return fmt.Sprintf("%s%s CRIT %s", bgRed, fgWhite, colorReset)
	case LevelInfo:
		return fmt.Sprintf("%s%s INFO %s", bgBlue, fgWhite, colorReset)
	default:
		return fmt.Sprintf("[%s]", level)
	}
}

// Bold returns text in bold if colors are active.
func (u *UI) Bold(text string) string {
	if !u.UseColors {
		return text
	}
	return colorBold + text + colorReset
}

// Dim returns text dimmed if colors are active.
func (u *UI) Dim(text string) string {
	if !u.UseColors {
		return text
	}
	return colorDim + text + colorReset
}

// Cyan returns text in cyan.
func (u *UI) Cyan(text string) string {
	if !u.UseColors {
		return text
	}
	return colorCyan + text + colorReset
}

// Green returns text in green.
func (u *UI) Green(text string) string {
	if !u.UseColors {
		return text
	}
	return colorGreen + text + colorReset
}

// Yellow returns text in yellow.
func (u *UI) Yellow(text string) string {
	if !u.UseColors {
		return text
	}
	return colorYellow + text + colorReset
}

// Red returns text in red.
func (u *UI) Red(text string) string {
	if !u.UseColors {
		return text
	}
	return colorRed + text + colorReset
}

// PrintFooter renders the official subtle branding footer.
func (u *UI) PrintFooter(version string) {
	if u.JSONMode {
		return
	}
	footer := fmt.Sprintf("\nminepulse %s • Built by PixelEzz (https://github.com/PixelEzz) • Run with --json for raw output\n", version)
	fmt.Fprint(u.Out, u.Dim(footer))
}

// OutputJSON serializes data as indented JSON to Out.
func (u *UI) OutputJSON(data interface{}) error {
	enc := json.NewEncoder(u.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// KeyValue prints a standard aligned key-value pair.
func (u *UI) KeyValue(key string, val interface{}) {
	if u.JSONMode {
		return
	}
	fmt.Fprintf(u.Out, "  %-24s %v\n", u.Dim(key+":"), val)
}

// Header prints a section header.
func (u *UI) Header(title string) {
	if u.JSONMode {
		return
	}
	fmt.Fprintf(u.Out, "\n%s\n%s\n", u.Bold(title), u.Dim(strings.Repeat("─", len(title)+4)))
}
