# GET /clients

List active clients in the system. Archived clients are available through `GET /clients/archived`.

## ApiClient method
`listClients`

## MCP tool
`list_clients`

## Request
No parameters or request body required.

## Response
- Status: `200 OK`
- Type: `Client[]`
- Returns active client objects ordered by name.

## Errors
| Status | Code | Condition |
|---|---|---|
| 500 | `INTERNAL_ERROR` | Database query failure |

## Rules and side effects
Read-only operation. Does not modify any state.

## Example
Request: `GET /api/v1/clients`

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
Clients screen, Document editor client selector, Register client filter.
