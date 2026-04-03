package playmock

import (
	"context"
	"time"
)

type Server struct {
	Compile func(*CompileRequest) (*CompileResponse, error)
	Share   func(*ShareRequest) (*ShareResponse, error)
}

type CompileRequest struct {
	Body    string `json:"Body"`
	WithVet bool   `json:"WithVet"`
}

type CompileResponse struct {
	Errors      string         `json:"Errors,omitempty"`
	Events      []CompileEvent `json:"Events,omitempty"`
	Status      int            `json:"Status,omitempty"`
	IsTest      bool           `json:"IsTest,omitempty"`
	TestsFailed int            `json:"TestsFailed,omitempty"`
	VetErrors   string         `json:"VetErrors,omitempty"`
	VetOK       bool           `json:"VetOK,omitempty"`
}

type CompileEvent struct {
	Message string        `json:"Message,omitempty"`
	Kind    string        `json:"Kind,omitempty"`
	Delay   time.Duration `json:"Delay,omitempty"`
}

type ShareRequest struct {
	Body string
}

type ShareResponse struct {
	ID string
}

// Run launches an HTTP server mocking the Go Playground.
func (*Server) Run(ctx context.Context) (u string, _ error) {
	return "", nil
}
