# POST /documents/{id}/reveal

Open the client folder for a document in the operating system file manager (e.g. Windows Explorer).

## ApiClient method
`revealFolder`

## MCP tool
`reveal_folder`

## Request
Path parameter:
- `id` (string, required): document ID

Body: empty.

## Response
- Status: `200 OK`
- Type: `{ "success": boolean, "path": string }`
- Returns confirmed folder path that was opened.

## Errors
| Status | Code | Condition |
|---|---|---|
| 404 | `NOT_FOUND` | Document or client folder not found |
| 501 | `NOT_SUPPORTED` | Reveal in folder capability is false |

## Rules and side effects
Instructs the host system to launch the native file manager focusing on the client's folder.

## Example
Request: `POST /api/v1/documents/doc-1/reveal`

Response:
```json
{
  "success": true,
  "path": "C:\\Users\\User\\JobCard\\Clients\\Apex Logistics"
}
```

## Used by
Document View ExportMenu ("Show in folder").
