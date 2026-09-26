// Package plugin is the small runtime SDK for daoliyu MCP plugins.
package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

type Request struct {
	Method    string          `json:"method"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	Config    json.RawMessage `json:"config,omitempty"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type Result struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

type Handler func(context.Context, json.RawMessage, json.RawMessage) (any, error)

type Server struct{ handlers map[string]Handler }

func NewServer() *Server { return &Server{handlers: map[string]Handler{}} }

func (s *Server) Handle(name string, handler Handler) { s.handlers[name] = handler }

func (s *Server) Run(ctx context.Context, r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64<<10), 8<<20)
	enc := json.NewEncoder(w)
	for scanner.Scan() {
		var req Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			if err := enc.Encode(errorResult(err)); err != nil {
				return err
			}
			continue
		}
		result := s.call(ctx, req)
		if err := enc.Encode(result); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s *Server) call(ctx context.Context, req Request) Result {
	if req.Method != "tools/call" {
		return errorResult(fmt.Errorf("unsupported method %q", req.Method))
	}
	handler := s.handlers[req.Name]
	if handler == nil {
		return errorResult(fmt.Errorf("unknown tool %q", req.Name))
	}
	value, err := handler(ctx, req.Arguments, req.Config)
	if err != nil {
		return errorResult(err)
	}
	b, err := json.Marshal(value)
	if err != nil {
		return errorResult(err)
	}
	return Result{Content: []Content{{Type: "text", Text: string(b)}}}
}

func errorResult(err error) Result {
	return Result{IsError: true, Content: []Content{{Type: "text", Text: err.Error()}}}
}
