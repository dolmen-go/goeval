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

package testexe

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// CaptureResult represents the result of a captured command execution for regression testing.
// It includes the command line arguments, environment variables, stdin, stdout, stderr and exit status.
//
// The representation format on disk is inspired by [txtar]:
//
//	command arg1 arg2
//
//	-- env --
//	ENV_VAR=value
//	-- stdin --
//	stdin content
//	-- exit status: N --
//	-- stdout --
//	stdout content
//	-- stderr --
//	stderr content
//
// All sections are optional except the command line.
//
// In the stdin, stdout and stderr sections, if the content does not end with a newline,
// a "^D" marker is appended to indicate the end of the content.
//
// [txtar]: https://pkg.go.dev/golang.org/x/tools/txtar
type CaptureResult struct {
	Args  []string
	Env   []string
	Stdin string

	ExitStatus int
	Stdout     string
	Stderr     string
}

// Capture runs the given command and captures its stdin, stdout,
// stderr and exit status.
//
// The command's Args field is expected to be set to the full command
// line, with Args[0] being the executable name.
// The command's Stdin may point to a reader that will also be captured.
// The command's Env may be set to a custom environment, which will be
// captured as well but only the differences with the system environment
// are recorded).
func Capture(cmd *exec.Cmd) (*CaptureResult, error) {
	var stdin, stdout, stderr bytes.Buffer
	var withStdin bool
	if cmd.Stdin != nil {
		withStdin = true
		cmd.Stdin = io.TeeReader(cmd.Stdin, &stdin)
	}
	if cmd.Stdout != nil && cmd.Stdout != io.Discard {
		cmd.Stdout = io.MultiWriter(cmd.Stdout, &stdout)
	} else {
		cmd.Stdout = &stdout
	}
	if cmd.Stderr != nil && cmd.Stderr != io.Discard {
		cmd.Stderr = io.MultiWriter(cmd.Stderr, &stderr)
	} else {
		cmd.Stderr = &stderr
	}
	err := cmd.Run()
	var exitStatus int
	switch err := err.(type) {
	case nil:
	case *exec.ExitError:
		exitStatus = err.ExitCode()
	default:
		return nil, err
	}

	args := slices.Clone(cmd.Args)
	args[0] = filepath.Base(cmd.Path)
	if runtime.GOOS == "windows" {
		args[0] = strings.TrimSuffix(args[0], ".exe")
	}

	var cap CaptureResult
	cap.Args = args
	cap.ExitStatus = exitStatus
	if withStdin {
		cap.Stdin = stdin.String()
	}
	cap.Stdout = stdout.String()
	cap.Stderr = stderr.String()

	sysEnv := make(map[string]string)
	for _, e := range os.Environ() {
		k, v, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}
		sysEnv[k] = v
	}
	// Record only the differences between the command's environment and the system environment
	for _, e := range cmd.Env {
		k, v, ok := strings.Cut(e, "=")
		if !ok || v == sysEnv[k] {
			continue
		}
		if runtime.GOOS == "windows" && k == "SYSTEMROOT" {
			// Ignore SYSTEMROOT, which is always set on Windows and may differ between test runs.
			continue
		}
		override := false
		for i, e2 := range cap.Env {
			if strings.HasPrefix(e2, k+"=") {
				cap.Env[i] = e
				override = true
				break
			}
		}
		if !override {
			cap.Env = append(cap.Env, e)
		}
	}
	slices.Sort(cap.Env)

	return &cap, nil
}

func writeStream(w io.Writer, title string, content string) (n int64, err error) {
	if len(content) == 0 {
		return 0, nil
	}
	nn, err := fmt.Fprintf(w, "-- %s --\n", title)
	n += int64(nn)
	if err != nil {
		return
	}
	// nn, err = io.Copy(w, &withEOLReader{r: strings.NewReader(content)})
	// n += nn
	// if err != nil {
	//	return n, err
	//}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "^D\n"
	}
	nn, err = w.Write([]byte(content))
	n += int64(nn)
	return
}

func readStream(lines []string) (content string, remaining []string) {
	var w strings.Builder
	i := 0
	for i < len(lines) && !strings.HasPrefix(lines[i], "-- ") {
		w.WriteString(lines[i] + "\n")
		i++
	}
	content = w.String()
	// If last line ends with ^D, trim \n
	content = strings.TrimSuffix(content, "^D\n")

	remaining = lines[i:]
	return
}

// WriteTo writes the capture result to the given writer in a format
// that can be parsed by [ParseCapture].
func (r *CaptureResult) WriteTo(output io.Writer) (n int64, err error) {
	args := slices.Clone(r.Args)
	for i, a := range args {
		ascii := strconv.QuoteToASCII(a)
		if len(ascii) == len(a)+2 && !strings.ContainsAny(a, "\"`' \t\r\n") {
			continue
		}
		if strconv.CanBackquote(a) {
			a = "`" + a + "`"
		} else {
			a = strconv.QuoteToGraphic(a)
		}
		args[i] = a
	}

	nn, err := fmt.Fprintln(output, strings.Join(args, " "))
	n += int64(nn)
	if err != nil {
		return
	}
	nn, err = fmt.Fprintln(output)
	n += int64(nn)
	if err != nil {
		return
	}
	if len(r.Env) > 0 {
		nn, err = fmt.Fprintln(output, "-- env --")
		n += int64(nn)
		if err != nil {
			return
		}
		slices.Sort(r.Env)
		for _, e := range r.Env {
			nn, err = fmt.Fprintln(output, e)
			n += int64(nn)
			if err != nil {
				return
			}
		}
	}
	nnn, err := writeStream(output, "stdin", r.Stdin)
	n += nnn
	if err != nil {
		return
	}
	if r.ExitStatus != 0 {
		nn, err = fmt.Fprintf(output, "-- exit status: %d --\n", r.ExitStatus)
		n += int64(nn)
		if err != nil {
			return
		}
	}
	nnn, err = writeStream(output, "stdout", r.Stdout)
	n += nnn
	if err != nil {
		return
	}
	nnn, err = writeStream(output, "stderr", r.Stderr)
	n += nnn
	return
}

// WriteCapture executes the given command and writes its captured
// result to the given path in a format that can be parsed by
// [ParseCapture].
func WriteCapture(cmd *exec.Cmd, path string) error {
	result, err := Capture(cmd)
	if err != nil {
		return err
	}

	osPath := path
	if !filepath.IsAbs(path) {
		osPath = filepath.FromSlash(path)
	}

	f, err := os.Create(osPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = result.WriteTo(f)
	return err

}

// ParseCapture parses a capture result from the given reader.
func ParseCapture(r io.Reader) (*CaptureResult, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	buf = bytes.ReplaceAll(buf, []byte("\r\n"), []byte("\n"))
	buf = bytes.ReplaceAll(buf, []byte("\r"), []byte("\n"))
	buf = bytes.TrimSuffix(buf, []byte("\n"))

	lines := strings.Split(string(buf), "\n")
	i := 0
	for i < len(lines) {
		if lines[i] != "" || !strings.HasPrefix(lines[i], "#") {
			break
		}
		i++
	}
	if i == len(lines) {
		return nil, fmt.Errorf("invalid capture: no command line")
	}

	var result CaptureResult

	args := strings.Fields(lines[i])
	for i, a := range args {
		if len(a) >= 2 && (a[0] == '"' || a[0] == '`') && a[len(a)-1] == a[0] {
			unquoted, err := strconv.Unquote(a)
			if err != nil {
				return nil, fmt.Errorf("invalid capture: invalid argument %q: %w", a, err)
			}
			a = unquoted
		}
		args[i] = a
	}
	result.Args = args

	i++
	for i < len(lines) && (lines[i] == "" || strings.HasPrefix(lines[i], "#")) {
		i++
	}
	for i < len(lines) {
		switch lines[i] {
		case "-- stdin --":
			result.Stdin, lines = readStream(lines[i+1:])
			i = 0
		case "-- stdout --":
			result.Stdout, lines = readStream(lines[i+1:])
			i = 0
		case "-- stderr --":
			result.Stderr, lines = readStream(lines[i+1:])
			i = 0
		case "-- env --":
			var env []string
			i++
			for i < len(lines) && !strings.HasPrefix(lines[i], "-- ") {
				env = append(env, lines[i])
				i++
			}
			slices.Sort(env)
			result.Env = env
		default:
			const prefix = "-- exit status: "
			if strings.HasPrefix(lines[i], prefix) {
				n, err := strconv.Atoi(strings.TrimSuffix(lines[i][len(prefix):], " --"))
				if err != nil || n < 0 || n > 255 {
					return nil, fmt.Errorf("invalid capture: invalid exit status: %w", err)
				}
				result.ExitStatus = n
				i++
				continue
			}
			return nil, fmt.Errorf("invalid capture: unexpected line %q", lines[i])
		}
	}
	return &result, nil
}

// CommandAssert executes the given command and asserts that its
// captured result matches the expectation.
// cmd.Args and cmd.Stdin are ignored and replaced by expected.Args and expected.Stdin for the execution.
func CommandAssert(cmd *exec.Cmd, expected *CaptureResult) error {
	if len(expected.Env) > 0 {
		if cmd.Env == nil {
			cmd.Env = os.Environ()
		}
		cmd.Env = append(cmd.Env, expected.Env...)
	}
	cmd.Stdin = strings.NewReader(expected.Stdin)
	cmd.Args = append(append(make([]string, 0, len(expected.Args)), cmd.Args[0]), expected.Args[1:]...)

	result, err := Capture(cmd)
	if err != nil {
		return fmt.Errorf("failed to capture command: %w", err)
	}

	if result.ExitStatus != expected.ExitStatus {
		return fmt.Errorf("unexpected exit status: got %d, expected %d", result.ExitStatus, expected.ExitStatus)
	}
	if result.Stderr != expected.Stderr {
		return fmt.Errorf("unexpected stderr: got %q, expected %q", result.Stderr, expected.Stderr)
	}
	if result.Stdout != expected.Stdout {
		return fmt.Errorf("unexpected stdout: got %q, expected %q", result.Stdout, expected.Stdout)
	}
	return nil
}

// TestCommandAssert executes the given command and asserts that its
// captured result matches the expectation.
// cmd.Args and cmd.Stdin are ignored and replaced by expected.Args and expected.Stdin for the execution.
// Errors are reported to tb.
func TestCommandAssert(tb testing.TB, cmd *exec.Cmd, expected *CaptureResult) {
	tb.Helper()

	err := CommandAssert(cmd, expected)
	if err != nil {
		tb.Fatal(err)
	}
}
