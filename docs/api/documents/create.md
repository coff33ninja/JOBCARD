# POST /documents

Create a new draft document (invoice, quotation, or job card).

## ApiClient method
`createDraft`

## MCP tool
`create_draft`

## Request
Body:
- `type` ("quotation" | "invoice" | "jobcard", required)
- `clientId` (string, required)
- `reference` (string, required)
- `scope` (string, optional)
- `jobCardId` (string, optional)
- `quotationId` (string, optional)
- `workItems` (WorkItem[], optional)
- `lines` (LineItem[], optional)
- `depositCents` (Cents, optional)
- `terms` (string, optional)
- `exclusions` (string, optional)
- `turnaround` (string, optional)
- `notes` (string, optional)

## Response
- Status: `201 Created`
- Type: `Doc`
- Returns draft document with `number: null` and `status: "draft"`.

## Errors
| Status | Code | Condition |
|---|---|---|
| 400 | `VALIDATION_FAILED` | Missing type, clientId, or reference |

## Rules and side effects
- Document is saved with `number: null` and `status: "draft"`.
- Calculates subtotal, tax, and total based on provided lines and settings tax rate.

## Example
Request: `POST /api/v1/documents`
```json
{
  "type": "quotation",
  "clientId": "cli-1",
  "reference": "Dell Latitude 5420",
  "scope": "Battery replacement",
  "lines": [{ "description": "OEM Battery", "qty": 1, "unitPriceCents": 120000, "totalCents": 120000 }]
}
```

Response:
```json
{
  "id": "doc-draft-1",
  "type": "quotation",
  "number": null,
  "clientId": "cli-1",
  "status": "draft",
  "totalCents": 120000
}
```

## Used by
Document Editor create flow.
