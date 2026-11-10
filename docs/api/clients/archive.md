# DELETE /clients/{id}

Archive a client without deleting the client record or its documents.

## ApiClient method

`archiveClient`

## MCP tool

`archive_client`

## Request

Path parameter:

- `id` (string, required)

No request body required.

## Response

- Status: `200 OK`
- Type: `Client`
- Returns the archived client record.

## Errors

| Status | Code | Condition |
|---|---|---|
| 404 | `NOT_FOUND` | Client ID does not exist |

## Rules and side effects

The client is removed from active lists and cannot be selected for new documents. Existing documents remain linked and readable. Use `POST /clients/{id}/restore` to reverse this action.

## Example

Request: `DELETE /api/v1/clients/cli-1`

Response:
```json
{
  "id": "cli-1",
  "name": "Apex Logistics",
  "contact": "John Smith",
  "phone": "0821234567",
  "email": "billing@apex.co.za",
  "address": "12 High St, Cape Town",
  "notes": "Gate code 4040"
}
```

## Used by

Client directory and `archive_client`.
