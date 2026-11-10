// Package domain holds the pure business rules. It mirrors src/domain/*.ts in the UI;
// behaviour must stay identical (money in integer cents, ISO dates, derived statuses).
package domain

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Contact string `json:"contact"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Address string `json:"address"`
	Notes   string `json:"notes"`
}

type Line struct {
	Description    string  `json:"description"`
	Qty            float64 `json:"qty"`
	UnitPriceCents int64   `json:"unitPriceCents"`
	TotalCents     int64   `json:"totalCents"`
}

type Payment struct {
	ID          string `json:"id"`
	Date        string `json:"date"`
	AmountCents int64  `json:"amountCents"`
	Method      string `json:"method"`
	Reference   string `json:"reference"`
}

type WorkItem struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type Doc struct {
	ID            string     `json:"id"`
	Type          string     `json:"type"`
	Number        *string    `json:"number"`
	ClientID      string     `json:"clientId"`
	JobCardID     *string    `json:"jobCardId"`
	QuotationID   *string    `json:"quotationId"`
	IssueDate     *string    `json:"issueDate"`
	DueDate       *string    `json:"dueDate"`
	ValidUntil    *string    `json:"validUntil"`
	FollowUpDate  *string    `json:"followUpDate"`
	Reference     string     `json:"reference"`
	Scope         string     `json:"scope"`
	WorkItems     []WorkItem `json:"workItems"`
	Lines         []Line     `json:"lines"`
	TaxCents      int64      `json:"taxCents"`
	SubtotalCents int64      `json:"subtotalCents"`
	TotalCents    int64      `json:"totalCents"`
	DepositCents  *int64     `json:"depositCents"`
	Payments      []Payment  `json:"payments"`
	Status        string     `json:"status"`
	Terms         string     `json:"terms"`
	Exclusions    string     `json:"exclusions"`
	Turnaround    string     `json:"turnaround"`
	Notes         string     `json:"notes"`
	FolderPath    *string    `json:"folderPath"`
	CreatedAt     string     `json:"createdAt"`
	UpdatedAt     string     `json:"updatedAt"`
}

type Settings struct {
	BusinessName      string  `json:"businessName"`
	Person            string  `json:"person"`
	Phone             string  `json:"phone"`
	Email             string  `json:"email"`
	Location          string  `json:"location"`
	BankName          string  `json:"bankName"`
	AccountHolder     string  `json:"accountHolder"`
	AccountNumber     string  `json:"accountNumber"`
	BranchCode        string  `json:"branchCode"`
	TaxRate           float64 `json:"taxRate"`
	DefaultTermsDays  int     `json:"defaultTermsDays"`
	QuoteValidityDays int     `json:"quoteValidityDays"`
}

type Storage struct {
	DataDir    string `json:"dataDir"`
	DBPath     string `json:"dbPath"`
	ClientsDir string `json:"clientsDir"`
}

type Capabilities struct {
	Docx           bool `json:"docx"`
	PDF            bool `json:"pdf"`
	RevealInFolder bool `json:"revealInFolder"`
}

// ---------- money ----------

// Round matches JavaScript Math.round (halves round up).
func Round(x float64) int64 { return int64(math.Floor(x + 0.5)) }

func LineTotal(qty float64, unitPriceCents int64) int64 { return Round(qty * float64(unitPriceCents)) }

// NormalizeLines recomputes every line total; the server is authoritative for money.
func NormalizeLines(in []Line) []Line {
	out := make([]Line, 0, len(in))
	for _, l := range in {
		l.TotalCents = LineTotal(l.Qty, l.UnitPriceCents)
		out = append(out, l)
	}
	return out
}

func PaidSum(p []Payment) int64 {
	var s int64
	for _, x := range p {
		s += x.AmountCents
	}
	return s
}

type Totals struct{ Subtotal, Tax, Total, Paid, Balance int64 }

func CalcTotals(lines []Line, taxRate float64, payments []Payment) Totals {
	var sub int64
	for _, l := range lines {
		sub += l.TotalCents
	}
	tax := Round(float64(sub) * taxRate)
	t := Totals{Subtotal: sub, Tax: tax, Total: sub + tax, Paid: PaidSum(payments)}
	t.Balance = t.Total - t.Paid
	return t
}

// FormatCents renders cents as R1,000.00 (the style of the existing invoices).
func FormatCents(c int64) string {
	neg := c < 0
	if neg {
		c = -c
	}
	whole := strconv.FormatInt(c/100, 10)
	var b strings.Builder
	for i, ch := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(ch)
	}
	s := fmt.Sprintf("R%s.%02d", b.String(), c%100)
	if neg {
		return "-" + s
	}
	return s
}

// ---------- dates (ISO yyyy-mm-dd, UTC) ----------

const dateLayout = "2006-01-02"

func ParseDate(s string) (time.Time, error) { return time.ParseInLocation(dateLayout, s, time.UTC) }

func ValidDate(s string) bool { _, err := ParseDate(s); return err == nil }

func AddDays(s string, n int) string {
	t, err := ParseDate(s)
	if err != nil {
		return s
	}
	return t.AddDate(0, 0, n).Format(dateLayout)
}

// DaysBetween returns to - from in whole days.
func DaysBetween(from, to string) int {
	a, e1 := ParseDate(from)
	b, e2 := ParseDate(to)
	if e1 != nil || e2 != nil {
		return 0
	}
	return int(math.Round(b.Sub(a).Hours() / 24))
}

var relDate = regexp.MustCompile(`^today([+-]\d+)$`)

// ResolveRelative turns "today", "today-45", "today+7" into a date; other strings pass through.
func ResolveRelative(s, today string) string {
	if !strings.HasPrefix(s, "today") {
		return s
	}
	if s == "today" {
		return today
	}
	m := relDate.FindStringSubmatch(s)
	if m == nil {
		return today
	}
	n, _ := strconv.Atoi(m[1])
	return AddDays(today, n)
}

// DaysOverdue = today - due when balance > 0 and due < today, else nil.
func DaysOverdue(due *string, today string, balance int64) *int {
	if due == nil || balance <= 0 {
		return nil
	}
	d := DaysBetween(*due, today)
	if d > 0 {
		return &d
	}
	return nil
}

// ---------- statuses ----------

func DeriveInvoiceStatus(d *Doc, today string) string {
	if d.Status == "cancelled" {
		return "cancelled"
	}
	if d.Status == "draft" || d.Number == nil {
		return "draft"
	}
	total := d.TotalCents
	paid := PaidSum(d.Payments)
	balance := total - paid
	pastDue := d.DueDate != nil && DaysBetween(*d.DueDate, today) > 0

	if balance <= 0 && total > 0 {
		return "paid"
	}
	if paid > 0 && paid < total {
		if pastDue && balance > 0 {
			return "overdue"
		}
		return "partially_paid"
	}
	if paid == 0 {
		if pastDue && balance > 0 {
			return "overdue"
		}
		return "sent"
	}
	if d.Status == "" {
		return "sent"
	}
	return d.Status
}

func DeriveQuotationStatus(d *Doc, today string) string {
	switch d.Status {
	case "accepted", "declined", "converted", "draft":
		return d.Status
	}
	if d.ValidUntil != nil && DaysBetween(*d.ValidUntil, today) > 0 {
		return "expired"
	}
	if d.Status == "" {
		return "sent"
	}
	return d.Status
}

func DerivedStatus(d *Doc, today string) string {
	switch d.Type {
	case "invoice":
		return DeriveInvoiceStatus(d, today)
	case "quotation":
		return DeriveQuotationStatus(d, today)
	}
	return d.Status
}

var transitions = map[string]map[string][]string{
	"invoice": {
		"draft": {"cancelled"}, "sent": {"cancelled"}, "overdue": {"cancelled"},
	},
	"quotation": {
		"draft": {"cancelled"}, "sent": {"accepted", "declined", "cancelled"}, "accepted": {"cancelled"},
	},
	"jobcard": {
		"open":        {"in_progress"},
		"in_progress": {"completed", "open"},
		"completed":   {"invoiced", "closed", "in_progress"},
		"invoiced":    {"closed"},
		"closed":      {"open"},
	},
}

// TransitionAllowed reports whether POST /documents/{id}/status may move from -> to.
// Payment-driven statuses (partially_paid, paid) and converted are never set directly.
func TransitionAllowed(docType, from, to string) bool {
	for _, t := range transitions[docType][from] {
		if t == to {
			return true
		}
	}
	return false
}

// ---------- numbering ----------

func Prefix(docType string) string {
	switch docType {
	case "jobcard":
		return "JC"
	case "quotation":
		return "QUO"
	}
	return "INV"
}

func FormatNumber(docType string, year, seq int) string {
	return fmt.Sprintf("%s-%d-%03d", Prefix(docType), year, seq)
}

// NextNumber continues after the highest issued number of that type and year.
func NextNumber(docs []Doc, docType string, year int) string {
	re := regexp.MustCompile(fmt.Sprintf(`^%s-%d-(\d+)$`, Prefix(docType), year))
	max := 0
	for _, d := range docs {
		if d.Type != docType || d.Number == nil {
			continue
		}
		if m := re.FindStringSubmatch(*d.Number); m != nil {
			if n, _ := strconv.Atoi(m[1]); n > max {
				max = n
			}
		}
	}
	return FormatNumber(docType, year, max+1)
}

// ---------- stamps (rendered onto exported files) ----------

type Stamp struct {
	Text   string
	Color  string // green | amber | red | grey
	Struck bool
	Strong bool
}

func StampFor(d *Doc, today string) *Stamp {
	switch d.Type {
	case "invoice":
		if d.Status == "cancelled" {
			return &Stamp{Text: "CANCELLED", Color: "grey", Struck: true}
		}
		if d.Status == "draft" || d.Number == nil {
			return &Stamp{Text: "DRAFT", Color: "grey"}
		}
		switch DeriveInvoiceStatus(d, today) {
		case "paid":
			return &Stamp{Text: "PAID", Color: "green"}
		case "partially_paid":
			return &Stamp{Text: "PARTIALLY PAID", Color: "amber"}
		case "overdue":
			return &Stamp{Text: "OVERDUE", Color: "red", Strong: true}
		}
		return &Stamp{Text: "UNPAID", Color: "red"}
	case "quotation":
		if d.Status == "draft" || d.Number == nil {
			return &Stamp{Text: "DRAFT", Color: "grey"}
		}
		switch DeriveQuotationStatus(d, today) {
		case "accepted", "converted":
			return &Stamp{Text: "ACCEPTED", Color: "green"}
		case "declined":
			return &Stamp{Text: "DECLINED", Color: "red"}
		case "expired":
			return &Stamp{Text: "EXPIRED", Color: "grey"}
		}
		return &Stamp{Text: "DRAFT", Color: "grey"}
	}
	return nil
}

// ---------- dashboard ----------

type MonthRow struct {
	Month            int   `json:"month"`
	Jobs             int   `json:"jobs"`
	InvoicedCents    int64 `json:"invoicedCents"`
	PaidCents        int64 `json:"paidCents"`
	OutstandingCents int64 `json:"outstandingCents"`
}

type Dashboard struct {
	Year   int `json:"year"`
	Totals struct {
		InvoicedCents    int64 `json:"invoicedCents"`
		PaidCents        int64 `json:"paidCents"`
		OutstandingCents int64 `json:"outstandingCents"`
		OverdueCents     int64 `json:"overdueCents"`
	} `json:"totals"`
	StatusCounts map[string]int `json:"statusCounts"`
	Attention    struct {
		FollowUpsDue  int   `json:"followUpsDue"`
		DueNext7Count int   `json:"dueNext7Count"`
		DueNext7Cents int64 `json:"dueNext7Cents"`
	} `json:"attention"`
	Ageing struct {
		D1_30   int64 `json:"d1_30"`
		D31_60  int64 `json:"d31_60"`
		D61_90  int64 `json:"d61_90"`
		D90Plus int64 `json:"d90plus"`
	} `json:"ageing"`
	Monthly     []MonthRow `json:"monthly"`
	NextNumbers struct {
		JobCard   string `json:"jobcard"`
		Quotation string `json:"quotation"`
		Invoice   string `json:"invoice"`
	} `json:"nextNumbers"`
}

// BuildDashboard mirrors MockApi.getDashboard. followUpsDue counts unpaid invoices
// whose follow-up reminder date has arrived.
func BuildDashboard(docs []Doc, year int, today string) Dashboard {
	var d Dashboard
	d.Year = year
	d.StatusCounts = map[string]int{"draft": 0, "sent": 0, "partially_paid": 0, "paid": 0, "overdue": 0, "cancelled": 0}
	d.Monthly = make([]MonthRow, 12)
	for i := range d.Monthly {
		d.Monthly[i].Month = i + 1
	}
	todayT, _ := ParseDate(today)

	for i := range docs {
		doc := &docs[i]
		if doc.Type == "jobcard" && doc.IssueDate != nil {
			if t, err := ParseDate(*doc.IssueDate); err == nil && t.Year() == year {
				d.Monthly[int(t.Month())-1].Jobs++
			}
		}
	}
	for i := range docs {
		inv := &docs[i]
		if inv.Type != "invoice" {
			continue
		}
		st := DeriveInvoiceStatus(inv, today)
		if _, ok := d.StatusCounts[st]; ok {
			d.StatusCounts[st]++
		}
		if st == "cancelled" {
			continue
		}
		paid := PaidSum(inv.Payments)
		bal := inv.TotalCents - paid

		if inv.IssueDate != nil {
			if t, err := ParseDate(*inv.IssueDate); err == nil && t.Year() == year {
				d.Totals.InvoicedCents += inv.TotalCents
				d.Totals.PaidCents += paid
				d.Totals.OutstandingCents += bal
				m := &d.Monthly[int(t.Month())-1]
				m.InvoicedCents += inv.TotalCents
				m.PaidCents += paid
				m.OutstandingCents += bal
			}
		}
		if od := DaysOverdue(inv.DueDate, today, bal); od != nil {
			d.Totals.OverdueCents += bal
			switch {
			case *od <= 30:
				d.Ageing.D1_30 += bal
			case *od <= 60:
				d.Ageing.D31_60 += bal
			case *od <= 90:
				d.Ageing.D61_90 += bal
			default:
				d.Ageing.D90Plus += bal
			}
		}
		if inv.FollowUpDate != nil && bal > 0 && *inv.FollowUpDate <= today {
			d.Attention.FollowUpsDue++
		}
		if inv.DueDate != nil && bal > 0 {
			if dueT, err := ParseDate(*inv.DueDate); err == nil {
				diff := int(math.Round(dueT.Sub(todayT).Hours() / 24))
				if diff >= 0 && diff <= 7 {
					d.Attention.DueNext7Count++
					d.Attention.DueNext7Cents += bal
				}
			}
		}
	}
	d.NextNumbers.JobCard = NextNumber(docs, "jobcard", year)
	d.NextNumbers.Quotation = NextNumber(docs, "quotation", year)
	d.NextNumbers.Invoice = NextNumber(docs, "invoice", year)
	return d
}
