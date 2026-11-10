# POST /documents/{id}/restore

Restore an archived, unissued draft to the active register.

## ApiClient method

`restoreDraft`

## MCP tool

`restore_draft`

## Request

Path parameter:

- `id` (string, required): archived draft ID

No request body is required by the HTTP API. The MCP tool requires `confirm: true`.

## Response

- Status: `200 OK`
- Type: `Doc`
- Returns the draft with `archivedAt: null`.

## Errors

| Status | Code | Condition |
|---|---|---|
| 404 | `NOT_FOUND` | Document ID not found |
| 409 | `INVALID_STATE` | Document is issued or is not in draft status |

## Rules and side effects

- Restoring removes the archive marker and returns the draft to the active register.
- Dashboard totals are recalculated from active documents.

## Example

Request: `POST /api/v1/documents/doc-draft-1/restore`

Response:
```json
{
  "id": "doc-draft-1",
  "number": null,
  "status": "draft",
  "archivedAt": null
}
```

## Used by

Archived documents register.
