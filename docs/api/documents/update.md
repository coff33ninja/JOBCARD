# PUT /documents/{id}

Update a draft document.

## ApiClient method
`updateDraft`

## MCP tool
`update_draft`

## Request
Path parameter:
- `id` (string, required): document ID

Body: Partial document fields (client, lines, work items, terms, notes).

## Response
- Status: `200 OK`
- Type: `Doc`
- Returns updated document with recalculated totals.

## Errors
| Status | Code | Condition |
|---|---|---|
| 400 | `VALIDATION_FAILED` | Invalid fields |
| 404 | `NOT_FOUND` | Document ID not found |
| 409 | `CANNOT_EDIT_ISSUED` | Attempting to edit an already issued document |

## Rules and side effects
- Recalculates subtotal, tax, and total.
- Rejects edits if document has already been issued.

## Example
Request: `PUT /api/v1/documents/doc-draft-1`
```json
{
  "scope": "Battery and fan cleaning",
  "notes": "Completed test"
}
```

Response:
```json
{
  "id": "doc-draft-1",
  "type": "quotation",
  "scope": "Battery and fan cleaning",
  "notes": "Completed test"
}
```

## Used by
Document Editor edit draft flow.
