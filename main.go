/*
   Copyright 2019-2023 Olivier Mengué.

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
	"time"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	goimp "golang.org/x/tools/imports"
)

type imports struct {
	packages   map[string]string // alias => import path
	modules    map[string]string // module path => version
	onlySemVer bool
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
		tmpPath, _, _ := strings.Cut(path, "@")
		alias = alias + " " + tmpPath // special alias
	} else if strings.Contains(alias, " ") {
		return fmt.Errorf("%q: invalid alias", s)
	}
	var p2 string
	if p2, version, ok = strings.Cut(path, "@"); ok {
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
		imp.onlySemVer = imp.onlySemVer && semver.IsValid(version) && version == semver.Canonical(version)
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

var run = runSilent

func runSilent(cmd *exec.Cmd) error {
	return cmd.Run()
}

func runX(cmd *exec.Cmd) error {
	// Inject -x in go commands
	if cmd.Args[0] == "go" && cmd.Args[1] != "env" {
		cmd.Args = append([]string{"go", cmd.Args[1], "-x"}, cmd.Args[2:]...)
	}
	fmt.Printf("%s\n", cmd.Args)
	return cmd.Run()
}

func runTime(cmd *exec.Cmd) error {
	defer func(start time.Time) {
		fmt.Fprintf(os.Stderr, "run %v %v\n", time.Since(start), cmd.Args)
	}(time.Now())
	return cmd.Run()
}

var goCmd = "go"

func getGOMODCACHE(env []string) (string, error) {
	var out bytes.Buffer
	cmd := exec.Command(goCmd, "env", "GOMODCACHE")
	cmd.Stderr = os.Stderr
	cmd.Stdout = &out
	cmd.Env = env
	err := run(cmd)
	if err != nil {
		return "", err
	}
	b := bytes.TrimRight(out.Bytes(), "\r\n")
	if len(b) == 0 {
		return "", errors.New("can't retrieve GOMODCACHE")
	}
	return string(b), nil
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
		packages:   map[string]string{"  ": "os"},
		onlySemVer: true,
	}
	flag.Var(&imports, "i", "import package: [alias=]import-path")

	var goimports string
	flag.StringVar(&goimports, "goimports", "goimports", "goimports tool name, to use an alternate tool or just disable it")

	var noRun bool // -E, like "cc -E"
	flag.BoolVar(&noRun, "E", false, "just dump the assembled source, without running it")

	showCmds := flag.Bool("x", false, "print the commands")

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
			"Copyright 2019-2023 Olivier Mengué.\n"+
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

	if *showCmds {
		run = runX
	}

	moduleMode := imports.modules != nil

	env := os.Environ()
	if moduleMode {
		env = append(env, "GO111MODULE=on")
	} else {
		// Run in GOPATH mode, ignoring any code in the current directory
		env = append(env, "GO111MODULE=off")
	}

	var dir, origDir string

	if moduleMode {
		// "go get" is not yet as smart as we want, so let's help
		// https://go.dev/issue/43646
		preferCache := imports.onlySemVer
		var gomodcache string
		if preferCache {
			var err error
			gomodcache, err = getGOMODCACHE(env)
			preferCache = err == nil
		}

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

		gomod := dir + "/go.mod"
		if err := os.WriteFile(gomod, []byte("module "+moduleName+"\n"), 0600); err != nil {
			log.Fatal("go.mod:", err)
		}
		defer os.Remove(gomod)

		var gogetArgs []string
		gogetArgs = append(gogetArgs, "get", "--")
		for mod, ver := range imports.modules {
			gogetArgs = append(gogetArgs, mod+"@"+ver)
			if preferCache {
				// Keep preferCache as long as we find modules in the cache
				_, err := os.Stat(gomodcache + "/cache/download/" + mod + "/@v/" + ver + ".mod")
				preferCache = err == nil
			}
		}
		for _, path := range imports.packages {
			if _, seen := imports.modules[path]; !seen {
				gogetArgs = append(gogetArgs, path)
			}
		}

		// fmt.Println("preferCache", preferCache)

		cmd := exec.Command("go", gogetArgs...)
		if preferCache {
			// As we found all modules in the cache, tell "go get" to not use the proxy.
			// See https://go.dev/issue/43646
			cmd.Env = append(env, "GOPROXY=file://"+filepath.ToSlash(gomodcache)+"/cache/download")
		} else {
			cmd.Env = env
		}
		cmd.Dir = dir
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stdout = os.Stdout
		// go get is too verbose :(
		cmd.Stderr = nil
		if err = run(cmd); err != nil {
			log.Fatal("go get failure:", err)
		}
		// log.Println("go get OK.")
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
	if moduleMode {
		fmt.Fprintf(&src, "_ = os.Chdir(%q)\n", origDir)
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
		cmd.Dir = dir
		cmd.Stdin = &src
		cmd.Stdout = f
		cmd.Stderr = os.Stderr
		err = run(cmd)
	}
	if err != nil {
		return err
	}

	if noRun {
		// dump go.mod, go.sum
		if moduleMode {
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

	if moduleMode {
		/*
			// Do we need to run "go get" again after "goimports"?
			goget := exec.Command("go", "get", ".")
			goget.Env = env
			goget.Dir = dir
			goget.Stdout = os.Stdout
			goget.Stderr = os.Stderr
			run(goget)
		*/

		/*
			// Debug
			log.Println(origDir)
			showDir := exec.Command("sh", "-c", "ls -l;cat "+f.Name()+";echo '-- go.mod --';cat go.mod;echo '-- go.sum --';cat go.sum")
			showDir.Env = env
			showDir.Dir = dir
			showDir.Stdout = os.Stdout
			run(showDir)
		*/
	}

	var runArgs = make([]string, 0, 3+len(args))
	runArgs = append(runArgs, "run", f.Name(), "--")
	runArgs = append(runArgs, args...)

	// log.Println("go", runArgs)

	cmd := exec.Command("go", runArgs...)
	cmd.Env = env
	cmd.Dir = dir // In Go module mode we run from the temp module dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// exec.ExitError is handled in caller
	return run(cmd)
}
