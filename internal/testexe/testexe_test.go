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
	"testing"

	"github.com/dolmen-go/goeval/internal/testexe"
)

// See other tests in the 'echo' package.
// See cover.sh to run echo's testsuite with coverage for both . and ./echo.

func ExampleMain_Command() {
	echo := testexe.Main{
		PackagePath: "./echo",
	}

	cmd, cleanup := echo.Command("-stdout", "foo")
	defer cleanup()

	cmd.Stdout = os.Stdout

	cmd.Run()

	// Output:
	// foo
}

func TestMain_TestCommand(t *testing.T) {
	exampleMain_TestCommand(t)
}

func exampleMain_TestCommand(t *testing.T) {
	echo := testexe.Main{
		PackagePath: "./echo",
	}

	cmd := echo.TestCommand(t, "-stdout", "foo")

	var buf bytes.Buffer
	cmd.Stdout = &buf

	err := cmd.Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.String() != "foo\n" {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}
