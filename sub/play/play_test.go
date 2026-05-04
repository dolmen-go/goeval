//go:build !goeval.offline

/*
   Copyright 2025-2026 Olivier Mengué.

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
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/dolmen-go/goeval/internal/playmock"
	"github.com/dolmen-go/goeval/internal/testexe"
)

const userAgent = "goeval.play.test/v0.0.0 (github.com/dolmen-go/goeval/sub/play_test)"

var play testexe.Main

func Example_fmt() {
	cmd, cleanup := play.Command(userAgent)
	defer cleanup()
	cmd.Stdin = strings.NewReader(`package main;import"fmt";func main(){fmt.Println("OK")}`)
	cmd.Stdout = os.Stdout
	cmd.Run()

	// Output:
	// OK
}

func Example_stderr() {
	cmd, cleanup := play.Command(userAgent)
	defer cleanup()
	cmd.Stdin = strings.NewReader(`package main;import("fmt";"os");func main(){fmt.Fprintln(os.Stderr,"OK Err")}`)
	cmd.Stderr = os.Stdout
	cmd.Run()

	// Output:
	// OK Err
}

func Example_time() {
	cmd, cleanup := play.Command(userAgent)
	defer cleanup()
	cmd.Stdin = strings.NewReader(`package main;import("fmt";"time");func main(){fmt.Println(time.Now().Format(time.RFC3339))}`)
	cmd.Stdout = os.Stdout
	cmd.Run()

	// Output:
	// 2009-11-10T23:00:00Z
}

func Example_compileError() {
	cmd, cleanup := play.Command(userAgent)
	defer cleanup()
	cmd.Stdin = strings.NewReader(`@`)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout // We want to capture stderr, so redirect stderr to the Example's stdout
	err := cmd.Run()
	if err, ok := err.(*exec.ExitError); ok {
		if err.ExitCode() != 1 {
			fmt.Println("Status 1 expected, got ", err.ExitCode())
		}
	}

	// Note: the output check is fragile as it relies on the Go compiler output which might change.

	// Output:
	// prog.go:1:1: illegal character U+0040 '@'
}

func Example_exit42() {
	cmd, cleanup := play.Command(userAgent)
	defer cleanup()
	cmd.Stdin = strings.NewReader(`package main;import("fmt";"os");func main(){fmt.Fprintln(os.Stderr,"Err 42");os.Exit(42)}`)
	cmd.Stderr = os.Stdout
	err := cmd.Run()
	if err, ok := err.(*exec.ExitError); ok {
		if err.ExitCode() != 42 {
			fmt.Println("Status 42 expected, got ", err.ExitCode())
		}
	}

	// Output:
	// Err 42
}

// TestMock starts a mock of play.golang.org exposed to main via a proxy.
func TestMock(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"package main\nimport \"fmt\"\nfunc main() {\n  fmt.Println(\"Hello, world!\")\n}\n":                       "Hello, world!\n",
		"package main\nimport (\n  \"fmt\"\n  \"math/rand\"\n)\nfunc main() {\n  fmt.Println(rand.Intn(100))\n}\n": "42\n",
	}

	srv := &playmock.Server{
		Compile: func(req *playmock.CompileRequest) (*playmock.CompileResponse, error) {
			out, ok := tests[req.Body]
			if !ok {
				t.Errorf("unexpected body: got %q", req.Body)
				return nil, fmt.Errorf("unexpected input: %q", req.Body)
			}

			return &playmock.CompileResponse{
				Errors: "",
				Events: []playmock.CompileEvent{
					{Message: out, Kind: "stdout", Delay: 0},
				},
				Status:      0,
				IsTest:      false,
				TestsFailed: 0,
			}, nil
		},
	}

	proxyURL, caPEM := playmock.TestRunProxy(t, "https://play.golang.org", srv.Handler())

	env := append(
		os.Environ(),
		playmock.ProxyEnv(t, proxyURL, caPEM)...,
	)

	t.Logf("Proxy started at %v", proxyURL)

	for in, out := range tests {
		t.Run(out, func(t *testing.T) {
			t.Parallel()

			t.Log("\n" + in)

			cmd := play.TestCommand(t, userAgent)
			cmd.Env = env
			cmd.Stdin = strings.NewReader(in)

			play.Locked(t)

			cap, err := testexe.Capture(cmd)
			if err != nil {
				t.Fatal("capture:", err)
			}

			if cap.ExitStatus != 0 {
				t.Errorf("Exit status: %d", cap.ExitStatus)
			}
			if cap.Stderr != "" {
				t.Error(cap.Stderr)
			}

			if cap.Stdout != out {
				t.Log("Expected:", out)
				t.Error("Got:", cap.Stdout)
			}
		})
	}
}
