/*
   Copyright 2026 Olivier Mengué.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package testexe

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// WithCoverage checks if the test is called with active coverage collection.
// This doesn't check that the test binary itself has been built for coverage collection,
// but just that the binary that we'll build must have coverage enabled.
func WithCoverage() bool {
	if coverdir := os.Getenv("GOCOVERDIR"); coverdir != "" {
		fi, err := os.Stat(coverdir)
		return !os.IsNotExist(err) && fi.IsDir()
	}
	return false
}

type Main struct {
	PackagePath string
	BuildArgs   []string
	BuildTags   []string
	BuildEnv    []string
	BuildVCS    bool
	Verbose     bool

	running  atomic.Int32
	building sync.Mutex

	exeDir  string
	exePath string
}

func (m *Main) reset() {
	m.exeDir = ""
	m.exePath = ""
}

func (m *Main) Cleanup() {
	if m.running.Add(-1) > 0 {
		return
	}

	m.building.Lock()
	defer m.building.Unlock()

	if m.exeDir != "" {
		os.RemoveAll(m.exeDir)
		m.reset()
	}
}

// UsedBy increases the reference count for the test duration.
//
// This allows to declare a [Main] in a test, but use it from multiple parallel subtests
// and have the binary deleted only at the end of all subtests.
func (m *Main) UsedBy(tb testing.TB) {
	m.running.Add(1)
	tb.Cleanup(m.Cleanup)
}

func (m *Main) build(log io.Writer) {
	m.building.Lock()
	defer m.building.Unlock()

	if m.exePath != "" {
		if m.Verbose {
			fmt.Fprintf(log, "Using %s...\n", m.exePath)
		}
		return
	}

	exeName := "testexe"

	if m.PackagePath == "" || m.PackagePath == "." {
		m.PackagePath = "."

		// Build the exeName from the directory of the caller
		// Note: we can't just use the caller's package name because it's "main" or "main_test".
		if _, srcFile, _, ok := runtime.Caller(0); ok {
			callers := make([]uintptr, 10)
			nFrames := runtime.Callers(1, callers)
			frames := runtime.CallersFrames(callers[:nFrames])
			for {
				frame, more := frames.Next()
				if frame.File != srcFile {
					exeName = path.Base(path.Dir(frame.File))
					break
				}
				if !more {
					break
				}
			}
		}
	} else {
		m.PackagePath = filepath.ToSlash(m.PackagePath)
		exeName = path.Base(m.PackagePath)
	}

	var err error
	m.exeDir, err = os.MkdirTemp("", exeName+".*")
	if err != nil {
		panic(err)
	}

	built := false

	defer func() {
		if !built {
			os.RemoveAll(m.exeDir)
			m.reset()
		}
	}()

	m.exePath = filepath.Join(m.exeDir, exeName)
	if runtime.GOOS == "windows" {
		m.exePath += ".exe"
	}

	// fmt.Fprintf(os.Stderr, "Building %s...\n", m.exePath)

	argsBuild := []string{
		"build",
		"-o", m.exePath,
		// "-x",
		"-trimpath",
	}

	// Enforce an explicit choice of -buildvcs
	// The default is false, for faster builds
	if m.BuildVCS {
		argsBuild = append(argsBuild, "-buildvcs=true")
	} else {
		argsBuild = append(argsBuild, "-buildvcs=false")
	}

	if len(m.BuildTags) > 0 {
		slices.Sort(m.BuildTags)
		argsBuild = append(argsBuild,
			"-tags", strings.Join(m.BuildTags, ","),
		)
	}

	// If GOCOVERDIR is set and valid, inject "-cover".
	if WithCoverage() {
		if m.Verbose {
			fmt.Fprintf(log, "GOCOVERDIR=%s\n", os.Getenv("GOCOVERDIR"))
		}
		argsBuild = append(argsBuild, "-cover")
	}

	if len(m.BuildArgs) > 0 {
		argsBuild = append(argsBuild, m.BuildArgs...)
	}

	argsBuild = append(argsBuild, m.PackagePath)

	if m.Verbose {
		fmt.Fprintf(log, "Building %s: go %v\n", exeName, argsBuild)
	}

	cmdBuild := exec.Command("go", argsBuild...)
	cmdBuild.Env = append(os.Environ(), m.BuildEnv...)
	cmdBuild.Stdout = os.Stderr
	cmdBuild.Stderr = os.Stderr
	if err := cmdBuild.Run(); err != nil {
		panic(fmt.Errorf("failed to build %s: %w", exeName, err))
	}

	built = true
}

func (m *Main) command(log io.Writer, args []string) (cmd *exec.Cmd, cleanup func()) {
	if logFlush, ok := log.(interface{ Flush() error }); ok {
		defer logFlush.Flush()
	}
	m.running.Add(1)

	m.build(log)

	cmd = exec.Command(m.exePath, args...)
	cmd.Env = os.Environ() // Ensure GOCOVERDIR is passed to the execution of the binary

	return cmd, m.Cleanup
}

func (m *Main) Command(args ...string) (cmd *exec.Cmd, cleanup func()) {
	return m.command(os.Stderr, args)
}

// tbOutput replaces testing.TB.Output() when the test doesn't implement it (Go < 1.25).
type tbOutput struct {
	tb  testing.TB
	buf []byte
}

func (o *tbOutput) Write(p []byte) (n int, err error) {
	o.buf = append(o.buf, p...)
	return len(p), nil
}

func (o *tbOutput) Flush() error {
	if len(o.buf) > 0 {
		o.tb.Log(string(o.buf))
		o.buf = nil
	}
	return nil
}

// TestCommand returns a command to execute the test binary with the given arguments.
func (m *Main) TestCommand(tb testing.TB, args ...string) *exec.Cmd {
	tb.Helper()

	var log io.Writer
	// testing.TB.Output() was added in Go 1.25
	if tbWithOutput, ok := any(tb).(interface{ Output() io.Writer }); ok {
		log = tbWithOutput.Output()
	} else {
		log = &tbOutput{tb: tb}
	}
	cmd, cleanup := m.command(log, args)
	if cleanup != nil {
		tb.Cleanup(cleanup)
	}
	return cmd
}

// TestCapture executes the test binary with the given arguments
// and captures its stdout, stderr and exit status for reproduction.
// See [Capture] for more details.
func (m *Main) TestCapture(tb testing.TB, args ...string) *CaptureResult {
	tb.Helper()

	cmd := m.TestCommand(tb, args...)
	res, err := Capture(cmd)
	if err != nil {
		tb.Fatal(err)
	}
	return res
}

// TestLogCapture executes the test binary with the given arguments
// and logs its captured stdout, stderr and exit status for debugging.
func (m *Main) TestLogCapture(tb testing.TB, args ...string) {
	tb.Helper()

	cmd := m.TestCommand(tb, args...)
	res, err := Capture(cmd)
	if err != nil {
		tb.Fatal(err)
	}
	var buf strings.Builder
	res.WriteTo(&buf)
	tb.Log("\n" + buf.String() + "EOF")
}

// TestWriteCapture executes the test binary with the given arguments
// and writes its captured stdout, stderr and exit status to the given path for reproduction.
// If the file already exists, it is just replayed.
func (m *Main) TestWriteCapture(tb testing.TB, path string, args ...string) {
	tb.Helper()

	osPath := path
	if !filepath.IsAbs(path) {
		osPath = filepath.FromSlash(path)
	}
	_, err := os.Stat(osPath)
	if os.IsNotExist(err) {
		tb.Log("Capturing output to create " + path + "...")
		err := WriteCapture(m.TestCommand(tb, args...), path)
		if err != nil {
			tb.Fatal(err)
		}
	} else if err != nil {
		tb.Fatal(err)
	}

	m.TestAssert(tb, path)
}

// TestAssert executes the test binary with the given script and asserts
// that its stdout, stderr and exit status match the expected values.
func (m *Main) TestAssert(tb testing.TB, path string) {
	tb.Helper()

	if !filepath.IsAbs(path) {
		path = filepath.FromSlash(path)
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		tb.Fatalf("file %s does not exist", path)
	} else if err != nil {
		tb.Fatalf("failed to stat file %s: %v", path, err)
	}
	defer f.Close()

	var expected *CaptureResult
	chanErr := make(chan error)
	go func(chanErr chan<- error) {
		defer close(chanErr)
		chanErr <- func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic: %v", r)
				}
			}()
			expected, err = ParseCapture(f)
			return
		}()
	}(chanErr)

	cmd := m.TestCommand(tb)

	if err := <-chanErr; err != nil {
		tb.Fatalf("failed to parse capture: %v", err)
	}

	TestCommandAssert(tb, cmd, expected)
}
