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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/dolmen-go/goeval/internal/playmock"
)

func TestServerCompile(t *testing.T) {
	t.Parallel()

	expectedBody := "package main\nimport \"fmt\"\nfunc main() {\n  fmt.Println(\"Hello, world!\")\n}\n"
	expectedResp := &playmock.CompileResponse{
		Errors: "",
		Events: []playmock.CompileEvent{
			{Message: "Hello, world!\n", Kind: "stdout", Delay: 0},
		},
		Status:      0,
		IsTest:      false,
		TestsFailed: 0,
	}

	srv := &playmock.Server{
		Compile: func(req *playmock.CompileRequest) (*playmock.CompileResponse, error) {
			if req.Body != expectedBody {
				t.Errorf("unexpected body: got %q, want %q", req.Body, expectedBody)
			}
			return expectedResp, nil
		},
	}

	urlStr := srv.TestRun(t)

	checkResponse := func(t *testing.T, resp *http.Response) {
		t.Helper()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var gotResp playmock.CompileResponse
		if err := json.NewDecoder(resp.Body).Decode(&gotResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if gotResp.Errors != expectedResp.Errors {
			t.Errorf("Errors: got %q, want %q", gotResp.Errors, expectedResp.Errors)
		}
		if len(gotResp.Events) != len(expectedResp.Events) {
			t.Errorf("Events: got %d, want %d", len(gotResp.Events), len(expectedResp.Events))
		} else {
			for i := range expectedResp.Events {
				if gotResp.Events[i] != expectedResp.Events[i] {
					t.Errorf("Event[%d]: got %+v, want %+v", i, gotResp.Events[i], expectedResp.Events[i])
				}
			}
		}
		if gotResp.Status != expectedResp.Status {
			t.Errorf("Status: got %d, want %d", gotResp.Status, expectedResp.Status)
		}
		if gotResp.IsTest != expectedResp.IsTest {
			t.Errorf("IsTest: got %v, want %v", gotResp.IsTest, expectedResp.IsTest)
		}
		if gotResp.TestsFailed != expectedResp.TestsFailed {
			t.Errorf("TestsFailed: got %d, want %d", gotResp.TestsFailed, expectedResp.TestsFailed)
		}
	}

	t.Run("JSON", func(t *testing.T) {
		t.Parallel()

		reqBody, _ := json.Marshal(playmock.CompileRequest{
			Body: expectedBody,
		})
		resp, err := http.Post(urlStr+"/compile", "application/json", bytes.NewReader(reqBody))
		if err != nil {
			t.Fatalf("POST /compile failed: %v", err)
		}
		defer resp.Body.Close()
		checkResponse(t, resp)
	})

	t.Run("Form", func(t *testing.T) {
		t.Parallel()

		form := url.Values{}
		form.Add("body", expectedBody)
		form.Add("withVet", "false")

		resp, err := http.PostForm(urlStr+"/compile", form)
		if err != nil {
			t.Fatalf("POST /compile failed: %v", err)
		}
		defer resp.Body.Close()
		checkResponse(t, resp)
	})
}

func TestServerShare(t *testing.T) {
	t.Parallel()

	expectedBody := "package main"
	expectedID := "abcdef"

	srv := &playmock.Server{
		Share: func(req *playmock.ShareRequest) (*playmock.ShareResponse, error) {
			if req.Body != expectedBody {
				t.Errorf("unexpected body: got %q, want %q", req.Body, expectedBody)
			}
			return &playmock.ShareResponse{ID: expectedID}, nil
		},
	}

	urlStr := srv.TestRun(t)

	resp, err := http.Post(urlStr+"/share", "text/plain", strings.NewReader(expectedBody))
	if err != nil {
		t.Fatalf("POST /share failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != expectedID {
		t.Errorf("unexpected ID: got %q, want %q", string(body), expectedID)
	}
}

func TestServerNotImplemented(t *testing.T) {
	srv := &playmock.Server{}
	urlStr := srv.TestRun(t)

	resp, _ := http.Post(urlStr+"/compile", "application/json", strings.NewReader("{}"))
	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", resp.StatusCode)
	}

	resp, _ = http.Post(urlStr+"/share", "text/plain", strings.NewReader("test"))
	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", resp.StatusCode)
	}
}

func proxyClient(proxyURL string, caPEM []byte) (*http.Client, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("failed to load CA certificate")
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", proxyURL, err)
	}

	c := *http.DefaultClient
	c.Transport = &http.Transport{
		Proxy: func(*http.Request) (*url.URL, error) {
			return u, nil
		},
		TLSClientConfig: &tls.Config{RootCAs: pool},
	}

	return &c, nil
}

func TestProxyNotImplemented(t *testing.T) {
	t.Parallel()

	srv := &playmock.Server{}
	urlBase := "https://play.golang.org/_"

	proxyURL, caPEM := srv.TestRunProxy(t, urlBase)

	c, err := proxyClient(proxyURL, caPEM)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.Post(urlBase+"/compile", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", resp.StatusCode)
	}
}
