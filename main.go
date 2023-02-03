/*
   Copyright 2019-2022 Olivier Mengué.

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
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec" // Go 1.19 behaviour enforced in go.mod. See https://blog.golang.org/path-security and https://pkg.go.dev/os/exec
	"path/filepath"
	"strings"

	"golang.org/x/mod/module"
	goimp "golang.org/x/tools/imports"
)

type imports struct {
	packages map[string]string // alias => import path
	modules  map[string]string // module path => version
}

func (*imports) String() string {
	return "" // irrelevant
}

func (imp *imports) Set(s string) error {
	var alias, path, version string
	var ok bool
	if alias, path, ok = strings.Cut(s, "="); !ok {
		alias = ""
		path = s
	} else if alias == "" {
		return fmt.Errorf("%q: empty alias", s)
	} else if alias == "_" || alias == "." {
		alias = alias + " " + path // special alias
	} else if strings.Contains(alias, " ") {
		return fmt.Errorf("%q: invalid alias", s)
	}
	var p2 string
	if p2, version, ok = strings.Cut(s, "@"); ok {
		if version == "" {
			return fmt.Errorf("%q: empty module version", s)
		}
		path = p2
		if err := module.CheckPath(path); err != nil {
			return fmt.Errorf("%q: %w", s, err)
		}
		// TODO check for duplicates
		if imp.modules == nil {
			imp.modules = make(map[string]string)
		}
		imp.modules[path] = version
	} else if alias == "" {
		alias = "  " + path // special alias
	}

	switch path {
	case "":
		return fmt.Errorf("%q: empty path", s)
	case "embed":
		return errors.New("use of package 'embed' is not allowed")

	default:
		if err := module.CheckImportPath(path); err != nil {
			return fmt.Errorf("%q: %w", s, err)
		}
	}

	if alias != "" {
		imp.packages[alias] = path
	}

	// log.Printf("alias=%s path=%s version=%s", alias, path, version)

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
	imports := imports{
		packages: map[string]string{"  ": "os"},
	}
	flag.Var(&imports, "i", "import package: [alias=]import-path")

	var goimports string
	flag.StringVar(&goimports, "goimports", "goimports", "goimports tool name, to use an alternate tool or just disable it")

	var noRun bool // -E, like "cc -E"
	flag.BoolVar(&noRun, "E", false, "just dump the assembled source, without running it")

	flag.Usage = func() {
		prog := os.Args[0]
		fmt.Fprintf(flag.CommandLine.Output(), ""+
			"\n"+
			"Usage: %s [<options>...] <code> [<args>...]\n"+
			"\n"+
			"Options:\n",
			prog)
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), ""+
			"\n"+
			"Example:\n"+
			"  %s -i fmt 'fmt.Println(\"Hello, world!\")'\n"+
			"\n"+
			"Copyright 2019-2022 Olivier Mengué.\n"+
			"Source code: https://github.com/dolmen-go/goeval\n",
			prog)
		os.Exit(1)
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
	}
	code := flag.Arg(0)
	if code == "-" {
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		code = string(b)
	}

	args := flag.Args()[1:]

	var origDir string

	var dir string
	if imports.modules != nil {
		var err error
		if dir, err = os.MkdirTemp("", "goeval*"); err != nil {
			log.Fatal(err)
		}
		defer os.Remove(dir)

		moduleName := filepath.Base(dir)

		origDir, err = os.Getwd()
		if err != nil {
			log.Fatal("getwd:", err)
		}

		if err := os.Chdir(dir); err != nil {
			log.Fatalf("chdir(%q): %v", dir, err)
		}

		gomod, err := os.Create(dir + "/go.mod")
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			gomod.Close()
			os.Remove(gomod.Name())
		}()
		fmt.Fprintf(gomod, "module %s\n\nrequire (\n", moduleName)
		for path, version := range imports.modules {
			fmt.Fprintf(gomod, "\t%s %s\n", path, version)
		}
		gomod.WriteString(")\n")
		gomod.Close()

		// log.Printf("go.mod: %s", gomod.Name())
		cmd := exec.Command("go", "mod", "download")
		cmd.Env = os.Environ()
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = os.Stderr
		if err = cmd.Run(); err != nil {
			log.Fatal("go mod download failure:", err)
		}
		log.Println("go mod download OK.")
		defer os.Remove(dir + "/go.sum")
	}

	var src bytes.Buffer
	src.WriteString("package main\n")
	for alias, path := range imports.packages {
		if len(alias) > 2 && alias[1] == ' ' {
			switch alias[0] {
			case '.', '_':
				alias = alias[:1]
			case ' ': // no alias
				fmt.Fprintf(&src, "import %q\n", path)
				continue
			}
		}
		fmt.Fprintf(&src, "import %s %q\n", alias, path)
	}
	src.WriteString("func main() {\nos.Args[1] = os.Args[0]\nos.Args = os.Args[1:]\n")
	if origDir != "" {
		fmt.Fprintf(&src, "os.Chdir(%q)\n", origDir)
	}
	src.WriteString("//line :1\n")
	src.WriteString(code)
	src.WriteString("\n}\n")

	// fmt.Print(src.String())

	var f *os.File
	var err error
	if !noRun {
		f, err = ioutil.TempFile(dir, "*.go")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		defer os.Remove(f.Name())
	} else {
		f = os.Stdout
	}

	env := os.Environ()

	if imports.modules == nil {
		// Run in GOPATH mode, ignoring any code in the current directory
		env = append(env, "GO111MODULE=off")
	} else {
		env = append(env, "GO111MODULE=on")
	}

	switch goimports {
	case "goimports":
		var out []byte
		var filename string // filename is used to locate the relevant go.mod
		if imports.packages != nil {
			filename = f.Name()
		}
		out, err = goimp.Process(filename, src.Bytes(), &goimp.Options{
			Fragment:   false,
			AllErrors:  false,
			Comments:   true,
			TabIndent:  true,
			TabWidth:   8,
			FormatOnly: false,
		})
		if err == nil {
			_, err = f.Write(out)
		}
	case "":
		_, err = f.Write(src.Bytes())
	default:
		cmd := exec.Command(goimports)
		cmd.Env = env
		cmd.Stdin = &src
		cmd.Stdout = f
		cmd.Stderr = os.Stderr
		err = cmd.Run()
	}
	if err != nil {
		return err
	}

	if noRun {
		// dump go.mod, go.sum
		if imports.modules != nil {
			gomod, err := os.Open(dir + "/go.mod")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("-- go.mod --")
			defer gomod.Close()
			io.Copy(os.Stdout, gomod)
			gosum, err := os.Open(dir + "/go.sum")
			switch {

			case errors.Is(err, os.ErrNotExist):
			case err != nil:
				log.Fatal(err)
			default:
				fmt.Println("-- go.sum --")
				defer gosum.Close()
				io.Copy(os.Stdout, gosum)
			}
		}

		return nil
	}
	err = f.Close()
	if err != nil {
		return err
	}

	log.Println(origDir)
	if origDir != "" {
		cmd1 := exec.Command("sh", "-c", "ls -l;cat "+f.Name()+";echo '-- go.mod --';cat go.mod;echo '-- go.sum --';cat go.sum")
		cmd1.Stdout = os.Stdout
		cmd1.Run()
		/*
			cmd2 := exec.Command("cat", f.Name())
			cmd2.Stdout = os.Stdout
			cmd2.Run()
		*/
	}

	var runArgs = make([]string, 0, 1+2+2+len(args))
	runArgs = append(runArgs, "run")
	if imports.modules != nil {
		log.Println(f.Name())
		//runArgs = append(runArgs, "-modfile", dir+"/go.mod", f.Name())
		//runArgs = append(runArgs, "-modfile", dir+"/go.mod", dir+"@v1.0.0")
		// runArgs = append(runArgs, f.Name())
		// runArgs = append(runArgs, filepath.Base(f.Name()))
		runArgs = append(runArgs, ".")
	} else {
		runArgs = append(runArgs, f.Name())
	}
	runArgs = append(runArgs, "--")
	runArgs = append(runArgs, args...)

	log.Println(runArgs)

	/*
		cmdSh := exec.Command("bash")
		cmdSh.Env = append(os.Environ(), "PS1=++>")
		cmdSh.Stdin = os.Stdin
		cmdSh.Stdout = os.Stdout
		cmdSh.Stderr = os.Stderr
		cmdSh.Run()
	*/

	cmd := exec.Command("go", runArgs...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// exec.ExitError is handled in caller
	return cmd.Run()
}
