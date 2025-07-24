//go:build !goeval.offline

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

package main_test

import (
	"os"
	"os/exec"
	"strings"
)

const userAgent = "goeval.play.test/v0.0.0 (github.com/dolmen-go/goeval/sub/play_test)"

func Example_fmt() {
	cmd := exec.Command("go", "run", "play.go", userAgent)
	cmd.Env = append(os.Environ(), "GO111MODULE=off")
	cmd.Stdin = strings.NewReader(`package main;import"fmt";func main(){fmt.Println("OK")}`)
	cmd.Stdout = os.Stdout
	cmd.Run()

	// Output:
	// OK
}

func Example_time() {
	cmd := exec.Command("go", "run", "play.go", userAgent)
	cmd.Env = append(os.Environ(), "GO111MODULE=off")
	cmd.Stdin = strings.NewReader(`package main;import("fmt";"time");func main(){fmt.Println(time.Now().Format(time.RFC3339))}`)
	cmd.Stdout = os.Stdout
	cmd.Run()

	// Output:
	// 2009-11-10T23:00:00Z
}
