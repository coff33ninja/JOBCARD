# PUT /documents/{id}

Update a draft document.

## ApiClient method
`updateDraft`

## MCP tool
`update_draft`

## Request
Path parameter:
- `id` (string, required): document ID

Body: Partial document fields (client, lines, work items, intake items, diagnostic sections, authorization/sign-off, terms, notes). Send `intakeItems` or `diagnosticSections` as complete replacement arrays. `authorization` replaces the existing authorization object; omit a field to retain its current value.

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

Money is server-authoritative: line totals, subtotal, tax and total are recomputed on every save (see create). Fields omitted from the body keep their value; send `null` to clear `jobCardId`, `quotationId` or `depositCents`.

## Example
Request: `PUT /api/v1/documents/doc-draft-1`
```json
{
  "scope": "Battery and fan cleaning",
  "intakeItems": [{ "label": "Charger", "value": "Received with laptop" }],
  "diagnosticSections": [{ "title": "Battery test", "status": "PASSED", "details": "Replacement battery tested successfully." }],
  "authorization": {
    "technicianSignature": "Example Technician",
    "technicianDate": "2026-10-12",
    "clientSignature": "Example Client",
    "clientDate": "2026-10-12"
  },
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
