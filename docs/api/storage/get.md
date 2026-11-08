# GET /storage

Retrieve read-only data folder paths and database location on the host machine.

## ApiClient method
`getStorage`

## MCP tool
`get_storage`

## Request
No query parameters or body required.

## Response
- Status: `200 OK`
- Type: `Storage`
- Contains `dataDir`, `dbPath`, and `clientsDir`.

## Errors
| Status | Code | Condition |
|---|---|---|
| 500 | `INTERNAL_ERROR` | Unable to resolve filesystem paths |

## Rules and side effects
Read-only inspection. The UI only displays these locations and cannot alter them.

## Example
Request: `GET /api/v1/storage`

Response:
```json
{
  "dataDir": "%APPDATA%\\JobCard",
  "dbPath": "%APPDATA%\\JobCard\\jobcard.sqlite",
  "clientsDir": "%APPDATA%\\JobCard\\Clients"
}
```

## Used by
Settings screen Data Location card.
