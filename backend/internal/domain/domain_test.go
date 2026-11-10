package domain

import "testing"

func sp(s string) *string { return &s }

func TestRoundMatchesJS(t *testing.T) {
	if Round(2.5) != 3 || Round(-2.5) != -2 || Round(1.4) != 1 {
		t.Fatal("Round must match Math.round")
	}
}

func TestCalcTotals(t *testing.T) {
	lines := NormalizeLines([]Line{{Qty: 2, UnitPriceCents: 1999}, {Qty: 0.5, UnitPriceCents: 333}})
	if lines[0].TotalCents != 3998 || lines[1].TotalCents != 167 {
		t.Fatalf("line totals %+v", lines)
	}
	got := CalcTotals(lines, 0.15, []Payment{{AmountCents: 1000}})
	if got.Subtotal != 4165 || got.Tax != 625 || got.Total != 4790 || got.Balance != 3790 {
		t.Fatalf("totals %+v", got)
	}
	zero := CalcTotals(lines, 0, nil)
	if zero.Tax != 0 || zero.Total != 4165 {
		t.Fatalf("zero tax %+v", zero)
	}
}

func TestFormatCents(t *testing.T) {
	cases := map[int64]string{0: "R0.00", 40000: "R400.00", 100000: "R1,000.00", 123456789: "R1,234,567.89", -550: "-R5.50"}
	for in, want := range cases {
		if got := FormatCents(in); got != want {
			t.Errorf("FormatCents(%d)=%s want %s", in, got, want)
		}
	}
}

func TestDates(t *testing.T) {
	if AddDays("2026-11-08", 7) != "2026-11-15" || AddDays("2026-12-30", 3) != "2027-01-02" {
		t.Fatal("AddDays")
	}
	if DaysBetween("2026-11-01", "2026-11-08") != 7 || DaysBetween("2026-11-08", "2026-11-01") != -7 {
		t.Fatal("DaysBetween")
	}
	if ResolveRelative("today-45", "2026-11-08") != "2026-09-24" || ResolveRelative("today", "2026-11-08") != "2026-11-08" || ResolveRelative("2026-01-01", "x") != "2026-01-01" {
		t.Fatal("ResolveRelative")
	}
	if DaysOverdue(sp("2026-11-01"), "2026-11-08", 100) == nil || *DaysOverdue(sp("2026-11-01"), "2026-11-08", 100) != 7 {
		t.Fatal("overdue 7")
	}
	if DaysOverdue(sp("2026-11-01"), "2026-11-08", 0) != nil || DaysOverdue(sp("2026-11-09"), "2026-11-08", 100) != nil || DaysOverdue(nil, "2026-11-08", 100) != nil {
		t.Fatal("not overdue cases")
	}
}

func inv(status string, number *string, total int64, paid int64, due *string) *Doc {
	d := &Doc{Type: "invoice", Status: status, Number: number, TotalCents: total, DueDate: due}
	if paid > 0 {
		d.Payments = []Payment{{AmountCents: paid}}
	}
	return d
}

func TestDeriveInvoiceStatus(t *testing.T) {
	today := "2026-11-08"
	n := sp("INV-2026-001")
	cases := []struct {
		name string
		d    *Doc
		want string
	}{
		{"cancelled stays", inv("cancelled", n, 100, 0, sp("2026-01-01")), "cancelled"},
		{"draft stays", inv("draft", nil, 100, 0, nil), "draft"},
		{"no number is draft", inv("sent", nil, 100, 0, nil), "draft"},
		{"paid", inv("sent", n, 100, 100, sp("2026-01-01")), "paid"},
		{"partial not due", inv("sent", n, 100, 40, sp("2026-11-20")), "partially_paid"},
		{"partial overdue", inv("sent", n, 100, 40, sp("2026-11-01")), "overdue"},
		{"unpaid not due", inv("sent", n, 100, 0, sp("2026-11-08")), "sent"},
		{"unpaid overdue", inv("sent", n, 100, 0, sp("2026-11-07")), "overdue"},
	}
	for _, c := range cases {
		if got := DeriveInvoiceStatus(c.d, today); got != c.want {
			t.Errorf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}

func TestDeriveQuotationStatus(t *testing.T) {
	q := &Doc{Type: "quotation", Status: "sent", ValidUntil: sp("2026-11-01")}
	if DeriveQuotationStatus(q, "2026-11-08") != "expired" {
		t.Fatal("expired")
	}
	q.Status = "accepted"
	if DeriveQuotationStatus(q, "2026-11-08") != "accepted" {
		t.Fatal("accepted stays")
	}
}

func TestStamps(t *testing.T) {
	today := "2026-11-08"
	n := sp("INV-2026-001")
	if s := StampFor(inv("sent", n, 100, 100, nil), today); s.Text != "PAID" || s.Color != "green" {
		t.Fatal("paid stamp")
	}
	if s := StampFor(inv("sent", n, 100, 0, sp("2026-12-01")), today); s.Text != "UNPAID" {
		t.Fatal("unpaid stamp")
	}
	if s := StampFor(inv("sent", n, 100, 0, sp("2026-01-01")), today); s.Text != "OVERDUE" || !s.Strong {
		t.Fatal("overdue stamp")
	}
	if StampFor(&Doc{Type: "jobcard"}, today) != nil {
		t.Fatal("jobcard has no stamp")
	}
}

func TestTransitions(t *testing.T) {
	ok := [][3]string{{"quotation", "sent", "accepted"}, {"quotation", "sent", "declined"}, {"invoice", "sent", "cancelled"}, {"jobcard", "open", "in_progress"}, {"jobcard", "closed", "open"}}
	for _, c := range ok {
		if !TransitionAllowed(c[0], c[1], c[2]) {
			t.Errorf("should allow %v", c)
		}
	}
	bad := [][3]string{{"invoice", "paid", "cancelled"}, {"invoice", "sent", "paid"}, {"invoice", "partially_paid", "cancelled"}, {"quotation", "converted", "accepted"}, {"jobcard", "open", "closed"}}
	for _, c := range bad {
		if TransitionAllowed(c[0], c[1], c[2]) {
			t.Errorf("should reject %v", c)
		}
	}
}

func TestNextNumber(t *testing.T) {
	docs := []Doc{{Type: "invoice", Number: sp("INV-2026-009")}, {Type: "invoice", Number: sp("INV-2026-011")}, {Type: "invoice", Number: sp("INV-2025-050")}, {Type: "quotation", Number: sp("QUO-2026-004")}}
	if NextNumber(docs, "invoice", 2026) != "INV-2026-012" {
		t.Fatal("continues after max")
	}
	if NextNumber(docs, "invoice", 2027) != "INV-2027-001" {
		t.Fatal("restarts each year")
	}
	if NextNumber(docs, "jobcard", 2026) != "JC-2026-001" {
		t.Fatal("empty type")
	}
}

func TestDashboard(t *testing.T) {
	today := "2026-11-08"
	n1, n2 := sp("INV-2026-001"), sp("INV-2026-002")
	docs := []Doc{
		{Type: "invoice", Status: "sent", Number: n1, TotalCents: 1000, IssueDate: sp("2026-10-01"), DueDate: sp("2026-10-08"), FollowUpDate: sp("2026-11-07")},
		{Type: "invoice", Status: "sent", Number: n2, TotalCents: 500, IssueDate: sp("2026-11-05"), DueDate: sp("2026-11-12"), Payments: []Payment{{AmountCents: 200}}},
		{Type: "invoice", Status: "cancelled", Number: sp("INV-2026-003"), TotalCents: 9999, IssueDate: sp("2026-11-01")},
	}
	d := BuildDashboard(docs, 2026, today)
	if d.Totals.InvoicedCents != 1500 || d.Totals.PaidCents != 200 || d.Totals.OutstandingCents != 1300 || d.Totals.OverdueCents != 1000 {
		t.Fatalf("totals %+v", d.Totals)
	}
	if d.Ageing.D31_60 != 1000 || d.Attention.DueNext7Count != 1 || d.Attention.DueNext7Cents != 300 || d.Attention.FollowUpsDue != 1 {
		t.Fatalf("ageing/attention %+v %+v", d.Ageing, d.Attention)
	}
	if d.StatusCounts["cancelled"] != 1 || d.StatusCounts["overdue"] != 1 || len(d.Monthly) != 12 || d.Monthly[9].InvoicedCents != 1000 {
		t.Fatalf("counts/monthly %+v", d.StatusCounts)
	}
	if d.NextNumbers.Invoice != "INV-2026-004" {
		t.Fatalf("next %s", d.NextNumbers.Invoice)
	}
}
