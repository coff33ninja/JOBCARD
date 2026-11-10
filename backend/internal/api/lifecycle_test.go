package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestDocumentLifecycleThroughAPI(t *testing.T) {
	ts := testServer(t, "")
	base := ts.URL + "/api/v1"

	clientResponse := do(t, http.MethodPost, base+"/clients", `{"name":"Lifecycle Test"}`, jsonHdr)
	var client struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(clientResponse.Body).Decode(&client); err != nil {
		t.Fatal(err)
	}
	if clientResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create client: status %d", clientResponse.StatusCode)
	}

	create := func(reference string) string {
		t.Helper()
		response := do(t, http.MethodPost, base+"/documents",
			`{"type":"quotation","clientId":"`+client.ID+`","reference":"`+reference+`","lines":[{"description":"Repair","qty":1,"unitPriceCents":12500}]}`,
			jsonHdr)
		var doc struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(response.Body).Decode(&doc); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusCreated {
			t.Fatalf("create draft: status %d", response.StatusCode)
		}
		return doc.ID
	}
	issue := func(id string) {
		t.Helper()
		response := do(t, http.MethodPost, base+"/documents/"+id+"/issue", "", nil)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("issue document: status %d", response.StatusCode)
		}
	}

	archivedDraft := create("Archive this")
	archiveResponse := do(t, http.MethodPost, base+"/documents/"+archivedDraft+"/archive", "", nil)
	if archiveResponse.StatusCode != http.StatusOK {
		t.Fatalf("archive draft: status %d", archiveResponse.StatusCode)
	}
	archivedResponse := do(t, http.MethodGet, base+"/documents/archived", "", nil)
	var archived []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(archivedResponse.Body).Decode(&archived); err != nil {
		t.Fatal(err)
	}
	if len(archived) != 1 || archived[0].ID != archivedDraft {
		t.Fatalf("archived documents: %v", archived)
	}
	restoreResponse := do(t, http.MethodPost, base+"/documents/"+archivedDraft+"/restore", "", nil)
	if restoreResponse.StatusCode != http.StatusOK {
		t.Fatalf("restore draft: status %d", restoreResponse.StatusCode)
	}
	deleteWithoutConfirmation := do(t, http.MethodDelete, base+"/documents/"+archivedDraft, `{"confirm":false}`, jsonHdr)
	if deleteWithoutConfirmation.StatusCode != http.StatusBadRequest {
		t.Fatalf("delete without confirmation: status %d", deleteWithoutConfirmation.StatusCode)
	}
	deleteResponse := do(t, http.MethodDelete, base+"/documents/"+archivedDraft, `{"confirm":true}`, jsonHdr)
	if deleteResponse.StatusCode != http.StatusOK {
		t.Fatalf("delete confirmed draft: status %d", deleteResponse.StatusCode)
	}

	quoteID := create("Revise and replace")
	issue(quoteID)
	editWithoutConfirmation := do(t, http.MethodPut, base+"/quotations/"+quoteID+"/revision",
		`{"scope":"Updated scope","confirm":false}`, jsonHdr)
	if editWithoutConfirmation.StatusCode != http.StatusBadRequest {
		t.Fatalf("revision without confirmation: status %d", editWithoutConfirmation.StatusCode)
	}
	editResponse := do(t, http.MethodPut, base+"/quotations/"+quoteID+"/revision",
		`{"scope":"Updated scope","confirm":true}`, jsonHdr)
	if editResponse.StatusCode != http.StatusOK {
		t.Fatalf("confirmed in-place revision: status %d", editResponse.StatusCode)
	}
	historyResponse := do(t, http.MethodGet, base+"/quotations/"+quoteID+"/revisions", "", nil)
	var revisions []struct {
		Snapshot struct {
			Scope string `json:"scope"`
		} `json:"snapshot"`
	}
	if err := json.NewDecoder(historyResponse.Body).Decode(&revisions); err != nil {
		t.Fatal(err)
	}
	if historyResponse.StatusCode != http.StatusOK || len(revisions) != 1 || revisions[0].Snapshot.Scope != "" {
		t.Fatalf("revision history: status %d, revisions %v", historyResponse.StatusCode, revisions)
	}

	replacementResponse := do(t, http.MethodPost, base+"/quotations/"+quoteID+"/replacement", "", nil)
	var replacement struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(replacementResponse.Body).Decode(&replacement); err != nil {
		t.Fatal(err)
	}
	if replacementResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create replacement draft: status %d", replacementResponse.StatusCode)
	}
	issue(replacement.ID)
	sourceResponse := do(t, http.MethodGet, base+"/documents/"+quoteID, "", nil)
	var source struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(sourceResponse.Body).Decode(&source); err != nil {
		t.Fatal(err)
	}
	if source.Status != "cancelled" {
		t.Fatalf("original quotation status after replacement issue = %q", source.Status)
	}
}
