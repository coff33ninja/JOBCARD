# POST /documents/{id}/archive

Archive an unissued draft without deleting it.

## ApiClient method

`archiveDraft`

## MCP tool

`archive_draft`

## Request

Path parameter:

- `id` (string, required): draft ID

No request body is required by the HTTP API. The MCP tool requires `confirm: true`.

## Response

- Status: `200 OK`
- Type: `Doc`
- Returns the archived draft with `archivedAt` set.

## Errors

| Status | Code | Condition |
|---|---|---|
| 404 | `NOT_FOUND` | Document ID not found |
| 409 | `INVALID_STATE` | Document is issued or is not in draft status |

## Rules and side effects

- The draft remains stored and can be restored.
- It is excluded from the active register and dashboard totals while archived.
- Issued documents cannot be archived through this operation.

## Example

Request: `POST /api/v1/documents/doc-draft-1/archive`

Response:
```json
{
  "id": "doc-draft-1",
  "number": null,
  "status": "draft",
  "archivedAt": "2026-11-10T12:00:00Z"
}
```

## Used by

Document detail draft actions.
