// Package export renders documents (docx from editable templates, pdf via Word, register xlsx).
package export

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

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

// ---------- tiny WordprocessingML builder (used only to write the default templates) ----------

func esc(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

type run struct {
	text   string
	bold   bool
	italic bool
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
	rule   bool // bottom rule (section heading)
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
	text   string
	detail string
	bold   bool
	align  string
	shade  string
	color  string
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
			if c.detail != "" {
				b.WriteString(para{runs: []run{{text: c.detail, italic: true, size: 18, color: "666666"}}, after: 40}.xml())
			}
			b.WriteString(`</w:tc>`)
		}
		b.WriteString(`</w:tr>`)
	}
	b.WriteString(`</w:tbl>`)
	return b.String()
}

// ---------- values the templates can use ----------

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

func money(c int64) string { return strings.TrimPrefix(domain.FormatCents(c), "R") }

func trimFloat(f float64) string {
	s := fmt.Sprintf("%.2f", f)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return s
}

func itemText(w domain.WorkItem) string {
	if strings.TrimSpace(w.Detail) == "" {
		return w.Title
	}
	return w.Title + ": " + w.Detail
}

func diagnosticText(section domain.DiagnosticSection) string {
	var parts []string
	if title := strings.TrimSpace(section.Title); title != "" {
		parts = append(parts, title)
	}
	if status := strings.TrimSpace(section.Status); status != "" {
		parts = append(parts, "STATUS: "+status)
	}
	if details := strings.TrimSpace(section.Details); details != "" {
		parts = append(parts, details)
	}
	return strings.Join(parts, "\n")
}

func signatureLine(value string) string {
	if strings.TrimSpace(value) == "" {
		return "______________________________"
	}
	return value
}

func signOffDate(value string) string {
	if strings.TrimSpace(value) == "" {
		return "______________________________"
	}
	if parsed, err := domain.ParseDate(value); err == nil {
		return parsed.Format("2 January 2006")
	}
	return value
}

// Values returns every {{placeholder}} the document-level templates understand.
func (c Ctx) Values() map[string]string {
	d, s, cl := c.Doc, c.Settings, c.Client
	num := "-"
	if d.Number != nil {
		num = *d.Number
	}
	var contact []string
	for _, v := range []string{s.Person, s.Phone, s.Email} {
		if strings.TrimSpace(v) != "" {
			contact = append(contact, v)
		}
	}
	rq := c.QuotationNumber
	if rq == "" {
		rq = "Not applicable"
	}
	ref := d.Reference
	if c.JobCardNumber != "" {
		ref = c.JobCardNumber + " - " + d.Reference
	}
	vat := "VAT (Non-VAT Vendor)"
	if s.TaxRate > 0 {
		vat = fmt.Sprintf("VAT (%s%%)", trimFloat(s.TaxRate*100))
	}
	paid := domain.PaidSum(d.Payments)
	deposit := "-"
	if d.DepositCents != nil {
		deposit = domain.FormatCents(*d.DepositCents)
	}
	validFor := "-"
	if d.IssueDate != nil && d.ValidUntil != nil {
		validFor = fmt.Sprintf("%d days (until %s)", domain.DaysBetween(*d.IssueDate, *d.ValidUntil), longDate(d.ValidUntil))
	}
	terms := d.Terms
	if strings.TrimSpace(terms) == "" {
		if d.Type == "quotation" {
			terms = "Deposit and balance as agreed on acceptance"
		} else {
			terms = "Due by " + longDate(d.DueDate)
		}
	}
	stampText, stampCol, stampStrike := "", "", ""
	if st := domain.StampFor(&d, c.Today); st != nil {
		stampText, stampCol = st.Text, stampColor[st.Color]
		if st.Struck {
			stampStrike = "1"
		}
	}
	return map[string]string{
		"business.name": s.BusinessName, "business.person": s.Person, "business.phone": s.Phone, "business.email": s.Email,
		"business.location": s.Location, "business.contact_line": strings.Join(contact, " | "),
		"bank.name": dash(s.BankName), "bank.holder": dash(s.AccountHolder), "bank.account": dash(s.AccountNumber), "bank.branch": dash(s.BranchCode),
		"doc.number": num, "doc.issue_date": longDate(d.IssueDate), "doc.due_date": longDate(d.DueDate), "doc.valid_until": longDate(d.ValidUntil),
		"doc.valid_for": validFor, "doc.reference": dash(d.Reference), "doc.reference_full": dash(ref), "doc.scope": dash(d.Scope),
		"doc.status": d.Status, "doc.terms": dash(d.Terms), "doc.exclusions": dash(d.Exclusions), "doc.turnaround": dash(d.Turnaround),
		"doc.notes": dash(d.Notes), "doc.related_quotation": rq, "doc.job_card": dash(c.JobCardNumber),
		"client.name": dash(cl.Name), "client.contact": dash(cl.Contact), "client.phone": dash(cl.Phone), "client.email": dash(cl.Email), "client.address": dash(cl.Address),
		"authorization.technician_signature": signatureLine(d.Authorization.TechnicianSignature),
		"authorization.technician_date":      signOffDate(d.Authorization.TechnicianDate),
		"authorization.client_signature":     signatureLine(d.Authorization.ClientSignature),
		"authorization.client_date":          signOffDate(d.Authorization.ClientDate),
		"stamp":                              stampText, "stamp.color": stampCol, "stamp.strike": stampStrike,
		"subtotal": domain.FormatCents(d.SubtotalCents), "tax_label": vat, "tax": domain.FormatCents(d.TaxCents), "total": domain.FormatCents(d.TotalCents),
		"paid": domain.FormatCents(paid), "balance": domain.FormatCents(d.TotalCents - paid), "deposit": deposit,
		"payment_reference": num + " / " + cl.Name, "payment_terms": terms,
	}
}

func (c Ctx) lineValues(l domain.Line) map[string]string {
	return map[string]string{
		"line.description": l.Description, "line.detail": l.Detail, "line.qty": trimFloat(l.Qty),
		"line.unit_price": money(l.UnitPriceCents), "line.total": money(l.TotalCents),
	}
}
