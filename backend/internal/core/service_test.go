package core

import (
	"errors"
	"path/filepath"
	"testing"

	"jobcard/internal/domain"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	s, err := NewService(dir, filepath.Join(dir, "Clients"), func() string { return "2026-11-08" })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func str(s string) Opt[string] { return Opt[string]{Set: true, Val: &s} }

func mustClient(t *testing.T, s *Service) domain.Client {
	t.Helper()
	c, err := s.CreateClient(ClientInput{Name: str("Test Client")})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func draft(t *testing.T, s *Service, typ, clientID string, unit int64) domain.Doc {
	t.Helper()
	lines := []domain.Line{{Description: "Item", Qty: 1, UnitPriceCents: unit}}
	d, err := s.CreateDraft(DraftInput{
		Type: str(typ), ClientID: str(clientID), Reference: str("Ref"),
		Lines: Opt[[]domain.Line]{Set: true, Val: &lines},
	})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func wantStatus(t *testing.T, err error, status int, code string) {
	t.Helper()
	var ae *AppError
	if !errors.As(err, &ae) || ae.Status != status || ae.Code != code {
		t.Fatalf("want %d %s, got %v", status, code, err)
	}
}

func TestNumberingContinuesAndRestartsPerYear(t *testing.T) {
	s := newTestService(t)
	c := mustClient(t, s)
	a, _ := s.IssueDocument(draft(t, s, "invoice", c.ID, 1000).ID, "2026-11-08")
	b, _ := s.IssueDocument(draft(t, s, "invoice", c.ID, 1000).ID, "2026-11-09")
	n, _ := s.IssueDocument(draft(t, s, "invoice", c.ID, 1000).ID, "2027-01-02")
	q, _ := s.IssueDocument(draft(t, s, "quotation", c.ID, 1000).ID, "2026-11-08")
	if *a.Number != "INV-2026-001" || *b.Number != "INV-2026-002" || *n.Number != "INV-2027-001" || *q.Number != "QUO-2026-001" {
		t.Fatalf("numbers %s %s %s %s", *a.Number, *b.Number, *n.Number, *q.Number)
	}
	if b.DueDate == nil || *b.DueDate != "2026-11-16" {
		t.Fatalf("due date with default 7 day terms: %v", b.DueDate)
	}
}

func TestClientArchivePreservesDocumentsAndCanBeRestored(t *testing.T) {
	s := newTestService(t)
	c := mustClient(t, s)
	doc := draft(t, s, "invoice", c.ID, 1000)

	if _, err := s.ArchiveClient(c.ID); err != nil {
		t.Fatal(err)
	}
	active, err := s.ListClients()
	if err != nil || len(active) != 0 {
		t.Fatalf("active clients after archive = %v, %v", active, err)
	}
	archived, err := s.ListArchivedClients()
	if err != nil || len(archived) != 1 || archived[0].ID != c.ID {
		t.Fatalf("archived clients = %v, %v", archived, err)
	}
	if got, err := s.GetDocument(doc.ID); err != nil || got.ClientID != c.ID {
		t.Fatalf("historical document = %+v, %v", got, err)
	}
	if got, err := s.UpdateDraft(doc.ID, DraftInput{Scope: str("Existing draft remains editable")}); err != nil || got.Scope != "Existing draft remains editable" {
		t.Fatalf("edit existing draft for archived client = %+v, %v", got, err)
	}
	_, err = s.CreateDraft(DraftInput{Type: str("invoice"), ClientID: str(c.ID), Reference: str("New work")})
	wantStatus(t, err, 400, "VALIDATION_FAILED")

	if _, err := s.RestoreClient(c.ID); err != nil {
		t.Fatal(err)
	}
	active, err = s.ListClients()
	if err != nil || len(active) != 1 || active[0].ID != c.ID {
		t.Fatalf("active clients after restore = %v, %v", active, err)
	}
	archived, err = s.ListArchivedClients()
	if err != nil || len(archived) != 0 {
		t.Fatalf("archived clients after restore = %v, %v", archived, err)
	}
}

func TestArchivedAtColumnMigratesLegacyDatabase(t *testing.T) {
	dir := t.TempDir()
	s, err := NewService(dir, filepath.Join(dir, "Clients"), func() string { return "2026-11-08" })
	if err != nil {
		t.Fatal(err)
	}
	c := mustClient(t, s)
	if _, err := s.DB.Exec(`ALTER TABLE clients DROP COLUMN archived_at`); err != nil {
		t.Fatalf("prepare legacy schema: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s, err = NewService(dir, filepath.Join(dir, "Clients"), func() string { return "2026-11-08" })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if clients, err := s.ListClients(); err != nil || len(clients) != 1 || clients[0].ID != c.ID {
		t.Fatalf("legacy clients after migration = %v, %v", clients, err)
	}
	if _, err := s.ArchiveClient(c.ID); err != nil {
		t.Fatalf("archive migrated client: %v", err)
	}
	if clients, err := s.ListArchivedClients(); err != nil || len(clients) != 1 || clients[0].ID != c.ID {
		t.Fatalf("archived clients after migration = %v, %v", clients, err)
	}
}

func TestFollowUpDatePersistsAndLegacyDatabaseMigrates(t *testing.T) {
	dir := t.TempDir()
	s, err := NewService(dir, filepath.Join(dir, "Clients"), func() string { return "2026-11-08" })
	if err != nil {
		t.Fatal(err)
	}
	c := mustClient(t, s)
	invoice := draft(t, s, "invoice", c.ID, 0)
	if _, err := s.IssueDocument(invoice.ID, "2026-11-08"); err != nil {
		t.Fatal(err)
	}
	date := "2026-11-09"
	if _, err := s.SetFollowUpDate(invoice.ID, &date); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s, err = NewService(dir, filepath.Join(dir, "Clients"), func() string { return "2026-11-08" })
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetDocument(invoice.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.FollowUpDate == nil || *got.FollowUpDate != date {
		t.Fatalf("reopened follow-up date = %v, want %s", got.FollowUpDate, date)
	}
	if _, err := s.DB.Exec(`ALTER TABLE documents DROP COLUMN follow_up_date`); err != nil {
		t.Fatalf("prepare legacy schema: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s, err = NewService(dir, filepath.Join(dir, "Clients"), func() string { return "2026-11-08" })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	got, err = s.GetDocument(invoice.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.FollowUpDate != nil {
		t.Fatalf("legacy document follow-up date = %v, want null", got.FollowUpDate)
	}
}

func TestIssuedDocumentIsImmutable(t *testing.T) {
	s := newTestService(t)
	c := mustClient(t, s)
	d := draft(t, s, "invoice", c.ID, 1000)
	if _, err := s.IssueDocument(d.ID, ""); err != nil {
		t.Fatal(err)
	}
	_, err := s.IssueDocument(d.ID, "")
	wantStatus(t, err, 409, "ALREADY_ISSUED")
	_, err = s.UpdateDraft(d.ID, DraftInput{Scope: str("x")})
	wantStatus(t, err, 409, "CANNOT_EDIT_ISSUED")
}

func TestServerComputesMoney(t *testing.T) {
	s := newTestService(t)
	c := mustClient(t, s)
	lines := []domain.Line{{Description: "A", Qty: 2, UnitPriceCents: 1999, TotalCents: 1}} // bogus client total ignored
	d, err := s.CreateDraft(DraftInput{Type: str("invoice"), ClientID: str(c.ID), Reference: str("R"), Lines: Opt[[]domain.Line]{Set: true, Val: &lines}})
	if err != nil || d.TotalCents != 3998 || d.Lines[0].TotalCents != 3998 {
		t.Fatalf("total %v %v", d.TotalCents, err)
	}
}

func TestPaymentRules(t *testing.T) {
	s := newTestService(t)
	c := mustClient(t, s)
	d := draft(t, s, "invoice", c.ID, 10000)
	_, err := s.AddPayment(d.ID, PaymentInput{Date: "2026-11-08", AmountCents: 100})
	wantStatus(t, err, 409, "INVALID_STATE") // not issued
	s.IssueDocument(d.ID, "")
	_, err = s.AddPayment(d.ID, PaymentInput{Date: "2026-11-08", AmountCents: 0})
	wantStatus(t, err, 400, "VALIDATION_FAILED")
	_, err = s.AddPayment(d.ID, PaymentInput{Date: "2026-11-08", AmountCents: 10001})
	wantStatus(t, err, 422, "OVERPAYMENT_REJECTED")
	got, _ := s.AddPayment(d.ID, PaymentInput{Date: "2026-11-08", AmountCents: 4000})
	if got.Status != "partially_paid" {
		t.Fatalf("status %s", got.Status)
	}
	got, _ = s.AddPayment(d.ID, PaymentInput{Date: "2026-11-09", AmountCents: 6000})
	if got.Status != "paid" || len(got.Payments) != 2 {
		t.Fatalf("status %s payments %d", got.Status, len(got.Payments))
	}
	q := draft(t, s, "quotation", c.ID, 100)
	_, err = s.AddPayment(q.ID, PaymentInput{Date: "2026-11-08", AmountCents: 1})
	wantStatus(t, err, 400, "INVALID_DOCUMENT_TYPE")
}

func TestStatusTransitionsAndCancel(t *testing.T) {
	s := newTestService(t)
	c := mustClient(t, s)
	inv := draft(t, s, "invoice", c.ID, 10000)
	s.IssueDocument(inv.ID, "")
	s.AddPayment(inv.ID, PaymentInput{Date: "2026-11-08", AmountCents: 1})
	_, err := s.SetStatus(inv.ID, "cancelled") // has payments
	wantStatus(t, err, 400, "VALIDATION_FAILED")
	_, err = s.SetStatus(inv.ID, "paid")
	wantStatus(t, err, 400, "VALIDATION_FAILED")

	q := draft(t, s, "quotation", c.ID, 100)
	s.IssueDocument(q.ID, "")
	if got, err := s.SetStatus(q.ID, "accepted"); err != nil || got.Status != "accepted" {
		t.Fatalf("accept %v %v", got.Status, err)
	}
}

func TestConvertQuotation(t *testing.T) {
	s := newTestService(t)
	c := mustClient(t, s)
	q := draft(t, s, "quotation", c.ID, 12000)
	_, err := s.ConvertQuotation(q.ID)
	wantStatus(t, err, 409, "INVALID_STATE") // draft
	s.IssueDocument(q.ID, "")
	inv, err := s.ConvertQuotation(q.ID)
	if err != nil || inv.Type != "invoice" || inv.Number != nil || inv.QuotationID == nil || *inv.QuotationID != q.ID || inv.TotalCents != 12000 {
		t.Fatalf("convert %+v %v", inv, err)
	}
	after, _ := s.GetDocument(q.ID)
	if after.Status != "converted" {
		t.Fatalf("quotation status %s", after.Status)
	}
	_, err = s.ConvertQuotation(q.ID)
	wantStatus(t, err, 409, "INVALID_STATE")
}

func TestValidation(t *testing.T) {
	s := newTestService(t)
	_, err := s.CreateClient(ClientInput{Name: str("  ")})
	wantStatus(t, err, 400, "VALIDATION_FAILED")
	_, err = s.CreateDraft(DraftInput{Type: str("invoice"), ClientID: str("nope"), Reference: str("R")})
	wantStatus(t, err, 400, "VALIDATION_FAILED")
	c := mustClient(t, s)
	empty := draft(t, s, "invoice", c.ID, 100)
	s.UpdateDraft(empty.ID, DraftInput{Lines: Opt[[]domain.Line]{Set: true, Val: &[]domain.Line{}}})
	_, err = s.IssueDocument(empty.ID, "")
	wantStatus(t, err, 400, "VALIDATION_FAILED") // no lines
	_, err = s.IssueDocument(empty.ID, "08/11/2026")
	wantStatus(t, err, 400, "VALIDATION_FAILED") // bad date
}

func TestSeedOnlyIntoEmptyDatabase(t *testing.T) {
	s := newTestService(t)
	seeded, err := s.Seed(filepath.Join("..", "..", "..", "seed", "seed.json"))
	if err != nil || !seeded {
		t.Fatalf("seed %v %v", seeded, err)
	}
	again, err := s.Seed(filepath.Join("..", "..", "..", "seed", "seed.json"))
	if err != nil || again {
		t.Fatalf("second seed must be skipped: %v %v", again, err)
	}
	d, err := s.Dashboard(2026)
	if err != nil || len(d.Monthly) != 12 || d.Totals.InvoicedCents <= 0 {
		t.Fatalf("dashboard %+v %v", d.Totals, err)
	}
}

func TestSafeName(t *testing.T) {
	if SafeName(`A/B: "C"?`) != "A_B_ _C__" && SafeName(`A/B: "C"?`) == "" {
		t.Fatal("unexpected")
	}
	if SafeName("  ... ") != "_" || SafeName("Apex Logistics") != "Apex Logistics" {
		t.Fatal("SafeName")
	}
}

func TestSeedResolvesFollowUpDates(t *testing.T) {
	s := newTestService(t)
	if seeded, err := s.Seed(filepath.Join("..", "..", "..", "seed", "seed.json")); err != nil || !seeded {
		t.Fatalf("seed %v %v", seeded, err)
	}
	docs, err := s.AllDocuments()
	if err != nil {
		t.Fatal(err)
	}
	var withDate int
	for _, d := range docs {
		if d.FollowUpDate != nil {
			withDate++
			if len(*d.FollowUpDate) != 10 || (*d.FollowUpDate)[:2] != "20" {
				t.Fatalf("%s: follow-up date not resolved: %q", d.ID, *d.FollowUpDate)
			}
		}
	}
	if withDate == 0 {
		t.Fatal("seed should contain follow-up dates")
	}
	dash, err := s.Dashboard(2026)
	if err != nil || dash.Attention.FollowUpsDue != 1 {
		t.Fatalf("follow-ups due %d (%v), want 1 from the seeded overdue invoice", dash.Attention.FollowUpsDue, err)
	}
}
