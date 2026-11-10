# PUT /quotations/{id}/revision

Edit an open, issued quotation in place while preserving its prior version.

## ApiClient method

`updateIssuedQuotation`

## MCP tool

`update_quotation_revision`

## Request

Path parameter:

- `id` (string, required): issued quotation ID

Body: Editable document fields accepted by `PUT /documents/{id}` plus:

- `confirm` (boolean, required): must be `true`

The MCP tool also requires `confirm: true`, representing explicit user approval.

## Response

- Status: `200 OK`
- Type: `Doc`
- Returns the updated quotation. Its issued number and status are unchanged.

## Errors

| Status | Code | Condition |
|---|---|---|
| 400 | `CONFIRMATION_REQUIRED` | `confirm` is not `true` |
| 400 | `VALIDATION_FAILED` | No editable fields or invalid field values |
| 404 | `NOT_FOUND` | Quotation ID not found |
| 409 | `INVALID_STATE` | Document is not an open, issued quotation |

## Rules and side effects

- Each successful edit saves a complete snapshot of the quotation before the edit.
- The snapshot and document update are committed together; a failed update does not create a revision.
- The quotation number and status remain unchanged. Other issued document types cannot be revised in place.
- Read prior versions with `GET /quotations/{id}/revisions`.

## Example

Request: `PUT /api/v1/quotations/quo-1/revision`
```json
{
  "confirm": true,
  "scope": "Updated repair scope",
  "lines": [
    { "description": "Replacement part", "qty": 1, "unitPriceCents": 125000 }
  ]
}
```

## Used by

Issued quotation edit flow.
