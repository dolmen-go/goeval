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

package testexe_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/dolmen-go/goeval/internal/testexe"
)

var echo = testexe.Main{
	PackagePath: "./echo",
}

func TestCapture(t *testing.T) {
	t.Parallel()

	// testexe.Capture(t.Output(), echo.TestCommand(t, "-stdout", "hello"))
	res, err := testexe.Capture(echo.TestCommand(t, "-stdout", "hello"))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	res.WriteTo(&buf)
	t.Log("\n" + buf.String() + "EOF")
}

func TestLogCapture(t *testing.T) {
	t.Parallel()
	echo.TestLogCapture(t, "-stderr", "err")
}

func TestAssert(t *testing.T) {
	t.Parallel()
	echo.TestAssert(t, "echo/testdata/echo_hello.golden")
}

func TestWriteCaptureCreate(t *testing.T) {
	t.Parallel()
	echo.TestWriteCapture(t, filepath.Join(t.TempDir(), t.Name()+".golden"), "-exit=2", "-stdout=Hello X", "-stderr", "err")
}

func TestWriteCaptureStderr(t *testing.T) {
	t.Parallel()
	echo.TestWriteCapture(t, "echo/testdata/echo_stderr.golden", "-stderr", "err")
}

func TestWriteCaptureExit(t *testing.T) {
	t.Parallel()
	echo.TestWriteCapture(t, "echo/testdata/echo_exit42.golden", "-exit=42", "-stderr=Exit 42")
}
