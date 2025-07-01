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
	"os"
	"os/exec"
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
	// import (
	//	"fmt"
	//	"os"
	// )
	//
	// func main() {
	//	os.Args[1] = os.Args[0]
	//	os.Args = os.Args[1:]
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
