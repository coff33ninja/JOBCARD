# POST /documents/{id}/payments

Record a payment received against an invoice.

## ApiClient method
`addPayment`

## MCP tool
`add_payment`

## Request
Path parameter:
- `id` (string, required): invoice document ID

Body:
- `date` (ISODate, required): payment date
- `amountCents` (Cents, required, integer > 0)
- `method` (string, required): payment method, e.g. "EFT", "Cash", "Card"
- `reference` (string, optional): bank or transaction reference

## Response
- Status: `201 Created`
- Type: `Doc`
- Returns updated invoice with new payment appended and updated status.

## Errors
| Status | Code | Condition |
|---|---|---|
| 400 | `VALIDATION_FAILED` | Missing payment date or amount |
| 404 | `NOT_FOUND` | Document not found |
| 422 | `OVERPAYMENT_REJECTED` | Payment amount exceeds remaining balance |

## Rules and side effects
- Appends payment to document payments list.
- Calculates new balance.
- If balance reaches 0, status transitions to `paid`.
- If balance > 0 and paid > 0, status transitions to `partially_paid`.

## Example
Request: `POST /api/v1/documents/doc-1/payments`
```json
{
  "date": "2026-11-08",
  "amountCents": 150000,
  "method": "EFT",
  "reference": "EFT-8849"
}
```

Response:
```json
{
  "id": "doc-1",
  "status": "paid",
  "totalCents": 150000,
  "payments": [
    {
      "id": "pay-1",
      "date": "2026-11-08",
      "amountCents": 150000,
      "method": "EFT",
      "reference": "EFT-8849"
    }
  ]
}
```

## Used by
Document View payment dialog.
