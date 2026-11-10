// Package core implements the JobCard business rules.
package core

import (
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strings"

	"jobcard/internal/domain"
)

// StockIntakeLineInput describes one product and its purchase cost and markup.
type StockIntakeLineInput struct {
	Description   string `json:"description"`
	Quantity      int    `json:"quantity"`
	UnitCostCents int64  `json:"unitCostCents"`
	MarkupPercent int    `json:"markupPercent"`
}

// StockIntakeInput records a purchase or existing stock on hand.
type StockIntakeInput struct {
	Source           string                 `json:"source"`
	Supplier         string                 `json:"supplier"`
	Couriered        bool                   `json:"couriered"`
	CourierCostCents int64                  `json:"courierCostCents"`
	Items            []StockIntakeLineInput `json:"items"`
}

// StockItem describes an inventory line and its remaining available quantity.
type StockItem struct {
	ID                    string `json:"id"`
	IntakeID              string `json:"intakeId"`
	Source                string `json:"source"`
	Supplier              string `json:"supplier"`
	Couriered             bool   `json:"couriered"`
	CourierCostCents      int64  `json:"courierCostCents"`
	Description           string `json:"description"`
	QuantityReceived      int    `json:"quantityReceived"`
	QuantityAvailable     int    `json:"quantityAvailable"`
	UnitCostCents         int64  `json:"unitCostCents"`
	MarkupPercent         int    `json:"markupPercent"`
	CourierAllocatedCents int64  `json:"courierAllocatedCents"`
	CreatedAt             string `json:"createdAt"`
}

// StockAllocationResult reports the inventory change and its client draft invoice.
type StockAllocationResult struct {
	Item             StockItem  `json:"item"`
	Document         domain.Doc `json:"document"`
	QuantityAssigned int        `json:"quantityAssigned"`
}

type stockItemRecord struct {
	StockItem
	AllocatedQuantity int
}

func ensureStockTables(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS stock_intakes(
  id TEXT PRIMARY KEY, source TEXT NOT NULL CHECK(source IN ('purchase','on_hand')),
  supplier TEXT NOT NULL DEFAULT '', couriered INTEGER NOT NULL CHECK(couriered IN (0,1)),
  courier_cost_cents INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS stock_items(
  id TEXT PRIMARY KEY, intake_id TEXT NOT NULL REFERENCES stock_intakes(id),
  description TEXT NOT NULL, quantity_received INTEGER NOT NULL CHECK(quantity_received > 0),
  unit_cost_cents INTEGER NOT NULL CHECK(unit_cost_cents > 0),
  markup_percent INTEGER NOT NULL CHECK(markup_percent BETWEEN 8 AND 10),
  courier_allocated_cents INTEGER NOT NULL DEFAULT 0);
CREATE INDEX IF NOT EXISTS idx_stock_items_intake ON stock_items(intake_id);
CREATE TABLE IF NOT EXISTS stock_allocations(
  id TEXT PRIMARY KEY, item_id TEXT NOT NULL REFERENCES stock_items(id),
  client_id TEXT NOT NULL REFERENCES clients(id), quantity INTEGER NOT NULL CHECK(quantity > 0),
  courier_cost_cents INTEGER NOT NULL DEFAULT 0,
  invoice_id TEXT REFERENCES documents(id) ON DELETE SET NULL, created_at TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS idx_stock_allocations_item ON stock_allocations(item_id);
CREATE TABLE IF NOT EXISTS stock_invoice_links(
  client_id TEXT PRIMARY KEY REFERENCES clients(id),
  invoice_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE, updated_at TEXT NOT NULL);
`)
	return err
}

// CreateStockIntake records purchased stock or stock already on hand.
func (s *Service) CreateStockIntake(in StockIntakeInput) ([]StockItem, error) {
	in.Source = strings.TrimSpace(in.Source)
	in.Supplier = strings.TrimSpace(in.Supplier)
	if in.Source != "purchase" && in.Source != "on_hand" {
		return nil, invalid("Source must be purchase or on_hand", map[string]string{"source": "Invalid source"})
	}
	if in.Source == "purchase" && in.Supplier == "" {
		return nil, invalid("Supplier is required for purchased stock", map[string]string{"supplier": "Supplier is required"})
	}
	if in.CourierCostCents < 0 {
		return nil, invalid("Courier cost cannot be negative", map[string]string{"courierCostCents": "Cannot be negative"})
	}
	if in.Source == "on_hand" && (in.Couriered || in.CourierCostCents != 0) {
		return nil, invalid("Existing stock cannot include a courier charge", map[string]string{"couriered": "Not allowed for existing stock"})
	}
	if !in.Couriered && in.CourierCostCents != 0 {
		return nil, invalid("Courier cost requires couriered stock", map[string]string{"courierCostCents": "Couriered must be true"})
	}
	if len(in.Items) == 0 {
		return nil, invalid("At least one stock item is required", map[string]string{"items": "Add at least one item"})
	}

	totalCost, weights, err := validateStockLines(in.Items)
	if err != nil {
		return nil, err
	}
	courierShares, err := allocateCourierCost(in.CourierCostCents, totalCost, weights)
	if err != nil {
		return nil, err
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	intakeID := newID("stock-intake")
	createdAt := nowISO()
	if _, err := tx.Exec(`INSERT INTO stock_intakes(id,source,supplier,couriered,courier_cost_cents,created_at) VALUES(?,?,?,?,?,?)`,
		intakeID, in.Source, in.Supplier, in.Couriered, in.CourierCostCents, createdAt); err != nil {
		return nil, err
	}

	items := make([]StockItem, 0, len(in.Items))
	for i, line := range in.Items {
		item := StockItem{
			ID:                    newID("stock-item"),
			IntakeID:              intakeID,
			Source:                in.Source,
			Supplier:              in.Supplier,
			Couriered:             in.Couriered,
			CourierCostCents:      in.CourierCostCents,
			Description:           strings.TrimSpace(line.Description),
			QuantityReceived:      line.Quantity,
			QuantityAvailable:     line.Quantity,
			UnitCostCents:         line.UnitCostCents,
			MarkupPercent:         line.MarkupPercent,
			CourierAllocatedCents: courierShares[i],
			CreatedAt:             createdAt,
		}
		if _, err := tx.Exec(`INSERT INTO stock_items(id,intake_id,description,quantity_received,unit_cost_cents,markup_percent,courier_allocated_cents) VALUES(?,?,?,?,?,?,?)`,
			item.ID, item.IntakeID, item.Description, item.QuantityReceived, item.UnitCostCents, item.MarkupPercent, item.CourierAllocatedCents); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return items, nil
}

func validateStockLines(lines []StockIntakeLineInput) (int64, []int64, error) {
	weights := make([]int64, len(lines))
	var totalCost int64
	for i, line := range lines {
		field := fmt.Sprintf("items.%d", i)
		if strings.TrimSpace(line.Description) == "" {
			return 0, nil, invalid("Stock item description is required", map[string]string{field + ".description": "Required"})
		}
		if line.Quantity < 1 || line.Quantity > 1_000_000 {
			return 0, nil, invalid("Stock quantity must be between 1 and 1,000,000", map[string]string{field + ".quantity": "Out of range"})
		}
		if line.UnitCostCents <= 0 || line.UnitCostCents > int64(^uint64(0)>>1)/int64(line.Quantity) {
			return 0, nil, invalid("Unit cost must be positive and within the supported range", map[string]string{field + ".unitCostCents": "Out of range"})
		}
		if line.MarkupPercent < 8 || line.MarkupPercent > 10 {
			return 0, nil, invalid("Markup must be between 8 and 10 percent", map[string]string{field + ".markupPercent": "Must be 8, 9 or 10"})
		}
		weight := line.UnitCostCents * int64(line.Quantity)
		if totalCost > int64(^uint64(0)>>1)-weight {
			return 0, nil, invalid("Stock intake value is too large", map[string]string{"items": "Value exceeds supported range"})
		}
		weights[i] = weight
		totalCost += weight
	}
	return totalCost, weights, nil
}

func allocateCourierCost(courierCost, totalCost int64, weights []int64) ([]int64, error) {
	shares := make([]int64, len(weights))
	if courierCost == 0 {
		return shares, nil
	}
	if totalCost == 0 {
		return nil, invalid("Courier cost cannot be split without item costs", map[string]string{"courierCostCents": "No positive item cost"})
	}

	type remainder struct {
		index int
		value *big.Int
	}
	remainders := make([]remainder, 0, len(weights))
	total := big.NewInt(totalCost)
	assigned := int64(0)
	for i, weight := range weights {
		product := new(big.Int).Mul(big.NewInt(courierCost), big.NewInt(weight))
		quotient, rem := new(big.Int), new(big.Int)
		quotient.QuoRem(product, total, rem)
		shares[i] = quotient.Int64()
		assigned += shares[i]
		remainders = append(remainders, remainder{index: i, value: rem})
	}
	sort.Slice(remainders, func(i, j int) bool {
		if comparison := remainders[i].value.Cmp(remainders[j].value); comparison != 0 {
			return comparison > 0
		}
		return remainders[i].index < remainders[j].index
	})
	for i := 0; assigned < courierCost; i++ {
		if i >= len(remainders) {
			return nil, fmt.Errorf("could not distribute courier cost remainder")
		}
		shares[remainders[i].index]++
		assigned++
	}
	return shares, nil
}

// ListStockItems returns inventory entries, including entries with no stock left.
func (s *Service) ListStockItems() ([]StockItem, error) {
	return listStockItems(s.DB)
}

func listStockItems(q queryer) ([]StockItem, error) {
	rows, err := q.Query(`
SELECT i.id,i.intake_id,t.source,t.supplier,t.couriered,t.courier_cost_cents,
       i.description,i.quantity_received,i.unit_cost_cents,i.markup_percent,
       i.courier_allocated_cents,i.quantity_received-COALESCE(SUM(a.quantity),0),t.created_at
FROM stock_items i
JOIN stock_intakes t ON t.id=i.intake_id
LEFT JOIN stock_allocations a ON a.item_id=i.id
GROUP BY i.id
ORDER BY i.rowid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []StockItem{}
	for rows.Next() {
		var item StockItem
		var couriered int
		if err := rows.Scan(
			&item.ID, &item.IntakeID, &item.Source, &item.Supplier, &couriered, &item.CourierCostCents,
			&item.Description, &item.QuantityReceived, &item.UnitCostCents, &item.MarkupPercent,
			&item.CourierAllocatedCents, &item.QuantityAvailable, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.Couriered = couriered != 0
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// AllocateStock removes available stock and adds its all-in price to the client's stock draft invoice.
func (s *Service) AllocateStock(itemID, clientID string, quantity int) (StockAllocationResult, error) {
	if quantity < 1 || quantity > 1_000_000 {
		return StockAllocationResult{}, invalid("Assignment quantity must be between 1 and 1,000,000", map[string]string{"quantity": "Out of range"})
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return StockAllocationResult{}, err
	}
	defer tx.Rollback()

	if _, err := getActiveClient(tx, clientID); err != nil {
		var appError *AppError
		if errors.As(err, &appError) && appError.Status == http.StatusNotFound {
			return StockAllocationResult{}, invalid("Client not found", map[string]string{"clientId": "Client not found"})
		}
		return StockAllocationResult{}, err
	}
	item, allocatedQuantity, allocatedCourier, err := getStockItem(tx, itemID)
	if err != nil {
		return StockAllocationResult{}, err
	}
	available := item.QuantityReceived - allocatedQuantity
	if quantity > available {
		return StockAllocationResult{}, appErr(http.StatusConflict, "INSUFFICIENT_STOCK", "Requested quantity exceeds available stock")
	}

	courierShare := allocatedCourier
	if quantity < available {
		courierShare, err = roundedRatio(allocatedCourier, int64(quantity), int64(available))
		if err != nil {
			return StockAllocationResult{}, err
		}
	}
	lines, err := stockInvoiceLines(item, quantity, courierShare)
	if err != nil {
		return StockAllocationResult{}, err
	}

	doc, err := linkedStockInvoice(s, tx, clientID)
	if err != nil {
		return StockAllocationResult{}, err
	}
	if doc.ID == "" {
		typeName, client, reference := "invoice", clientID, "Stock items"
		doc, err = createDraft(s, tx, DraftInput{
			Type:      Opt[string]{Set: true, Val: &typeName},
			ClientID:  Opt[string]{Set: true, Val: &client},
			Reference: Opt[string]{Set: true, Val: &reference},
		})
		if err != nil {
			return StockAllocationResult{}, err
		}
	}
	doc.Lines = append(doc.Lines, lines...)
	doc.Lines = domain.NormalizeLines(doc.Lines)
	settings, err := getSettings(tx)
	if err != nil {
		return StockAllocationResult{}, err
	}
	totals := domain.CalcTotals(doc.Lines, settings.TaxRate, doc.Payments)
	doc.SubtotalCents, doc.TaxCents, doc.TotalCents = totals.Subtotal, totals.Tax, totals.Total
	doc.UpdatedAt = nowISO()
	if err := updateDoc(tx, &doc); err != nil {
		return StockAllocationResult{}, err
	}
	if _, err := tx.Exec(`INSERT INTO stock_invoice_links(client_id,invoice_id,updated_at) VALUES(?,?,?)
ON CONFLICT(client_id) DO UPDATE SET invoice_id=excluded.invoice_id,updated_at=excluded.updated_at`,
		clientID, doc.ID, doc.UpdatedAt); err != nil {
		return StockAllocationResult{}, err
	}
	if _, err := tx.Exec(`INSERT INTO stock_allocations(id,item_id,client_id,quantity,courier_cost_cents,invoice_id,created_at) VALUES(?,?,?,?,?,?,?)`,
		newID("stock-allocation"), itemID, clientID, quantity, courierShare, doc.ID, nowISO()); err != nil {
		return StockAllocationResult{}, err
	}

	updatedItem, _, _, err := getStockItem(tx, itemID)
	if err != nil {
		return StockAllocationResult{}, err
	}
	updatedItem.QuantityAvailable = item.QuantityReceived - allocatedQuantity - quantity
	doc, err = s.getDoc(tx, doc.ID)
	if err != nil {
		return StockAllocationResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return StockAllocationResult{}, err
	}
	return StockAllocationResult{Item: updatedItem, Document: doc, QuantityAssigned: quantity}, nil
}

func linkedStockInvoice(s *Service, q queryer, clientID string) (domain.Doc, error) {
	var invoiceID string
	err := q.QueryRow(`SELECT invoice_id FROM stock_invoice_links WHERE client_id=?`, clientID).Scan(&invoiceID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Doc{}, nil
	}
	if err != nil {
		return domain.Doc{}, err
	}
	doc, err := s.getDoc(q, invoiceID)
	if err != nil {
		return domain.Doc{}, err
	}
	if doc.Type != "invoice" || doc.ClientID != clientID || doc.Number != nil || doc.ArchivedAt != nil || doc.Status != "draft" {
		return domain.Doc{}, nil
	}
	return doc, nil
}

func getStockItem(q queryer, id string) (StockItem, int, int64, error) {
	var item StockItem
	var couriered, allocatedQuantity int
	var allocatedCourier int64
	err := q.QueryRow(`
SELECT i.id,i.intake_id,t.source,t.supplier,t.couriered,t.courier_cost_cents,
       i.description,i.quantity_received,i.unit_cost_cents,i.markup_percent,
       i.courier_allocated_cents,t.created_at,
       COALESCE(SUM(a.quantity),0),COALESCE(SUM(a.courier_cost_cents),0)
FROM stock_items i
JOIN stock_intakes t ON t.id=i.intake_id
LEFT JOIN stock_allocations a ON a.item_id=i.id
WHERE i.id=?
GROUP BY i.id`, id).Scan(
		&item.ID, &item.IntakeID, &item.Source, &item.Supplier, &couriered, &item.CourierCostCents,
		&item.Description, &item.QuantityReceived, &item.UnitCostCents, &item.MarkupPercent,
		&item.CourierAllocatedCents, &item.CreatedAt, &allocatedQuantity, &allocatedCourier,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return StockItem{}, 0, 0, notFound("Stock item", id)
	}
	if err != nil {
		return StockItem{}, 0, 0, err
	}
	item.Couriered = couriered != 0
	item.QuantityAvailable = item.QuantityReceived - allocatedQuantity
	return item, allocatedQuantity, item.CourierAllocatedCents - allocatedCourier, nil
}

func roundedRatio(value, numerator, denominator int64) (int64, error) {
	if denominator <= 0 || value < 0 || numerator < 0 {
		return 0, invalid("Invalid stock cost allocation", map[string]string{"quantity": "Invalid cost allocation"})
	}
	product := new(big.Int).Mul(big.NewInt(value), big.NewInt(numerator))
	divisor := big.NewInt(denominator)
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(product, divisor, remainder)
	if remainder.Mul(remainder, big.NewInt(2)).Cmp(divisor) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if !quotient.IsInt64() {
		return 0, invalid("Stock cost allocation exceeds the supported range", map[string]string{"quantity": "Out of range"})
	}
	return quotient.Int64(), nil
}

func applyMarkup(cost int64, markupPercent int) (int64, error) {
	if cost < 0 || markupPercent < 8 || markupPercent > 10 {
		return 0, invalid("Invalid stock markup", map[string]string{"markupPercent": "Must be 8, 9 or 10"})
	}
	product := new(big.Int).Mul(big.NewInt(cost), big.NewInt(int64(100+markupPercent)))
	product.Add(product, big.NewInt(50))
	product.Div(product, big.NewInt(100))
	if !product.IsInt64() {
		return 0, invalid("Stock price exceeds the supported range", map[string]string{"unitCostCents": "Out of range"})
	}
	return product.Int64(), nil
}

func stockInvoiceLines(item StockItem, quantity int, courierShare int64) ([]domain.Line, error) {
	baseCourierPerUnit := courierShare / int64(quantity)
	unitsWithExtraCent := int(courierShare % int64(quantity))
	basePrice, err := applyMarkup(item.UnitCostCents+baseCourierPerUnit, item.MarkupPercent)
	if err != nil {
		return nil, err
	}
	baseQuantity := quantity - unitsWithExtraCent
	if unitsWithExtraCent == 0 {
		return []domain.Line{{Description: item.Description, Qty: float64(quantity), UnitPriceCents: basePrice}}, nil
	}
	extraPrice, err := applyMarkup(item.UnitCostCents+baseCourierPerUnit+1, item.MarkupPercent)
	if err != nil {
		return nil, err
	}
	if extraPrice == basePrice {
		return []domain.Line{{Description: item.Description, Qty: float64(quantity), UnitPriceCents: basePrice}}, nil
	}
	lines := make([]domain.Line, 0, 2)
	if baseQuantity > 0 {
		lines = append(lines, domain.Line{Description: item.Description, Qty: float64(baseQuantity), UnitPriceCents: basePrice})
	}
	lines = append(lines, domain.Line{Description: item.Description, Qty: float64(unitsWithExtraCent), UnitPriceCents: extraPrice})
	return lines, nil
}
