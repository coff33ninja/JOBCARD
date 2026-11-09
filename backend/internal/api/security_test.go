package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"jobcard/internal/core"
)

func testServer(t *testing.T, token string) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	svc, err := core.NewService(dir, filepath.Join(dir, "Clients"), func() string { return "2026-11-08" })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { svc.Close() })
	ts := httptest.NewServer((&Server{Svc: svc, Loopback: true, Token: token}).Handler())
	t.Cleanup(ts.Close)
	return ts
}

func do(t *testing.T, method, url, body string, hdr map[string]string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(method, url, strings.NewReader(body))
	if body == "" {
		req.Body = nil
		req.ContentLength = 0
	}
	for k, v := range hdr {
		if k == "Host" {
			req.Host = v
			continue
		}
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

var jsonHdr = map[string]string{"Content-Type": "application/json"}

func TestGuardBlocksBrowserAttacks(t *testing.T) {
	ts := testServer(t, "")
	api := ts.URL + "/api/v1"

	if r := do(t, "POST", api+"/clients", `{"name":"A"}`, jsonHdr); r.StatusCode != 201 {
		t.Fatalf("normal request: %d", r.StatusCode)
	}
	local := map[string]string{"Content-Type": "application/json", "Origin": "http://localhost:3000"}
	if r := do(t, "POST", api+"/clients", `{"name":"B"}`, local); r.StatusCode != 201 {
		t.Fatalf("local dev origin must work: %d", r.StatusCode)
	}
	// cross-site form POST: browsers send it without a preflight
	evil := map[string]string{"Content-Type": "text/plain", "Origin": "https://evil.example"}
	if r := do(t, "POST", api+"/clients", `{"name":"X"}`, evil); r.StatusCode != 403 {
		t.Fatalf("cross-site origin: %d", r.StatusCode)
	}
	// cross-site request with no body at all (e.g. issue, convert)
	if r := do(t, "POST", api+"/documents/x/issue", "", map[string]string{"Origin": "https://evil.example"}); r.StatusCode != 403 {
		t.Fatalf("cross-site empty POST: %d", r.StatusCode)
	}
	// simple-request content type without an Origin header
	if r := do(t, "POST", api+"/clients", `{"name":"X"}`, map[string]string{"Content-Type": "text/plain"}); r.StatusCode != 415 {
		t.Fatalf("text/plain body: %d", r.StatusCode)
	}
	// DNS rebinding: attacker hostname resolving to 127.0.0.1
	if r := do(t, "GET", api+"/clients", "", map[string]string{"Host": "evil.example:8080"}); r.StatusCode != 403 {
		t.Fatalf("rebinding host: %d", r.StatusCode)
	}
	// bodyless POST from the UI passes the guard and reaches the app (404 = unknown id)
	if r := do(t, "POST", api+"/documents/missing/issue", "", nil); r.StatusCode != 404 {
		t.Fatalf("bodyless POST: %d", r.StatusCode)
	}
}

func TestTokenMode(t *testing.T) {
	ts := testServer(t, "s3cret")
	api := ts.URL + "/api/v1/clients"
	if r := do(t, "GET", api, "", nil); r.StatusCode != 401 {
		t.Fatalf("no token: %d", r.StatusCode)
	}
	if r := do(t, "GET", api, "", map[string]string{"Authorization": "Bearer nope"}); r.StatusCode != 401 {
		t.Fatalf("wrong token: %d", r.StatusCode)
	}
	if r := do(t, "GET", api, "", map[string]string{"Authorization": "s3cret"}); r.StatusCode != 401 {
		t.Fatalf("token without Bearer scheme: %d", r.StatusCode)
	}
	if r := do(t, "GET", api, "", map[string]string{"Authorization": "Bearer s3cret"}); r.StatusCode != 200 {
		t.Fatalf("right token: %d", r.StatusCode)
	}
}

func TestLoopbackAddr(t *testing.T) {
	for addr, want := range map[string]bool{"127.0.0.1:8080": true, "localhost:8080": true, "[::1]:8080": true, "0.0.0.0:8080": false, ":8080": false, "192.168.0.5:8080": false} {
		if IsLoopbackAddr(addr) != want {
			t.Errorf("IsLoopbackAddr(%s) want %v", addr, want)
		}
	}
}
