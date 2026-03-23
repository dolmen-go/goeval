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

package main_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/dolmen-go/goeval/internal/testexe"
)

func Example() {
	var echo testexe.Main
	echo.Verbose = true

	cmd, cleanup := echo.Command("-stdout", "foo")
	defer cleanup()

	cmd.Stdout = os.Stdout

	cmd.Run()

	cmd, cleanup = echo.Command("-stderr", "bar")
	defer cleanup()

	cmd.Stderr = os.Stdout

	cmd.Run()

	cmd, cleanup = echo.Command("-exit", "42")
	defer cleanup()

	err := cmd.Run()
	fmt.Println(err.(*exec.ExitError).ExitCode())

	// Output:
	// foo
	// bar
	// 42
}

var echo = testexe.Main{
	Verbose: true,
}

func TestEchoStdout(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	cmd := echo.TestCommand(t, "-stdout", "foo")
	cmd.Stdout = &buf
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
	out := strings.TrimRight(buf.String(), "\r\n")
	if out != "foo" {
		t.Fatalf(`-stdout: got %q, expected "foo"`, out)
	}
}

func TestEchoStderr(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	cmd := echo.TestCommand(t, "-stderr", "bar")
	cmd.Stderr = &buf
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
	out := strings.TrimRight(buf.String(), "\r\n")
	if out != "bar" {
		t.Fatalf(`-stdout: got %q, expected "bar"`, out)
	}
}

func TestEchoStdin(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	cmd := echo.TestCommand(t, "-stdin")
	cmd.Stdin = strings.NewReader("baz\n")
	cmd.Stdout = &buf
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
	out := strings.TrimRight(buf.String(), "\r\n")
	if out != "baz" {
		t.Fatalf(`-stdout: got %q, expected "baz"`, out)
	}
}
