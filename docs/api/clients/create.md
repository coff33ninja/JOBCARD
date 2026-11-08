# POST /clients

Create a new client profile.

## ApiClient method
`createClient`

## MCP tool
`create_client`

## Request
Body:
- `name` (string, required, 1-120 chars)
- `contact` (string, optional)
- `phone` (string, optional)
- `email` (string, optional)
- `address` (string, optional)
- `notes` (string, optional)

## Response
- Status: `201 Created`
- Type: `Client`
- Returns the newly created client record with assigned `id`.

## Errors
| Status | Code | Condition |
|---|---|---|
| 400 | `VALIDATION_FAILED` | Missing name or invalid data format |

## Rules and side effects
Generates a unique client ID. Creates client folder in storage when supported.

## Example
Request: `POST /api/v1/clients`
```json
{
  "name": "Beacon Media",
  "contact": "Sarah Connor",
  "phone": "0835551234",
  "email": "accounts@beacon.co.za",
  "address": "44 Long St, Cape Town",
  "notes": "Net 30 days"
}
```

Response:
```json
{
  "id": "cli-2",
  "name": "Beacon Media",
  "contact": "Sarah Connor",
  "phone": "0835551234",
  "email": "accounts@beacon.co.za",
  "address": "44 Long St, Cape Town",
  "notes": "Net 30 days"
}
```

## Used by
Clients screen modal, Document editor inline new client modal.
