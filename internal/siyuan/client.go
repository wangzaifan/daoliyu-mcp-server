package siyuan

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL, Token string
	HTTP           *http.Client
}
type response struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func NewClient(baseURL, token string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Token: token, HTTP: &http.Client{Timeout: 12 * time.Second}}
}

func (c *Client) Post(ctx context.Context, path string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Token "+c.Token)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("SiYuan HTTP %s", res.Status)
	}
	var envelope response
	if err := json.NewDecoder(io.LimitReader(res.Body, 16<<20)).Decode(&envelope); err != nil {
		return err
	}
	if envelope.Code != 0 {
		return fmt.Errorf("SiYuan API %d: %s", envelope.Code, envelope.Msg)
	}
	if out != nil && len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			return err
		}
	}
	return nil
}

type Notebook struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Closed bool   `json:"closed"`
}

func (c *Client) ListNotebooks(ctx context.Context) ([]Notebook, error) {
	var out struct {
		Notebooks []Notebook `json:"notebooks"`
	}
	err := c.Post(ctx, "/api/notebook/lsNotebooks", map[string]any{}, &out)
	return out.Notebooks, err
}
func (c *Client) Query(ctx context.Context, stmt string, out any) error {
	return c.Post(ctx, "/api/query/sql", map[string]string{"stmt": stmt}, out)
}
func (c *Client) ExportDocument(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	err := c.Post(ctx, "/api/export/exportMdContent", map[string]string{"id": id}, &out)
	return out, err
}
func (c *Client) CreateDocument(ctx context.Context, notebook, path, markdown string) (string, error) {
	var out string
	err := c.Post(ctx, "/api/filetree/createDocWithMd", map[string]string{"notebook": notebook, "path": path, "markdown": markdown}, &out)
	return out, err
}

func (c *Client) UpdateBlock(ctx context.Context, id, markdown string) error {
	return c.Post(ctx, "/api/block/updateBlock", map[string]any{
		"dataType": "markdown",
		"data":     markdown,
		"id":       id,
		"lockType": false,
	}, nil)
}

func (c *Client) SetBlockAttrs(ctx context.Context, id string, attrs map[string]string) error {
	return c.Post(ctx, "/api/attr/setBlockAttrs", map[string]any{"id": id, "attrs": attrs}, nil)
}

func (c *Client) GetBlockAttrs(ctx context.Context, id string) (map[string]string, error) {
	var out map[string]string
	err := c.Post(ctx, "/api/attr/getBlockAttrs", map[string]string{"id": id}, &out)
	return out, err
}
