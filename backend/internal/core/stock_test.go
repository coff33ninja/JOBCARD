package core

import (
	"testing"

	"jobcard/internal/domain"
)

func TestStockIntakeCourierAllocationAndClientAssignment(t *testing.T) {
	s := newTestService(t)
	client := mustClient(t, s)

	_, err := s.CreateStockIntake(StockIntakeInput{
		Source:           "purchase",
		Supplier:         "Parts Shop",
		Couriered:        true,
		CourierCostCents: 1000,
		Items: []StockIntakeLineInput{
			{Description: "SSD", Quantity: 2, UnitCostCents: 10000, MarkupPercent: 10},
			{Description: "RAM", Quantity: 1, UnitCostCents: 10000, MarkupPercent: 8},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	items, err := s.ListStockItems()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].CourierAllocatedCents+items[1].CourierAllocatedCents != 1000 {
		t.Fatalf("courier fee was not fully allocated: %+v", items)
	}
	ssd := items[0]
	if ssd.Description != "SSD" || ssd.CourierAllocatedCents != 667 {
		t.Fatalf("courier fee should be split by item cost: %+v", ssd)
	}

	first, err := s.AllocateStock(ssd.ID, client.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.Document.Type != "invoice" || first.Document.Status != "draft" || first.QuantityAssigned != 1 {
		t.Fatalf("unexpected stock invoice: %+v", first)
	}
	if got := first.Document.Lines[0].UnitPriceCents; got != 11367 {
		t.Fatalf("expected all-in price with 10%% markup, got %d cents", got)
	}
	if first.Item.QuantityAvailable != 1 {
		t.Fatalf("expected one SSD left in inventory, got %d", first.Item.QuantityAvailable)
	}

	second, err := s.AllocateStock(ssd.ID, client.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if second.Document.ID != first.Document.ID || len(second.Document.Lines) != 2 {
		t.Fatalf("stock allocations should accumulate in one client draft invoice: %+v", second.Document)
	}
	if got := second.Document.Lines[1].UnitPriceCents; got != 11366 {
		t.Fatalf("expected remaining courier cents to be included, got %d cents", got)
	}
}

func TestStockAllocationRejectsExcessQuantityWithoutChangingInventory(t *testing.T) {
	s := newTestService(t)
	client := mustClient(t, s)
	items, err := s.CreateStockIntake(StockIntakeInput{
		Source: "on_hand",
		Items: []StockIntakeLineInput{
			{Description: "Keyboard", Quantity: 2, UnitCostCents: 5000, MarkupPercent: 9},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.AllocateStock(items[0].ID, client.ID, 3)
	wantStatus(t, err, 409, "INSUFFICIENT_STOCK")

	inventory, err := s.ListStockItems()
	if err != nil {
		t.Fatal(err)
	}
	if inventory[0].QuantityAvailable != 2 {
		t.Fatalf("failed allocation changed inventory: %+v", inventory[0])
	}
	docs, err := s.AllDocuments()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 0 {
		t.Fatalf("failed allocation created a document: %+v", docs)
	}
}

func TestStockIntakeValidatesMarkupAndSupportsExistingStock(t *testing.T) {
	s := newTestService(t)

	_, err := s.CreateStockIntake(StockIntakeInput{
		Source: "on_hand",
		Items: []StockIntakeLineInput{
			{Description: "Mouse", Quantity: 1, UnitCostCents: 1000, MarkupPercent: 11},
		},
	})
	wantStatus(t, err, 400, "VALIDATION_FAILED")

	items, err := s.CreateStockIntake(StockIntakeInput{
		Source: "on_hand",
		Items: []StockIntakeLineInput{
			{Description: "Mouse", Quantity: 1, UnitCostCents: 1000, MarkupPercent: 8},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Source != "on_hand" || items[0].Couriered || items[0].Supplier != "" {
		t.Fatalf("unexpected existing-stock entry: %+v", items[0])
	}
	if items[0].CourierAllocatedCents != 0 {
		t.Fatalf("existing stock should not have a courier allocation: %+v", items[0])
	}
}

func TestStockInvoiceUsesFinalPriceOnly(t *testing.T) {
	s := newTestService(t)
	client := mustClient(t, s)
	items, err := s.CreateStockIntake(StockIntakeInput{
		Source:   "purchase",
		Supplier: "Parts Shop",
		Items: []StockIntakeLineInput{
			{Description: "Network adapter", Quantity: 1, UnitCostCents: 10000, MarkupPercent: 8},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := s.AllocateStock(items[0].ID, client.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Document.Lines) != 1 || result.Document.Lines[0].Description != "Network adapter" {
		t.Fatalf("invoice should show the item, not internal cost details: %+v", result.Document.Lines)
	}
	if result.Document.Lines[0].UnitPriceCents != domain.Round(10000*1.08) {
		t.Fatalf("unexpected all-in price: %+v", result.Document.Lines[0])
	}
}
