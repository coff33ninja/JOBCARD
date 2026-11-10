# POST /stock/intakes

Record a purchase or stock that was already on hand.

## ApiClient method

`createStockIntake`

## MCP tool

`create_stock_intake`

## Request

```json
{
  "source": "purchase",
  "supplier": "Parts Shop",
  "couriered": true,
  "courierCostCents": 1000,
  "items": [
    {
      "description": "SSD",
      "quantity": 2,
      "unitCostCents": 10000,
      "markupPercent": 10
    },
    {
      "description": "RAM",
      "quantity": 1,
      "unitCostCents": 10000,
      "markupPercent": 8
    }
  ]
}
```

Use `source: "on_hand"` for stock being entered from existing inventory; supplier and courier details are omitted or empty.

## Response

`201 Created` with the stock entries created by the intake.

## Errors

- `400 VALIDATION_FAILED` — invalid source, missing purchase supplier, invalid quantity/cost/markup, or courier details inconsistent with the source.
- `500 INTERNAL_ERROR` — the intake could not be stored.

## Rules and side effects

- Every item must have a positive quantity and unit cost, and a markup of 8%, 9%, or 10%.
- Purchased stock must include its supplier. Existing stock on hand cannot include a courier charge.
- When a purchase is couriered, the courier cost is split between its items in proportion to each item's purchase value. Rounding cents are fully distributed.
- The intake and all its item entries are saved atomically.

## Example

```http
POST /api/v1/stock/intakes
Content-Type: application/json

{"source":"purchase","supplier":"Parts Shop","couriered":false,"courierCostCents":0,"items":[{"description":"SSD","quantity":2,"unitCostCents":10000,"markupPercent":9}]}
```

## Used by

- Inventory intake form
- `create_stock_intake` MCP tool
