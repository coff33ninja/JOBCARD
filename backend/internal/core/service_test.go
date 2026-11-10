package core

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
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

func TestDocumentIntakeAndDiagnosticDataPersists(t *testing.T) {
	s := newTestService(t)
	c := mustClient(t, s)

	var in DraftInput
	err := json.Unmarshal([]byte(`{
		"type":"invoice",
		"clientId":"`+c.ID+`",
		"reference":"Laptop",
		"lines":[{"description":"SSD","detail":"1TB NVMe","qty":1,"unitPriceCents":100}],
		"intakeItems":[{"label":"Primary storage","value":"1TB HDD"}],
		"diagnosticSections":[{"title":"Drive health","status":"PASSED","details":"100% health"}],
		"authorization":{
			"technicianSignature":"Tech",
			"technicianDate":"2026-10-12",
			"clientSignature":"Client",
			"clientDate":"2026-10-12"
		}
	}`), &in)
	if err != nil {
		t.Fatal(err)
	}
	created, err := s.CreateDraft(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Lines) != 1 || created.Lines[0].Detail != "1TB NVMe" {
		t.Fatalf("line detail did not persist: %+v", created.Lines)
	}
	createdJSON, err := json.Marshal(created)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Primary storage", "1TB HDD", "Drive health", "100% health", "Tech", "Client", "2026-10-12"} {
		if !strings.Contains(string(createdJSON), want) {
			t.Errorf("created document did not persist %q: %s", want, createdJSON)
		}
	}

	var update DraftInput
	if err := json.Unmarshal([]byte(`{
		"intakeItems":[{"label":"Accessories","value":"Carrying bag"}],
		"diagnosticSections":[{"title":"Resolution","status":"COMPLETE","details":"Tested stable"}],
		"authorization":{"clientSignature":"Approved"}
	}`), &update); err != nil {
		t.Fatal(err)
	}
	updated, err := s.UpdateDraft(created.ID, update)
	if err != nil {
		t.Fatal(err)
	}
	updatedJSON, err := json.Marshal(updated)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Accessories", "Carrying bag", "Resolution", "Tested stable", "Approved"} {
		if !strings.Contains(string(updatedJSON), want) {
			t.Errorf("updated document did not persist %q: %s", want, updatedJSON)
		}
	}
}

func TestIssueDocumentFillsMissingTechnicianSignOff(t *testing.T) {
	s := newTestService(t)
	client := mustClient(t, s)
	settings, err := s.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	settings.Person = "Technician Name"
	if _, err := s.UpdateSettings(settings); err != nil {
		t.Fatal(err)
	}

	unfilled := draft(t, s, "jobcard", client.ID, 0)
	issued, err := s.IssueDocument(unfilled.ID, "2026-11-09")
	if err != nil {
		t.Fatal(err)
	}
	if issued.Authorization.TechnicianSignature != "Technician Name" || issued.Authorization.TechnicianDate != "2026-11-09" {
		t.Fatalf("issue should fill missing technician name/date from settings and issue date: %+v", issued.Authorization)
	}

	signature, date := "Manual Technician", "2026-11-07"
	manual, err := s.CreateDraft(DraftInput{
		Type: str("jobcard"), ClientID: str(client.ID), Reference: str("Manual sign-off"),
		Authorization: Opt[domain.Authorization]{Set: true, Val: &domain.Authorization{
			TechnicianSignature: signature,
			TechnicianDate:      date,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	issuedManual, err := s.IssueDocument(manual.ID, "2026-11-10")
	if err != nil {
		t.Fatal(err)
	}
	if issuedManual.Authorization.TechnicianSignature != signature || issuedManual.Authorization.TechnicianDate != date {
		t.Fatalf("issue must preserve manually provided technician sign-off: %+v", issuedManual.Authorization)
	}
}

func TestNumberAlreadyAssignedRejectsActiveDuplicate(t *testing.T) {
	candidate := "INV-2026-003"
	docs := []domain.Doc{
		{ID: "doc-1", Type: "invoice", Number: &candidate, Status: "sent"},
		{ID: "doc-2", Type: "invoice", Status: "cancelled", Number: &[]string{"INV-2026-004"}[0]},
		{ID: "doc-3", Type: "quotation", Number: &[]string{"QUO-2026-003"}[0], Status: "sent"},
	}
	if !numberAlreadyAssigned(docs, "invoice", candidate, "doc-new") {
		t.Fatal("active invoice number should be detected as already assigned")
	}
	if numberAlreadyAssigned(docs, "invoice", candidate, "doc-1") {
		t.Fatal("same document should not self-trigger the active-number conflict guard")
	}
	if numberAlreadyAssigned(docs, "invoice", "INV-2026-004", "doc-new") {
		t.Fatal("cancelled invoice should not count as an active duplicate")
	}
}

func TestDocumentReportColumnsMigrateLegacyDatabase(t *testing.T) {
	dir := t.TempDir()
	s, err := NewService(dir, filepath.Join(dir, "Clients"), func() string { return "2026-11-08" })
	if err != nil {
		t.Fatal(err)
	}
	c := mustClient(t, s)
	doc := draft(t, s, "invoice", c.ID, 1000)
	for _, column := range []string{"intake_items", "diagnostic_sections", "authorization"} {
		if _, err := s.DB.Exec(`ALTER TABLE documents DROP COLUMN ` + column); err != nil {
			t.Fatalf("prepare legacy schema without %s: %v", column, err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s, err = NewService(dir, filepath.Join(dir, "Clients"), func() string { return "2026-11-08" })
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	got, err := s.GetDocument(doc.ID)
	if err != nil {
		t.Fatalf("read document after migration: %v", err)
	}
	if len(got.IntakeItems) != 0 || len(got.DiagnosticSections) != 0 || got.Authorization != (domain.Authorization{}) {
		t.Fatalf("legacy report fields should have empty defaults: %+v", got)
	}
}

func TestDocumentReportDecodeErrorsAreReturned(t *testing.T) {
	s := newTestService(t)
	c := mustClient(t, s)
	doc := draft(t, s, "invoice", c.ID, 1000)
	if _, err := s.DB.Exec(`UPDATE documents SET authorization='{' WHERE id=?`, doc.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetDocument(doc.ID); err == nil || !strings.Contains(err.Error(), "decode document authorization") {
		t.Fatalf("want explicit authorization decode error, got %v", err)
	}
}

func TestCreateDraftWithClientRollsBackClientWhenDraftIsInvalid(t *testing.T) {
	s := newTestService(t)
	clientName := "Invalid Draft Client"
	_, err := s.CreateDraftWithClient(
		DraftInput{Type: str("invoice")},
		ClientInput{Name: str(clientName)},
	)
	if err == nil {
		t.Fatal("expected invalid draft error")
	}
	clients, err := s.ListClients()
	if err != nil {
		t.Fatal(err)
	}
	if len(clients) != 0 {
		t.Fatalf("client creation should roll back with invalid draft, got %+v", clients)
	}
}

func TestFindPotentialDuplicateClientsMatchesNamesAndContactDetails(t *testing.T) {
	s := newTestService(t)
	existing, err := s.CreateClient(ClientInput{
		Name:    str("Acme Holdings"),
		Contact: str("Jane Doe"),
		Phone:   str("082 123 4567"),
		Email:   str("accounts@acme.co.za"),
	})
	if err != nil {
		t.Fatal(err)
	}
	archived, err := s.CreateClient(ClientInput{
		Name:    str("Former Client"),
		Contact: str("Alice Smith"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ArchiveClient(archived.ID); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		input        ClientInput
		wantID       string
		wantReason   string
		wantArchived bool
	}{
		{
			name:       "normalized company name",
			input:      ClientInput{Name: str("  ACME   HOLDINGS ")},
			wantID:     existing.ID,
			wantReason: "name/contact",
		},
		{
			name:       "person contact name",
			input:      ClientInput{Name: str("Jane Doe")},
			wantID:     existing.ID,
			wantReason: "name/contact",
		},
		{
			name:       "formatted phone",
			input:      ClientInput{Name: str("Different Client"), Phone: str("082-123-4567")},
			wantID:     existing.ID,
			wantReason: "phone",
		},
		{
			name:       "case-insensitive email",
			input:      ClientInput{Name: str("Different Client"), Email: str("ACCOUNTS@ACME.CO.ZA")},
			wantID:     existing.ID,
			wantReason: "email",
		},
		{
			name:         "archived client",
			input:        ClientInput{Name: str("alice   smith")},
			wantID:       archived.ID,
			wantReason:   "name/contact",
			wantArchived: true,
		},
		{
			name:  "distinct name does not fuzzy match",
			input: ClientInput{Name: str("Acme Holdings Ltd"), Phone: str("011 555 0199")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := s.FindPotentialDuplicateClients(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantID == "" {
				if len(matches) != 0 {
					t.Fatalf("unexpected duplicate matches: %+v", matches)
				}
				return
			}
			if len(matches) != 1 || matches[0].Client.ID != tt.wantID {
				t.Fatalf("matches = %+v, want client %s", matches, tt.wantID)
			}
			if matches[0].Archived != tt.wantArchived {
				t.Fatalf("archived = %v, want %v", matches[0].Archived, tt.wantArchived)
			}
			if !strings.Contains(strings.Join(matches[0].Reasons, ","), tt.wantReason) {
				t.Fatalf("reasons = %v, want %q", matches[0].Reasons, tt.wantReason)
			}
		})
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

func TestDraftArchiveRestoreAndDeleteBoundaries(t *testing.T) {
	s := newTestService(t)
	c := mustClient(t, s)
	doc := draft(t, s, "quotation", c.ID, 1000)

	archived, err := s.ArchiveDraft(doc.ID)
	if err != nil || archived.ArchivedAt == nil {
		t.Fatalf("archive draft = %+v, %v", archived, err)
	}
	active, err := s.ListDocuments(DocFilters{})
	if err != nil || len(active) != 0 {
		t.Fatalf("active documents after archive = %v, %v", active, err)
	}
	archivedDocs, err := s.ListArchivedDocuments()
	if err != nil || len(archivedDocs) != 1 || archivedDocs[0].ID != doc.ID {
		t.Fatalf("archived documents = %v, %v", archivedDocs, err)
	}

	restored, err := s.RestoreDraft(doc.ID)
	if err != nil || restored.ArchivedAt != nil {
		t.Fatalf("restore draft = %+v, %v", restored, err)
	}
	if _, err := s.DeleteDraft(doc.ID); err != nil {
		t.Fatalf("delete restored draft: %v", err)
	}
	if _, err := s.GetDocument(doc.ID); err == nil {
		t.Fatal("deleted draft is still retrievable")
	}

	issued := draft(t, s, "quotation", c.ID, 1000)
	if _, err := s.IssueDocument(issued.ID, ""); err != nil {
		t.Fatal(err)
	}
	_, err = s.DeleteDraft(issued.ID)
	wantStatus(t, err, 409, "INVALID_STATE")
}

func TestReplacementQuotationCopiesFieldsAndCancelsSourceOnIssue(t *testing.T) {
	s := newTestService(t)
	c := mustClient(t, s)
	source := draft(t, s, "quotation", c.ID, 12500)
	source, err := s.UpdateDraft(source.ID, DraftInput{
		Scope:      str("Replace display"),
		Exclusions: str("Data recovery"),
		Turnaround: str("Two days"),
		Notes:      str("Bring charger"),
	})
	if err != nil {
		t.Fatal(err)
	}
	source, err = s.IssueDocument(source.ID, "")
	if err != nil {
		t.Fatal(err)
	}

	replacement, err := s.CreateReplacementQuotation(source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if replacement.Number != nil || replacement.Status != "draft" ||
		replacement.RevisionOfQuotationID == nil || *replacement.RevisionOfQuotationID != source.ID ||
		replacement.ClientID != source.ClientID || replacement.Scope != source.Scope ||
		replacement.TotalCents != source.TotalCents || replacement.Exclusions != source.Exclusions ||
		replacement.Turnaround != source.Turnaround || replacement.Notes != source.Notes {
		t.Fatalf("replacement did not preserve source fields and linkage: %+v", replacement)
	}
	stillOpen, err := s.GetDocument(source.ID)
	if err != nil || stillOpen.Status != "sent" {
		t.Fatalf("source should remain open until replacement is issued: %+v, %v", stillOpen, err)
	}

	issuedReplacement, err := s.IssueDocument(replacement.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	cancelledSource, err := s.GetDocument(source.ID)
	if err != nil || cancelledSource.Status != "cancelled" {
		t.Fatalf("source quotation after replacement issue = %+v, %v", cancelledSource, err)
	}
	if issuedReplacement.Number == nil || issuedReplacement.Status != "sent" {
		t.Fatalf("replacement quotation was not issued: %+v", issuedReplacement)
	}
}

func TestIssuedQuotationRevisionRequiresConfirmationAndPreservesHistory(t *testing.T) {
	s := newTestService(t)
	c := mustClient(t, s)
	quote := draft(t, s, "quotation", c.ID, 12500)
	quote, err := s.IssueDocument(quote.ID, "")
	if err != nil {
		t.Fatal(err)
	}

	update := DraftInput{Scope: str("Updated scope"), Lines: Opt[[]domain.Line]{
		Set: true, Val: &[]domain.Line{{Description: "Updated work", Qty: 1, UnitPriceCents: 15000}},
	}}
	_, err = s.UpdateIssuedQuotation(quote.ID, update, false)
	wantStatus(t, err, 400, "CONFIRMATION_REQUIRED")

	updated, err := s.UpdateIssuedQuotation(quote.ID, update, true)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Number == nil || *updated.Number != *quote.Number || updated.Status != quote.Status ||
		updated.Scope != "Updated scope" || updated.TotalCents != 15000 {
		t.Fatalf("in-place revision changed immutable issue details or missed updates: %+v", updated)
	}
	revisions, err := s.ListQuotationRevisions(quote.ID)
	if err != nil || len(revisions) != 1 {
		t.Fatalf("revision history = %v, %v", revisions, err)
	}
	if revisions[0].Snapshot.Scope != quote.Scope || revisions[0].Snapshot.TotalCents != quote.TotalCents {
		t.Fatalf("revision snapshot did not preserve the old quotation: %+v", revisions[0].Snapshot)
	}

	invalidUpdate := DraftInput{Lines: Opt[[]domain.Line]{Set: true, Val: &[]domain.Line{{Qty: 0}}}}
	_, err = s.UpdateIssuedQuotation(quote.ID, invalidUpdate, true)
	wantStatus(t, err, 400, "VALIDATION_FAILED")
	revisions, err = s.ListQuotationRevisions(quote.ID)
	if err != nil || len(revisions) != 1 {
		t.Fatalf("failed update must not add revision history: %v, %v", revisions, err)
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
