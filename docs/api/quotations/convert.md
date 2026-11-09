# POST /quotations/{id}/convert

Convert a quotation into a new draft invoice.

## ApiClient method
`convertQuotation`

## MCP tool
`convert_quotation`

## Request
Path parameter:
- `id` (string, required): quotation document ID

Body: empty.

## Response
- Status: `201 Created`
- Type: `Doc`
- Returns newly created draft invoice with `quotationId` linked.

## Errors
| Status | Code | Condition |
|---|---|---|
| 404 | `NOT_FOUND` | Quotation ID not found |
| 409 | `INVALID_STATE` | Document is not a quotation or already converted |
| 409 | `INVALID_STATE` | Not a quotation, still a draft, declined, cancelled, or already converted |

## Rules and side effects
- Copies client ID, reference, scope, work items, and line items into a new draft invoice (`number: null`, `status: "draft"`).
- Sets source quotation's status to `converted`.
- Does not mutate any other fields on the source quotation.

## Example
Request: `POST /api/v1/quotations/quo-1/convert`

Response:
```json
{
  "id": "inv-from-quo-1",
  "type": "invoice",
  "number": null,
  "quotationId": "quo-1",
  "clientId": "cli-1",
  "status": "draft",
  "totalCents": 450000
}
```

## Used by
Document View and Document Editor "Convert to invoice" button.
