# POST /clients/{id}/restore

Restore an archived client.

## ApiClient method

`restoreClient`

## MCP tool

`restore_client`

## Request

Path parameter:

- `id` (string, required)

No request body required.

## Response

- Status: `200 OK`
- Type: `Client`
- Returns the restored client record.

## Errors

| Status | Code | Condition |
|---|---|---|
| 404 | `NOT_FOUND` | Client ID does not exist |

## Rules and side effects

The client reappears in active lists and can be selected for new documents. Existing documents remain linked.

## Example

Request: `POST /api/v1/clients/cli-1/restore`

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

Archived clients view and `restore_client`.
