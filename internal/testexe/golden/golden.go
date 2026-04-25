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
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/dolmen-go/goeval/internal/testexe"
)

var (
	withStdin  = flag.Bool("i", false, "capture stdin ([i]nteractive)")
	withUpdate = flag.Bool("u", false, "update: replay and overwrite with the new output")

	envVars []string // -D
)

func main() {
	flag.Func("D", "capture environnment variable (`<name>[=<value>]`)", func(env string) error {
		// If value is not given, take it from the environment
		if strings.IndexByte(env, '=') == -1 {
			env += "=" + os.Getenv(env)
		}
		// Replace a previous variable with the same name
		if len(envVars) > 0 {
			prefix := env[:strings.IndexByte(env, '=')+1]
			for i := range envVars {
				if strings.HasPrefix(envVars[i], prefix) {
					envVars[i] = env
					return nil
				}
			}
		}
		envVars = append(envVars, env)
		return nil
	})

	if len(os.Args) <= 1 {
		usage()
	}
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 1 {
		replay(flag.CommandLine.Args())
	} else {
		capture(flag.CommandLine.Args())
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, ""+
		"usage: %s"+" [-D <name>[=<value>] ...] [-i] <out.golden> <cmd> [<args>...]\n"+
		"       %[1]s [-u] <in.golden>\n"+
		"\n"+
		"With 2 or more arguments, %[1]s captures the output of the given command and\n"+
		"writes it to the given golden file.\n"+
		"With exactly 1 argument, %[1]s replays the given golden file and asserts that\n"+
		"the command's output matches the captured one.\n"+
		"\n"+
		"  -i    capture stdin ([i]nteractive)\n"+
		"  -D    capture environment variable `<name>[=<value>]`\n"+
		"        If just a name is given, the value is taken from the environment.\n"+
		"  -u    update: replay and overwrite with the new output\n",
		filepath.Base(os.Args[0]))

	os.Exit(1)
}

func fatal(status int, message string, args ...any) {
	cmd := strings.ReplaceAll(filepath.Base(os.Args[0]), "%", "%%")
	fmt.Fprintf(os.Stderr, cmd+": "+message+"\n", args...)
	os.Exit(status)
}

func replay(args []string) {
	f, err := os.Open(args[0])
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
			fatal(5, "lookpath: %v", err)
		}
		res.Args[0] = c
	} else {
		fatal(5, "executable not found: %s", res.Args[0])
	}

	cmd := exec.Command(res.Args[0], res.Args[1:]...)

	if *withUpdate {
		res.PrepareCmd(cmd)
		res2, err := testexe.Capture(cmd)
		if err != nil {
			fatal(2, "capture: %v", err)
		}

		// Be sure that we preserve original inputs
		res2.Env = res.Env
		res2.Stdin = res.Stdin
		// TODO restore comments

		saveCapture(res2, args[0])
		return
	}

	err = testexe.CommandAssert(cmd, res)
	if err != nil {
		fmt.Fprintf(os.Stderr, "assert: %v\n", err)
		res.WriteTo(os.Stderr)
		os.Exit(1)
	}
}

func capture(args []string) {
	if args[0] != "-" {
		fi, err := os.Stat(args[0])
		if err == nil {
			ftype := fi.Mode().Type()
			// Disallow overriding an existing golden file.
			// But allow to send to an irregular file such as a TTY or /dev/null.
			if ftype.IsRegular() {
				fatal(2, "%s: file exists", args[0])
			}
		} else if !os.IsNotExist(err) {
			fatal(2, "%v", err)
		}
	}

	// fmt.Println("Launching:", args[1:])
	cmd := exec.Command(args[1], args[2:]...)

	if len(envVars) > 0 {
		cmd.Env = append(os.Environ(), envVars...)
	}

	if *withStdin {
		cmd.Stdin = os.Stdin
	} else {
		// Check if data is available on Stdin

		type R struct {
			buf []byte
			err error
		}
		ch := make(chan *R, 1)
		go func() {
			b := []byte{0} // 1-byte buffer
			n, err := os.Stdin.Read(b)
			ch <- &R{buf: b[:n], err: err}
			close(ch)
		}()

		select {
		case res := <-ch:
			if len(res.buf) > 0 {
				cmd.Stdin = bytes.NewReader(res.buf)
				if res.err == nil {
					cmd.Stdin = io.MultiReader(cmd.Stdin, os.Stdin)
				}
			}
		case <-time.After(10 * time.Millisecond):
			// Do not capture stdin

			// Close stdin to force the Read to fail, and so release the channel and goroutine.
			// Note: the next open will reuse fd 0.
			os.Stdin.Close()
			go func() {
				<-ch
			}()
		}
	}

	res, err := testexe.Capture(cmd)
	if err != nil {
		fatal(2, "capture: %v", err)
	}
	// fmt.Println("Done.")

	// If a set of environment variables was given, capture just them.
	if len(envVars) > 0 {
		res.Env = envVars
	}

	if args[0] == "-" {
		if _, err := res.WriteTo(os.Stdout); err != nil {
			fatal(5, "write: %v", err)
		}
	} else {
		saveCapture(res, args[0])
	}
}

func saveCapture(res *testexe.CaptureResult, out string) {
	f, err := os.Create(out)
	if err != nil {
		fatal(4, "create: %v", err)
	}
	defer f.Close()

	if _, err := res.WriteTo(f); err != nil {
		fatal(5, "write: %v", err)
	}
}
