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

// Info prints an informational log line to stderr
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

// Result prints a matched URL to stdout
func Result(url string) {
	fmt.Println(url)
}

// FormatStatus returns a color-coded status code string [200], [301], [403]
func FormatStatus(code int) string {
	codeStr := fmt.Sprintf("[%d]", code)
	if NoColor {
		return codeStr
	}
	switch {
	case code >= 200 && code < 300:
		return colorGreen + codeStr + colorReset
	case code >= 300 && code < 400:
		return colorYellow + codeStr + colorReset
	case code >= 400 && code < 500:
		return colorRed + codeStr + colorReset
	default:
		return colorCyan + codeStr + colorReset
	}
}

// ResultWithStatus prints URL alongside its status code
func ResultWithStatus(url string, code int) {
	fmt.Printf("%s %s\n", url, FormatStatus(code))
}

// Dorkf prints a dork banner line to stderr
func Dorkf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorize(colorBold, "[DORK] ")+format, args...)
}

// Grayf prints dimmed text to stderr
func Grayf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorize(colorGray, "")+format+colorReset, args...)
}
