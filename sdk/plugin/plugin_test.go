package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestServerRunsDocumentedCallProtocol(t *testing.T) {
	s := NewServer()
	s.Handle("echo", func(_ context.Context, _ json.RawMessage, config json.RawMessage) (any, error) {
		var cfg map[string]string
		if err := json.Unmarshal(config, &cfg); err != nil {
			return nil, err
		}
		return map[string]string{"ok": "yes" + cfg["suffix"]}, nil
	})
	var out bytes.Buffer
	if err := s.Run(context.Background(), strings.NewReader(`{"method":"tools/call","name":"echo","arguments":{},"config":{"suffix":"!"}}`+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `yes!`) || !strings.Contains(out.String(), `"type":"text"`) {
		t.Fatalf("unexpected result: %s", out.String())
	}
}
