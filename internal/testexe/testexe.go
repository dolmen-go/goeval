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
		"-buildvcs=false",
		"-trimpath",
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
	m.running.Add(1)

	m.build(log)

	cmd = exec.Command(m.exePath, args...)
	cmd.Env = os.Environ() // Ensure GOCOVERDIR is passed to the execution of the binary

	return cmd, m.Cleanup
}

func (m *Main) Command(args ...string) (cmd *exec.Cmd, cleanup func()) {
	return m.command(os.Stderr, args)
}

func (m *Main) TestCommand(t testing.TB, args ...string) *exec.Cmd {
	cmd, cleanup := m.command(t.Output(), args)
	if cleanup != nil {
		t.Cleanup(cleanup)
	}
	return cmd
}
