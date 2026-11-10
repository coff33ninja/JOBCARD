package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientArchiveAndRestoreThroughAPI(t *testing.T) {
	ts := testServer(t, "")
	base := ts.URL + "/api/v1"

	preflight := do(t, http.MethodOptions, base+"/clients/cli-test", "", map[string]string{
		"Origin": "http://localhost:3000", "Access-Control-Request-Method": http.MethodDelete,
	})
	if preflight.StatusCode != http.StatusNoContent || !strings.Contains(preflight.Header.Get("Access-Control-Allow-Methods"), "DELETE") {
		t.Fatalf("DELETE preflight: status %d, methods %q", preflight.StatusCode, preflight.Header.Get("Access-Control-Allow-Methods"))
	}

	clientResponse := do(t, http.MethodPost, base+"/clients", `{"name":"Archive Test"}`, jsonHdr)
	var client struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(clientResponse.Body).Decode(&client); err != nil {
		t.Fatal(err)
	}
	if clientResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create client: status %d", clientResponse.StatusCode)
	}

	draftResponse := do(t, http.MethodPost, base+"/documents", `{"type":"jobcard","clientId":"`+client.ID+`","reference":"Archive history"}`, jsonHdr)
	var doc struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(draftResponse.Body).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if draftResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create document: status %d", draftResponse.StatusCode)
	}

	archiveResponse := do(t, http.MethodDelete, base+"/clients/"+client.ID, "", nil)
	if archiveResponse.StatusCode != http.StatusOK {
		t.Fatalf("archive client: status %d", archiveResponse.StatusCode)
	}
	activeResponse := do(t, http.MethodGet, base+"/clients", "", nil)
	var active []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(activeResponse.Body).Decode(&active); err != nil {
		t.Fatal(err)
	}
	if len(active) != 0 {
		t.Fatalf("active clients after archive = %v", active)
	}
	archivedResponse := do(t, http.MethodGet, base+"/clients/archived", "", nil)
	var archived []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(archivedResponse.Body).Decode(&archived); err != nil {
		t.Fatal(err)
	}
	if len(archived) != 1 || archived[0].ID != client.ID {
		t.Fatalf("archived clients = %v", archived)
	}

	historyResponse := do(t, http.MethodGet, base+"/documents/"+doc.ID, "", nil)
	if historyResponse.StatusCode != http.StatusOK {
		t.Fatalf("get historical document: status %d", historyResponse.StatusCode)
	}
	rejectedDraft := do(t, http.MethodPost, base+"/documents", `{"type":"jobcard","clientId":"`+client.ID+`","reference":"New work"}`, jsonHdr)
	if rejectedDraft.StatusCode != http.StatusBadRequest {
		t.Fatalf("create draft for archived client: status %d", rejectedDraft.StatusCode)
	}

	restoreResponse := do(t, http.MethodPost, base+"/clients/"+client.ID+"/restore", "", nil)
	if restoreResponse.StatusCode != http.StatusOK {
		t.Fatalf("restore client: status %d", restoreResponse.StatusCode)
	}
	activeResponse = do(t, http.MethodGet, base+"/clients", "", nil)
	if err := json.NewDecoder(activeResponse.Body).Decode(&active); err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].ID != client.ID {
		t.Fatalf("active clients after restore = %v", active)
	}
}

func TestFindPotentialDuplicateClientsThroughAPI(t *testing.T) {
	ts := testServer(t, "")
	base := ts.URL + "/api/v1"
	created := do(t, http.MethodPost, base+"/clients", `{"name":"Acme Holdings","contact":"Jane Doe"}`, jsonHdr)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create client: status %d", created.StatusCode)
	}
	var existing struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(created.Body).Decode(&existing); err != nil {
		t.Fatal(err)
	}

	response := do(t, http.MethodPost, base+"/clients/check-duplicates",
		`{"name":"  ACME   HOLDINGS ","contact":"Jane Doe"}`, jsonHdr)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("check duplicates: status %d", response.StatusCode)
	}
	var matches []struct {
		Client struct {
			ID string `json:"id"`
		} `json:"client"`
		Reasons []string `json:"reasons"`
	}
	if err := json.NewDecoder(response.Body).Decode(&matches); err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Client.ID != existing.ID {
		t.Fatalf("duplicate matches = %+v, want client %s", matches, existing.ID)
	}
	if !strings.Contains(strings.Join(matches[0].Reasons, ","), "name/contact") {
		t.Fatalf("duplicate reasons = %v", matches[0].Reasons)
	}
}

func TestDuplicateClientCheckIsNotAuditedAsWrite(t *testing.T) {
	handler := (&Server{}).auditWrites(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/clients/check-duplicates", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("check response status = %d, want %d", response.Code, http.StatusOK)
	}
}
