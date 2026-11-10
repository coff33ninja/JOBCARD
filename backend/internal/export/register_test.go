package export

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"

	"jobcard/internal/domain"
)

func TestRegisterIncludesFollowUpDate(t *testing.T) {
	n := "INV-2026-001"
	issue, due, follow := "2026-10-01", "2026-10-08", "2026-11-05"
	docs := []domain.Doc{{
		ID: "d1", Type: "invoice", Number: &n, ClientID: "c1", Status: "sent", Reference: "R",
		IssueDate: &issue, DueDate: &due, FollowUpDate: &follow, TotalCents: 10000,
	}}
	data, err := RegisterXLSX(docs, []domain.Client{{ID: "c1", Name: "Acme"}}, "2026-11-08")
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := f.GetRows("Register")
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows %v %v", len(rows), err)
	}
	col := -1
	for i, h := range rows[0] {
		if h == "Follow-up Date" {
			col = i
		}
	}
	if col < 0 || len(rows[1]) <= col || rows[1][col] != follow {
		t.Fatalf("header %v, row %v", rows[0], rows[1])
	}
}
