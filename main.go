/*
   Copyright 2019 Olivier Mengué.

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
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

type imports map[string]string

func (i imports) String() string {
	return "" // irrelevant
}

func (imp *imports) Set(s string) error {
	i := strings.IndexByte(s, '=')
	var name, path string
	if i == -1 {
		path = s
		i = strings.LastIndexByte(s, '/')
		if i > 0 {
			name = s[i+1:]
		} else {
			name = path
		}
	} else {
		name = s[:i]
		path = s[i+1:]
	}

	// FIXME check that name and path have len > 0

	if *imp == nil {
		*imp = imports{name: path}
	} else {
		(*imp)[name] = path
	}
	return nil
}

func main() {
	err := _main()
	if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() > 0 {
		os.Exit(exit.ExitCode())
	} else if err != nil {
		log.Fatal(err)
	}
}

func _main() error {
	var imports imports
	flag.Var(&imports, "i", "import package: [alias=]import-path")

	var goimports string
	flag.StringVar(&goimports, "goimports", "goimports", "goimports tool name, to use an alternate tool or just disable it")

	var noRun bool // -E, like "cc -E"
	flag.BoolVar(&noRun, "E", false, "just dump the assembled source, without running it")

	flag.Usage = func() {
		prog := os.Args[0]
		fmt.Fprintf(flag.CommandLine.Output(), "\nUsage: %s [<options>...] <code> [<args>...]\n\nOptions:\n", prog)
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nExample:\n  %s -i fmt 'fmt.Println(\"Hello, world!\")'\n\n", prog)
		os.Exit(1)
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
	}
	code := flag.Arg(0)
	args := flag.Args()[1:]

	var src bytes.Buffer
	src.WriteString("package main\n")
	for name, path := range imports {
		fmt.Fprintf(&src, "import %s %q\n", name, path)
	}
	src.WriteString("func main() {\n//line :1\n")
	src.WriteString(code)
	src.WriteString("\n}\n")

	// fmt.Print(src.String())

	var f *os.File
	var err error
	if !noRun {
		f, err = ioutil.TempFile("", "*.go")
		if err != nil {
			log.Fatal(err)
		}
		defer os.Remove(f.Name())
		defer f.Close()
	} else {
		f = os.Stdout
	}

	if goimports != "" {
		cmd := exec.Command("goimports")
		cmd.Stdin = &src
		cmd.Stdout = f
		cmd.Stderr = os.Stderr
		err = cmd.Run()
	} else {
		_, err = f.Write(src.Bytes())
	}
	if err != nil {
		return err
	}
	if noRun {
		return nil
	}
	err = f.Close()
	if err != nil {
		return err
	}

	cmd := exec.Command("go", append([]string{
		"run",
		f.Name(),
	}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// exec.ExitError is handled in caller
	return cmd.Run()
}
