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
	"os"
	"runtime"
	"testing"

	"github.com/dolmen-go/goeval/internal/testexe"
)

var golden = testexe.Main{
	Verbose: true,
}

func TestUsage(t *testing.T) {
	t.Parallel()

	golden.TestWriteCapture(t, "testdata/usage.golden", "-h")
}

func TestCapture(t *testing.T) {
	t.Parallel()

	golden.TestWriteCapture(t, "testdata/golden-echo."+runtime.GOOS+".golden", os.DevNull, "go", "run", "../echo", "-stdout=OK", "-stderr=err", "-exit=2")
}

func TestReplay(t *testing.T) {
	t.Parallel()

	golden.TestAssert(t, "testdata/replay.golden")
}
