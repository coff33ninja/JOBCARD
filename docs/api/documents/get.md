# GET /documents/{id}

Retrieve a single document by its unique identifier.

## ApiClient method
`getDocument`

## MCP tool
`get_document`

## Request
Path parameter:
- `id` (string, required): document ID

## Response
- Status: `200 OK`
- Type: `Doc`
- Returns complete document object including line items, work items, and payments.

## Errors
| Status | Code | Condition |
|---|---|---|
| 404 | `NOT_FOUND` | Document ID not found |

## Rules and side effects
Read-only query.

## Example
Request: `GET /api/v1/documents/doc-1`

Response:
```json
{
  "id": "doc-1",
  "type": "invoice",
  "number": "INV-2026-001",
  "clientId": "cli-1",
  "jobCardId": null,
  "quotationId": null,
  "issueDate": "2026-11-01",
  "dueDate": "2026-11-08",
  "validUntil": null,
  "reference": "Lenovo T480",
  "scope": "SSD upgrade",
  "workItems": [{ "title": "Hardware", "detail": "Installed 1TB NVMe" }],
  "lines": [{ "description": "1TB SSD", "qty": 1, "unitPriceCents": 150000, "totalCents": 150000 }],
  "taxCents": 0,
  "subtotalCents": 150000,
  "totalCents": 150000,
  "depositCents": null,
  "payments": [],
  "status": "sent",
  "terms": "7 days",
  "exclusions": "",
  "turnaround": "",
  "notes": "",
  "folderPath": null,
  "createdAt": "2026-11-01T08:00:00Z",
  "updatedAt": "2026-11-01T08:00:00Z"
}
```

## Used by
Document View screen, Document Editor edit mode.
