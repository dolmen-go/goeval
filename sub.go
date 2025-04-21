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
func prepareSub(appCode string) (stdin *bytes.Buffer, cmd *exec.Cmd, cleanup func()) {
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
	cmd = exec.Command(goCmd, "run", fName, getUserAgent())
	cmd.Env = append(os.Environ(), "GO111MODULE=off") // We must not use the 'env' built for local run here
	cmd.Stdin = stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return
}
