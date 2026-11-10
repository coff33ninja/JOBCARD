# GET /quotations/{id}/revisions

List the saved pre-edit versions of an issued quotation.

## ApiClient method

`listQuotationRevisions`

## MCP tool

`list_quotation_revisions`

## Request

Path parameter:

- `id` (string, required): quotation ID

## Response

- Status: `200 OK`
- Type: `DocumentRevision[]`
- Returns complete snapshots, newest first. Each revision has `id`, `documentId`, `recordedAt`, and `snapshot` fields.

## Errors

| Status | Code | Condition |
|---|---|---|
| 404 | `NOT_FOUND` | Quotation ID not found |
| 409 | `INVALID_STATE` | Document exists but is not a quotation |

## Rules and side effects

Read-only operation. Revisions are created by successful in-place edits through `PUT /quotations/{id}/revision`.

## Example

Request: `GET /api/v1/quotations/quo-1/revisions`

Response:
```json
[
  {
    "id": "rev-1",
    "documentId": "quo-1",
    "recordedAt": "2026-11-10T12:00:00Z",
    "snapshot": {
      "id": "quo-1",
      "type": "quotation",
      "number": "QUO-2026-001",
      "status": "sent",
      "scope": "Original repair scope"
    }
  }
]
```

## Used by

Issued quotation revision history.
