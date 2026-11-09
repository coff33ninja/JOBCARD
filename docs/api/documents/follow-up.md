# POST /documents/{id}/follow-up

Set or clear the payment follow-up date for an issued invoice. This is used to flag an invoice that needs a manual follow-up reminder.

## ApiClient method
`setFollowUp`

## MCP tool
`set_follow_up`

## Request
Path parameter:
- `id` (string, required): document ID

Body:
- `followUpDate` (string or null, required): ISO date for the follow-up reminder, or `null` to clear it

## Response
- Status: `200 OK`
- Type: `Doc`
- Returns the updated document with the new `followUpDate` value.

## Errors
| Status | Code | Condition |
|---|---|---|
| 400 | `VALIDATION_FAILED` | `followUpDate` is not a valid `YYYY-MM-DD` date or `null` |
| 404 | `NOT_FOUND` | Document not found |
| 409 | `CONFLICT` | Follow-up date is being set on a document that is not an issued invoice |

## Rules and side effects
Updates the follow-up reminder date only on an issued invoice. Clearing the date removes the reminder without changing the invoice's financial state.

- `followUpDate` accepts either a valid ISO date or `null`.
- A follow-up date is only valid for issued invoices; quotations and job cards reject it.
- The dashboard counts overdue unpaid invoices with a due follow-up reminder when they are past the scheduled date.

## Example
Request: `POST /api/v1/documents/doc-1/follow-up`
```json
{
  "followUpDate": "2025-05-15"
}
```

Response:
```json
{
  "id": "doc-1",
  "status": "sent",
  "followUpDate": "2025-05-15"
}
```

## Used by
Invoice detail actions for setting or clearing payment follow-up reminders.
