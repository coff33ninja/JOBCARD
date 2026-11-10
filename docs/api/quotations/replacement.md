# POST /quotations/{id}/replacement

Create a linked replacement draft from an open, issued quotation.

## ApiClient method

`createReplacementQuotation`

## MCP tool

`create_replacement_quotation`

## Request

Path parameter:

- `id` (string, required): source quotation ID

No request body is required.

## Response

- Status: `201 Created`
- Type: `Doc`
- Returns a new quotation draft linked to the source by `revisionOfQuotationId`.

## Errors

| Status | Code | Condition |
|---|---|---|
| 404 | `NOT_FOUND` | Source quotation not found |
| 409 | `INVALID_STATE` | Source is not an open, issued quotation |

## Rules and side effects

- The source quotation remains open when the replacement draft is created.
- The replacement copies the source quotation's client, job-card link, and quote content. It has no number until issued.
- Issuing the replacement cancels the source quotation in the same transaction. The source remains stored as a reference record and is excluded from active totals.
- If the source quotation is no longer open when the replacement is issued, issuing the replacement is rejected.

## Example

Request: `POST /api/v1/quotations/quo-1/replacement`

Response:
```json
{
  "id": "doc-draft-2",
  "type": "quotation",
  "number": null,
  "status": "draft",
  "revisionOfQuotationId": "quo-1"
}
```

## Used by

Document detail quotation actions.
