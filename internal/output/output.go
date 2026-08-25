package output

import (
	"fmt"
	"os"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

var NoColor bool

func colorize(color, text string) string {
	if NoColor {
		return text
	}
	return color + text + colorReset
}

// Info prints an informational log line to stderr (won't pollute piped stdout)
func Info(msg string) {
	fmt.Fprintln(os.Stderr, colorize(colorCyan, "[INFO] ")+msg)
}

func Infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorize(colorCyan, "[INFO] ")+format, args...)
}

// Success prints a success log line to stderr
func Success(msg string) {
	fmt.Fprintln(os.Stderr, colorize(colorGreen, "[+] ")+msg)
}

func Successf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorize(colorGreen, "[+] ")+format, args...)
}

// Warn prints a warning to stderr
func Warn(msg string) {
	fmt.Fprintln(os.Stderr, colorize(colorYellow, "[WARN] ")+msg)
}

func Warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorize(colorYellow, "[WARN] ")+format, args...)
}

// Errorf prints an error to stderr
func Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorize(colorRed, "[ERR] ")+format, args...)
}

// Result prints a matched URL to stdout (the only thing going to STDOUT for pipe compatibility)
func Result(url string) {
	fmt.Println(url)
}

// Dorkf prints a dork banner line to stderr
func Dorkf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorize(colorBold, "[DORK] ")+format, args...)
}

// Gray prints dimmed text to stderr (used for stats / debug)
func Grayf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorize(colorGray, "")+format+colorReset, args...)
}
