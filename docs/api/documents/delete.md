# DELETE /documents/{id}

Permanently delete an unissued draft.

## ApiClient method

`deleteDraft`

## MCP tool

`delete_draft`

## Request

Path parameter:

- `id` (string, required): draft ID

Body:

- `confirm` (boolean, required): must be `true`

The MCP tool also requires `confirm: true`.

## Response

- Status: `200 OK`
- Type: `Doc`
- Returns the deleted draft record as confirmation.

## Errors

| Status | Code | Condition |
|---|---|---|
| 400 | `CONFIRMATION_REQUIRED` | `confirm` is not `true` |
| 404 | `NOT_FOUND` | Document ID not found |
| 409 | `INVALID_STATE` | Document is issued or is not in draft status |
| 409 | `DOCUMENT_IN_USE` | Another document or a payment references the draft |

## Rules and side effects

- Deletion is permanent; use archive instead when the draft may be needed later.
- Issued documents cannot be deleted.
- Drafts referenced by another document or payment cannot be deleted.

## Example

Request: `DELETE /api/v1/documents/doc-draft-1`
```json
{
  "confirm": true
}
```

## Used by

Document detail draft actions.
