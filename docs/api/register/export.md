# GET /register/export.xlsx

Download the complete documents register as a spreadsheet (.xlsx file).

## ApiClient method
`exportRegisterSpreadsheet`

## MCP tool
`export_register_spreadsheet`

## Request
No parameters required.

## Response
- Status: `200 OK`
- Type: `Blob` (raw Excel binary file)
- Headers:
  - `Content-Type`: `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
  - `Content-Disposition`: `attachment; filename="register-2026.xlsx"`

## Errors
| Status | Code | Condition |
|---|---|---|
| 500 | `INTERNAL_ERROR` | Spreadsheet generation failure |

## Rules and side effects
Generates and downloads an Excel workbook summarizing all invoices, quotations, and job cards.

## Example
Request: `GET /api/v1/register/export.xlsx`

Response: Binary Excel file stream (`PK...`)

## Used by
Register screen "Export (.xlsx)" button, Settings screen "Download register (.xlsx)" button.
