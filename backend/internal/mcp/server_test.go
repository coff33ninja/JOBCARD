package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"jobcard/internal/core"
	"jobcard/internal/export"
)

func newServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	svc, err := core.NewService(dir, filepath.Join(dir, "Clients"), func() string { return "2026-11-08" })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { svc.Close() })
	return &Server{Svc: svc, Templates: export.Templates{Dir: filepath.Join(dir, "Templates")}, Version: "test"}
}

// rpc sends newline-delimited requests and returns the decoded response lines.
func rpc(t *testing.T, s *Server, lines ...string) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := s.Serve(strings.NewReader(strings.Join(lines, "\n")+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	var res []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("response is not JSON: %q", l)
		}
		res = append(res, m)
	}
	return res
}

func call(t *testing.T, s *Server, id int, name string, args any) (text string, isErr bool) {
	t.Helper()
	b, _ := json.Marshal(args)
	r := rpc(t, s, fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, id, name, b))
	if len(r) != 1 {
		t.Fatalf("want 1 response, got %d", len(r))
	}
	res, _ := r[0]["result"].(map[string]any)
	if res == nil {
		t.Fatalf("no result: %v", r[0])
	}
	content := res["content"].([]any)[0].(map[string]any)
	return content["text"].(string), res["isError"] == true
}

func TestHandshakeAndNotifications(t *testing.T) {
	s := newServer(t)
	r := rpc(t, s,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"x","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":3,"method":"nope"}`,
		`not json`)
	if len(r) != 4 { // initialize, ping, unknown method, parse error. The notification gets no reply.
		t.Fatalf("responses: %d %v", len(r), r)
	}
	init := r[0]["result"].(map[string]any)
	if init["protocolVersion"] != "2025-03-26" || init["serverInfo"].(map[string]any)["name"] != "jobcard" {
		t.Fatalf("initialize: %v", init)
	}
	if r[2]["error"].(map[string]any)["code"].(float64) != -32601 || r[3]["error"].(map[string]any)["code"].(float64) != -32700 {
		t.Fatalf("errors: %v %v", r[2], r[3])
	}
}

func TestToolsListMatchesRegistryAndHidesSettingsWrite(t *testing.T) {
	s := newServer(t)
	r := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	tools := r[0]["result"].(map[string]any)["tools"].([]any)
	names := map[string]bool{}
	for _, x := range tools {
		m := x.(map[string]any)
		names[m["name"].(string)] = true
		if m["inputSchema"].(map[string]any)["type"] != "object" || m["description"] == "" {
			t.Errorf("bad tool %v", m["name"])
		}
	}
	for _, want := range []string{"list_clients", "list_archived_clients", "find_duplicate_clients", "create_client", "update_client", "archive_client", "restore_client", "list_documents", "get_document", "create_draft", "update_draft",
		"issue_document", "set_follow_up", "add_payment", "update_status", "convert_quotation", "export_document", "reveal_folder", "get_dashboard",
		"get_settings", "get_storage", "get_capabilities", "export_register_spreadsheet", "get_audit_log"} {
		if !names[want] {
			t.Errorf("missing tool %s", want)
		}
	}
	if names["update_settings"] {
		t.Error("update_settings must not be exposed to the AI")
	}
	if rpc(t, s, `{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"nope","arguments":{}}}`)[0]["error"] == nil {
		t.Error("unknown tool must be a protocol error")
	}
}

func TestFindDuplicateClientsToolReturnsPossibleMatches(t *testing.T) {
	s := newServer(t)
	name, contact := "Acme Holdings", "Jane Doe"
	existing, err := s.Svc.CreateClient(core.ClientInput{
		Name:    core.Opt[string]{Set: true, Val: &name},
		Contact: core.Opt[string]{Set: true, Val: &contact},
	})
	if err != nil {
		t.Fatal(err)
	}

	text, isErr := call(t, s, 2, "find_duplicate_clients", map[string]any{"name": "Jane Doe"})
	if isErr {
		t.Fatalf("find_duplicate_clients returned an error: %s", text)
	}
	var matches []core.ClientDuplicateMatch
	if err := json.Unmarshal([]byte(text), &matches); err != nil {
		t.Fatalf("decode duplicate matches: %v; text: %s", err, text)
	}
	if len(matches) != 1 || matches[0].Client.ID != existing.ID {
		t.Fatalf("duplicate matches = %+v, want client %s", matches, existing.ID)
	}
}

func TestLifecycleToolsAreRegisteredAndRequireConfirmation(t *testing.T) {
	s := newServer(t)
	stringPtr := func(value string) *string { return &value }
	response := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	tools := response[0]["result"].(map[string]any)["tools"].([]any)
	names := map[string]bool{}
	for _, entry := range tools {
		names[entry.(map[string]any)["name"].(string)] = true
	}
	for _, name := range []string{
		"list_archived_documents", "archive_draft", "restore_draft", "delete_draft",
		"create_replacement_quotation", "update_quotation_revision", "list_quotation_revisions",
	} {
		if !names[name] {
			t.Errorf("missing lifecycle tool %s", name)
		}
	}

	client, err := s.Svc.CreateClient(core.ClientInput{Name: core.Opt[string]{Set: true, Val: stringPtr("Lifecycle")}})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := s.Svc.CreateDraft(core.DraftInput{
		Type:      core.Opt[string]{Set: true, Val: stringPtr("jobcard")},
		ClientID:  core.Opt[string]{Set: true, Val: stringPtr(client.ID)},
		Reference: core.Opt[string]{Set: true, Val: stringPtr("Delete safely")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, isErr := call(t, s, 2, "delete_draft", map[string]any{"id": doc.ID, "confirm": false}); !isErr {
		t.Fatal("delete_draft must reject calls without explicit confirmation")
	}
	if _, isErr := call(t, s, 3, "delete_draft", map[string]any{"id": doc.ID, "confirm": true}); isErr {
		t.Fatal("confirmed delete_draft should succeed")
	}
}

func TestDraftToolsExposeAndPersistReportFields(t *testing.T) {
	s := newServer(t)
	response := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	tools := response[0]["result"].(map[string]any)["tools"].([]any)
	for _, entry := range tools {
		tool := entry.(map[string]any)
		name := tool["name"].(string)
		if name != "create_draft" && name != "update_draft" {
			continue
		}
		schema := tool["inputSchema"].(map[string]any)
		properties := schema["properties"].(map[string]any)
		for _, field := range []string{"intakeItems", "diagnosticSections", "authorization"} {
			if _, ok := properties[field]; !ok {
				t.Errorf("%s input schema is missing %s", name, field)
			}
		}
		lines := properties["lines"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
		if _, ok := lines["detail"]; !ok {
			t.Errorf("%s line schema is missing detail", name)
		}
		if name == "create_draft" {
			if _, ok := properties["newClient"]; !ok {
				t.Error("create_draft input schema is missing newClient")
			}
			if _, ok := tool["inputSchema"].(map[string]any)["oneOf"]; !ok {
				t.Error("create_draft input schema must require exactly one client option")
			}
		}
	}

	text, isErr := call(t, s, 2, "create_draft", map[string]any{
		"type": "invoice", "reference": "Laptop",
		"newClient": map[string]string{
			"name": "Report Client", "contact": "Example Contact", "phone": "000",
		},
		"intakeItems":        []map[string]string{{"label": "Charger", "value": "Received"}},
		"diagnosticSections": []map[string]string{{"title": "Battery test", "status": "FAILED", "details": "No charge"}},
		"authorization":      map[string]string{"technicianSignature": "Tech", "technicianDate": "2026-11-10", "clientSignature": "Client", "clientDate": "2026-11-10"},
		"lines":              []map[string]any{{"description": "Battery", "detail": "OEM replacement", "qty": 1, "unitPriceCents": 1000}},
	})
	if isErr {
		t.Fatal(text)
	}
	var document struct {
		ID       string `json:"id"`
		ClientID string `json:"clientId"`
	}
	if err := json.Unmarshal([]byte(text), &document); err != nil {
		t.Fatal(err)
	}
	if document.ClientID == "" {
		t.Fatalf("newly created draft has no linked client: %s", text)
	}

	text, isErr = call(t, s, 3, "update_draft", map[string]any{
		"id":          document.ID,
		"intakeItems": []map[string]string{{"label": "Power adapter", "value": "Included"}},
	})
	if isErr {
		t.Fatal(text)
	}
	got, err := s.Svc.GetDocument(document.ID)
	if err != nil {
		t.Fatal(err)
	}
	client, err := s.Svc.GetClient(got.ClientID)
	if err != nil || client.Name != "Report Client" || client.Contact != "Example Contact" || client.Phone != "000" {
		t.Fatalf("inline client was not created and linked: %+v, %v", client, err)
	}
	if len(got.IntakeItems) != 1 || got.IntakeItems[0].Label != "Power adapter" ||
		len(got.DiagnosticSections) != 1 || got.DiagnosticSections[0].Details != "No charge" ||
		got.Authorization.TechnicianSignature != "Tech" || len(got.Lines) != 1 || got.Lines[0].Detail != "OEM replacement" {
		t.Fatalf("MCP report fields were not persisted as expected: %+v", got)
	}

	if text, isErr = call(t, s, 4, "create_draft", map[string]any{
		"type": "invoice", "clientId": got.ClientID, "newClient": map[string]string{"name": "Duplicate"},
		"reference": "Conflicting client choices",
	}); !isErr || !strings.Contains(text, "Provide either clientId or newClient") {
		t.Fatalf("supplying both client options should fail: %s, error %v", text, isErr)
	}

	if text, isErr = call(t, s, 5, "create_draft", map[string]any{
		"type": "invoice", "newClient": map[string]string{"name": "Rolled Back Client"},
	}); !isErr {
		t.Fatalf("draft without reference should fail: %s", text)
	}
	clients, err := s.Svc.ListClients()
	if err != nil {
		t.Fatal(err)
	}
	if len(clients) != 1 || clients[0].ID != got.ClientID {
		t.Fatalf("invalid draft left an orphan client: %+v", clients)
	}
}

func TestArchivedClientsCanBeRestored(t *testing.T) {
	s := newServer(t)
	text, isErr := call(t, s, 1, "create_client", map[string]any{"name": "Archive Test"})
	if isErr {
		t.Fatal(text)
	}
	var client struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(text), &client); err != nil {
		t.Fatal(err)
	}

	text, isErr = call(t, s, 2, "create_draft", map[string]any{
		"type": "jobcard", "clientId": client.ID, "reference": "Archive history",
	})
	if isErr {
		t.Fatal(text)
	}
	var doc struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		t.Fatal(err)
	}
	if text, isErr = call(t, s, 3, "archive_client", map[string]any{"id": client.ID}); isErr {
		t.Fatal(text)
	}
	if text, isErr = call(t, s, 4, "list_clients", map[string]any{}); isErr || text != "[]" {
		t.Fatalf("active clients = %s, error %v", text, isErr)
	}
	if text, isErr = call(t, s, 5, "list_archived_clients", map[string]any{}); isErr || !strings.Contains(text, client.ID) {
		t.Fatalf("archived clients = %s, error %v", text, isErr)
	}
	if text, isErr = call(t, s, 6, "get_document", map[string]any{"id": doc.ID}); isErr || !strings.Contains(text, client.ID) {
		t.Fatalf("historical document = %s, error %v", text, isErr)
	}
	if text, isErr = call(t, s, 7, "create_draft", map[string]any{
		"type": "jobcard", "clientId": client.ID, "reference": "New work",
	}); !isErr || !strings.Contains(text, "VALIDATION_FAILED") {
		t.Fatalf("draft for archived client = %s, error %v", text, isErr)
	}
	if text, isErr = call(t, s, 8, "restore_client", map[string]any{"id": client.ID}); isErr {
		t.Fatal(text)
	}
	if text, isErr = call(t, s, 9, "list_clients", map[string]any{}); isErr || !strings.Contains(text, client.ID) {
		t.Fatalf("restored clients = %s, error %v", text, isErr)
	}
}

func TestDraftThenConfirmedIssueAndPayment(t *testing.T) {
	s := newServer(t)
	text, isErr := call(t, s, 1, "create_client", map[string]any{"name": "Acme", "email": "a@acme.test"})
	if isErr {
		t.Fatal(text)
	}
	var client struct{ ID string }
	json.Unmarshal([]byte(text), &client)

	text, isErr = call(t, s, 2, "create_draft", map[string]any{
		"type": "invoice", "clientId": client.ID, "reference": "Router setup",
		"lines": []map[string]any{{"description": "Labour", "qty": 2, "unitPriceCents": 45000, "totalCents": 1}},
	})
	if isErr {
		t.Fatal(text)
	}
	var d struct {
		ID         string
		TotalCents int64
		Number     *string
	}
	json.Unmarshal([]byte(text), &d)
	if d.TotalCents != 90000 || d.Number != nil {
		t.Fatalf("server must compute money and leave drafts unnumbered: %s", text)
	}

	// issuing without approval is refused and changes nothing
	text, isErr = call(t, s, 3, "issue_document", map[string]any{"id": d.ID})
	if !isErr || !strings.Contains(text, "CONFIRMATION_REQUIRED") {
		t.Fatalf("issue without confirm: %v %s", isErr, text)
	}
	if got, _ := s.Svc.GetDocument(d.ID); got.Number != nil {
		t.Fatal("document was issued without confirmation")
	}
	text, isErr = call(t, s, 4, "issue_document", map[string]any{"id": d.ID, "confirm": true})
	if isErr || !strings.Contains(text, "INV-2026-001") {
		t.Fatalf("issue: %v %s", isErr, text)
	}
	text, isErr = call(t, s, 5, "add_payment", map[string]any{"id": d.ID, "date": "2026-11-08", "amountCents": 10000})
	if !isErr || !strings.Contains(text, "CONFIRMATION_REQUIRED") {
		t.Fatalf("payment without confirm: %s", text)
	}
	text, isErr = call(t, s, 6, "add_payment", map[string]any{"id": d.ID, "date": "2026-11-08", "amountCents": 10000, "confirm": true, "reference": "EFT 1"})
	if isErr || !strings.Contains(text, "partially_paid") {
		t.Fatalf("payment: %v %s", isErr, text)
	}
	// app errors reach the model with their code and message so it can correct itself
	text, isErr = call(t, s, 7, "add_payment", map[string]any{"id": d.ID, "date": "2026-11-08", "amountCents": 99999999, "confirm": true})
	if !isErr || !strings.Contains(text, "OVERPAYMENT_REJECTED") {
		t.Fatalf("overpayment: %s", text)
	}
	text, isErr = call(t, s, 8, "update_status", map[string]any{"id": d.ID, "status": "cancelled"})
	if !isErr || !strings.Contains(text, "CONFIRMATION_REQUIRED") {
		t.Fatalf("cancel without confirm: %s", text)
	}

	// everything written through MCP is in the audit log
	text, isErr = call(t, s, 9, "get_audit_log", map[string]any{})
	if isErr || !strings.Contains(text, `"source":"mcp"`) || !strings.Contains(text, "issue_document") || !strings.Contains(text, "create_draft") {
		t.Fatalf("audit: %s", text)
	}
}

func TestReadToolsAndValidation(t *testing.T) {
	s := newServer(t)
	for i, name := range []string{"list_clients", "list_documents", "get_dashboard", "get_settings", "get_storage", "get_capabilities"} {
		if text, isErr := call(t, s, i+1, name, map[string]any{}); isErr {
			t.Errorf("%s failed: %s", name, text)
		}
	}
	if text, isErr := call(t, s, 20, "create_client", map[string]any{"name": " "}); !isErr || !strings.Contains(text, "VALIDATION_FAILED") {
		t.Errorf("empty name: %s", text)
	}
	if text, isErr := call(t, s, 21, "create_client", "not an object"); !isErr {
		t.Errorf("bad arguments should be an error result: %s", text)
	}
	if text, isErr := call(t, s, 22, "get_document", map[string]any{"id": "missing"}); !isErr || !strings.Contains(text, "NOT_FOUND") {
		t.Errorf("missing doc: %s", text)
	}
}
