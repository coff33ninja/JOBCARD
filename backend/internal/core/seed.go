package core

import (
	"encoding/json"
	"fmt"
	"os"

	"jobcard/internal/domain"
)

type seedFile struct {
	Settings  domain.Settings `json:"settings"`
	Clients   []domain.Client `json:"clients"`
	Documents []domain.Doc    `json:"documents"`
}

// Seed loads sample data from a seed file (development only, behind --seed).
// It never touches a database that already holds clients or documents.
// Dates like "today-45" are resolved against the service's today.
func (s *Service) Seed(path string) (bool, error) {
	var n int
	if err := s.DB.QueryRow(`SELECT (SELECT COUNT(*) FROM clients)+(SELECT COUNT(*) FROM documents)`).Scan(&n); err != nil {
		return false, err
	}
	if n > 0 {
		return false, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	var sf seedFile
	if err := json.Unmarshal(raw, &sf); err != nil {
		return false, fmt.Errorf("parse seed: %w", err)
	}
	today := s.Today()
	tx, err := s.DB.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	settings := sf.Settings
	if settings.DefaultTermsDays == 0 {
		settings.DefaultTermsDays = 7
	}
	if settings.QuoteValidityDays == 0 {
		settings.QuoteValidityDays = 14
	}
	js, _ := json.Marshal(settings)
	if _, err := tx.Exec(`UPDATE settings SET json=? WHERE id=1`, string(js)); err != nil {
		return false, err
	}
	for _, c := range sf.Clients {
		if _, err := tx.Exec(`INSERT INTO clients(id,name,contact,phone,email,address,notes,created_at) VALUES(?,?,?,?,?,?,?,?)`,
			c.ID, c.Name, c.Contact, c.Phone, c.Email, c.Address, c.Notes, nowISO()); err != nil {
			return false, err
		}
	}
	res := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := domain.ResolveRelative(*p, today)
		return &v
	}
	for _, d := range sf.Documents {
		d.IssueDate, d.DueDate, d.ValidUntil = res(d.IssueDate), res(d.DueDate), res(d.ValidUntil)
		if d.CreatedAt == "" {
			d.CreatedAt = nowISO()
		} else {
			d.CreatedAt = domain.ResolveRelative(d.CreatedAt, today)
		}
		if d.UpdatedAt == "" {
			d.UpdatedAt = d.CreatedAt
		} else {
			d.UpdatedAt = domain.ResolveRelative(d.UpdatedAt, today)
		}
		d.Lines = domain.NormalizeLines(d.Lines)
		if d.WorkItems == nil {
			d.WorkItems = []domain.WorkItem{}
		}
		t := domain.CalcTotals(d.Lines, settings.TaxRate, nil)
		d.SubtotalCents, d.TaxCents, d.TotalCents = t.Subtotal, t.Tax, t.Total
		if err := insertDoc(tx, &d); err != nil {
			return false, fmt.Errorf("seed document %s: %w", d.ID, err)
		}
		for i, p := range d.Payments {
			pid := p.ID
			if pid == "" {
				pid = fmt.Sprintf("pay-%s-%d", d.ID, i+1)
			}
			if _, err := tx.Exec(`INSERT INTO payments(id,document_id,date,amount_cents,method,reference) VALUES(?,?,?,?,?,?)`,
				pid, d.ID, domain.ResolveRelative(p.Date, today), p.AmountCents, p.Method, p.Reference); err != nil {
				return false, err
			}
		}
	}
	return true, tx.Commit()
}
