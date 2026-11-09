package terminal_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/PixelEzz/minepulse/internal/terminal"
)

func TestUIChipPlain(t *testing.T) {
	buf := new(bytes.Buffer)
	ui := terminal.New(buf, buf, false, false)
	ui.UseColors = false

	chip := ui.Chip(terminal.LevelOK)
	if chip != "[OK]" {
		t.Errorf("expected [OK], got %q", chip)
	}

	chip = ui.Chip(terminal.LevelWarn)
	if chip != "[WARN]" {
		t.Errorf("expected [WARN], got %q", chip)
	}
}

func TestUIChipColor(t *testing.T) {
	buf := new(bytes.Buffer)
	ui := terminal.New(buf, buf, false, false)
	ui.UseColors = true

	chip := ui.Chip(terminal.LevelOK)
	if !strings.Contains(chip, "\033[") {
		t.Errorf("expected ANSI escape code in chip, got %q", chip)
	}
}

func TestUIFooter(t *testing.T) {
	buf := new(bytes.Buffer)
	ui := terminal.New(buf, buf, false, false)
	ui.PrintFooter("v0.1.0")

	out := buf.String()
	if !strings.Contains(out, "minepulse v0.1.0 • Built by PixelEzz") {
		t.Errorf("footer missing expected text: %s", out)
	}
}
