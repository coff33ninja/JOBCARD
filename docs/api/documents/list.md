# GET /documents

List and filter documents across quotations, invoices, and job cards.

## ApiClient method
`listDocuments`

## MCP tool
`list_documents`

## Request
Query parameters:
- `type` (DocType, optional): "quotation" | "invoice" | "jobcard"
- `status` (string, optional): status filter
- `clientId` (string, optional): filter by client
- `q` (string, optional): text search on reference, number, or scope
- `from` (ISODate, optional): issue date range start
- `to` (ISODate, optional): issue date range end

## Response
- Status: `200 OK`
- Type: `Doc[]`
- Returns an array of matching documents sorted by issueDate / createdAt descending.

## Errors
| Status | Code | Condition |
|---|---|---|
| 500 | `INTERNAL_ERROR` | Database query failure |

## Rules and side effects
Read-only search operation.

## Example
Request: `GET /api/v1/documents?type=invoice`

Response:
```json
[
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
    "workItems": [],
    "lines": [],
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
]
```

## Used by
Register screen, Client detail documents list, Dashboard summary loader.
