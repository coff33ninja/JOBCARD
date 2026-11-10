# GET /clients/archived

List archived clients.

## ApiClient method

`listArchivedClients`

## MCP tool

`list_archived_clients`

## Request

No parameters or request body required.

## Response

- Status: `200 OK`
- Type: `Client[]`
- Returns archived client objects ordered by name.

## Errors

| Status | Code | Condition |
|---|---|---|
| 500 | `INTERNAL_ERROR` | Database query failure |

## Rules and side effects

Read-only operation. Archived clients remain linked to their existing documents but cannot be used for new documents until restored.

## Example

Request: `GET /api/v1/clients/archived`

Response:
```json
[
  {
    "id": "cli-1",
    "name": "Apex Logistics",
    "contact": "John Smith",
    "phone": "0821234567",
    "email": "billing@apex.co.za",
    "address": "12 High St, Cape Town",
    "notes": "Gate code 4040"
  }
]
```

## Used by

Archived clients view and document history.
