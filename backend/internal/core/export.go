package core

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"jobcard/internal/domain"
	"jobcard/internal/export"
)

// ExportResult is a rendered document plus the copy saved in the client folder.
type ExportResult struct {
	Data      []byte
	Mime      string
	Filename  string // suggested download name, e.g. INV-2026-001.pdf
	SavedPath string // copy written to the client folder
}

var filePrefix = map[string]string{"invoice": "Invoice", "quotation": "Quotation", "jobcard": "JobCard"}

// ExportCtx gathers everything the templates need for one document.
func (s *Service) ExportCtx(d domain.Doc) (export.Ctx, error) {
	cl, err := s.GetClient(d.ClientID)
	if err != nil {
		return export.Ctx{}, err
	}
	st, err := s.GetSettings()
	if err != nil {
		return export.Ctx{}, err
	}
	c := export.Ctx{Doc: d, Client: cl, Settings: st, Today: s.Today()}
	all, err := s.AllDocuments()
	if err != nil {
		return c, err
	}
	for _, o := range all {
		if o.Number == nil {
			continue
		}
		if d.QuotationID != nil && o.ID == *d.QuotationID {
			c.QuotationNumber = *o.Number
		}
		if d.JobCardID != nil && o.ID == *d.JobCardID {
			c.JobCardNumber = *o.Number
		}
		if d.Type == "jobcard" && o.JobCardID != nil && *o.JobCardID == d.ID {
			c.Related = append(c.Related, *o.Number)
		}
	}
	return c, nil
}

// ExportDocument renders an issued document from its template as docx or pdf and saves a copy
// into the client's folder (overwriting the previous copy of the same name).
func (s *Service) ExportDocument(id, format string, tpls export.Templates) (*ExportResult, error) {
	d, err := s.GetDocument(id)
	if err != nil {
		return nil, err
	}
	if d.Number == nil {
		return nil, appErr(http.StatusBadRequest, "DRAFT_CANNOT_EXPORT", "Drafts cannot be exported. Only issued documents can be exported.")
	}
	if format != "pdf" && format != "docx" {
		return nil, &AppError{Status: 400, Code: "VALIDATION_FAILED", Message: "format must be pdf or docx", Fields: map[string]string{"format": "Must be pdf or docx"}}
	}
	if format == "pdf" && !export.DetectCapabilities().PDF {
		return nil, appErr(http.StatusNotImplemented, "FORMAT_NOT_SUPPORTED", "PDF export needs Microsoft Word on the host machine")
	}
	ctx, err := s.ExportCtx(d)
	if err != nil {
		return nil, err
	}
	tpl, err := tpls.Load(d.Type)
	if err != nil {
		log.Printf("template load: %v", err)
		return nil, appErr(http.StatusInternalServerError, "EXPORT_FAILED", "Could not read the "+d.Type+" template")
	}
	data, warn, err := export.Render(tpl, ctx)
	if err != nil {
		log.Printf("template %s: %v", tpls.File(d.Type), err)
		return nil, appErr(http.StatusInternalServerError, "EXPORT_FAILED",
			fmt.Sprintf("The %s template cannot be used (%v). Fix it, or delete %s to restore the default.", d.Type, err, tpls.File(d.Type)))
	}
	if len(warn) > 0 {
		log.Printf("template %s has unknown placeholders: %v", tpls.File(d.Type), warn)
	}
	mime := "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	if format == "pdf" {
		if data, err = export.DocxToPDF(data); err != nil {
			log.Printf("pdf export failed: %v", err)
			return nil, appErr(http.StatusInternalServerError, "EXPORT_FAILED", "PDF conversion failed")
		}
		mime = "application/pdf"
	}
	folder, err := s.EnsureClientFolder(d.ClientID)
	if err != nil {
		log.Printf("client folder: %v", err)
		return nil, appErr(http.StatusInternalServerError, "EXPORT_FAILED", "Could not create the client folder")
	}
	saved := filepath.Join(folder, fmt.Sprintf("%s_%s_%s.%s", filePrefix[d.Type], *d.Number, strings.ReplaceAll(SafeName(ctx.Client.Name), " ", "_"), format))
	if err := os.WriteFile(saved, data, 0o644); err != nil {
		log.Printf("save copy: %v", err)
		return nil, appErr(http.StatusInternalServerError, "EXPORT_FAILED", "Could not save a copy to the client folder")
	}
	return &ExportResult{Data: data, Mime: mime, Filename: *d.Number + "." + format, SavedPath: saved}, nil
}

// RevealFolder opens the document's client folder in the file manager and returns its path.
func (s *Service) RevealFolder(id string) (string, error) {
	d, err := s.GetDocument(id)
	if err != nil {
		return "", err
	}
	if !export.DetectCapabilities().RevealInFolder {
		return "", appErr(http.StatusNotImplemented, "NOT_SUPPORTED", "Reveal in folder is not supported on this machine")
	}
	folder, err := s.EnsureClientFolder(d.ClientID)
	if err != nil {
		return "", err
	}
	if err := export.Reveal(folder); err != nil {
		log.Printf("reveal: %v", err)
		return "", appErr(http.StatusInternalServerError, "INTERNAL_ERROR", "Could not open the folder")
	}
	return folder, nil
}

// RegisterSpreadsheet builds the register .xlsx and a suggested file name.
func (s *Service) RegisterSpreadsheet() ([]byte, string, error) {
	docs, err := s.AllDocuments()
	if err != nil {
		return nil, "", err
	}
	clients, err := s.ListClients()
	if err != nil {
		return nil, "", err
	}
	today := s.Today()
	data, err := export.RegisterXLSX(docs, clients, today)
	if err != nil {
		log.Printf("register xlsx: %v", err)
		return nil, "", appErr(http.StatusInternalServerError, "INTERNAL_ERROR", "Spreadsheet generation failed")
	}
	return data, fmt.Sprintf("register-%s.xlsx", today[:4]), nil
}
