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

// Command echo is a target for testing package [github.com/dolmen-go/goeval/internal/testexe].
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
)

func main() {
	flag.Func("stdout", "Print on stdout", func(v string) error {
		_, err := fmt.Println(v)
		return err
	})
	flag.Func("stderr", "Print on stderr", func(v string) error {
		_, err := fmt.Fprintln(os.Stderr, v)
		return err
	})
	flag.BoolFunc("stdin", "Copy stdin to stdout", func(v string) error {
		_, err := io.Copy(os.Stdout, os.Stdin)
		return err
	})
	flag.Func("exit", "Exit with the given status code", func(v string) error {
		n, err := strconv.Atoi(v)
		if err == nil {
			os.Exit(n)
		}
		return err
	})
	flag.Usage = usage
	flag.Parse()
	if flag.NFlag() == 0 {
		flag.Usage()
	}
	if flag.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "echo: no arguments expected.")
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(flag.CommandLine.Output(), "usage: echo [options...]\n\noptions:")
	flag.PrintDefaults()
}
