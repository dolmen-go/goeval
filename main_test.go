/*
   Copyright 2019-2025 Olivier Mengué.

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
	"bytes"
	"os"
	"os/exec"
	"testing"
)

func goeval(args ...string) {
	cmd := exec.Command("go", append([]string{"run", "."}, args...)...)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func Example() {
	goeval(`fmt.Println("OK")`)

	// Output:
	// OK
}

func Example_dump() {
	goeval("-E", `fmt.Println("OK")`)

	// Output:
	// package main
	//
	// import "fmt"
	//
	// func main() {
	// //line :1
	//	fmt.Println("OK")
	// }
}

func Example_flag() {
	goeval(`fmt.Println(os.Args[1])`, `--`)
	goeval(`fmt.Println(os.Args[1])`, `-x`)      // -x is also a "go run" flag
	goeval(`fmt.Println(os.Args[1])`, `toto.go`) // toto.go could be caught by "go run"

	// Output:
	// --
	// -x
	// toto.go
}

func Example_import() {
	goeval(`-goimports=`, `-i`, `fmt`, `fmt.Println("OK")`)
	goeval(`-goimports=`, `-i=fmt`, `fmt.Println("OK")`)
	goeval(`-goimports=`, `-i`, `fmt`, `-i`, `time`, `fmt.Println(time.Time{}.In(time.UTC))`)
	goeval(`-goimports=`, `-i`, `fmt,time`, `fmt.Println(time.Time{}.In(time.UTC))`)
	goeval(`-goimports=`, `-i=fmt,time`, `fmt.Println(time.Time{}.In(time.UTC))`)

	// Output:
	// OK
	// OK
	// 0001-01-01 00:00:00 +0000 UTC
	// 0001-01-01 00:00:00 +0000 UTC
	// 0001-01-01 00:00:00 +0000 UTC
}

// printlnWriter writes each line to a [fmt.Println]-like function.
// [testing.T.Log] is such a function.
type printlnWriter func(...any)

func (tl printlnWriter) Write(b []byte) (int, error) {
	for len(b) > 0 {
		p := bytes.IndexByte(b, '\n')
		if p == -1 {
			tl(string(b))
			break
		}
		line := b[:p]
		if len(line) > 1 && line[p-1] == '\r' {
			line = line[:p-1]
		}
		tl(string(line))
		b = b[p+1:]
	}
	return len(b), nil
}

// goevalT runs goeval with the given arguments, and sends each line from stdout to tb.Log.
func goevalT(tb testing.TB, args ...string) {
	cmd := exec.Command("go", append([]string{"run", "."}, args...)...)
	cmd.Stdin = nil
	cmd.Stdout = printlnWriter(tb.Log)
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func TestShowRuntimeBuildInfo(t *testing.T) {
	goevalT(t, `-i=fmt,runtime/debug,os`, `-goimports=`, `bi,ok:=debug.ReadBuildInfo(); if !ok {os.Exit(1)}; fmt.Print(bi)`)
}

func TestPrintStack(t *testing.T) {
	goevalT(t, `-i=runtime/debug`, `-goimports=`, `debug.PrintStack()`)
}
