//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	procPeekNamedPipe = kernel32.NewProc("PeekNamedPipe")
)

// hasStdinData checks if standard input is a pipe that currently has data available.
func hasStdinData() bool {
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeNamedPipe == 0) {
		return false
	}
	var totalBytes uint32
	r1, _, _ := procPeekNamedPipe.Call(
		os.Stdin.Fd(),
		0, 0, 0,
		uintptr(unsafe.Pointer(&totalBytes)),
		0,
	)
	return r1 != 0 && totalBytes > 0
}
