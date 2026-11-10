package export

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Templates manages the editable .docx templates (invoice.docx, quotation.docx, jobcard.docx).
// Missing files are created from the built-in defaults; existing files are never overwritten.
type Templates struct{ Dir string }

var templateTypes = []string{"invoice", "quotation", "jobcard"}

func (t Templates) path(docType string) string { return filepath.Join(t.Dir, docType+".docx") }

// Ensure creates the folder, any missing default template, and README.txt.
func (t Templates) Ensure() ([]string, error) {
	if err := os.MkdirAll(t.Dir, 0o755); err != nil {
		return nil, err
	}
	var created []string
	for _, typ := range templateTypes {
		p := t.path(typ)
		if _, err := os.Stat(p); err == nil {
			continue
		}
		b, err := DefaultTemplate(typ)
		if err != nil {
			return created, err
		}
		if err := os.WriteFile(p, b, 0o644); err != nil {
			return created, err
		}
		created = append(created, p)
	}
	readme := filepath.Join(t.Dir, "README.txt")
	if _, err := os.Stat(readme); err != nil {
		if err := os.WriteFile(readme, []byte(templateReadme), 0o644); err != nil {
			return created, err
		}
		created = append(created, readme)
	}
	return created, nil
}

// Load returns the template for a document type (creating the defaults first if needed).
func (t Templates) Load(docType string) ([]byte, error) {
	if _, err := t.Ensure(); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(t.path(docType))
	if err != nil {
		return nil, fmt.Errorf("template %s: %w", t.path(docType), err)
	}
	return b, nil
}

func (t Templates) File(docType string) string { return t.path(docType) }

const templateReadme = `JobCard templates
=================
invoice.docx, quotation.docx and jobcard.docx in this folder are the layouts used for exports.
Open them in Word and change anything: fonts, colours, logo, wording, page layout.
Where a value should appear, type a placeholder in double curly braces. Keep the braces.
JobCard never overwrites a template that already exists. To go back to the original layout,
delete the file and it is recreated the next time a document is exported.

Business (Settings)
  {{business.name}} {{business.person}} {{business.phone}} {{business.email}} {{business.location}}
  {{business.contact_line}}   person | phone | email (empty parts left out)
  {{bank.name}} {{bank.holder}} {{bank.account}} {{bank.branch}}

Document
  {{doc.number}} {{doc.issue_date}} {{doc.due_date}} {{doc.valid_until}} {{doc.valid_for}}
  {{doc.reference}} {{doc.reference_full}} (job card number - reference) {{doc.job_card}}
  {{doc.scope}} {{doc.status}} {{doc.terms}} {{doc.exclusions}} {{doc.turnaround}} {{doc.notes}}
  {{doc.related_quotation}}
  {{stamp}}   PAID, UNPAID, OVERDUE, ACCEPTED and so on. Coloured automatically. Empty for job cards.

Client
  {{client.name}} {{client.contact}} {{client.phone}} {{client.email}} {{client.address}}

Money (formatted like R1,000.00)
  {{subtotal}} {{tax_label}} {{tax}} {{total}} {{paid}} {{balance}} {{deposit}}
  {{payment_reference}} {{payment_terms}}

Repeating rows
  A table row containing any {{line.*}} placeholder is repeated once per line item
  (put the line values you need in the same row).
  {{line.detail}} is the optional secondary description.
  A paragraph containing {{item.text}} (or {{item.title}} / {{item.detail}}) is repeated once per work item.
  A paragraph containing {{intake_item.label}} and {{intake_item.value}} is repeated once per intake item.
  A paragraph containing {{diagnostic_section.text}} is repeated once per diagnostic section.
  A paragraph containing {{related}} is repeated once per linked document number (job cards).
  Authorization placeholders: {{authorization.technician_signature}}, {{authorization.technician_date}},
  {{authorization.client_signature}}, and {{authorization.client_date}}.

A placeholder that is not recognised is left visible in the output so you can spot a typo.
The built-in invoice template puts job intake and completed work before FINANCIAL BREAKDOWN,
then lists diagnostics and authorization after payment details. The financial breakdown starts on page 2.
`

// DefaultTemplate builds the built-in layout for a document type, using placeholders.
func DefaultTemplate(docType string) ([]byte, error) {
	var x builder
	f := func(label, key string) { x.field(label, "{{"+key+"}}") }
	x.p(para{runs: []run{{text: "{{business.name}}", bold: true, size: 40, color: navy}}, after: 40})
	x.p(para{runs: []run{{text: "{{business.contact_line}}", color: "444444"}}, after: 20})
	x.p(para{runs: []run{{text: "{{business.location}}", color: "444444"}}, after: 120})
	x.p(para{runs: []run{{text: "{{stamp}}", bold: true, size: 36}}, align: "right", after: 120})

	clientBlock := func() {
		x.heading("CLIENT & JOB DETAILS")
		f("Client", "client.name")
		f("Contact person", "client.contact")
		f("Phone", "client.phone")
		f("Email", "client.email")
		f("Billing address", "client.address")
		f("Job / device reference", "doc.reference_full")
		f("Service scope", "doc.scope")
	}
	items := func(title string) {
		x.heading(title)
		x.p(para{runs: []run{{text: "\u2022 {{item.text}}"}}, after: 40, indent: 240})
	}
	intake := func() {
		x.heading("INTAKE INVENTORY & EQUIPMENT LOG")
		x.p(para{runs: []run{{text: "\u2022 {{intake_item.label}}: {{intake_item.value}}"}}, after: 40, indent: 240})
	}
	diagnostics := func() {
		x.heading("TECHNICAL DIAGNOSTIC REPORT")
		x.p(para{runs: []run{{text: "{{diagnostic_section.text}}"}}, after: 100})
	}
	authorization := func() {
		x.heading("AUTHORIZATION & SIGN-OFF")
		f("Technician signature", "authorization.technician_signature")
		f("Technician date", "authorization.technician_date")
		f("Client signature", "authorization.client_signature")
		f("Client date", "authorization.client_date")
	}
	financials := func(totalLabel string) {
		x.heading("FINANCIAL BREAKDOWN")
		head := func(t, a string) cell { return cell{text: t, bold: true, shade: navy, color: "FFFFFF", align: a} }
		x.raw(table([]int{4400, 1000, 2100, 2138}, [][]cell{
			{head("Description", ""), head("Qty", "center"), head("Unit price (R)", "right"), head("Total (R)", "right")},
			{{text: "{{line.description}}", detail: "{{line.detail}}"}, {text: "{{line.qty}}", align: "center"}, {text: "{{line.unit_price}}", align: "right"}, {text: "{{line.total}}", align: "right"}},
		}))
		x.p(para{after: 60})
		x.right("Subtotal", "{{subtotal}}", false)
		x.right("{{tax_label}}", "{{tax}}", false)
		x.right(totalLabel, "{{total}}", true)
	}
	paymentDetails := func() {
		x.heading("PAYMENT DETAILS")
		f("Bank", "bank.name")
		f("Account holder", "bank.holder")
		f("Account number", "bank.account")
		f("Branch code", "bank.branch")
		f("Payment reference", "payment_reference")
		f("Payment terms", "payment_terms")
	}

	switch docType {
	case "invoice":
		x.title("INVOICE & SERVICE SUMMARY")
		f("Invoice number", "doc.number")
		f("Invoice date", "doc.issue_date")
		f("Payment due", "doc.due_date")
		f("Related quotation", "doc.related_quotation")
		clientBlock()
		intake()
		items("COMPLETED SERVICES & WORK DONE")
		x.raw(`<w:p><w:r><w:br w:type="page"/></w:r></w:p>`)
		financials("TOTAL INVOICE AMOUNT")
		x.right("Payments received", "{{paid}}", false)
		x.right("BALANCE DUE", "{{balance}}", true)
		paymentDetails()
		diagnostics()
		authorization()
		x.heading("NOTES & RECOMMENDATIONS")
		x.text("{{doc.notes}}")
	case "quotation":
		x.title("QUOTATION & PROPOSED SERVICES")
		f("Quotation number", "doc.number")
		f("Quotation date", "doc.issue_date")
		f("Valid for", "doc.valid_for")
		clientBlock()
		intake()
		items("PROPOSED SERVICES & WORK TO BE DONE")
		f("Exclusions / assumptions", "doc.exclusions")
		f("Estimated turnaround", "doc.turnaround")
		financials("TOTAL QUOTED AMOUNT")
		x.right("Deposit, if agreed", "{{deposit}}", false)
		paymentDetails()
		x.heading("TERMS & CLIENT ACCEPTANCE")
		x.text("{{doc.terms}}")
		x.field("Client representative", "______________________________")
		x.field("Acceptance signature / approval reference", "______________________________")
		x.field("Acceptance date", "______________________________")
		x.heading("NOTES")
		x.text("{{doc.notes}}")
	case "jobcard":
		x.title("JOB CARD")
		f("Job card number", "doc.number")
		f("Date opened", "doc.issue_date")
		f("Job status", "doc.status")
		clientBlock()
		intake()
		items("WORK LOG")
		x.heading("LINKED DOCUMENTS")
		x.p(para{runs: []run{{text: "\u2022 {{related}}"}}, after: 40, indent: 240})
		diagnostics()
		authorization()
		x.heading("NOTES")
		x.text("{{doc.notes}}")
	default:
		return nil, fmt.Errorf("unknown document type %q", docType)
	}
	return assembleDocx(x.b.String())
}

type builder struct{ b strings.Builder }

func (x *builder) p(p para)     { x.b.WriteString(p.xml()) }
func (x *builder) raw(s string) { x.b.WriteString(s) }
func (x *builder) title(t string) {
	x.p(para{runs: []run{{text: t, bold: true, size: 30, color: navy}}, after: 120})
}
func (x *builder) heading(t string) {
	x.p(para{runs: []run{{text: t, bold: true, size: 24, color: navy}}, before: 240, after: 80, rule: true, keep: true})
}
func (x *builder) field(label, value string) {
	x.p(para{runs: []run{{text: label + ": ", bold: true}, {text: value}}, after: 40})
}
func (x *builder) text(t string) { x.p(para{runs: []run{{text: t}}, after: 60}) }
func (x *builder) right(label, value string, bold bool) {
	x.p(para{runs: []run{{text: label + ": ", bold: bold}, {text: value, bold: bold}}, align: "right", after: 40})
}

const wns = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"`

func assembleDocx(body string) ([]byte, error) {
	doc := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document ` + wns + `><w:body>` + body +
		`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1000" w:right="1134" w:bottom="1000" w:left="1134" w:header="708" w:footer="708" w:gutter="0"/></w:sectPr></w:body></w:document>`
	styles := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:styles ` + wns + `><w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri" w:cs="Calibri"/><w:sz w:val="21"/><w:szCs w:val="21"/></w:rPr></w:rPrDefault></w:docDefaults></w:styles>`
	files := []struct{ name, content string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/></Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`},
		{"word/_rels/document.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`},
		{"word/document.xml", doc},
		{"word/styles.xml", styles},
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: f.name, Method: zip.Deflate, Modified: time.Now()})
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(f.content)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
