package siyuan

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientUsesSiYuanEnvelopeAndToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Token secret" {
			t.Errorf("missing token")
		}
		if !strings.HasSuffix(r.URL.Path, "/api/notebook/lsNotebooks") {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"msg":"","data":{"notebooks":[{"id":"1","name":"n"}]}}`))
	}))
	defer ts.Close()
	ns, err := NewClient(ts.URL, "secret").ListNotebooks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ns) != 1 || ns[0].ID != "1" {
		t.Fatalf("%+v", ns)
	}
}
