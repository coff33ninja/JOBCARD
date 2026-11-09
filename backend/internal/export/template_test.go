package export

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jobcard/internal/domain"
)

func sp(s string) *string { return &s }

func sample(docType string) Ctx {
	d := domain.Doc{
		ID: "d1", Type: docType, Number: sp("INV-2026-001"), IssueDate: sp("2026-11-08"), DueDate: sp("2026-11-15"),
		Reference: "Laptop repair", Scope: "Fix screen\nand keyboard", Status: "sent", Notes: "Thank you",
		WorkItems: []domain.WorkItem{{Title: "Screen", Detail: "Replaced panel"}, {Title: "Keyboard"}},
		Lines: []domain.Line{
			{Description: "Panel & fitting", Qty: 1, UnitPriceCents: 100000, TotalCents: 100000},
			{Description: "Keyboard", Qty: 2, UnitPriceCents: 25000, TotalCents: 50000},
		},
		SubtotalCents: 150000, TotalCents: 150000,
		Payments: []domain.Payment{{AmountCents: 40000}},
	}
	switch docType {
	case "quotation":
		n := "QUO-2026-001"
		d.Number, d.ValidUntil = &n, sp("2026-11-22")
	case "jobcard":
		n := "JC-2026-001"
		d.Number, d.Status = &n, "open"
	}
	return Ctx{
		Doc: d, Client: domain.Client{Name: "Acme <Pty> & Co", Contact: "Jane"}, Today: "2026-11-08",
		Settings: domain.Settings{BusinessName: "Example IT", Person: "Alex", Phone: "000", BankName: "Example Bank"},
		Related:  []string{"INV-2026-001", "QUO-2026-001"},
	}
}

func part(t *testing.T, docx []byte, name string) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(docx), int64(len(docx)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range zr.File {
		if f.Name == name {
			rc, _ := f.Open()
			defer rc.Close()
			b, _ := io.ReadAll(rc)
			return string(b)
		}
	}
	t.Fatalf("part %s missing", name)
	return ""
}

func wellFormed(t *testing.T, x string) {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(x))
	for {
		if _, err := dec.Token(); err == io.EOF {
			return
		} else if err != nil {
			t.Fatalf("malformed xml: %v", err)
		}
	}
}

func TestDefaultTemplatesRender(t *testing.T) {
	for _, typ := range []string{"invoice", "quotation", "jobcard"} {
		tpl, err := DefaultTemplate(typ)
		if err != nil {
			t.Fatal(err)
		}
		out, warn, err := Render(tpl, sample(typ))
		if err != nil || len(warn) != 0 {
			t.Fatalf("%s: %v %v", typ, err, warn)
		}
		x := part(t, out, "word/document.xml")
		wellFormed(t, x)
		if strings.Contains(x, "{{") {
			t.Fatalf("%s: placeholders left over", typ)
		}
		if !strings.Contains(x, "Acme &lt;Pty&gt; &amp; Co") || !strings.Contains(x, "Example IT") {
			t.Fatalf("%s: values missing/unescaped", typ)
		}
		if !strings.Contains(x, "Replaced panel") && typ != "jobcard" || !strings.Contains(x, "Screen: Replaced panel") {
			t.Fatalf("%s: work items", typ)
		}
		if !strings.Contains(x, `Fix screen</w:t><w:br/><w:t xml:space="preserve">and keyboard`) {
			t.Fatalf("%s: newline not converted to a line break", typ)
		}
	}
}

func TestInvoiceRepeatsLinesAndMoney(t *testing.T) {
	tpl, _ := DefaultTemplate("invoice")
	out, _, _ := Render(tpl, sample("invoice"))
	x := part(t, out, "word/document.xml")
	for _, want := range []string{"Panel &amp; fitting", ">Keyboard<", "1,000.00", "R1,500.00", "R400.00", "R1,100.00", "INV-2026-001 / Acme"} {
		if !strings.Contains(x, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Count(x, "<w:tr>") != 3 { // header + 2 line items
		t.Errorf("rows = %d", strings.Count(x, "<w:tr>"))
	}
	if !strings.Contains(x, "PARTIALLY PAID") || !strings.Contains(x, `w:val="B26A00"`) {
		t.Error("stamp text/colour")
	}
	j, _, _ := Render(mustTpl("jobcard"), sample("jobcard"))
	jx := part(t, j, "word/document.xml")
	if !strings.Contains(jx, "QUO-2026-001") || strings.Contains(jx, "PAID") {
		t.Error("jobcard related docs / no stamp")
	}
}

func mustTpl(typ string) []byte { b, _ := DefaultTemplate(typ); return b }

func TestPlaceholderSplitAcrossRuns(t *testing.T) {
	body := `<w:p><w:r><w:rPr><w:b/></w:rPr><w:t xml:space="preserve">Name: </w:t></w:r>` +
		`<w:r><w:rPr><w:i/></w:rPr><w:t>{{cli</w:t></w:r><w:r><w:t>ent.na</w:t></w:r><w:r><w:rPr><w:b/></w:rPr><w:t>me}} (ok)</w:t></w:r></w:p>`
	tpl, _ := assembleDocx(body)
	out, warn, err := Render(tpl, sample("invoice"))
	if err != nil || len(warn) != 0 {
		t.Fatalf("%v %v", err, warn)
	}
	x := part(t, out, "word/document.xml")
	wellFormed(t, x)
	if !strings.Contains(x, "<w:i/></w:rPr><w:t xml:space=\"preserve\">Acme &lt;Pty&gt; &amp; Co</w:t>") {
		t.Fatalf("value should sit in the run where the placeholder started:\n%s", x)
	}
	if !strings.Contains(x, " (ok)") || strings.Contains(x, "{{") || strings.Contains(x, "ent.na") {
		t.Fatalf("leftovers:\n%s", x)
	}
}

func TestUnknownPlaceholderIsKeptAndReported(t *testing.T) {
	tpl, _ := assembleDocx(`<w:p><w:r><w:t>{{ nope.key }} and {{ client.name }}</w:t></w:r></w:p>`)
	out, warn, _ := Render(tpl, sample("invoice"))
	x := part(t, out, "word/document.xml")
	if !strings.Contains(x, "{{nope.key}}") || !strings.Contains(x, "Acme") || len(warn) != 1 || warn[0] != "nope.key" {
		t.Fatalf("%v\n%s", warn, x)
	}
}

func TestStampStrikeAndInvalidTemplate(t *testing.T) {
	c := sample("invoice")
	c.Doc.Status = "cancelled"
	out, _, _ := Render(mustTpl("invoice"), c)
	x := part(t, out, "word/document.xml")
	if !strings.Contains(x, "CANCELLED") || !strings.Contains(x, "<w:strike/>") {
		t.Error("cancelled stamp should be struck through")
	}
	if _, _, err := Render([]byte("not a zip"), c); err == nil {
		t.Error("invalid template must error")
	}
}

func TestEnsureNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	tp := Templates{Dir: filepath.Join(dir, "Templates")}
	created, err := tp.Ensure()
	if err != nil || len(created) != 4 { // 3 templates + README
		t.Fatalf("%v %v", created, err)
	}
	custom := []byte("user edited this")
	os.WriteFile(tp.File("invoice"), custom, 0o644)
	again, _ := tp.Ensure()
	if len(again) != 0 {
		t.Fatalf("second Ensure created %v", again)
	}
	b, _ := os.ReadFile(tp.File("invoice"))
	if string(b) != string(custom) {
		t.Fatal("existing template was overwritten")
	}
}
