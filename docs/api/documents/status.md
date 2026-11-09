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

Allowed transitions (from the stored status): invoice `draft`/`sent`/`overdue` to `cancelled` (not when payments exist); quotation `draft` to `cancelled`, `sent` to `accepted`/`declined`/`cancelled`, `accepted` to `cancelled`; job card `open` to `in_progress`, `in_progress` to `completed`/`open`, `completed` to `invoiced`/`closed`/`in_progress`, `invoiced` to `closed`, `closed` to `open`. `partially_paid`, `paid` and `converted` are never set directly: payments and conversion set them.

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
