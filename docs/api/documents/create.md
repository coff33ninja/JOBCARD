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
- `intakeItems` (`{ label: string, value: string }[]`, optional): equipment and condition received
- `diagnosticSections` (`{ title: string, status: string, details: string }[]`, optional)
- `authorization` (`{ technicianSignature, technicianDate, clientSignature, clientDate }`, optional; all values are strings)
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
- When using `newClient`, check for existing clients with [POST /clients/check-duplicates](../clients/check-duplicates.md) first; this warns about possible matches but does not block creation.

Money is server-authoritative: each line `totalCents` is recomputed as round(qty x unitPriceCents) and any supplied total is ignored; tax is calculated from the current `taxRate` setting. `jobCardId` and `quotationId`, when given, must reference an existing job card or quotation.

## Example
Request: `POST /api/v1/documents`
```json
{
  "type": "quotation",
  "clientId": "cli-1",
  "reference": "Dell Latitude 5420",
  "scope": "Battery replacement",
  "intakeItems": [{ "label": "Charger", "value": "Received with laptop" }],
  "diagnosticSections": [{ "title": "Battery test", "status": "FAILED", "details": "Battery does not hold charge." }],
  "authorization": {
    "technicianSignature": "Example Technician",
    "technicianDate": "2026-10-12",
    "clientSignature": "Example Client",
    "clientDate": "2026-10-12"
  },
  "lines": [{ "description": "OEM Battery", "detail": "Replacement part", "qty": 1, "unitPriceCents": 120000, "totalCents": 120000 }]
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
