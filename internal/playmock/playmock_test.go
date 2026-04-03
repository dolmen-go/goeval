package playmock_test

import (
	"bytes"
	"encoding/json"
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
