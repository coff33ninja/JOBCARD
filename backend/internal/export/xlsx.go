package export

import (
	"sort"

	"github.com/xuri/excelize/v2"

	"jobcard/internal/domain"
)

// RegisterXLSX writes all invoices, quotations and job cards as a spreadsheet.
func RegisterXLSX(docs []domain.Doc, clients []domain.Client, today string) ([]byte, error) {
	names := map[string]string{}
	for _, c := range clients {
		names[c.ID] = c.Name
	}
	jobNumber := map[string]string{}
	for _, d := range docs {
		if d.Type == "jobcard" && d.Number != nil {
			jobNumber[d.ID] = *d.Number
		}
	}
	sorted := append([]domain.Doc(nil), docs...)
	key := func(d domain.Doc) string {
		if d.IssueDate != nil {
			return *d.IssueDate
		}
		return d.CreatedAt
	}
	sort.SliceStable(sorted, func(i, j int) bool { return key(sorted[i]) < key(sorted[j]) })

	f := excelize.NewFile()
	defer f.Close()
	sheet := "Register"
	f.SetSheetName("Sheet1", sheet)
	head := []string{"Number", "Type", "Client", "Job Card No", "Reference", "Issue Date", "Due / Valid Until", "Total", "Paid", "Balance", "Status", "Days Overdue", "Follow-up Date"}
	for i, h := range head {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	bold, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "FFFFFF"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"18374E"}, Pattern: 1}})
	f.SetCellStyle(sheet, "A1", "M1", bold)
	money, _ := f.NewStyle(&excelize.Style{CustomNumFmt: strPtr(`"R"#,##0.00`)})

	row := 2
	for i := range sorted {
		d := &sorted[i]
		set := func(col int, v any) {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			f.SetCellValue(sheet, cell, v)
		}
		num := ""
		if d.Number != nil {
			num = *d.Number
		}
		due := ""
		if d.DueDate != nil {
			due = *d.DueDate
		} else if d.ValidUntil != nil {
			due = *d.ValidUntil
		}
		issue := ""
		if d.IssueDate != nil {
			issue = *d.IssueDate
		}
		paid := domain.PaidSum(d.Payments)
		set(1, num)
		set(2, d.Type)
		set(3, names[d.ClientID])
		if d.JobCardID != nil {
			set(4, jobNumber[*d.JobCardID])
		}
		set(5, d.Reference)
		set(6, issue)
		set(7, due)
		set(11, domain.DerivedStatus(d, today))
		if d.FollowUpDate != nil {
			set(13, *d.FollowUpDate)
		}
		if d.Type != "jobcard" {
			set(8, float64(d.TotalCents)/100)
			if d.Type == "invoice" {
				set(9, float64(paid)/100)
				set(10, float64(d.TotalCents-paid)/100)
				if od := domain.DaysOverdue(d.DueDate, today, d.TotalCents-paid); od != nil && d.Status != "cancelled" {
					set(12, *od)
				}
			}
		}
		row++
	}
	f.SetCellStyle(sheet, "H2", "J"+itoa(row), money)
	widths := []float64{16, 11, 26, 14, 34, 12, 16, 14, 14, 14, 15, 13, 15}
	for i, w := range widths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, w)
	}
	f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	f.AutoFilter(sheet, "A1:M"+itoa(row-1), nil)
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func strPtr(s string) *string { return &s }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
