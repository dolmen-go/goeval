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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dolmen-go/goeval/internal/testexe"
)

var echo = testexe.Main{
	PackagePath: "./echo",
}

func TestCapture(t *testing.T) {
	echo.UsedBy(t)
	t.Parallel()

	echo.Locked(t)

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
	echo.UsedBy(t)
	t.Parallel()
	echo.TestLogCapture(t, "-stderr", "err")
}

func TestAssert(t *testing.T) {
	echo.UsedBy(t)
	t.Parallel()
	echo.TestAssert(t, "echo/testdata/echo_hello.golden")
}

func TestWriteCaptureCreate(t *testing.T) {
	echo.UsedBy(t)
	t.Parallel()
	echo.TestWriteCapture(t, filepath.Join(t.TempDir(), t.Name()+".golden"), "-exit=2", "-stdout=Hello X", "-stderr", "err")
}

func TestWriteCaptureStderr(t *testing.T) {
	echo.UsedBy(t)
	t.Parallel()
	echo.TestWriteCapture(t, "echo/testdata/echo_stderr.golden", "-stderr", "err")
}

func TestWriteCaptureExit(t *testing.T) {
	echo.UsedBy(t)
	t.Parallel()
	echo.TestWriteCapture(t, "echo/testdata/echo_exit42.golden", "-exit=42", "-stderr=Exit 42")
}

func TestCaptureEnv(t *testing.T) {
	echo.UsedBy(t)
	t.Parallel()

	const (
		envVar1  = "TEST_CAPTURE_ENV_VAR1"
		envVar2  = "TEST_CAPTURE_ENV_VAR2"
		envValue = "42"
	)

	goldenPath := filepath.Clean(filepath.Join(t.TempDir(), t.Name()+".golden"))

	cmdArgs := []string{"-stdout=hello"}
	cmd := echo.TestCommand(t, cmdArgs...)

	// Append env vars in reverse order to check that they are sorted in the capture result.
	cmd.Env = append(os.Environ(), envVar2+"="+envValue, envVar1+"="+envValue)

	echo.Locked(t)

	err := testexe.WriteCapture(cmd, goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(goldenPath)

	content, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("\n" + string(content))
	cap, err := testexe.ParseCapture(bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	if len(cap.Env) != 2 || cap.Env[0] != envVar1+"="+envValue || cap.Env[1] != envVar2+"="+envValue {
		t.Fatal("env mismatch: " + strings.Join(cap.Env, ", "))
	}

	testexe.TestCommandAssert(t, echo.TestCommand(t, cmdArgs...), cap)
}

func TestGoldenEnv(t *testing.T) {
	t.Parallel()

	const golden = "echo/testdata/echo_env.golden"
	/*
		// Initial creation of the golden file:
		cmd := echo.TestCommand(t, "-stdout=OK")
		cmd.Env = append(os.Environ(), "TEST_CAPTURE_ENV_VAR2=42", "TEST_CAPTURE_ENV_VAR1=42")
		err := testexe.WriteCapture(cmd, golden)
		if err != nil {
			t.Fatal(err)
		}
	*/

	echo.TestAssert(t, golden)
}
