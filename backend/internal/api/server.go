// Package api is the HTTP layer: thin handlers over core.Service, JSON in and out.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"jobcard/internal/core"
	"jobcard/internal/export"
)

// Server serves the JobCard API and optional static frontend.
type Server struct {
	Svc       *core.Service
	Templates export.Templates // editable .docx layouts
	Static    string           // optional: directory with the built UI (dist)

	// Loopback is true when the listener is bound to localhost only (enables the Host check).
	Loopback bool
	// Token, when set, is required as "Authorization: Bearer <token>" on every /api/ request.
	Token string
}

// Handler returns the secured HTTP handler for the API and optional frontend.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	const v = "/api/v1"
	mux.HandleFunc("GET "+v+"/clients", s.listClients)
	mux.HandleFunc("GET "+v+"/clients/archived", s.listArchivedClients)
	mux.HandleFunc("POST "+v+"/clients", s.createClient)
	mux.HandleFunc("PUT "+v+"/clients/{id}", s.updateClient)
	mux.HandleFunc("DELETE "+v+"/clients/{id}", s.archiveClient)
	mux.HandleFunc("POST "+v+"/clients/{id}/restore", s.restoreClient)
	mux.HandleFunc("GET "+v+"/documents", s.listDocuments)
	mux.HandleFunc("GET "+v+"/documents/{id}", s.getDocument)
	mux.HandleFunc("POST "+v+"/documents", s.createDraft)
	mux.HandleFunc("PUT "+v+"/documents/{id}", s.updateDraft)
	mux.HandleFunc("POST "+v+"/documents/{id}/issue", s.issue)
	mux.HandleFunc("POST "+v+"/documents/{id}/follow-up", s.setFollowUp)
	mux.HandleFunc("POST "+v+"/documents/{id}/payments", s.addPayment)
	mux.HandleFunc("POST "+v+"/documents/{id}/status", s.setStatus)
	mux.HandleFunc("POST "+v+"/quotations/{id}/convert", s.convert)
	mux.HandleFunc("POST "+v+"/documents/{id}/export", s.exportDoc)
	mux.HandleFunc("POST "+v+"/documents/{id}/reveal", s.reveal)
	mux.HandleFunc("GET "+v+"/dashboard", s.dashboard)
	mux.HandleFunc("GET "+v+"/settings", s.getSettings)
	mux.HandleFunc("PUT "+v+"/settings", s.putSettings)
	mux.HandleFunc("GET "+v+"/storage", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, s.Svc.Storage()) })
	mux.HandleFunc("GET "+v+"/capabilities", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, export.DetectCapabilities()) })
	mux.HandleFunc("GET "+v+"/register/export.xlsx", s.registerXLSX)
	mux.HandleFunc(v+"/", func(w http.ResponseWriter, _ *http.Request) {
		writeErr(w, &core.AppError{Status: 404, Code: "NOT_FOUND", Message: "No such endpoint"})
	})
	if s.Static != "" {
		mux.HandleFunc("/", s.static)
	}
	return s.secure(cors(s.auditWrites(mux)))
}

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	var ae *core.AppError
	if !errors.As(err, &ae) {
		log.Printf("internal error: %v", err)
		ae = &core.AppError{Status: 500, Code: "INTERNAL_ERROR", Message: "Internal server error"}
	}
	body := map[string]any{"code": ae.Code, "message": ae.Message}
	if len(ae.Fields) > 0 {
		body["fields"] = ae.Fields
	}
	writeJSON(w, ae.Status, map[string]any{"error": body})
}

func bad(msg string) error {
	return &core.AppError{Status: 400, Code: "VALIDATION_FAILED", Message: msg}
}

func decode(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return bad("Request body must be valid JSON")
	}
	return nil
}

// cors lets the Vite dev server (any localhost port) call the API.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" {
			if u, err := url.Parse(o); err == nil {
				h, _, _ := net.SplitHostPort(u.Host)
				if h == "" {
					h = u.Host
				}
				if h == "localhost" || h == "127.0.0.1" || h == "[::1]" || h == "::1" {
					w.Header().Set("Access-Control-Allow-Origin", o)
					w.Header().Set("Vary", "Origin")
					w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
					w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition, X-Saved-Path")
				}
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	p := filepath.Join(s.Static, filepath.FromSlash(strings.TrimPrefix(filepath.ToSlash(filepath.Clean("/"+r.URL.Path)), "/")))
	if st, err := os.Stat(p); err == nil && !st.IsDir() {
		http.ServeFile(w, r, p)
		return
	}
	http.ServeFile(w, r, filepath.Join(s.Static, "index.html"))
}

// ---------- clients ----------

func (s *Server) listClients(w http.ResponseWriter, _ *http.Request) {
	v, err := s.Svc.ListClients()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) listArchivedClients(w http.ResponseWriter, _ *http.Request) {
	v, err := s.Svc.ListArchivedClients()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) createClient(w http.ResponseWriter, r *http.Request) {
	var in core.ClientInput
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	c, err := s.Svc.CreateClient(in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 201, c)
}

func (s *Server) updateClient(w http.ResponseWriter, r *http.Request) {
	var in core.ClientInput
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	c, err := s.Svc.UpdateClient(r.PathValue("id"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, c)
}

func (s *Server) archiveClient(w http.ResponseWriter, r *http.Request) {
	c, err := s.Svc.ArchiveClient(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, c)
}

func (s *Server) restoreClient(w http.ResponseWriter, r *http.Request) {
	c, err := s.Svc.RestoreClient(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, c)
}

// ---------- documents ----------

func (s *Server) listDocuments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	docs, err := s.Svc.ListDocuments(core.DocFilters{
		Type: q.Get("type"), Status: q.Get("status"), ClientID: q.Get("clientId"), Q: q.Get("q"), From: q.Get("from"), To: q.Get("to"),
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, docs)
}

func (s *Server) getDocument(w http.ResponseWriter, r *http.Request) {
	d, err := s.Svc.GetDocument(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, d)
}

func (s *Server) createDraft(w http.ResponseWriter, r *http.Request) {
	var in core.DraftInput
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	d, err := s.Svc.CreateDraft(in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 201, d)
}

func (s *Server) updateDraft(w http.ResponseWriter, r *http.Request) {
	var in core.DraftInput
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	d, err := s.Svc.UpdateDraft(r.PathValue("id"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, d)
}

func (s *Server) issue(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IssueDate string `json:"issueDate"`
	}
	if r.ContentLength != 0 {
		if err := decode(r, &body); err != nil {
			writeErr(w, err)
			return
		}
	}
	d, err := s.Svc.IssueDocument(r.PathValue("id"), body.IssueDate)
	if err != nil {
		writeErr(w, err)
		return
	}
	if _, ferr := s.Svc.EnsureClientFolder(d.ClientID); ferr != nil {
		log.Printf("could not create client folder: %v", ferr) // numbering already committed; folder is retried on export
	}
	writeJSON(w, 200, d)
}

func (s *Server) setFollowUp(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FollowUpDate json.RawMessage `json:"followUpDate"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	if body.FollowUpDate == nil {
		writeErr(w, bad("followUpDate is required"))
		return
	}

	var followUpDate *string
	if string(body.FollowUpDate) != "null" {
		var value string
		if err := json.Unmarshal(body.FollowUpDate, &value); err != nil {
			writeErr(w, bad("followUpDate must be a date or null"))
			return
		}
		followUpDate = &value
	}
	d, err := s.Svc.SetFollowUpDate(r.PathValue("id"), followUpDate)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) addPayment(w http.ResponseWriter, r *http.Request) {
	var in core.PaymentInput
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	d, err := s.Svc.AddPayment(r.PathValue("id"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 201, d)
}

func (s *Server) setStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string `json:"status"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	if strings.TrimSpace(in.Status) == "" {
		writeErr(w, &core.AppError{Status: 400, Code: "VALIDATION_FAILED", Message: "status is required", Fields: map[string]string{"status": "Required"}})
		return
	}
	d, err := s.Svc.SetStatus(r.PathValue("id"), in.Status)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, d)
}

func (s *Server) convert(w http.ResponseWriter, r *http.Request) {
	d, err := s.Svc.ConvertQuotation(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 201, d)
}

// ---------- export / reveal ----------

func (s *Server) exportDoc(w http.ResponseWriter, r *http.Request) {
	res, err := s.Svc.ExportDocument(r.PathValue("id"), r.URL.Query().Get("format"), s.Templates)
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", res.Mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, res.Filename))
	w.Header().Set("X-Saved-Path", res.SavedPath)
	w.WriteHeader(200)
	_, _ = w.Write(res.Data)
}

func (s *Server) reveal(w http.ResponseWriter, r *http.Request) {
	folder, err := s.Svc.RevealFolder(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"success": true, "path": folder})
}

// ---------- dashboard / settings / register ----------

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	year := 0
	if y := r.URL.Query().Get("year"); y != "" {
		n, err := strconv.Atoi(y)
		if err != nil || n < 1970 || n > 9999 {
			writeErr(w, bad("year must be a four digit year"))
			return
		}
		year = n
	}
	d, err := s.Svc.Dashboard(year)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, d)
}

func (s *Server) getSettings(w http.ResponseWriter, _ *http.Request) {
	st, err := s.Svc.GetSettings()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, st)
}

func (s *Server) putSettings(w http.ResponseWriter, r *http.Request) {
	st, err := s.Svc.GetSettings() // start from current so omitted fields keep their value
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := decode(r, &st); err != nil {
		writeErr(w, err)
		return
	}
	out, err := s.Svc.UpdateSettings(st)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, out)
}

func (s *Server) registerXLSX(w http.ResponseWriter, _ *http.Request) {
	data, name, err := s.Svc.RegisterSpreadsheet()
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	w.WriteHeader(200)
	_, _ = w.Write(data)
}

// auditWrites records every state-changing API call (method, path, status). Bodies are not stored.
func (s *Server) auditWrites(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodDelete) || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		s.Svc.Audit("api", r.Method+" "+r.URL.Path, "", fmt.Sprintf("status %d", rec.status))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
