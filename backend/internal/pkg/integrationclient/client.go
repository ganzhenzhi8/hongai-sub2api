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
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create integration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Sub2API-Integration/1.0")
	req.Header.Set(integrationauth.HeaderID, c.ID)
	nonce, err := randomNonce()
	if err != nil {
		return fmt.Errorf("create integration nonce: %w", err)
	}
	integrationauth.SignRequest(req, c.Secret, body, time.Now(), nonce)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("integration request failed")
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return fmt.Errorf("read integration response")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("integration request returned status %d", resp.StatusCode)
	}
	if response != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, response); err != nil {
			return fmt.Errorf("decode integration response")
		}
	}
	return nil
}

func randomNonce() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
