# POST /stock/items/{id}/assign

Assign available stock to a client and add its all-in charge to a draft invoice.

## ApiClient method

`allocateStock`

## MCP tool

`allocate_stock`

## Request

Path parameter `id` is the stock item ID.

```json
{
  "clientId": "client-123",
  "quantity": 1
}
```

## Response

`201 Created` with the updated stock entry, draft invoice, and assigned quantity.

## Errors

- `400 VALIDATION_FAILED` — invalid quantity or client.
- `404 NOT_FOUND` — stock item does not exist.
- `409 INSUFFICIENT_STOCK` — requested quantity exceeds the available quantity.
- `500 INTERNAL_ERROR` — the assignment could not be completed.

## Rules and side effects

- The client must be active and the requested quantity must be available.
- Inventory is reduced and the client's stock draft invoice is created or updated in one transaction.
- Repeated assignments for the same client are added to the same unissued stock draft invoice. A new draft is created once the previous one is issued or archived.
- The invoice shows the final item price with the 8–10% markup and the allocated courier cost already included; purchase cost and markup are not shown separately.
- The invoice remains a draft and is not issued automatically.

## Example

```http
POST /api/v1/stock/items/stock-item-123/assign
Content-Type: application/json

{"clientId":"client-123","quantity":1}
```

## Used by

- Inventory screen
- `allocate_stock` MCP tool
