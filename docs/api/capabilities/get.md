# GET /capabilities

Query what export formats and file system actions the backend currently supports.

## ApiClient method
`getCapabilities`

## MCP tool
`get_capabilities`

## Request
No query parameters or body required.

## Response
- Status: `200 OK`
- Type: `Capabilities`
- Booleans: `docx`, `pdf`, `revealInFolder`.

## Errors
| Status | Code | Condition |
|---|---|---|
| 500 | `INTERNAL_ERROR` | Unable to detect host capabilities |

## Rules and side effects
Read-only. Used by the UI `exportOptions` pure decision function to enable/disable or provide fallbacks for export actions.

## Example
Request: `GET /api/v1/capabilities`

Response:
```json
{
  "docx": true,
  "pdf": true,
  "revealInFolder": true
}
```

## Used by
Document View ExportMenu.
