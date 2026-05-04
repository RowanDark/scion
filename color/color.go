package color

import (
	"os"

	"golang.org/x/term"
)

var enabled bool

func init() {
	enabled = term.IsTerminal(int(os.Stderr.Fd()))
}

// Disable turns off color output (called when --no-color is set).
func Disable() {
	enabled = false
}

// IsEnabled returns whether color is currently active.
func IsEnabled() bool {
	return enabled
}

const (
	CodeReset  = "\033[0m"
	CodeBold   = "\033[1m"
	CodeCyan   = "\033[36m"
	CodeGreen  = "\033[32m"
	CodeYellow = "\033[33m"
	CodeRed    = "\033[31m"
	CodeWhite  = "\033[97m"
)

// Colorize wraps text in the given ANSI code (no-op when color is disabled).
func Colorize(code, text string) string {
	if !enabled {
		return text
	}
	return code + text + CodeReset
}

func Cyan(text string) string   { return Colorize(CodeCyan, text) }
func Green(text string) string  { return Colorize(CodeGreen, text) }
func Yellow(text string) string { return Colorize(CodeYellow, text) }
func Red(text string) string    { return Colorize(CodeRed, text) }
func Bold(text string) string   { return Colorize(CodeBold, text) }
