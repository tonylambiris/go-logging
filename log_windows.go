// +build windows
// Copyright 2013, Örjan Persson. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

import (
	"bytes"
	"io"
	"log"
	"syscall"
)

var (
	kernel32DLL                 = syscall.NewLazyDLL("kernel32.dll")
	setConsoleTextAttributeProc = kernel32DLL.NewProc("SetConsoleTextAttribute")
)

type color int
type word uint16

// Character attributes
// Note:
// -- The attributes are combined to produce various colors (e.g., Blue + Green will create Cyan).
//    Clearing all foreground or background colors results in black; setting all creates white.
// See https://msdn.microsoft.com/en-us/library/windows/desktop/ms682088(v=vs.85).aspx#_win32_character_attributes.
const (
	fgBlack     word = 0x0000
	fgBlue      word = 0x0001
	fgGreen     word = 0x0002
	fgCyan      word = 0x0003
	fgRed       word = 0x0004
	fgMagenta   word = 0x0005
	fgYellow    word = 0x0006
	fgWhite     word = 0x0007
	fgIntensity word = 0x0008
	fgMask      word = 0x000F
)

var (
	colors = []word{
		INFO:     fgWhite,
		CRITICAL: fgMagenta,
		ERROR:    fgRed,
		WARNING:  fgYellow,
		NOTICE:   fgGreen,
		DEBUG:    fgCyan,
	}
	boldcolors = []word{
		INFO:     fgWhite | fgIntensity,
		CRITICAL: fgMagenta | fgIntensity,
		ERROR:    fgRed | fgIntensity,
		WARNING:  fgYellow | fgIntensity,
		NOTICE:   fgGreen | fgIntensity,
		DEBUG:    fgCyan | fgIntensity,
	}
)

type file interface {
	Fd() uintptr
}

// LogBackend utilizes the standard log module.
type LogBackend struct {
	Logger *log.Logger
	Color  bool

	// f is set to a non-nil value if the underlying writer which logs writes to
	// implements the file interface. This makes us able to colorise the output.
	f file
}

// NewLogBackend creates a new LogBackend.
func NewLogBackend(out io.Writer, prefix string, flag int) *LogBackend {
	b := &LogBackend{Logger: log.New(out, prefix, flag)}

	// Unfortunately, the API used only takes an io.Writer where the Windows API
	// need the actual fd to change colors.
	if f, ok := out.(file); ok {
		b.f = f
	}

	return b
}

func (b *LogBackend) Log(level Level, calldepth int, rec *Record) error {
	if b.Color && b.f != nil {
		buf := &bytes.Buffer{}
		setConsoleTextAttribute(b.f, colors[level])
		buf.Write([]byte(rec.Formatted(calldepth + 1)))
		// For some reason, the Go logger arbitrarily decided "2" was the correct
		// call depth...
		err := b.Logger.Output(calldepth+2, buf.String())
		setConsoleTextAttribute(b.f, fgWhite)
		return err
	}
	return b.Logger.Output(calldepth+2, rec.Formatted(calldepth+1))
}

// setConsoleTextAttribute sets the attributes of characters written to the
// console screen buffer by the WriteFile or WriteConsole function.
// See http://msdn.microsoft.com/en-us/library/windows/desktop/ms686047(v=vs.85).aspx.
func setConsoleTextAttribute(f file, attribute word) error {
	r1, r2, err := setConsoleTextAttributeProc.Call(f.Fd(), uintptr(attribute), 0)
	use(attribute)
	return checkError(r1, r2, err)
}

// checkError evaluates the results of a Windows API call and returns the error if it failed.
func checkError(r1, r2 uintptr, err error) error {
	// Windows APIs return non-zero to indicate success
	if r1 != 0 {
		return nil
	}

	// Return the error if provided, otherwise default to EINVAL
	if err != nil {
		return err
	}
	return syscall.EINVAL
}

// use is a no-op, but the compiler cannot see that it is.
// Calling use(p) ensures that p is kept live until that point.
func use(p interface{}) {}

func doFmtVerbLevelColor(layout string, level Level, output io.Writer) {
}
