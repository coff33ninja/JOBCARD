// Package mcp exposes the JobCard service over the Model Context Protocol.
package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jobcard/internal/core"
	"jobcard/internal/domain"
	"jobcard/internal/export"
)

type tool struct {
	Name        string
	Description string
	Schema      map[string]any
	Write       bool // recorded in the audit log
	Run         func(args json.RawMessage) (any, error)
}

// ---------- JSON schema helpers ----------

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
func num(desc string) map[string]any { return map[string]any{"type": "number", "description": desc} }
func integer(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}
func boolean(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}
func enum(desc string, vals ...string) map[string]any {
	return map[string]any{"type": "string", "description": desc, "enum": vals}
}
func arr(items map[string]any, desc string) map[string]any {
	return map[string]any{"type": "array", "description": desc, "items": items}
}
func obj(props map[string]any, required ...string) map[string]any {
	m := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		m["required"] = required
	}
	return m
}

var confirmProp = boolean("Must be true. Only set it after the user explicitly approved this exact action.")

var draftProps = func() map[string]any {
	return map[string]any{
		"reference":   str("Short job/device reference, e.g. 'Dell laptop screen replacement'"),
		"scope":       str("Scope of service"),
		"jobCardId":   str("ID (not number) of the related job card, or null to clear"),
		"quotationId": str("ID of the related quotation, or null to clear"),
		"workItems":   arr(obj(map[string]any{"title": str("Item"), "detail": str("What was or will be done")}, "title"), "Work done / proposed"),
		"lines": arr(obj(map[string]any{
			"description":    str("Line description"),
			"qty":            num("Quantity, greater than 0"),
			"unitPriceCents": integer("Unit price in cents, excluding tax"),
		}, "description", "qty", "unitPriceCents"), "Line items. Totals are computed by the server."),
		"depositCents": integer("Deposit in cents (quotations), or null"),
		"terms":        str("Payment terms text"),
		"exclusions":   str("Exclusions / assumptions (quotations)"),
		"turnaround":   str("Estimated turnaround (quotations)"),
		"notes":        str("Notes and recommendations"),
	}
}

func withProps(base map[string]any, extra map[string]any) map[string]any {
	for k, v := range extra {
		base[k] = v
	}
	return base
}

func clientProps() map[string]any {
	return map[string]any{
		"name": str("Client or business name"), "contact": str("Contact person"), "phone": str("Phone"),
		"email": str("Email"), "address": str("Billing address"), "notes": str("Notes"),
	}
}

// ---------- tool table (names match mcpTool in src/api/endpoints.ts) ----------

func decode(args json.RawMessage, v any) error {
	if len(args) == 0 {
		args = json.RawMessage("{}")
	}
	if err := json.Unmarshal(args, v); err != nil {
		return &core.AppError{Status: 400, Code: "VALIDATION_FAILED", Message: "invalid arguments: " + err.Error()}
	}
	return nil
}

func needConfirm(c bool, what string) error {
	if !c {
		return &core.AppError{Status: 400, Code: "CONFIRMATION_REQUIRED",
			Message: what + " is permanent. Ask the user to approve this exact action, then call again with confirm=true."}
	}
	return nil
}

type idArgs struct {
	ID string `json:"id"`
}

func (s *Server) tools() []tool {
	svc := s.Svc
	return []tool{
		{Name: "list_clients", Description: "List active clients. Archived clients are available with list_archived_clients.", Schema: obj(map[string]any{}),
			Run: func(_ json.RawMessage) (any, error) { return svc.ListClients() }},
		{Name: "list_archived_clients", Description: "List archived clients, including those with existing documents.", Schema: obj(map[string]any{}),
			Run: func(_ json.RawMessage) (any, error) { return svc.ListArchivedClients() }},
		{Name: "create_client", Description: "Create a client. Only name is required.", Write: true,
			Schema: obj(clientProps(), "name"),
			Run: func(a json.RawMessage) (any, error) {
				var in core.ClientInput
				if err := decode(a, &in); err != nil {
					return nil, err
				}
				return svc.CreateClient(in)
			}},
		{Name: "update_client", Description: "Update a client's details. Only fields you send change.", Write: true,
			Schema: obj(withProps(clientProps(), map[string]any{"id": str("Client ID")}), "id"),
			Run: func(a json.RawMessage) (any, error) {
				var id idArgs
				var in core.ClientInput
				if err := decode(a, &id); err != nil {
					return nil, err
				}
				if err := decode(a, &in); err != nil {
					return nil, err
				}
				return svc.UpdateClient(id.ID, in)
			}},
		{Name: "archive_client", Description: "Archive a client without deleting its documents. This can be undone with restore_client.", Write: true,
			Schema: obj(map[string]any{"id": str("Client ID")}, "id"),
			Run: func(a json.RawMessage) (any, error) {
				var id idArgs
				if err := decode(a, &id); err != nil {
					return nil, err
				}
				return svc.ArchiveClient(id.ID)
			}},
		{Name: "restore_client", Description: "Restore an archived client so it appears in active lists and can be used for new documents.", Write: true,
			Schema: obj(map[string]any{"id": str("Client ID")}, "id"),
			Run: func(a json.RawMessage) (any, error) {
				var id idArgs
				if err := decode(a, &id); err != nil {
					return nil, err
				}
				return svc.RestoreClient(id.ID)
			}},
		{Name: "list_documents", Description: "List quotations, invoices and job cards, newest first. Status filters use the derived status (e.g. overdue).",
			Schema: obj(map[string]any{
				"type":     enum("Document type", "quotation", "invoice", "jobcard"),
				"status":   str("Status, e.g. draft, sent, partially_paid, paid, overdue, cancelled, accepted, open"),
				"clientId": str("Client ID"), "q": str("Text search in number, reference, scope"),
				"from": str("Issue date from, YYYY-MM-DD"), "to": str("Issue date to, YYYY-MM-DD"),
				"limit": integer("Max results (default 50)"),
			}),
			Run: func(a json.RawMessage) (any, error) {
				var raw struct {
					Type, Status, ClientID, Q, From, To string
					Limit                               int
				}
				if err := decode(a, &raw); err != nil {
					return nil, err
				}
				docs, err := svc.ListDocuments(core.DocFilters{Type: raw.Type, Status: raw.Status, ClientID: raw.ClientID, Q: raw.Q, From: raw.From, To: raw.To})
				if err != nil {
					return nil, err
				}
				lim := raw.Limit
				if lim <= 0 {
					lim = 50
				}
				if len(docs) > lim {
					docs = docs[:lim]
				}
				return docs, nil
			}},
		{Name: "get_document", Description: "Get one document with its lines, payments and client folder path.",
			Schema: obj(map[string]any{"id": str("Document ID")}, "id"),
			Run: func(a json.RawMessage) (any, error) {
				var id idArgs
				if err := decode(a, &id); err != nil {
					return nil, err
				}
				return svc.GetDocument(id.ID)
			}},
		{Name: "create_draft", Description: "Create a draft quotation, invoice or job card. Drafts have no number yet and can be edited.", Write: true,
			Schema: obj(withProps(draftProps(), map[string]any{
				"type":     enum("Document type", "quotation", "invoice", "jobcard"),
				"clientId": str("Client ID (use list_clients or create_client first)"),
			}), "type", "clientId", "reference"),
			Run: func(a json.RawMessage) (any, error) {
				var in core.DraftInput
				if err := decode(a, &in); err != nil {
					return nil, err
				}
				return svc.CreateDraft(in)
			}},
		{Name: "update_draft", Description: "Edit a draft. Only fields you send change; line items are replaced as a whole. Issued documents cannot be edited.", Write: true,
			Schema: obj(withProps(draftProps(), map[string]any{"id": str("Document ID"), "clientId": str("Client ID")}), "id"),
			Run: func(a json.RawMessage) (any, error) {
				var id idArgs
				var in core.DraftInput
				if err := decode(a, &id); err != nil {
					return nil, err
				}
				if err := decode(a, &in); err != nil {
					return nil, err
				}
				return svc.UpdateDraft(id.ID, in)
			}},
		{Name: "issue_document", Description: "PERMANENT. Issue a draft: assigns its permanent number (JC-/QUO-/INV-), sets dates, locks it. Requires user approval and confirm=true.", Write: true,
			Schema: obj(map[string]any{"id": str("Document ID"), "issueDate": str("YYYY-MM-DD, defaults to today"), "confirm": confirmProp}, "id", "confirm"),
			Run: func(a json.RawMessage) (any, error) {
				var in struct {
					ID        string `json:"id"`
					IssueDate string `json:"issueDate"`
					Confirm   bool   `json:"confirm"`
				}
				if err := decode(a, &in); err != nil {
					return nil, err
				}
				if err := needConfirm(in.Confirm, "Issuing a document"); err != nil {
					return nil, err
				}
				d, err := svc.IssueDocument(in.ID, in.IssueDate)
				if err == nil {
					_, _ = svc.EnsureClientFolder(d.ClientID)
				}
				return d, err
			}},
		{Name: "set_follow_up", Description: "Set or clear a reminder date on an issued, non-cancelled invoice.", Write: true,
			Schema: obj(map[string]any{
				"id":           str("Invoice ID"),
				"followUpDate": map[string]any{"type": []string{"string", "null"}, "description": "ISO date YYYY-MM-DD, or null to clear the reminder"},
			}, "id", "followUpDate"),
			Run: func(a json.RawMessage) (any, error) {
				var in struct {
					ID           string  `json:"id"`
					FollowUpDate *string `json:"followUpDate"`
				}
				if err := decode(a, &in); err != nil {
					return nil, err
				}
				return svc.SetFollowUpDate(in.ID, in.FollowUpDate)
			}},
		{Name: "add_payment", Description: "PERMANENT. Record a payment against an issued invoice. Payments cannot be deleted. Requires user approval and confirm=true.", Write: true,
			Schema: obj(map[string]any{
				"id": str("Invoice ID"), "date": str("Payment date YYYY-MM-DD"), "amountCents": integer("Amount in cents, greater than 0"),
				"method": str("e.g. EFT, cash"), "reference": str("Bank or transaction reference"), "confirm": confirmProp,
			}, "id", "date", "amountCents", "confirm"),
			Run: func(a json.RawMessage) (any, error) {
				var in struct {
					ID      string `json:"id"`
					Confirm bool   `json:"confirm"`
					core.PaymentInput
				}
				if err := decode(a, &in); err != nil {
					return nil, err
				}
				if err := needConfirm(in.Confirm, "Recording a payment"); err != nil {
					return nil, err
				}
				return svc.AddPayment(in.ID, in.PaymentInput)
			}},
		{Name: "update_status", Description: "Change a document's status (quotation: accepted/declined/cancelled; invoice: cancelled; job card: open, in_progress, completed, invoiced, closed). Cancelling is permanent and needs confirm=true.", Write: true,
			Schema: obj(map[string]any{"id": str("Document ID"), "status": str("New status"), "confirm": boolean("Required (true) when cancelling, after user approval")}, "id", "status"),
			Run: func(a json.RawMessage) (any, error) {
				var in struct {
					ID      string `json:"id"`
					Status  string `json:"status"`
					Confirm bool   `json:"confirm"`
				}
				if err := decode(a, &in); err != nil {
					return nil, err
				}
				if in.Status == "cancelled" {
					if err := needConfirm(in.Confirm, "Cancelling a document"); err != nil {
						return nil, err
					}
				}
				return svc.SetStatus(in.ID, in.Status)
			}},
		{Name: "convert_quotation", Description: "PERMANENT. Create a draft invoice from an issued quotation and mark the quotation converted. Requires user approval and confirm=true.", Write: true,
			Schema: obj(map[string]any{"id": str("Quotation ID"), "confirm": confirmProp}, "id", "confirm"),
			Run: func(a json.RawMessage) (any, error) {
				var in struct {
					ID      string `json:"id"`
					Confirm bool   `json:"confirm"`
				}
				if err := decode(a, &in); err != nil {
					return nil, err
				}
				if err := needConfirm(in.Confirm, "Converting a quotation"); err != nil {
					return nil, err
				}
				return svc.ConvertQuotation(in.ID)
			}},
		{Name: "export_document", Description: "Render an issued document from its template and save it into the client's folder. Returns the saved path (not the file contents).", Write: true,
			Schema: obj(map[string]any{"id": str("Document ID"), "format": enum("File format", "pdf", "docx")}, "id", "format"),
			Run: func(a json.RawMessage) (any, error) {
				var in struct{ ID, Format string }
				if err := decode(a, &in); err != nil {
					return nil, err
				}
				res, err := svc.ExportDocument(in.ID, in.Format, s.Templates)
				if err != nil {
					return nil, err
				}
				return map[string]any{"savedPath": res.SavedPath, "filename": res.Filename, "bytes": len(res.Data)}, nil
			}},
		{Name: "reveal_folder", Description: "Open the document's client folder in Windows Explorer on the user's machine.",
			Schema: obj(map[string]any{"id": str("Document ID")}, "id"),
			Run: func(a json.RawMessage) (any, error) {
				var id idArgs
				if err := decode(a, &id); err != nil {
					return nil, err
				}
				p, err := svc.RevealFolder(id.ID)
				return map[string]any{"success": err == nil, "path": p}, err
			}},
		{Name: "get_dashboard", Description: "Totals invoiced/paid/outstanding/overdue, status counts, ageing, monthly figures and the next free numbers for a year.",
			Schema: obj(map[string]any{"year": integer("Four digit year, defaults to the current year")}),
			Run: func(a json.RawMessage) (any, error) {
				var in struct{ Year int }
				if err := decode(a, &in); err != nil {
					return nil, err
				}
				return svc.Dashboard(in.Year)
			}},
		{Name: "get_settings", Description: "Read business and bank details, tax rate and default terms (read-only here; the user edits them in the app).",
			Schema: obj(map[string]any{}),
			Run:    func(_ json.RawMessage) (any, error) { return svc.GetSettings() }},
		{Name: "get_storage", Description: "Where the database and client folders live.", Schema: obj(map[string]any{}),
			Run: func(_ json.RawMessage) (any, error) { return svc.Storage(), nil }},
		{Name: "get_capabilities", Description: "Which export formats and actions this machine supports (docx, pdf, reveal in folder).", Schema: obj(map[string]any{}),
			Run: func(_ json.RawMessage) (any, error) { return capabilities(), nil }},
		{Name: "export_register_spreadsheet", Description: "Write the full register (invoices, quotations, job cards) to an .xlsx file and return its path.", Write: true,
			Schema: obj(map[string]any{}),
			Run: func(_ json.RawMessage) (any, error) {
				data, name, err := svc.RegisterSpreadsheet()
				if err != nil {
					return nil, err
				}
				dir := filepath.Join(svc.DataDir, "Exports")
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return nil, err
				}
				p := filepath.Join(dir, name)
				if err := os.WriteFile(p, data, 0o644); err != nil {
					return nil, err
				}
				return map[string]any{"savedPath": p, "bytes": len(data)}, nil
			}},
		{Name: "get_audit_log", Description: "Recent changes made through the API and MCP (newest first). Use it to tell the user what was changed.",
			Schema: obj(map[string]any{"limit": integer("Max entries (default 50, max 500)")}),
			Run: func(a json.RawMessage) (any, error) {
				var in struct{ Limit int }
				if err := decode(a, &in); err != nil {
					return nil, err
				}
				return svc.ListAudit(in.Limit)
			}},
	}
}

func capabilities() domain.Capabilities { return export.DetectCapabilities() }

// call runs a tool and builds the MCP result. found=false means the tool name is unknown.
func (s *Server) call(name string, args json.RawMessage) (map[string]any, bool) {
	for _, t := range s.tools() {
		if t.Name != name {
			continue
		}
		val, err := t.Run(args)
		result := func(text string, isErr bool) map[string]any {
			return map[string]any{"content": []map[string]any{{"type": "text", "text": text}}, "isError": isErr}
		}
		if t.Write {
			outcome := "ok"
			if err != nil {
				outcome = "error: " + err.Error()
			}
			s.Svc.Audit("mcp", t.Name, "", fmt.Sprintf("%s | %s", strings.TrimSpace(string(args)), outcome))
		}
		if err != nil {
			var ae *core.AppError
			body := map[string]any{"code": "INTERNAL_ERROR", "message": "Internal error"}
			if errors.As(err, &ae) {
				body = map[string]any{"code": ae.Code, "message": ae.Message}
				if len(ae.Fields) > 0 {
					body["fields"] = ae.Fields
				}
			}
			b, _ := json.Marshal(map[string]any{"error": body})
			return result(string(b), true), true
		}
		b, mErr := json.Marshal(val)
		if mErr != nil {
			return result(`{"error":{"code":"INTERNAL_ERROR","message":"could not encode result"}}`, true), true
		}
		return result(string(b), false), true
	}
	return nil, false
}
