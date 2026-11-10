// Package core is the single place business rules are enforced. The HTTP API
// (and later the MCP server) are thin wrappers over Service.
package core

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"jobcard/internal/domain"
)

// ---------- errors ----------

type AppError struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
}

func (e *AppError) Error() string { return e.Code + ": " + e.Message }

func appErr(status int, code, msg string) *AppError {
	return &AppError{Status: status, Code: code, Message: msg}
}

func invalid(msg string, fields map[string]string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: "VALIDATION_FAILED", Message: msg, Fields: fields}
}

func notFound(what, id string) *AppError {
	return appErr(http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("%s %s not found", what, id))
}

// ---------- optional JSON fields (absent vs null vs value) ----------

type Opt[T any] struct {
	Set bool
	Val *T
}

func (o *Opt[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Val = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Val = &v
	return nil
}

// Or returns the value when present and non-null, else def.
func (o Opt[T]) Or(def T) T {
	if o.Set && o.Val != nil {
		return *o.Val
	}
	return def
}

// ---------- service ----------

type Service struct {
	DB         *sql.DB
	Today      func() string
	DataDir    string
	DBPath     string
	ClientsDir string
}

type queryer interface {
	Query(q string, args ...any) (*sql.Rows, error)
	QueryRow(q string, args ...any) *sql.Row
	Exec(q string, args ...any) (sql.Result, error)
}

const schema = `
CREATE TABLE IF NOT EXISTS clients(
  id TEXT PRIMARY KEY, name TEXT NOT NULL,
  contact TEXT NOT NULL DEFAULT '', phone TEXT NOT NULL DEFAULT '', email TEXT NOT NULL DEFAULT '',
  address TEXT NOT NULL DEFAULT '', notes TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, archived_at TEXT);
CREATE TABLE IF NOT EXISTS documents(
  id TEXT PRIMARY KEY, type TEXT NOT NULL, number TEXT UNIQUE,
  client_id TEXT NOT NULL REFERENCES clients(id),
  job_card_id TEXT, quotation_id TEXT,
  issue_date TEXT, due_date TEXT, valid_until TEXT, follow_up_date TEXT,
  reference TEXT NOT NULL, scope TEXT NOT NULL DEFAULT '',
  work_items TEXT NOT NULL DEFAULT '[]', lines TEXT NOT NULL DEFAULT '[]',
  tax_cents INTEGER NOT NULL DEFAULT 0, subtotal_cents INTEGER NOT NULL DEFAULT 0, total_cents INTEGER NOT NULL DEFAULT 0,
  deposit_cents INTEGER, status TEXT NOT NULL,
  terms TEXT NOT NULL DEFAULT '', exclusions TEXT NOT NULL DEFAULT '', turnaround TEXT NOT NULL DEFAULT '', notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS idx_documents_client ON documents(client_id);
CREATE INDEX IF NOT EXISTS idx_documents_type ON documents(type);
CREATE TABLE IF NOT EXISTS payments(
  id TEXT PRIMARY KEY, document_id TEXT NOT NULL REFERENCES documents(id),
  date TEXT NOT NULL, amount_cents INTEGER NOT NULL, method TEXT NOT NULL, reference TEXT NOT NULL DEFAULT '');
CREATE INDEX IF NOT EXISTS idx_payments_doc ON payments(document_id);
CREATE TABLE IF NOT EXISTS settings(id INTEGER PRIMARY KEY CHECK(id=1), json TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS audit_log(
  id INTEGER PRIMARY KEY AUTOINCREMENT, ts TEXT NOT NULL, source TEXT NOT NULL,
  action TEXT NOT NULL, target TEXT NOT NULL DEFAULT '', detail TEXT NOT NULL DEFAULT '');
`

func NewService(dataDir, clientsDir string, today func() string) (*Service, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dataDir, "jobcard.sqlite")
	dsn := "file:" + filepath.ToSlash(dbPath) + "?_txlock=immediate&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // one writer; transactions serialise numbering
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	if err := ensureFollowUpDateColumn(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate follow-up date column: %w", err)
	}
	if err := ensureClientArchivedAtColumn(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate client archive column: %w", err)
	}
	s := &Service{DB: db, Today: today, DataDir: dataDir, DBPath: dbPath, ClientsDir: clientsDir}
	def, _ := json.Marshal(defaultSettings())
	if _, err := db.Exec(`INSERT OR IGNORE INTO settings(id, json) VALUES(1, ?)`, string(def)); err != nil {
		return nil, err
	}
	return s, nil
}

func ensureFollowUpDateColumn(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(documents)`)
	if err != nil {
		return err
	}
	found := false
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return err
		}
		if name == "follow_up_date" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if found {
		return nil
	}
	_, err = db.Exec(`ALTER TABLE documents ADD COLUMN follow_up_date TEXT`)
	return err
}

func ensureClientArchivedAtColumn(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(clients)`)
	if err != nil {
		return err
	}
	found := false
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return err
		}
		if name == "archived_at" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if found {
		return nil
	}
	_, err = db.Exec(`ALTER TABLE clients ADD COLUMN archived_at TEXT`)
	return err
}

func (s *Service) Close() error { return s.DB.Close() }

func defaultSettings() domain.Settings {
	return domain.Settings{DefaultTermsDays: 7, QuoteValidityDays: 14}
}

func newID(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

func nowISO() string { return time.Now().UTC().Format(time.RFC3339) }

func nullStr(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullInt(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

func strPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	v := ns.String
	return &v
}

// ---------- settings ----------

func getSettings(q queryer) (domain.Settings, error) {
	var js string
	if err := q.QueryRow(`SELECT json FROM settings WHERE id=1`).Scan(&js); err != nil {
		return defaultSettings(), err
	}
	st := defaultSettings()
	if err := json.Unmarshal([]byte(js), &st); err != nil {
		return st, err
	}
	return st, nil
}

func (s *Service) GetSettings() (domain.Settings, error) { return getSettings(s.DB) }

func (s *Service) UpdateSettings(in domain.Settings) (domain.Settings, error) {
	f := map[string]string{}
	if in.TaxRate < 0 || in.TaxRate > 1 {
		f["taxRate"] = "Must be between 0 and 1 (0.15 = 15%)"
	}
	if in.DefaultTermsDays < 0 || in.DefaultTermsDays > 365 {
		f["defaultTermsDays"] = "Must be between 0 and 365"
	}
	if in.QuoteValidityDays < 1 || in.QuoteValidityDays > 365 {
		f["quoteValidityDays"] = "Must be between 1 and 365"
	}
	if len(f) > 0 {
		return in, invalid("Invalid settings", f)
	}
	js, _ := json.Marshal(in)
	if _, err := s.DB.Exec(`UPDATE settings SET json=? WHERE id=1`, string(js)); err != nil {
		return in, err
	}
	return in, nil
}

// ---------- clients ----------

type ClientInput struct {
	Name    Opt[string] `json:"name"`
	Contact Opt[string] `json:"contact"`
	Phone   Opt[string] `json:"phone"`
	Email   Opt[string] `json:"email"`
	Address Opt[string] `json:"address"`
	Notes   Opt[string] `json:"notes"`
}

func (s *Service) ListClients() ([]domain.Client, error) {
	return s.listClients(false)
}

func (s *Service) ListArchivedClients() ([]domain.Client, error) {
	return s.listClients(true)
}

func (s *Service) listClients(archived bool) ([]domain.Client, error) {
	query := `SELECT id,name,contact,phone,email,address,notes FROM clients WHERE archived_at IS NULL`
	if archived {
		query = `SELECT id,name,contact,phone,email,address,notes FROM clients WHERE archived_at IS NOT NULL`
	}
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Client{}
	for rows.Next() {
		var c domain.Client
		if err := rows.Scan(&c.ID, &c.Name, &c.Contact, &c.Phone, &c.Email, &c.Address, &c.Notes); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out, rows.Err()
}

func getClient(q queryer, id string) (domain.Client, error) {
	var c domain.Client
	err := q.QueryRow(`SELECT id,name,contact,phone,email,address,notes FROM clients WHERE id=?`, id).
		Scan(&c.ID, &c.Name, &c.Contact, &c.Phone, &c.Email, &c.Address, &c.Notes)
	if errors.Is(err, sql.ErrNoRows) {
		return c, notFound("Client", id)
	}
	return c, err
}

func (s *Service) GetClient(id string) (domain.Client, error) { return getClient(s.DB, id) }

func getActiveClient(q queryer, id string) (domain.Client, error) {
	var c domain.Client
	err := q.QueryRow(`SELECT id,name,contact,phone,email,address,notes FROM clients WHERE id=? AND archived_at IS NULL`, id).
		Scan(&c.ID, &c.Name, &c.Contact, &c.Phone, &c.Email, &c.Address, &c.Notes)
	if errors.Is(err, sql.ErrNoRows) {
		return c, notFound("Client", id)
	}
	return c, err
}

func (s *Service) CreateClient(in ClientInput) (domain.Client, error) {
	name := strings.TrimSpace(in.Name.Or(""))
	if name == "" {
		return domain.Client{}, invalid("Client name is required", map[string]string{"name": "Name is required"})
	}
	c := domain.Client{
		ID: newID("cli"), Name: name,
		Contact: strings.TrimSpace(in.Contact.Or("")), Phone: strings.TrimSpace(in.Phone.Or("")),
		Email: strings.TrimSpace(in.Email.Or("")), Address: strings.TrimSpace(in.Address.Or("")),
		Notes: strings.TrimSpace(in.Notes.Or("")),
	}
	_, err := s.DB.Exec(`INSERT INTO clients(id,name,contact,phone,email,address,notes,created_at) VALUES(?,?,?,?,?,?,?,?)`,
		c.ID, c.Name, c.Contact, c.Phone, c.Email, c.Address, c.Notes, nowISO())
	return c, err
}

func (s *Service) ArchiveClient(id string) (domain.Client, error) {
	c, err := getClient(s.DB, id)
	if err != nil {
		return c, err
	}
	_, err = s.DB.Exec(`UPDATE clients SET archived_at=COALESCE(archived_at, ?) WHERE id=?`, nowISO(), id)
	return c, err
}

func (s *Service) RestoreClient(id string) (domain.Client, error) {
	c, err := getClient(s.DB, id)
	if err != nil {
		return c, err
	}
	_, err = s.DB.Exec(`UPDATE clients SET archived_at=NULL WHERE id=?`, id)
	return c, err
}

func (s *Service) UpdateClient(id string, in ClientInput) (domain.Client, error) {
	c, err := getClient(s.DB, id)
	if err != nil {
		return c, err
	}
	if in.Name.Set {
		n := strings.TrimSpace(in.Name.Or(""))
		if n == "" {
			return c, invalid("Client name cannot be empty", map[string]string{"name": "Name is required"})
		}
		c.Name = n
	}
	c.Contact = strings.TrimSpace(in.Contact.Or(c.Contact))
	c.Phone = strings.TrimSpace(in.Phone.Or(c.Phone))
	c.Email = strings.TrimSpace(in.Email.Or(c.Email))
	c.Address = strings.TrimSpace(in.Address.Or(c.Address))
	c.Notes = strings.TrimSpace(in.Notes.Or(c.Notes))
	_, err = s.DB.Exec(`UPDATE clients SET name=?,contact=?,phone=?,email=?,address=?,notes=? WHERE id=?`,
		c.Name, c.Contact, c.Phone, c.Email, c.Address, c.Notes, id)
	return c, err
}

// ---------- folders ----------

var badName = strings.NewReplacer(`\`, "_", "/", "_", ":", "_", "*", "_", "?", "_", `"`, "_", "<", "_", ">", "_", "|", "_")

// SafeName makes a string usable as a Windows folder or file name.
func SafeName(s string) string {
	s = strings.Trim(strings.TrimSpace(badName.Replace(s)), ". ")
	if s == "" {
		return "_"
	}
	return s
}

func (s *Service) ClientFolder(clientName string) string {
	return filepath.Join(s.ClientsDir, SafeName(clientName))
}

// EnsureClientFolder creates (if needed) and returns the folder for a document's client.
func (s *Service) EnsureClientFolder(clientID string) (string, error) {
	c, err := getClient(s.DB, clientID)
	if err != nil {
		return "", err
	}
	p := s.ClientFolder(c.Name)
	return p, os.MkdirAll(p, 0o755)
}

// ---------- documents ----------

const docCols = `id,type,number,client_id,job_card_id,quotation_id,issue_date,due_date,valid_until,follow_up_date,reference,scope,work_items,lines,
tax_cents,subtotal_cents,total_cents,deposit_cents,status,terms,exclusions,turnaround,notes,created_at,updated_at`

func scanDoc(r interface{ Scan(...any) error }) (domain.Doc, error) {
	var d domain.Doc
	var number, jc, qu, issue, due, valid, followUp sql.NullString
	var deposit sql.NullInt64
	var wi, ln string
	err := r.Scan(&d.ID, &d.Type, &number, &d.ClientID, &jc, &qu, &issue, &due, &valid, &followUp, &d.Reference, &d.Scope, &wi, &ln,
		&d.TaxCents, &d.SubtotalCents, &d.TotalCents, &deposit, &d.Status, &d.Terms, &d.Exclusions, &d.Turnaround, &d.Notes,
		&d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return d, err
	}
	d.Number, d.JobCardID, d.QuotationID = strPtr(number), strPtr(jc), strPtr(qu)
	d.IssueDate, d.DueDate, d.ValidUntil = strPtr(issue), strPtr(due), strPtr(valid)
	d.FollowUpDate = strPtr(followUp)
	if deposit.Valid {
		v := deposit.Int64
		d.DepositCents = &v
	}
	d.WorkItems, d.Lines = []domain.WorkItem{}, []domain.Line{}
	_ = json.Unmarshal([]byte(wi), &d.WorkItems)
	_ = json.Unmarshal([]byte(ln), &d.Lines)
	if d.WorkItems == nil {
		d.WorkItems = []domain.WorkItem{}
	}
	if d.Lines == nil {
		d.Lines = []domain.Line{}
	}
	d.Payments = []domain.Payment{}
	return d, nil
}

// attach loads payments and computes the client folder path for each document.
func (s *Service) attach(q queryer, docs []domain.Doc) error {
	pay := map[string][]domain.Payment{}
	rows, err := q.Query(`SELECT id,document_id,date,amount_cents,method,reference FROM payments ORDER BY date, rowid`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var p domain.Payment
		var docID string
		if err := rows.Scan(&p.ID, &docID, &p.Date, &p.AmountCents, &p.Method, &p.Reference); err != nil {
			rows.Close()
			return err
		}
		pay[docID] = append(pay[docID], p)
	}
	rows.Close()

	names := map[string]string{}
	crows, err := q.Query(`SELECT id,name FROM clients`)
	if err != nil {
		return err
	}
	for crows.Next() {
		var id, name string
		if err := crows.Scan(&id, &name); err != nil {
			crows.Close()
			return err
		}
		names[id] = name
	}
	crows.Close()

	for i := range docs {
		if p, ok := pay[docs[i].ID]; ok {
			docs[i].Payments = p
		}
		if n, ok := names[docs[i].ClientID]; ok {
			f := s.ClientFolder(n)
			docs[i].FolderPath = &f
		}
	}
	return nil
}

func (s *Service) queryDocs(q queryer, where string, args ...any) ([]domain.Doc, error) {
	rows, err := q.Query(`SELECT `+docCols+` FROM documents `+where, args...)
	if err != nil {
		return nil, err
	}
	docs := []domain.Doc{}
	for rows.Next() {
		d, err := scanDoc(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		docs = append(docs, d)
	}
	rows.Close()
	if err := s.attach(q, docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (s *Service) getDoc(q queryer, id string) (domain.Doc, error) {
	docs, err := s.queryDocs(q, `WHERE id=?`, id)
	if err != nil {
		return domain.Doc{}, err
	}
	if len(docs) == 0 {
		return domain.Doc{}, notFound("Document", id)
	}
	return docs[0], nil
}

func (s *Service) GetDocument(id string) (domain.Doc, error) { return s.getDoc(s.DB, id) }

type DocFilters struct {
	Type, Status, ClientID, Q, From, To string
}

func (s *Service) ListDocuments(f DocFilters) ([]domain.Doc, error) {
	docs, err := s.queryDocs(s.DB, ``)
	if err != nil {
		return nil, err
	}
	today := s.Today()
	q := strings.ToLower(f.Q)
	out := []domain.Doc{}
	for i := range docs {
		d := &docs[i]
		if f.Type != "" && d.Type != f.Type {
			continue
		}
		if f.Status != "" && domain.DerivedStatus(d, today) != f.Status {
			continue
		}
		if f.ClientID != "" && d.ClientID != f.ClientID {
			continue
		}
		if q != "" {
			hit := (d.Number != nil && strings.Contains(strings.ToLower(*d.Number), q)) ||
				strings.Contains(strings.ToLower(d.Reference), q) || strings.Contains(strings.ToLower(d.Scope), q)
			if !hit {
				continue
			}
		}
		if f.From != "" && d.IssueDate != nil && *d.IssueDate < f.From {
			continue
		}
		if f.To != "" && d.IssueDate != nil && *d.IssueDate > f.To {
			continue
		}
		out = append(out, *d)
	}
	key := func(d domain.Doc) string {
		if d.IssueDate != nil {
			return *d.IssueDate
		}
		return d.CreatedAt
	}
	sort.SliceStable(out, func(i, j int) bool { return key(out[i]) > key(out[j]) })
	return out, nil
}

func (s *Service) AllDocuments() ([]domain.Doc, error) { return s.queryDocs(s.DB, ``) }

func insertDoc(q queryer, d *domain.Doc) error {
	wi, _ := json.Marshal(d.WorkItems)
	ln, _ := json.Marshal(d.Lines)
	_, err := q.Exec(`INSERT INTO documents(`+docCols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		d.ID, d.Type, nullStr(d.Number), d.ClientID, nullStr(d.JobCardID), nullStr(d.QuotationID),
		nullStr(d.IssueDate), nullStr(d.DueDate), nullStr(d.ValidUntil), nullStr(d.FollowUpDate), d.Reference, d.Scope, string(wi), string(ln),
		d.TaxCents, d.SubtotalCents, d.TotalCents, nullInt(d.DepositCents), d.Status,
		d.Terms, d.Exclusions, d.Turnaround, d.Notes, d.CreatedAt, d.UpdatedAt)
	return err
}

func updateDoc(q queryer, d *domain.Doc) error {
	wi, _ := json.Marshal(d.WorkItems)
	ln, _ := json.Marshal(d.Lines)
	_, err := q.Exec(`UPDATE documents SET number=?,client_id=?,job_card_id=?,quotation_id=?,issue_date=?,due_date=?,valid_until=?,follow_up_date=?,
reference=?,scope=?,work_items=?,lines=?,tax_cents=?,subtotal_cents=?,total_cents=?,deposit_cents=?,status=?,
terms=?,exclusions=?,turnaround=?,notes=?,updated_at=? WHERE id=?`,
		nullStr(d.Number), d.ClientID, nullStr(d.JobCardID), nullStr(d.QuotationID),
		nullStr(d.IssueDate), nullStr(d.DueDate), nullStr(d.ValidUntil), nullStr(d.FollowUpDate),
		d.Reference, d.Scope, string(wi), string(ln), d.TaxCents, d.SubtotalCents, d.TotalCents, nullInt(d.DepositCents), d.Status,
		d.Terms, d.Exclusions, d.Turnaround, d.Notes, d.UpdatedAt, d.ID)
	return err
}

type DraftInput struct {
	Type         Opt[string]            `json:"type"`
	ClientID     Opt[string]            `json:"clientId"`
	Reference    Opt[string]            `json:"reference"`
	Scope        Opt[string]            `json:"scope"`
	JobCardID    Opt[string]            `json:"jobCardId"`
	QuotationID  Opt[string]            `json:"quotationId"`
	WorkItems    Opt[[]domain.WorkItem] `json:"workItems"`
	Lines        Opt[[]domain.Line]     `json:"lines"`
	DepositCents Opt[int64]             `json:"depositCents"`
	Terms        Opt[string]            `json:"terms"`
	Exclusions   Opt[string]            `json:"exclusions"`
	Turnaround   Opt[string]            `json:"turnaround"`
	Notes        Opt[string]            `json:"notes"`
}

func validLines(lines []domain.Line) error {
	f := map[string]string{}
	for i, l := range lines {
		if strings.TrimSpace(l.Description) == "" {
			f["lines"] = fmt.Sprintf("Line %d: description is required", i+1)
		} else if l.Qty <= 0 {
			f["lines"] = fmt.Sprintf("Line %d: qty must be greater than 0", i+1)
		} else if l.UnitPriceCents < 0 {
			f["lines"] = fmt.Sprintf("Line %d: unit price cannot be negative", i+1)
		}
		if len(f) > 0 {
			return invalid("Invalid line items", f)
		}
	}
	return nil
}

func (s *Service) checkLink(q queryer, field string, id *string, wantType string) error {
	if id == nil {
		return nil
	}
	var t string
	err := q.QueryRow(`SELECT type FROM documents WHERE id=?`, *id).Scan(&t)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && t != wantType) {
		return invalid(field+" must reference an existing "+wantType, map[string]string{field: "Must reference an existing " + wantType})
	}
	return err
}

func (s *Service) CreateDraft(in DraftInput) (domain.Doc, error) {
	typ := in.Type.Or("")
	ref := strings.TrimSpace(in.Reference.Or(""))
	cid := in.ClientID.Or("")
	f := map[string]string{}
	if typ != "invoice" && typ != "quotation" && typ != "jobcard" {
		f["type"] = "Must be quotation, invoice or jobcard"
	}
	if cid == "" {
		f["clientId"] = "Client is required"
	}
	if ref == "" {
		f["reference"] = "Reference is required"
	}
	if len(f) > 0 {
		return domain.Doc{}, invalid("type, clientId and reference are required", f)
	}
	if _, err := getActiveClient(s.DB, cid); err != nil {
		var ae *AppError
		if errors.As(err, &ae) && ae.Status == 404 {
			return domain.Doc{}, invalid("Client not found", map[string]string{"clientId": "Client not found"})
		}
		return domain.Doc{}, err
	}
	var jc, qu *string
	if in.JobCardID.Val != nil {
		jc = in.JobCardID.Val
	}
	if in.QuotationID.Val != nil {
		qu = in.QuotationID.Val
	}
	if err := s.checkLink(s.DB, "jobCardId", jc, "jobcard"); err != nil {
		return domain.Doc{}, err
	}
	if err := s.checkLink(s.DB, "quotationId", qu, "quotation"); err != nil {
		return domain.Doc{}, err
	}
	lines := in.Lines.Or(nil)
	if err := validLines(lines); err != nil {
		return domain.Doc{}, err
	}
	lines = domain.NormalizeLines(lines)
	st, err := s.GetSettings()
	if err != nil {
		return domain.Doc{}, err
	}
	t := domain.CalcTotals(lines, st.TaxRate, nil)
	now := nowISO()
	d := domain.Doc{
		ID: newID("doc"), Type: typ, ClientID: cid, JobCardID: jc, QuotationID: qu,
		Reference: ref, Scope: strings.TrimSpace(in.Scope.Or("")),
		WorkItems: in.WorkItems.Or([]domain.WorkItem{}), Lines: lines,
		TaxCents: t.Tax, SubtotalCents: t.Subtotal, TotalCents: t.Total, DepositCents: in.DepositCents.Val,
		Payments: []domain.Payment{}, Status: "draft",
		Terms: in.Terms.Or(""), Exclusions: in.Exclusions.Or(""), Turnaround: in.Turnaround.Or(""), Notes: in.Notes.Or(""),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := insertDoc(s.DB, &d); err != nil {
		return domain.Doc{}, err
	}
	return s.GetDocument(d.ID)
}

func (s *Service) UpdateDraft(id string, in DraftInput) (domain.Doc, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return domain.Doc{}, err
	}
	defer tx.Rollback()
	d, err := s.getDoc(tx, id)
	if err != nil {
		return d, err
	}
	if d.Number != nil {
		return d, appErr(http.StatusConflict, "CANNOT_EDIT_ISSUED", "Cannot edit an already issued document")
	}
	if in.ClientID.Set {
		cid := in.ClientID.Or("")
		checkClient := getActiveClient
		if cid == d.ClientID {
			checkClient = getClient
		}
		if _, err := checkClient(tx, cid); err != nil {
			return d, invalid("Client not found", map[string]string{"clientId": "Client not found"})
		}
		d.ClientID = cid
	}
	if in.Reference.Set {
		r := strings.TrimSpace(in.Reference.Or(""))
		if r == "" {
			return d, invalid("Reference is required", map[string]string{"reference": "Reference is required"})
		}
		d.Reference = r
	}
	if in.JobCardID.Set {
		if err := s.checkLink(tx, "jobCardId", in.JobCardID.Val, "jobcard"); err != nil {
			return d, err
		}
		d.JobCardID = in.JobCardID.Val
	}
	if in.QuotationID.Set {
		if err := s.checkLink(tx, "quotationId", in.QuotationID.Val, "quotation"); err != nil {
			return d, err
		}
		d.QuotationID = in.QuotationID.Val
	}
	d.Scope = strings.TrimSpace(in.Scope.Or(d.Scope))
	d.WorkItems = in.WorkItems.Or(d.WorkItems)
	if in.DepositCents.Set {
		d.DepositCents = in.DepositCents.Val
	}
	d.Terms, d.Exclusions = in.Terms.Or(d.Terms), in.Exclusions.Or(d.Exclusions)
	d.Turnaround, d.Notes = in.Turnaround.Or(d.Turnaround), in.Notes.Or(d.Notes)
	if in.Lines.Set {
		lines := in.Lines.Or(nil)
		if err := validLines(lines); err != nil {
			return d, err
		}
		d.Lines = domain.NormalizeLines(lines)
	}
	st, err := getSettings(tx)
	if err != nil {
		return d, err
	}
	t := domain.CalcTotals(d.Lines, st.TaxRate, nil)
	d.SubtotalCents, d.TaxCents, d.TotalCents = t.Subtotal, t.Tax, t.Total
	d.UpdatedAt = nowISO()
	if err := updateDoc(tx, &d); err != nil {
		return d, err
	}
	if err := tx.Commit(); err != nil {
		return d, err
	}
	return s.GetDocument(id)
}

func (s *Service) SetFollowUpDate(id string, followUpDate *string) (domain.Doc, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return domain.Doc{}, err
	}
	defer tx.Rollback()

	d, err := s.getDoc(tx, id)
	if err != nil {
		return d, err
	}
	if d.Type != "invoice" {
		return d, appErr(http.StatusBadRequest, "INVALID_DOCUMENT_TYPE", "Follow-up dates only apply to invoices")
	}
	if d.Number == nil || d.Status == "cancelled" {
		return d, appErr(http.StatusConflict, "INVALID_STATE", "Follow-up dates require an issued, non-cancelled invoice")
	}
	if followUpDate != nil && !domain.ValidDate(*followUpDate) {
		return d, invalid("followUpDate must be YYYY-MM-DD or null", map[string]string{"followUpDate": "Must be YYYY-MM-DD or null"})
	}

	d.FollowUpDate = followUpDate
	d.UpdatedAt = nowISO()
	if err := updateDoc(tx, &d); err != nil {
		return d, err
	}
	if err := tx.Commit(); err != nil {
		return d, err
	}
	return s.GetDocument(id)
}

func nextNumber(q queryer, docType string, year int) (string, error) {
	prefix := fmt.Sprintf("%s-%d-", domain.Prefix(docType), year)
	rows, err := q.Query(`SELECT number FROM documents WHERE type=? AND number LIKE ?`, docType, prefix+"%")
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var docs []domain.Doc
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return "", err
		}
		docs = append(docs, domain.Doc{Type: docType, Number: &n})
	}
	return domain.NextNumber(docs, docType, year), rows.Err()
}

// IssueDocument assigns the next number (per type and year), sets dates and moves draft -> sent/open.
func (s *Service) IssueDocument(id, issueDate string) (domain.Doc, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return domain.Doc{}, err
	}
	defer tx.Rollback()
	d, err := s.getDoc(tx, id)
	if err != nil {
		return d, err
	}
	if d.Number != nil {
		return d, appErr(http.StatusConflict, "ALREADY_ISSUED", fmt.Sprintf("Document %s has already been issued with number %s", id, *d.Number))
	}
	if issueDate == "" {
		issueDate = s.Today()
	}
	if !domain.ValidDate(issueDate) {
		return d, invalid("issueDate must be YYYY-MM-DD", map[string]string{"issueDate": "Must be YYYY-MM-DD"})
	}
	if d.Status == "cancelled" {
		return d, appErr(http.StatusConflict, "INVALID_STATE", "A cancelled document cannot be issued")
	}
	if (d.Type == "invoice" || d.Type == "quotation") && len(d.Lines) == 0 {
		return d, invalid("At least one line item is required before issuing", map[string]string{"lines": "At least one line item is required"})
	}
	st, err := getSettings(tx)
	if err != nil {
		return d, err
	}
	t, _ := domain.ParseDate(issueDate)
	num, err := nextNumber(tx, d.Type, t.Year())
	if err != nil {
		return d, err
	}
	d.Number, d.IssueDate = &num, &issueDate
	switch d.Type {
	case "invoice":
		due := domain.AddDays(issueDate, st.DefaultTermsDays)
		d.DueDate = &due
		d.Status = "sent"
	case "quotation":
		vu := domain.AddDays(issueDate, st.QuoteValidityDays)
		d.ValidUntil = &vu
		d.Status = "sent"
	default:
		d.Status = "open"
	}
	d.UpdatedAt = nowISO()
	if err := updateDoc(tx, &d); err != nil {
		return d, err
	}
	if err := tx.Commit(); err != nil {
		return d, err
	}
	return s.GetDocument(id)
}

type PaymentInput struct {
	Date        string `json:"date"`
	AmountCents int64  `json:"amountCents"`
	Method      string `json:"method"`
	Reference   string `json:"reference"`
}

func (s *Service) AddPayment(id string, in PaymentInput) (domain.Doc, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return domain.Doc{}, err
	}
	defer tx.Rollback()
	d, err := s.getDoc(tx, id)
	if err != nil {
		return d, err
	}
	if d.Type != "invoice" {
		return d, appErr(http.StatusBadRequest, "INVALID_DOCUMENT_TYPE", "Payments can only be added to invoices")
	}
	f := map[string]string{}
	if !domain.ValidDate(in.Date) {
		f["date"] = "Must be YYYY-MM-DD"
	}
	if in.AmountCents <= 0 {
		f["amountCents"] = "Must be greater than 0"
	}
	if len(f) > 0 {
		return d, invalid("Valid payment date and positive amount in cents are required", f)
	}
	if d.Number == nil || d.Status == "cancelled" {
		return d, appErr(http.StatusConflict, "INVALID_STATE", "Payments can only be recorded against issued, non-cancelled invoices")
	}
	balance := d.TotalCents - domain.PaidSum(d.Payments)
	if in.AmountCents > balance {
		return d, appErr(http.StatusUnprocessableEntity, "OVERPAYMENT_REJECTED",
			fmt.Sprintf("Payment of %d cents exceeds outstanding balance of %d cents", in.AmountCents, balance))
	}
	method := strings.TrimSpace(in.Method)
	if method == "" {
		method = "EFT"
	}
	if _, err := tx.Exec(`INSERT INTO payments(id,document_id,date,amount_cents,method,reference) VALUES(?,?,?,?,?,?)`,
		newID("pay"), id, in.Date, in.AmountCents, method, in.Reference); err != nil {
		return d, err
	}
	if in.AmountCents == balance {
		d.Status = "paid"
	} else {
		d.Status = "partially_paid"
	}
	d.UpdatedAt = nowISO()
	if err := updateDoc(tx, &d); err != nil {
		return d, err
	}
	if err := tx.Commit(); err != nil {
		return d, err
	}
	return s.GetDocument(id)
}

func (s *Service) SetStatus(id, status string) (domain.Doc, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return domain.Doc{}, err
	}
	defer tx.Rollback()
	d, err := s.getDoc(tx, id)
	if err != nil {
		return d, err
	}
	if !domain.TransitionAllowed(d.Type, d.Status, status) || (d.Type == "invoice" && status == "cancelled" && len(d.Payments) > 0) {
		return d, invalid(fmt.Sprintf("Unsupported status transition from %s to %s", d.Status, status),
			map[string]string{"status": "Unsupported transition"})
	}
	d.Status = status
	d.UpdatedAt = nowISO()
	if err := updateDoc(tx, &d); err != nil {
		return d, err
	}
	if err := tx.Commit(); err != nil {
		return d, err
	}
	return s.GetDocument(id)
}

// ConvertQuotation creates a draft invoice from an issued quotation and marks the quotation converted.
func (s *Service) ConvertQuotation(id string) (domain.Doc, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return domain.Doc{}, err
	}
	defer tx.Rollback()
	q, err := s.getDoc(tx, id)
	if err != nil {
		return q, err
	}
	if q.Type != "quotation" {
		return q, appErr(http.StatusConflict, "INVALID_STATE", "Only quotations can be converted to invoices")
	}
	if q.Status == "converted" {
		return q, appErr(http.StatusConflict, "INVALID_STATE", "Quotation is already converted")
	}
	if q.Number == nil || q.Status == "draft" || q.Status == "declined" || q.Status == "cancelled" {
		return q, appErr(http.StatusConflict, "INVALID_STATE", "Only an issued, open quotation can be converted")
	}
	now := nowISO()
	qid := q.ID
	inv := domain.Doc{
		ID: newID("doc"), Type: "invoice", ClientID: q.ClientID, JobCardID: q.JobCardID, QuotationID: &qid,
		Reference: q.Reference, Scope: q.Scope,
		WorkItems: append([]domain.WorkItem{}, q.WorkItems...), Lines: append([]domain.Line{}, q.Lines...),
		TaxCents: q.TaxCents, SubtotalCents: q.SubtotalCents, TotalCents: q.TotalCents,
		Payments: []domain.Payment{}, Status: "draft", Terms: q.Terms, Notes: q.Notes,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := insertDoc(tx, &inv); err != nil {
		return inv, err
	}
	q.Status, q.UpdatedAt = "converted", now
	if err := updateDoc(tx, &q); err != nil {
		return inv, err
	}
	if err := tx.Commit(); err != nil {
		return inv, err
	}
	return s.GetDocument(inv.ID)
}

// ---------- dashboard / storage ----------

func (s *Service) Dashboard(year int) (domain.Dashboard, error) {
	docs, err := s.AllDocuments()
	if err != nil {
		return domain.Dashboard{}, err
	}
	today := s.Today()
	if year == 0 {
		t, _ := domain.ParseDate(today)
		year = t.Year()
	}
	return domain.BuildDashboard(docs, year, today), nil
}

func (s *Service) Storage() domain.Storage {
	return domain.Storage{DataDir: s.DataDir, DBPath: s.DBPath, ClientsDir: s.ClientsDir}
}
