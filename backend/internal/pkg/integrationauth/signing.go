// Package integrationauth provides authentication primitives for trusted
// Sub2API-to-Sub2API management calls. It deliberately does not log or expose
// the request body; callers must keep credentials out of logs and audit text.
package integrationauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	HeaderID        = "X-Sub2-Integration-Id"
	HeaderTimestamp = "X-Sub2-Integration-Timestamp"
	HeaderNonce     = "X-Sub2-Integration-Nonce"
	HeaderSignature = "X-Sub2-Integration-Signature"
	DefaultClockSkew = 5 * time.Minute
)

var (
	ErrMissingHeaders = errors.New("integration authentication headers are incomplete")
	ErrInvalidTimestamp = errors.New("integration authentication timestamp is invalid")
	ErrTimestampExpired = errors.New("integration authentication timestamp is expired")
	ErrInvalidSignature = errors.New("integration authentication signature is invalid")
)

// CanonicalString is intentionally stable and includes the body digest rather
// than the body itself, so signatures remain bounded for large imports.
func CanonicalString(method, path, timestamp, nonce string, body []byte) string {
	bodySum := sha256.Sum256(body)
	return strings.ToUpper(method) + "\n" + path + "\n" + timestamp + "\n" + nonce + "\n" + hex.EncodeToString(bodySum[:])
}

func Sign(secret, method, path, timestamp, nonce string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(CanonicalString(method, path, timestamp, nonce, body)))
	return hex.EncodeToString(mac.Sum(nil))
}

func SignRequest(req *http.Request, secret string, body []byte, now time.Time, nonce string) {
	timestamp := strconv.FormatInt(now.Unix(), 10)
	req.Header.Set(HeaderTimestamp, timestamp)
	req.Header.Set(HeaderNonce, nonce)
	req.Header.Set(HeaderSignature, Sign(secret, req.Method, requestPath(req), timestamp, nonce, body))
}

// Verify validates the signed request. Replay protection belongs to the
// integration handler because it must be shared with request idempotency.
func Verify(req *http.Request, secret string, body []byte, now time.Time, skew time.Duration) error {
	if strings.TrimSpace(secret) == "" {
		return ErrInvalidSignature
	}
	id := strings.TrimSpace(req.Header.Get(HeaderID))
	timestamp := strings.TrimSpace(req.Header.Get(HeaderTimestamp))
	nonce := strings.TrimSpace(req.Header.Get(HeaderNonce))
	signature := strings.TrimSpace(req.Header.Get(HeaderSignature))
	if id == "" || timestamp == "" || nonce == "" || signature == "" {
		return ErrMissingHeaders
	}
	parsed, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return ErrInvalidTimestamp
	}
	if skew <= 0 {
		skew = DefaultClockSkew
	}
	stamp := time.Unix(parsed, 0)
	if now.Sub(stamp) > skew || stamp.Sub(now) > skew {
		return ErrTimestampExpired
	}
	want := Sign(secret, req.Method, requestPath(req), timestamp, nonce, body)
	got, err := hex.DecodeString(signature)
	if err != nil || !hmac.Equal(got, mustDecodeHex(want)) {
		return fmt.Errorf("%w", ErrInvalidSignature)
	}
	return nil
}

func requestPath(req *http.Request) string {
	path := req.URL.EscapedPath()
	if path == "" {
		path = "/"
	}
	if req.URL.RawQuery != "" {
		path += "?" + req.URL.RawQuery
	}
	return path
}

func mustDecodeHex(value string) []byte {
	decoded, _ := hex.DecodeString(value)
	return decoded
}
