package integrationclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/integrationauth"
)

func TestDoRawPreservesSignedRequestAndResponse(t *testing.T) {
	const secret = "integration-test-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if r.URL.RequestURI() != "/api/v1/integration/admin/accounts?page=2" {
			t.Fatalf("request URI = %q", r.URL.RequestURI())
		}
		if err := integrationauth.Verify(r, secret, body, time.Now(), time.Minute); err != nil {
			t.Fatalf("verify signed request: %v", err)
		}
		if string(body) != `{"name":"remote"}` {
			t.Fatalf("request body = %q", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"remote-etag"`)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":42}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "hongai", secret)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	status, headers, body, err := client.DoRaw(
		context.Background(),
		http.MethodPost,
		"/api/v1/integration/admin/accounts?page=2",
		[]byte(`{"name":"remote"}`),
	)
	if err != nil {
		t.Fatalf("DoRaw() error = %v", err)
	}
	if status != http.StatusCreated {
		t.Fatalf("status = %d", status)
	}
	if headers.Get("ETag") != `"remote-etag"` {
		t.Fatalf("ETag = %q", headers.Get("ETag"))
	}
	if string(body) != `{"id":42}` {
		t.Fatalf("response body = %q", string(body))
	}
}
