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
	"reflect"
	"regexp" // Add regexp for error matching
	"testing"
)

func TestSplitWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     []string
		errMatch string // Regex to match the error message
	}{
		// Basic cases
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "single word",
			input: "hello",
			want:  []string{"hello"},
		},
		{
			name:  "multiple words",
			input: "hello world",
			want:  []string{"hello", "world"},
		},
		{
			name:  "multiple words with tabs",
			input: "hello\tworld",
			want:  []string{"hello", "world"},
		},
		{
			name:  "leading and trailing spaces",
			input: "  hello world  ",
			want:  []string{"hello", "world"},
		},
		{
			name:  "multiple spaces between words",
			input: "hello   big   world",
			want:  []string{"hello", "big", "world"},
		},
		{
			name:  "only spaces",
			input: "   ",
			want:  nil,
		},
		{
			name:  "word with hyphen",
			input: "hello-world",
			want:  []string{"hello-world"},
		},

		// Backquote (`) cases
		{
			name:  "simple backquoted string",
			input: "`hello world`",
			want:  []string{"hello world"},
		},
		{
			name:  "backquoted string with leading/trailing text",
			input: "prefix`hello world`suffix",
			want:  []string{"prefixhello worldsuffix"},
		},
		{
			name:  "multiple backquoted strings",
			input: "`hello world` `go lang`",
			want:  []string{"hello world", "go lang"},
		},
		{
			name:  "mixed backquoted and unquoted",
			input: "say `hello world` to me",
			want:  []string{"say", "hello world", "to", "me"},
		},
		{
			name:  "empty backquoted string",
			input: "``",
			want:  []string{""},
		},
		{
			name:     "unclosed backquote",
			input:    "`hello",
			errMatch: "unclosed backtick starting at column 1",
		},
		{
			name:     "backquote unclosed with words after",
			input:    "word `hello",
			errMatch: "unclosed backtick starting at column 6",
		},
		{
			name:  "backquote with backtick inside",
			input: "`hello ` `world`",
			want:  []string{"hello ", "world"},
		},

		// Double-quote (") cases
		{
			name:  "simple double-quoted string",
			input: `"hello world"`,
			want:  []string{"hello world"},
		},
		{
			name:  "double-quoted string with escape sequence newline",
			input: `"hello\nworld"`,
			want:  []string{"hello\nworld"},
		},
		{
			name:  "double-quoted string with escape sequence tab",
			input: `"hello\tworld"`,
			want:  []string{"hello\tworld"},
		},
		{
			name:  "double-quoted string with escaped double quote",
			input: `"hello\"world"`,
			want:  []string{`hello"world`},
		},
		{
			name:  "double-quoted string with escaped backslash",
			input: `"hello\\world"`,
			want:  []string{`hello\world`},
		},
		{
			name:  "multiple double-quoted strings",
			input: `"hello world" "go lang"`,
			want:  []string{"hello world", "go lang"},
		},
		{
			name:  "mixed double-quoted and unquoted",
			input: `say "hello world" to me`,
			want:  []string{"say", "hello world", "to", "me"},
		},
		{
			name:  "empty double-quoted string",
			input: `""`,
			want:  []string{""},
		},
		{
			name:     "unclosed double-quote",
			input:    `"hello`,
			errMatch: "unclosed double-quote starting at column 1",
		},
		{
			name:     "double-quote unclosed with words after",
			input:    `word "hello`,
			errMatch: "unclosed double-quote starting at column 6",
		},
		{
			name:     "invalid escape sequence in double quote",
			input:    `"hello\xworld"`,
			errMatch: "invalid double-quoted string starting at column 1: invalid syntax",
		},

		// Mixed cases
		{
			name:  "backquote and double-quote mixed",
			input: `a ` + "`b c`" + ` "d\ne" f`, // This is getting complex in syntax
			want:  []string{"a", "b c", "d\ne", "f"},
		},
		{
			name:  "backquote and double-quote mixed 2",
			input: "`hello world` \"`quoted`\"",
			want:  []string{"hello world", "`quoted`"},
		},
		{
			name:  "double-quote within backquote",
			input: "`hello \"world\"`",
			want:  []string{"hello \"world\""},
		},
		{
			name:     "invalid escape sequence (backslash at end) in double-quote",
			input:    `"hello\`,
			errMatch: `invalid escape sequence at column 7`,
		},
		{
			name:     "invalid escape sequence (backtick) in double-quote",
			input:    `"hello\gworld"`, // `\g` is an invalid escape sequence
			errMatch: `invalid double-quoted string starting at column 1: invalid syntax`,
		},
		{
			name:  "literal backtick in double-quote",
			input: `"hello` + "`" + `world"`,
			want:  []string{"hello`world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitWords(tt.input)

			if tt.errMatch != "" {
				if err == nil {
					t.Errorf("splitWords() error = nil, want error matching %q", tt.errMatch)
				} else if !regexp.MustCompile(tt.errMatch).MatchString(err.Error()) {
					t.Errorf("splitWords() error = %v, want error matching %q", err, tt.errMatch)
				}
				return
			}

			if err != nil {
				t.Errorf("splitWords() unexpected error = %v", err)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitWords() got = %v, want %v", got, tt.want)
				return
			}

			formatted := formatWords(got)
			got, err = splitWords(formatted)

			if err != nil {
				t.Fatalf("formatWords() roudtrip failure = %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitWords() roundtrip failure %q = , got %v, want %v", formatted, got, tt.want)
				return
			}
		})
	}
}

func FuzzWordsRoundtrip(f *testing.F) {
	seeds := [][]string{
		{"hello", "world"},
		{"foo bar", "baz"},
		{"", "empty"},
		{"back`tick", "double\"quote"},
		{"slash\\", "newline\n"},
		{"\t", "\r", "\x00"},
	}
	for _, seed := range seeds {
		f.Add(formatWords(seed))
	}

	f.Fuzz(func(t *testing.T, input string) {
		words, err := splitWords(input)
		if err != nil {
			return
		}

		formatted := formatWords(words)
		words2, err := splitWords(formatted)
		if err != nil {
			t.Fatalf("Failed to parse formatted words %q: %v", formatted, err)
		}

		if !reflect.DeepEqual(words, words2) {
			t.Errorf("Roundtrip mismatch!\nOriginal words: %q\nFormatted: %q\nParsed again: %q", words, formatted, words2)
		}
	})
}
