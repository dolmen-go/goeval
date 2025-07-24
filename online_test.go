//go:build !goeval.offline

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
