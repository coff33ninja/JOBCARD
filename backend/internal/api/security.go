package api

import (
	"crypto/subtle"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"

	"jobcard/internal/core"
)

// secure is the request guard. Defaults protect a localhost-only server from the browser
// (cross-site requests, DNS rebinding); a bearer token adds real authentication for LAN use.
//
//   - Host must be localhost / 127.0.0.1 / ::1 when the server is bound to loopback (blocks DNS rebinding).
//   - A browser Origin must be local or the server's own origin (blocks cross-site POSTs, which
//     browsers send without a preflight for simple requests).
//   - Requests with a body must be application/json.
//   - When Token is set, every /api/ request needs "Authorization: Bearer <token>".
func (s *Server) secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.Loopback && !isLoopbackHost(r.Host) {
			writeErr(w, &core.AppError{Status: 403, Code: "FORBIDDEN_HOST", Message: "Host not allowed"})
			return
		}
		if o := r.Header.Get("Origin"); o != "" && !originAllowed(o, r.Host) {
			writeErr(w, &core.AppError{Status: 403, Code: "FORBIDDEN_ORIGIN", Message: "Cross-site requests are not allowed"})
			return
		}
		if s.Token != "" && strings.HasPrefix(r.URL.Path, "/api/") && r.Method != http.MethodOptions {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if got == r.Header.Get("Authorization") || subtle.ConstantTimeCompare([]byte(got), []byte(s.Token)) != 1 {
				w.Header().Set("WWW-Authenticate", `Bearer realm="jobcard"`)
				writeErr(w, &core.AppError{Status: 401, Code: "UNAUTHORIZED", Message: "A valid bearer token is required"})
				return
			}
		}
		if (r.Method == http.MethodPost || r.Method == http.MethodPut) && r.ContentLength != 0 {
			mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mt != "application/json" {
				writeErr(w, &core.AppError{Status: 415, Code: "UNSUPPORTED_MEDIA_TYPE", Message: "Request bodies must be application/json"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func hostname(hostport string) string {
	h, _, err := net.SplitHostPort(hostport)
	if err != nil {
		h = hostport
	}
	return strings.Trim(h, "[]")
}

func isLoopbackName(h string) bool {
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

func isLoopbackHost(hostport string) bool { return isLoopbackName(hostname(hostport)) }

func originAllowed(origin, host string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return isLoopbackName(u.Hostname()) || u.Host == host
}

// IsLoopbackAddr reports whether a listen address (host:port) is bound to loopback only.
func IsLoopbackAddr(addr string) bool {
	h := hostname(addr)
	return isLoopbackName(h)
}
