# PUT /clients/{id}

Update an existing client profile.

## ApiClient method
`updateClient`

## MCP tool
`update_client`

## Request
Path parameter:
- `id` (string, required)

Body:
- `name` (string, required)
- `contact` (string, optional)
- `phone` (string, optional)
- `email` (string, optional)
- `address` (string, optional)
- `notes` (string, optional)

## Response
- Status: `200 OK`
- Type: `Client`
- Returns the updated client record.

## Errors
| Status | Code | Condition |
|---|---|---|
| 400 | `VALIDATION_FAILED` | Missing name |
| 404 | `NOT_FOUND` | Client ID does not exist |

## Rules and side effects
Updates the client information in database. Documents referencing this client reflect the updated details.

## Example
Request: `PUT /api/v1/clients/cli-1`
```json
{
  "name": "Apex Logistics (Pty) Ltd",
  "contact": "John Smith",
  "phone": "0821234567",
  "email": "accounts@apex.co.za",
  "address": "12 High St, Cape Town",
  "notes": "Updated accounts email"
}
```

Response:
```json
{
  "id": "cli-1",
  "name": "Apex Logistics (Pty) Ltd",
  "contact": "John Smith",
  "phone": "0821234567",
  "email": "accounts@apex.co.za",
  "address": "12 High St, Cape Town",
  "notes": "Updated accounts email"
}
```

## Used by
Clients screen edit modal.
