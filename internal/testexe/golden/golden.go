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

// Command golden-capture is a helper to capture the output of a command and write it to a golden file.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/dolmen-go/goeval/internal/testexe"
)

func main() {
	if len(os.Args) <= 1 {
		usage()
	}
	// No declared flags for now, but be ready to extend by disallowing
	// a direct golden file whose name starts with '-' (escape with the usual '--')
	if strings.HasPrefix(os.Args[1], "-") {
		if os.Args[1] != "--" {
			usage()
		}
		// --
		os.Args = slices.Delete(os.Args, 1, 2)
	}

	if len(os.Args) == 2 {
		replay()
	} else {
		capture()
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, ""+
		"usage: %s <out.golden> <cmd> [<args>...]\n"+
		"       %[1]s <in.golden>\n"+
		"\n"+
		"With 2 or more arguments, %[1]s captures the output of the given command and\n"+
		"writes it to the given golden file.\n"+
		"With exactly 1 argument, %[1]s replays the given golden file and asserts that\n"+
		"the command's output matches the captured one.\n",
		filepath.Base(os.Args[0]))

	os.Exit(1)
}

func replay() {
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(3)
	}
	defer f.Close()

	res, err := testexe.ParseCapture(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse: %v\n", err)
		os.Exit(4)
	}

	wd, _ := os.Getwd()
	defaultExe := filepath.Join(wd, res.Args[0])
	if runtime.GOOS == "windows" && filepath.Ext(defaultExe) == "" {
		defaultExe += ".exe"
	}

	if _, err := os.Stat(defaultExe); err == nil {
		res.Args[0] = defaultExe
	} else if os.IsNotExist(err) {
		c, err := exec.LookPath(res.Args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "lookpath: %v\n", err)
			os.Exit(5)
		}
		res.Args[0] = c
	} else {
		fmt.Fprintf(os.Stderr, "executable not found: %s\n", res.Args[0])
		os.Exit(5)
	}

	cmd := exec.Command(res.Args[0], res.Args[1:]...)

	err = testexe.CommandAssert(cmd, res)
	if err != nil {
		fmt.Fprintf(os.Stderr, "assert: %v\n", err)
		res.WriteTo(os.Stderr)
		os.Exit(1)
	}
}

func capture() {
	fi, err := os.Stat(os.Args[1])
	if err == nil {
		ftype := fi.Mode().Type()
		// Disallow overriding an existing golden file.
		// But allow to send to an irregular file such as a TTY or /dev/null.
		if ftype.IsRegular() {
			fmt.Fprintf(os.Stderr, "%s: file exists\n", os.Args[1])
			os.Exit(2)
		}
	} else if !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[1], err)
		os.Exit(2)
	}

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin

	res, err := testexe.Capture(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "capture: %v\n", err)
		os.Exit(3)
	}

	f, err := os.Create(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "create: %v\n", err)
		os.Exit(4)
	}
	defer f.Close()

	if _, err := res.WriteTo(f); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(5)
	}
}
