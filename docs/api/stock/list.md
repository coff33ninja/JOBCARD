# GET /stock

List stock intake entries, including quantities already assigned to clients.

## ApiClient method

`listStockItems`

## MCP tool

`list_stock_items`

## Request

No request body.

## Response

`200 OK` with stock entries in intake order.

```json
[
  {
    "id": "stock-item-123",
    "intakeId": "stock-intake-123",
    "source": "purchase",
    "supplier": "Parts Shop",
    "couriered": true,
    "courierCostCents": 1000,
    "description": "SSD",
    "quantityReceived": 2,
    "quantityAvailable": 1,
    "unitCostCents": 10000,
    "markupPercent": 10,
    "courierAllocatedCents": 667,
    "createdAt": "2026-11-10T12:00:00Z"
  }
]
```

## Errors

- `500 INTERNAL_ERROR` — the inventory could not be read.

## Rules and side effects

- Entries remain listed when their available quantity reaches zero.
- Costs and markup are internal inventory details; only final all-in prices go on client invoices.

## Example

```http
GET /api/v1/stock
```

## Used by

- Inventory screen
- Stock allocation workflows
