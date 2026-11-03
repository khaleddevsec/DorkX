//go:build !windows

package main

import "os"

// hasStdinData checks if standard input is a pipe that currently has data available.
func hasStdinData() bool {
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeNamedPipe == 0) {
		return false
	}
	return true
}

