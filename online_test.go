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

// Show "goeval -play": running code remotely on https://play.golang.org/
func Example_play() {
	goeval(`-play`, `fmt.Println(time.Now())`)

	// Output:
	// 2009-11-10 23:00:00 +0000 UTC m=+0.000000001
}

// Show "goeval -play", with arguments values sent with the program code
func Example_playWithArgs() {
	goeval(`-play`, `fmt.Println(os.Args[1])`, `toto`)

	// Output:
	// toto
}
