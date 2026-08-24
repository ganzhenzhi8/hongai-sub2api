package integrationauth

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSignAndVerify(t *testing.T) {
	now := time.Unix(1_750_000_000, 0)
	req, err := http.NewRequest(http.MethodPost, "https://example.test/api/v1/integration/accounts", strings.NewReader(`{"name":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"name":"x"}`)
	req.Header.Set(HeaderID, "hongai")
	SignRequest(req, "test-secret", body, now, "n-1")
	if err := Verify(req, "test-secret", body, now, 0); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestVerifyRejectsBodyChange(t *testing.T) {
	now := time.Unix(1_750_000_000, 0)
	req, _ := http.NewRequest(http.MethodPost, "https://example.test/api/v1/integration/accounts", nil)
	req.Header.Set(HeaderID, "hongai")
	body := []byte(`{"name":"x"}`)
	SignRequest(req, "test-secret", body, now, "n-1")
	if err := Verify(req, "test-secret", []byte(`{"name":"y"}`), now, 0); err == nil {
		t.Fatal("expected body tampering to be rejected")
	}
}

func TestVerifyRejectsQueryChange(t *testing.T) {
	now := time.Unix(1_750_000_000, 0)
	req, _ := http.NewRequest(http.MethodGet, "https://example.test/api/v1/integration/accounts?page=1", nil)
	req.Header.Set(HeaderID, "hongai")
	SignRequest(req, "test-secret", nil, now, "n-query")
	req.URL.RawQuery = "page=2"
	if err := Verify(req, "test-secret", nil, now, 0); err == nil {
		t.Fatal("expected query tampering to be rejected")
	}
}

func TestVerifyRejectsExpiredTimestamp(t *testing.T) {
	now := time.Unix(1_750_000_000, 0)
	req, _ := http.NewRequest(http.MethodGet, "https://example.test/api/v1/integration/groups", nil)
	req.Header.Set(HeaderID, "hongai")
	SignRequest(req, "test-secret", nil, now.Add(-10*time.Minute), "n-1")
	if err := Verify(req, "test-secret", nil, now, time.Minute); err == nil {
		t.Fatal("expected expired timestamp to be rejected")
	}
}
