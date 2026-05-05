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
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/dolmen-go/goeval/internal/playmock"
	"github.com/dolmen-go/goeval/internal/testexe"
)

const userAgent = "goeval.share.test/v0.0.0 (github.com/dolmen-go/goeval/sub/share_test)"

var share testexe.Main

// TestMock starts a mock of play.golang.org exposed to main via a proxy.
func TestMock(t *testing.T) {
	t.Parallel()

	const expectedInput = "package main\nimport \"fmt\"\nfunc main() {\n  fmt.Println(\"Hello, world!\")\n}\n"
	const expectedID = "Znr6hsgvdlB"
	const expectedOutput = "https://go.dev/play/p/" + expectedID + "\n"

	mock := &playmock.Mock{
		Share: func(req *playmock.ShareRequest) (*playmock.ShareResponse, error) {
			if req.Body != expectedInput {
				return nil, errors.New("unexpected input")
			}

			return &playmock.ShareResponse{
				ID: expectedID,
			}, nil
		},
	}

	proxyURL, caPEM := playmock.TestRunProxy(t, "https://play.golang.org", mock.Handler())

	env := append(
		os.Environ(),
		playmock.ProxyEnv(t, proxyURL, caPEM)...,
	)

	t.Logf("Proxy started at %v", proxyURL)

	cmd := share.TestCommand(t, userAgent)
	cmd.Env = env
	cmd.Stdin = strings.NewReader(expectedInput)

	share.Locked(t)

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

	if cap.Stdout != expectedOutput {
		t.Log("Expected:", expectedOutput)
		t.Error("Got:", cap.Stdout)
	}
}
