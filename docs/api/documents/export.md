# POST /documents/{id}/export

Export an issued document as PDF or DOCX file bytes.

## ApiClient method
`exportDocument`

## MCP tool
`export_document`

## Request
Path parameter:
- `id` (string, required): document ID

Query parameter:
- `format` ("pdf" | "docx", required): export file format

## Response
- Status: `200 OK`
- Type: `Blob` (raw file bytes)
- Headers:
  - `Content-Type`: `application/pdf` or `application/vnd.openxmlformats-officedocument.wordprocessingml.document`
  - `Content-Disposition`: `attachment; filename="INV-2026-001.pdf"`

## Errors
| Status | Code | Condition |
|---|---|---|
| 400 | `DRAFT_CANNOT_EXPORT` | Document is a draft |
| 404 | `NOT_FOUND` | Document not found |
| 501 | `FORMAT_NOT_SUPPORTED` | Backend capability for requested format is false |

## Rules and side effects
- Generates file rendered from document data and business templates.
- Backend saves a copy directly to the client's folder on disk.

## Example
Request: `POST /api/v1/documents/doc-1/export?format=pdf`

Response: Binary PDF stream (`%PDF-1.4...`)

## Used by
Document View ExportMenu ("Download PDF", "Download DOCX").
