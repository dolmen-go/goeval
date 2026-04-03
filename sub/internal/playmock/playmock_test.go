package playmock

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestServerCompileJSON(t *testing.T) {
	ctx := t.Context()

	expectedBody := "package main\nimport \"fmt\"\nfunc main() {\n  fmt.Println(\"Hello, world!\")\n}\n"
	expectedResp := &CompileResponse{
		Errors: "",
		Events: []CompileEvent{
			{Message: "Hello, world!\n", Kind: "stdout", Delay: 0},
		},
		Status:      0,
		IsTest:      false,
		TestsFailed: 0,
	}

	srv := &Server{
		Compile: func(req *CompileRequest) (*CompileResponse, error) {
			if req.Body != expectedBody {
				t.Errorf("unexpected body: got %q, want %q", req.Body, expectedBody)
			}
			return expectedResp, nil
		},
	}

	urlStr, err := srv.Run(ctx)
	if err != nil {
		t.Fatalf("failed to run server: %v", err)
	}

	reqBody, _ := json.Marshal(CompileRequest{
		Body: expectedBody,
	})
	resp, err := http.Post(urlStr+"/compile", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("POST /compile failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var gotResp CompileResponse
	if err := json.NewDecoder(resp.Body).Decode(&gotResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if gotResp.Errors != expectedResp.Errors {
		t.Errorf("Errors: got %q, want %q", gotResp.Errors, expectedResp.Errors)
	}
	if len(gotResp.Events) != len(expectedResp.Events) || gotResp.Events[0].Message != expectedResp.Events[0].Message {
		t.Errorf("Events: got %+v, want %+v", gotResp.Events, expectedResp.Events)
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

func TestServerCompileForm(t *testing.T) {
	ctx := t.Context()

	expectedBody := "package main\nfunc main() { println(\"hello\") }"
	srv := &Server{
		Compile: func(req *CompileRequest) (*CompileResponse, error) {
			if req.Body != expectedBody {
				t.Errorf("unexpected body: got %q, want %q", req.Body, expectedBody)
			}
			if req.WithVet {
				t.Errorf("expected WithVet to be false")
			}
			return &CompileResponse{}, nil
		},
	}

	urlStr, err := srv.Run(ctx)
	if err != nil {
		t.Fatalf("failed to run server: %v", err)
	}

	form := url.Values{}
	form.Add("body", expectedBody)
	form.Add("withVet", "false")

	resp, err := http.PostForm(urlStr+"/compile", form)
	if err != nil {
		t.Fatalf("POST /compile failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestServerShare(t *testing.T) {
	ctx := t.Context()

	expectedBody := "package main"
	expectedID := "abcdef"

	srv := &Server{
		Share: func(req *ShareRequest) (*ShareResponse, error) {
			if req.Body != expectedBody {
				t.Errorf("unexpected body: got %q, want %q", req.Body, expectedBody)
			}
			return &ShareResponse{ID: expectedID}, nil
		},
	}

	urlStr, err := srv.Run(ctx)
	if err != nil {
		t.Fatalf("failed to run server: %v", err)
	}

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
	ctx := t.Context()

	srv := &Server{}
	urlStr, err := srv.Run(ctx)
	if err != nil {
		t.Fatalf("failed to run server: %v", err)
	}

	resp, _ := http.Post(urlStr+"/compile", "application/json", strings.NewReader("{}"))
	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", resp.StatusCode)
	}

	resp, _ = http.Post(urlStr+"/share", "text/plain", strings.NewReader("test"))
	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", resp.StatusCode)
	}
}
