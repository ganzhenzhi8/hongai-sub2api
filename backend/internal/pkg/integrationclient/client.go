// Package integrationclient is the hongai-side HTTPS client for the opt-in
// Sub2API integration API. It keeps the shared secret server-side and never
// includes credentials in errors or logs.
package integrationclient

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/integrationauth"
)

type Client struct {
	BaseURL    string
	ID         string
	Secret     string
	HTTPClient *http.Client
}

func New(baseURL, id, secret string) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || strings.TrimSpace(id) == "" || strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("integration client is not configured")
	}
	return &Client{BaseURL: baseURL, ID: strings.TrimSpace(id), Secret: secret, HTTPClient: &http.Client{Timeout: 30 * time.Second}}, nil
}

func (c *Client) DoJSON(ctx context.Context, method, path string, request any, response any) error {
	var body []byte
	var err error
	if request != nil {
		body, err = json.Marshal(request)
		if err != nil {
			return fmt.Errorf("encode integration request: %w", err)
		}
	}
	status, _, respBody, err := c.DoRaw(ctx, method, path, body)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("integration request returned status %d", status)
	}
	if response != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, response); err != nil {
			return fmt.Errorf("decode integration response")
		}
	}
	return nil
}

// DoRaw performs one signed integration request while preserving the remote
// status, headers, and response body. It is used by the restricted admin
// bridge so the original Sub2API frontend receives the same payload shape as
// it would from a local admin endpoint.
func (c *Client) DoRaw(ctx context.Context, method, path string, body []byte) (int, http.Header, []byte, error) {
	resp, err := c.Do(ctx, method, path, body)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return 0, nil, nil, fmt.Errorf("read integration response")
	}
	return resp.StatusCode, resp.Header.Clone(), respBody, nil
}

// Do returns the signed response without consuming its body. Callers must
// close the body; the admin bridge uses this form to preserve SSE streaming.
func (c *Client) Do(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create integration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Sub2API-Integration/1.0")
	req.Header.Set(integrationauth.HeaderID, c.ID)
	nonce, err := randomNonce()
	if err != nil {
		return nil, fmt.Errorf("create integration nonce: %w", err)
	}
	integrationauth.SignRequest(req, c.Secret, body, time.Now(), nonce)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("integration request failed")
	}
	return resp, nil
}

func randomNonce() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
