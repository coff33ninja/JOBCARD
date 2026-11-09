// Package export renders documents (docx, pdf via Word, register xlsx).
package export

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"jobcard/internal/domain"
)

// Ctx is everything needed to render one document.
type Ctx struct {
	Doc             domain.Doc
	Client          domain.Client
	Settings        domain.Settings
	Today           string
	QuotationNumber string   // related quotation, if any
	JobCardNumber   string   // related job card, if any
	Related         []string // job cards: linked quotation/invoice numbers
}

const navy = "18374E"

var stampColor = map[string]string{"green": "2E7D32", "amber": "B26A00", "red": "C62828", "grey": "6B7280"}

// ---------- tiny WordprocessingML builder ----------

func esc(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

type run struct {
	text   string
	bold   bool
	italic bool
	strike bool
	size   int // half-points
	color  string
}

func (r run) xml() string {
	p := ""
	if r.bold {
		p += "<w:b/>"
	}
	if r.italic {
		p += "<w:i/>"
	}
	if r.strike {
		p += "<w:strike/>"
	}
	if r.color != "" {
		p += `<w:color w:val="` + r.color + `"/>`
	}
	if r.size > 0 {
		p += fmt.Sprintf(`<w:sz w:val="%d"/><w:szCs w:val="%d"/>`, r.size, r.size)
	}
	return `<w:r><w:rPr>` + p + `</w:rPr><w:t xml:space="preserve">` + esc(r.text) + `</w:t></w:r>`
}

type para struct {
	runs   []run
	align  string // left,center,right
	before int
	after  int
	rule   bool   // bottom rule (section heading)
	box    string // colour => bordered box (stamp)
	indent int
	keep   bool
}

func (p para) xml() string {
	ppr := ""
	if p.keep {
		ppr += "<w:keepNext/>"
	}
	if p.rule {
		ppr += `<w:pBdr><w:bottom w:val="single" w:sz="6" w:space="1" w:color="` + navy + `"/></w:pBdr>`
	}
	if p.box != "" {
		e := func(side string) string {
			return `<w:` + side + ` w:val="single" w:sz="12" w:space="4" w:color="` + p.box + `"/>`
		}
		ppr += `<w:pBdr>` + e("top") + e("left") + e("bottom") + e("right") + `</w:pBdr>`
	}
	ppr += fmt.Sprintf(`<w:spacing w:before="%d" w:after="%d"/>`, p.before, p.after)
	if p.indent > 0 {
		ppr += fmt.Sprintf(`<w:ind w:left="%d"/>`, p.indent)
	}
	if p.align != "" {
		ppr += `<w:jc w:val="` + p.align + `"/>`
	}
	var rs strings.Builder
	for _, r := range p.runs {
		rs.WriteString(r.xml())
	}
	return `<w:p><w:pPr>` + ppr + `</w:pPr>` + rs.String() + `</w:p>`
}

type cell struct {
	text  string
	bold  bool
	align string
	shade string
	color string
}

func table(widths []int, rows [][]cell) string {
	var b strings.Builder
	total := 0
	for _, w := range widths {
		total += w
	}
	bd := func(side string) string {
		return `<w:` + side + ` w:val="single" w:sz="4" w:space="0" w:color="BFBFBF"/>`
	}
	b.WriteString(fmt.Sprintf(`<w:tbl><w:tblPr><w:tblW w:w="%d" w:type="dxa"/><w:tblBorders>%s%s%s%s%s%s</w:tblBorders><w:tblLayout w:type="fixed"/><w:tblCellMar><w:left w:w="80" w:type="dxa"/><w:right w:w="80" w:type="dxa"/></w:tblCellMar></w:tblPr>`,
		total, bd("top"), bd("left"), bd("bottom"), bd("right"), bd("insideH"), bd("insideV")))
	b.WriteString(`<w:tblGrid>`)
	for _, w := range widths {
		b.WriteString(fmt.Sprintf(`<w:gridCol w:w="%d"/>`, w))
	}
	b.WriteString(`</w:tblGrid>`)
	for _, row := range rows {
		b.WriteString(`<w:tr>`)
		for i, c := range row {
			shade := ""
			if c.shade != "" {
				shade = `<w:shd w:val="clear" w:color="auto" w:fill="` + c.shade + `"/>`
			}
			b.WriteString(fmt.Sprintf(`<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/>%s</w:tcPr>`, widths[i], shade))
			b.WriteString(para{runs: []run{{text: c.text, bold: c.bold, color: c.color}}, align: c.align, before: 40, after: 40}.xml())
			b.WriteString(`</w:tc>`)
		}
		b.WriteString(`</w:tr>`)
	}
	b.WriteString(`</w:tbl>`)
	return b.String()
}

// ---------- document content ----------

func longDate(s *string) string {
	if s == nil {
		return "-"
	}
	t, err := domain.ParseDate(*s)
	if err != nil {
		return *s
	}
	return t.Format("2 January 2006")
}

func dash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

type builder struct{ b strings.Builder }

func (x *builder) p(p para)     { x.b.WriteString(p.xml()) }
func (x *builder) raw(s string) { x.b.WriteString(s) }
func (x *builder) heading(t string) {
	x.p(para{runs: []run{{text: t, bold: true, size: 24, color: navy}}, before: 240, after: 80, rule: true, keep: true})
}
func (x *builder) field(label, value string) {
	x.p(para{runs: []run{{text: label + ": ", bold: true}, {text: dash(value)}}, after: 40})
}
func (x *builder) text(t string) {
	x.p(para{runs: []run{{text: dash(t)}}, after: 60})
}
func (x *builder) right(label, value string, bold bool) {
	x.p(para{runs: []run{{text: label + ": ", bold: bold}, {text: value, bold: bold}}, align: "right", after: 40})
}

func (c Ctx) title() string {
	switch c.Doc.Type {
	case "quotation":
		return "QUOTATION & PROPOSED SERVICES"
	case "jobcard":
		return "JOB CARD"
	}
	return "INVOICE & SERVICE SUMMARY"
}

func (c Ctx) header(x *builder) {
	s := c.Settings
	x.p(para{runs: []run{{text: dash(s.BusinessName), bold: true, size: 40, color: navy}}, after: 40})
	var parts []string
	for _, v := range []string{s.Person, s.Phone, s.Email} {
		if strings.TrimSpace(v) != "" {
			parts = append(parts, v)
		}
	}
	x.p(para{runs: []run{{text: strings.Join(parts, " | "), color: "444444"}}, after: 20})
	if strings.TrimSpace(s.Location) != "" {
		x.p(para{runs: []run{{text: s.Location, color: "444444"}}, after: 120})
	}
	if st := domain.StampFor(&c.Doc, c.Today); st != nil {
		col := stampColor[st.Color]
		size := 32
		if st.Strong {
			size = 40
		}
		x.p(para{runs: []run{{text: st.Text, bold: true, strike: st.Struck, size: size, color: col}}, align: "center", box: col, indent: 6200, before: 60, after: 160})
	}
	x.p(para{runs: []run{{text: c.title(), bold: true, size: 30, color: navy}}, after: 120})
}

func (c Ctx) clientBlock(x *builder, title string) {
	d, cl := c.Doc, c.Client
	x.heading(title)
	x.field("Client", cl.Name)
	x.field("Contact person", cl.Contact)
	x.field("Phone", cl.Phone)
	x.field("Email", cl.Email)
	x.field("Billing address", cl.Address)
	ref := d.Reference
	if c.JobCardNumber != "" {
		ref = c.JobCardNumber + " - " + d.Reference
	}
	x.field("Job / device reference", ref)
	x.field("Service scope", d.Scope)
}

func (c Ctx) workItems(x *builder, title string) {
	x.heading(title)
	if len(c.Doc.WorkItems) == 0 {
		x.text("-")
		return
	}
	for _, w := range c.Doc.WorkItems {
		t := w.Title
		if strings.TrimSpace(w.Detail) != "" {
			t += ": " + w.Detail
		}
		x.p(para{runs: []run{{text: "\u2022 " + t}}, after: 40, indent: 240})
	}
}

func (c Ctx) financials(x *builder, totalLabel string) {
	d := c.Doc
	x.heading("FINANCIAL BREAKDOWN")
	rows := [][]cell{{
		{text: "Description", bold: true, shade: navy, color: "FFFFFF"},
		{text: "Qty", bold: true, shade: navy, color: "FFFFFF", align: "center"},
		{text: "Unit price (R)", bold: true, shade: navy, color: "FFFFFF", align: "right"},
		{text: "Total (R)", bold: true, shade: navy, color: "FFFFFF", align: "right"},
	}}
	for _, l := range d.Lines {
		rows = append(rows, []cell{
			{text: l.Description},
			{text: trimFloat(l.Qty), align: "center"},
			{text: money(l.UnitPriceCents), align: "right"},
			{text: money(l.TotalCents), align: "right"},
		})
	}
	x.raw(table([]int{4400, 1000, 2100, 2138}, rows))
	x.p(para{after: 60})
	x.right("Subtotal", domain.FormatCents(d.SubtotalCents), false)
	vat := "VAT (Non-VAT Vendor)"
	if c.Settings.TaxRate > 0 {
		vat = fmt.Sprintf("VAT (%s%%)", trimFloat(c.Settings.TaxRate*100))
	}
	x.right(vat, domain.FormatCents(d.TaxCents), false)
	x.right(totalLabel, domain.FormatCents(d.TotalCents), true)
	switch d.Type {
	case "invoice":
		paid := domain.PaidSum(d.Payments)
		x.right("Payments received", domain.FormatCents(paid), false)
		x.right("BALANCE DUE", domain.FormatCents(d.TotalCents-paid), true)
	case "quotation":
		dep := "-"
		if d.DepositCents != nil {
			dep = domain.FormatCents(*d.DepositCents)
		}
		x.right("Deposit, if agreed", dep, false)
	}
}

func (c Ctx) paymentDetails(x *builder, terms string) {
	s := c.Settings
	x.heading("PAYMENT DETAILS")
	x.field("Bank", s.BankName)
	x.field("Account holder", s.AccountHolder)
	x.field("Account number", s.AccountNumber)
	x.field("Branch code", s.BranchCode)
	num := "-"
	if c.Doc.Number != nil {
		num = *c.Doc.Number
	}
	x.field("Payment reference", num+" / "+c.Client.Name)
	x.field("Payment terms", terms)
}

func (c Ctx) body() string {
	var x builder
	d := c.Doc
	num := "-"
	if d.Number != nil {
		num = *d.Number
	}
	c.header(&x)
	switch d.Type {
	case "invoice":
		x.field("Invoice number", num)
		x.field("Invoice date", longDate(d.IssueDate))
		x.field("Payment due", longDate(d.DueDate))
		rq := c.QuotationNumber
		if rq == "" {
			rq = "Not applicable"
		}
		x.field("Related quotation", rq)
		c.clientBlock(&x, "CLIENT & JOB DETAILS")
		c.workItems(&x, "COMPLETED SERVICES & WORK DONE")
		c.financials(&x, "TOTAL INVOICE AMOUNT")
		terms := d.Terms
		if strings.TrimSpace(terms) == "" {
			terms = "Due by " + longDate(d.DueDate)
		}
		c.paymentDetails(&x, terms)
		x.heading("NOTES & RECOMMENDATIONS")
		x.text(d.Notes)
	case "quotation":
		x.field("Quotation number", num)
		x.field("Quotation date", longDate(d.IssueDate))
		if d.IssueDate != nil && d.ValidUntil != nil {
			x.field("Valid for", fmt.Sprintf("%d days (until %s)", domain.DaysBetween(*d.IssueDate, *d.ValidUntil), longDate(d.ValidUntil)))
		}
		c.clientBlock(&x, "CLIENT & JOB DETAILS")
		c.workItems(&x, "PROPOSED SERVICES & WORK TO BE DONE")
		x.field("Exclusions / assumptions", d.Exclusions)
		x.field("Estimated turnaround", d.Turnaround)
		c.financials(&x, "TOTAL QUOTED AMOUNT")
		terms := d.Terms
		if strings.TrimSpace(terms) == "" {
			terms = "Deposit and balance as agreed on acceptance"
		}
		c.paymentDetails(&x, terms)
		x.heading("TERMS & CLIENT ACCEPTANCE")
		x.text(d.Terms)
		x.field("Client representative", "______________________________")
		x.field("Acceptance signature / approval reference", "______________________________")
		x.field("Acceptance date", "______________________________")
		x.heading("NOTES")
		x.text(d.Notes)
	default:
		x.field("Job card number", num)
		x.field("Date opened", longDate(d.IssueDate))
		x.field("Job status", d.Status)
		c.clientBlock(&x, "CLIENT & JOB DETAILS")
		c.workItems(&x, "WORK LOG")
		x.heading("LINKED DOCUMENTS")
		if len(c.Related) == 0 {
			x.text("-")
		}
		for _, r := range c.Related {
			x.p(para{runs: []run{{text: "\u2022 " + r}}, after: 40, indent: 240})
		}
		x.heading("NOTES")
		x.text(d.Notes)
	}
	return x.b.String()
}

func money(c int64) string { return strings.TrimPrefix(domain.FormatCents(c), "R") }

func trimFloat(f float64) string {
	s := fmt.Sprintf("%.2f", f)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return s
}

// BuildDocx renders the document as a .docx file.
func BuildDocx(c Ctx) ([]byte, error) {
	const ns = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"`
	doc := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document ` + ns + `><w:body>` + c.body() +
		`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1000" w:right="1134" w:bottom="1000" w:left="1134" w:header="708" w:footer="708" w:gutter="0"/></w:sectPr></w:body></w:document>`
	styles := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:styles ` + ns + `><w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri" w:cs="Calibri"/><w:sz w:val="21"/><w:szCs w:val="21"/></w:rPr></w:rPrDefault></w:docDefaults></w:styles>`
	files := []struct{ name, content string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/></Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`},
		{"word/_rels/document.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`},
		{"word/document.xml", doc},
		{"word/styles.xml", styles},
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	mod := time.Now()
	for _, f := range files {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: f.name, Method: zip.Deflate, Modified: mod})
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
