package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"jobcard/internal/core"
	"jobcard/internal/domain"
)

func TestStockIntakeAssignmentAndInventoryEndpoints(t *testing.T) {
	ts := testServer(t, "")
	base := ts.URL + "/api/v1"

	clientResponse := do(t, http.MethodPost, base+"/clients", `{"name":"Stock Client"}`, jsonHdr)
	if clientResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create client: status %d", clientResponse.StatusCode)
	}
	var client domain.Client
	if err := json.NewDecoder(clientResponse.Body).Decode(&client); err != nil {
		t.Fatal(err)
	}

	intakeResponse := do(t, http.MethodPost, base+"/stock/intakes",
		`{"source":"purchase","supplier":"Parts Shop","couriered":true,"courierCostCents":500,"items":[{"description":"SSD","quantity":2,"unitCostCents":10000,"markupPercent":9}]}`,
		jsonHdr)
	if intakeResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create stock intake: status %d", intakeResponse.StatusCode)
	}
	var items []core.StockItem
	if err := json.NewDecoder(intakeResponse.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].QuantityAvailable != 2 {
		t.Fatalf("unexpected intake response: %+v", items)
	}

	assignResponse := do(t, http.MethodPost, base+"/stock/items/"+items[0].ID+"/assign",
		`{"clientId":"`+client.ID+`","quantity":1}`, jsonHdr)
	if assignResponse.StatusCode != http.StatusCreated {
		t.Fatalf("assign stock: status %d", assignResponse.StatusCode)
	}
	var assignment core.StockAllocationResult
	if err := json.NewDecoder(assignResponse.Body).Decode(&assignment); err != nil {
		t.Fatal(err)
	}
	if assignment.Document.Type != "invoice" || assignment.Document.ClientID != client.ID ||
		assignment.QuantityAssigned != 1 || assignment.Item.QuantityAvailable != 1 {
		t.Fatalf("unexpected assignment response: %+v", assignment)
	}

	inventoryResponse := do(t, http.MethodGet, base+"/stock", "", nil)
	if inventoryResponse.StatusCode != http.StatusOK {
		t.Fatalf("list stock: status %d", inventoryResponse.StatusCode)
	}
	if err := json.NewDecoder(inventoryResponse.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].QuantityAvailable != 1 {
		t.Fatalf("inventory did not reflect the assignment: %+v", items)
	}
}

func TestStockAssignmentRejectsInvalidRequest(t *testing.T) {
	ts := testServer(t, "")
	base := ts.URL + "/api/v1"

	response := do(t, http.MethodPost, base+"/stock/items/missing/assign",
		`{"clientId":"missing","quantity":0}`, jsonHdr)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid assignment: status %d", response.StatusCode)
	}
}
