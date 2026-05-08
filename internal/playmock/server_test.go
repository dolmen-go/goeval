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

package playmock_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

// testProxyCACert checks that the proxy serves its certificate authority's certificate.
func testProxyCACert(t testing.TB, client *http.Client, proxyURL string, caPEM []byte) {
	caCertURL := proxyURL + "/ca-certificates.crt"
	rsp, err := client.Get(caCertURL)
	if err != nil {
		t.Fatalf("%s: %v", caCertURL, err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode == http.StatusOK {
		caPEM2, err := io.ReadAll(rsp.Body)
		if err != nil {
			t.Fatalf("%s: %s", caCertURL, err)
		}
		if !bytes.Equal(caPEM2, caPEM) {
			t.Fatal("CA certificate mismatch")
		}
	} else {
		t.Errorf("%s: %s", caCertURL, rsp.Status)
	}
}
