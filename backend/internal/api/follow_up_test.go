package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestFollowUpDateIsReturnedAndCanBeSetThroughAPI(t *testing.T) {
	ts := testServer(t, "")
	api := ts.URL + "/api/v1"

	clientResponse := do(t, http.MethodPost, api+"/clients", `{"name":"Follow-up Test"}`, jsonHdr)
	if clientResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create client: status %d", clientResponse.StatusCode)
	}
	var client struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(clientResponse.Body).Decode(&client); err != nil {
		t.Fatal(err)
	}

	draftResponse := do(t, http.MethodPost, api+"/documents", `{"type":"invoice","clientId":"`+client.ID+`","reference":"Follow-up test","lines":[{"description":"Service","qty":1,"unitPriceCents":0}]}`, jsonHdr)
	if draftResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create invoice draft: status %d", draftResponse.StatusCode)
	}
	var draft struct {
		ID           string  `json:"id"`
		FollowUpDate *string `json:"followUpDate"`
	}
	if err := json.NewDecoder(draftResponse.Body).Decode(&draft); err != nil {
		t.Fatal(err)
	}
	if draft.FollowUpDate != nil {
		t.Fatalf("new draft follow-up date = %q, want null", *draft.FollowUpDate)
	}

	issuedResponse := do(t, http.MethodPost, api+"/documents/"+draft.ID+"/issue", `{}`, jsonHdr)
	if issuedResponse.StatusCode != http.StatusOK {
		t.Fatalf("issue invoice: status %d", issuedResponse.StatusCode)
	}

	setResponse := do(t, http.MethodPost, api+"/documents/"+draft.ID+"/follow-up", `{"followUpDate":"2026-11-09"}`, jsonHdr)
	if setResponse.StatusCode != http.StatusOK {
		t.Fatalf("set follow-up date: status %d", setResponse.StatusCode)
	}
	var updated struct {
		FollowUpDate *string `json:"followUpDate"`
	}
	if err := json.NewDecoder(setResponse.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.FollowUpDate == nil || *updated.FollowUpDate != "2026-11-09" {
		t.Fatalf("updated follow-up date = %v, want 2026-11-09", updated.FollowUpDate)
	}

	listResponse := do(t, http.MethodGet, api+"/documents", "", nil)
	if listResponse.StatusCode != http.StatusOK {
		t.Fatalf("list documents: status %d", listResponse.StatusCode)
	}
	var documents []struct {
		ID           string  `json:"id"`
		FollowUpDate *string `json:"followUpDate"`
	}
	if err := json.NewDecoder(listResponse.Body).Decode(&documents); err != nil {
		t.Fatal(err)
	}
	if len(documents) != 1 || documents[0].ID != draft.ID ||
		documents[0].FollowUpDate == nil || *documents[0].FollowUpDate != "2026-11-09" {
		t.Fatalf("listed document follow-up date = %+v, want 2026-11-09", documents)
	}
}
