/*
   Copyright 2025 Olivier Mengué.

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

package main

import (
	"bytes"
	_ "embed"
	"io"
	"log"
	"os"
	"os/exec"
)

var (
	//go:embed sub/play/play.go
	playClient string
	//go:embed sub/share/share.go
	shareClient string
)

// prepareSub prepares execution of a sub command via a "go run".
// The returned stdin buffer may be filled with data.
// cleanup must be called after cmd.Run() to clean the tempoary go source created.
func prepareSub(appCode string) (stdin *bytes.Buffer, tail func() error, cleanup func()) {
	f, err := os.CreateTemp("", "*.go")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	fName := f.Name()
	cleanup = func() {
		os.Remove(fName)
	}

	if _, err := io.WriteString(f, appCode); err != nil {
		log.Fatal(err)
	}

	// Prepare input that will be filled before executing the command
	stdin = new(bytes.Buffer)

	// Run "go run" with the code submitted on stdin and the userAgent as first argument
	cmd := exec.Command(goCmd, "run", fName, getUserAgent())
	cmd.Env = append(os.Environ(), "GO111MODULE=off") // We must not use the 'env' built for local run here
	cmd.Stdin = stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	tail = func() error {
		return run(cmd)
	}
	return
}
