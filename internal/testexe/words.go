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
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

func formatWords(args []string) string {
	args = slices.Clone(args)
	for i, a := range args {
		if len(a) == 0 {
			args[i] = `""`
			continue
		}
		quoted := strconv.QuoteToGraphic(a)
		if len(quoted) == len(a)+2 && !strings.ContainsAny(a, "\"\\`' \t\r\n") {
			continue
		}
		if strconv.CanBackquote(a) {
			a = "`" + a + "`"
		} else {
			a = quoted
		}
		args[i] = a
	}

	return strings.Join(args, " ")
}

// splitWords splits a string in words, like [strings.Fields], but with 2 escape
// mecanisms:
//   - backquote (`) allows to use a raw string, including spaces, tabs.
//   - double-quote (") allows to use backslash sequences of [strconv.Quote].
func splitWords(line string) ([]string, error) {
	var args []string
	inSpaces := true
	var arg strings.Builder
	i := 0
	for i < len(line) {
		r, n := utf8.DecodeRuneInString(line[i:])
		isSpace := r == ' ' || r == '\t'
		if inSpaces {
			if isSpace {
				i += n
				continue
			}
			inSpaces = false
			arg.Reset()
		} else if isSpace {
			args = append(args, arg.String())
			inSpaces = true
			i += n
			continue
		}
		switch r {
		case '`':
			j := strings.IndexByte(line[i+1:], '`')
			if j == -1 {
				return nil, fmt.Errorf("unclosed backtick starting at column %d", i+1)
			}
			j = i + 1 + j
			arg.WriteString(line[i+1 : j])
			i = j + 1
		case '"':
			j := i + 1
			for {
				if j == len(line) {
					return nil, fmt.Errorf("unclosed double-quote starting at column %d", i+1)
				}
				if line[j] == '"' {
					break
				}
				if line[j] == '\\' {
					j++
					if j == len(line) {
						return nil, fmt.Errorf("invalid escape sequence at column %d", j)
					}
				}
				j++
			}
			unquoted, err := strconv.Unquote(line[i : j+1])
			if err != nil {
				return nil, fmt.Errorf("invalid double-quoted string starting at column %d: %v", i+1, err)
			}
			arg.WriteString(unquoted)
			i = j + 1
		default:
			j := strings.IndexAny(line[i+n:], " \t`\"")
			if j == -1 {
				j = len(line)
			} else {
				j += i + n
			}
			arg.WriteString(line[i:j])
			i = j
		}
	}
	if !inSpaces {
		args = append(args, arg.String())
	}
	return args, nil
}
