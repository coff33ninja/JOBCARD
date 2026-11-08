# POST /documents/{id}/status

Update document status directly (such as accepting or declining quotations, or cancelling invoices).

## ApiClient method
`updateStatus`

## MCP tool
`update_status`

## Request
Path parameter:
- `id` (string, required): document ID

Body:
- `status` (string, required): target status e.g. "accepted", "declined", "cancelled", "completed", "closed"

## Response
- Status: `200 OK`
- Type: `Doc`
- Returns updated document with new status.

## Errors
| Status | Code | Condition |
|---|---|---|
| 400 | `VALIDATION_FAILED` | Unsupported status transition |
| 404 | `NOT_FOUND` | Document not found |

## Rules and side effects
Updates the document status.

## Example
Request: `POST /api/v1/documents/doc-1/status`
```json
{
  "status": "cancelled"
}
```

Response:
```json
{
  "id": "doc-1",
  "status": "cancelled"
}
```

## Used by
Document View status action buttons (Cancel, Accept, Decline).
