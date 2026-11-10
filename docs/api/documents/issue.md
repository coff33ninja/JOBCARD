# POST /documents/{id}/issue

Issue a draft document, assigning permanent number and dates.

## ApiClient method
`issueDocument`

## MCP tool
`issue_document`

## Request
Path parameter:
- `id` (string, required): document ID

Optional body:
- `issueDate` (ISODate, optional): defaults to current date if omitted.

## Response
- Status: `200 OK`
- Type: `Doc`
- Returns issued document with assigned sequential `number`, calculated dates, and status set to `sent` (or `open` for job card).

## Errors
| Status | Code | Condition |
|---|---|---|
| 404 | `NOT_FOUND` | Document ID not found |
| 409 | `ALREADY_ISSUED` | Document has already been issued |
| 400 | `VALIDATION_FAILED` | Invoice or quotation has no line items, or `issueDate` is not `YYYY-MM-DD` |
| 409 | `INVALID_STATE` | Document is cancelled |

## Rules and side effects
- Assigns sequential formatted number (`JC-YYYY-NNN`, `QUO-YYYY-NNN`, `INV-YYYY-NNN`).
- Year in number is derived from the issue date.
- Computes `dueDate` (invoice) or `validUntil` (quotation) using settings terms.
- Fills a blank technician name from the Settings `person` value and a blank technician date from the issue date; manually entered values are preserved.
- Status transitions from `draft` to `sent` (or `open` for jobcard).
- Number and issue date become permanently immutable.

## Example
Request: `POST /api/v1/documents/doc-draft-1/issue`

Response:
```json
{
  "id": "doc-draft-1",
  "type": "invoice",
  "number": "INV-2026-002",
  "issueDate": "2026-11-08",
  "dueDate": "2026-11-15",
  "status": "sent"
}
```

## Used by
Document Editor "Issue" button, Document View issue confirmation.
