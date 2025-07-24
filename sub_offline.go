//go:build goeval.offline

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

package main

import (
	"bytes"
	"errors"
	"flag"
)

const featureIsDisabled = "feature is disabled in offline build"

// registerOnlineFlags does nothing.
func registerOnlineFlags() {
	flag.BoolFunc("play", featureIsDisabled+".", disabledFeature)
	flag.BoolFunc("share", featureIsDisabled+".", disabledFeature)
}

func disabledFeature(string) error {
	return errors.New(featureIsDisabled)
}

func prepareSubPlay() (*bytes.Buffer, func() error, func()) {
	panic("dead code in offline mode")
}

func prepareSubShare() (*bytes.Buffer, func() error, func()) {
	panic("dead code in offline mode")
}
