# GET /documents/archived

List archived, unissued drafts.

## ApiClient method

`listArchivedDocuments`

## MCP tool

`list_archived_documents`

## Request

No query parameters.

## Response

- Status: `200 OK`
- Type: `Doc[]`
- Returns archived drafts ordered by their document dates.

## Errors

| Status | Code | Condition |
|---|---|---|
| 500 | `INTERNAL_ERROR` | Database query failure |

## Rules and side effects

- Only unissued drafts explicitly archived by a user are returned.
- Archived drafts are excluded from `GET /documents`, the active register, and dashboard totals.
- The documents remain stored and can be returned to the active register with `POST /documents/{id}/restore`.
- The spreadsheet export includes all stored documents, including archived drafts.

## Example

Request: `GET /api/v1/documents/archived`

Response:
```json
[
  {
    "id": "doc-draft-1",
    "type": "quotation",
    "number": null,
    "status": "draft",
    "archivedAt": "2026-11-10T12:00:00Z"
  }
]
```

## Used by

Archived documents register.
