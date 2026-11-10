# JobCard API Reference

Base path: `/api/v1`  
Protocol: JSON over HTTP  
Authentication: None (local single-user)  
Error shape: `{ "error": { "code": string, "message": string, "fields"?: Record<string, string> } }`

## Type Glossary

- `Cents`: Integer cents representing currency in South African Rand (ZAR). Example: `100000` is `R1,000.00`.
- `ISODate`: Date formatted as `YYYY-MM-DD`. Example: `2026-11-08`.
- Server-Assigned Numbers: `JC-YYYY-NNN`, `QUO-YYYY-NNN`, `INV-YYYY-NNN` (assigned on issue, null while draft, immutable once assigned).

## Endpoint Index

### Clients
- [GET /clients](clients/list.md) — List active clients
- [GET /clients/archived](clients/archived.md) — List archived clients
- [POST /clients](clients/create.md) — Create a new client
- [PUT /clients/{id}](clients/update.md) — Update an existing client
- [DELETE /clients/{id}](clients/archive.md) — Archive a client while preserving documents
- [POST /clients/{id}/restore](clients/restore.md) — Restore an archived client

### Documents
- [GET /documents](documents/list.md) — List and filter document register
- [GET /documents/{id}](documents/get.md) — Get document by ID
- [POST /documents](documents/create.md) — Create draft document
- [PUT /documents/{id}](documents/update.md) — Update draft document
- [POST /documents/{id}/issue](documents/issue.md) — Issue document and assign number
- [POST /documents/{id}/payments](documents/payments.md) — Record a payment against an invoice
- [POST /documents/{id}/status](documents/status.md) — Update document status (cancel, accept, decline)
- [POST /quotations/{id}/convert](quotations/convert.md) — Convert quotation to draft invoice
- [POST /documents/{id}/export](documents/export.md) — Export PDF or DOCX file bytes
- [POST /documents/{id}/reveal](documents/reveal.md) — Open client folder on disk

### Dashboard, Settings & Storage
- [GET /dashboard](dashboard/get.md) — Dashboard financial metrics and summaries
- [GET /settings](settings/get.md) — Get business and banking settings
- [PUT /settings](settings/update.md) — Update business and banking settings
- [GET /storage](storage/get.md) — Read-only data storage directory paths
- [GET /capabilities](capabilities/get.md) — Export engine capabilities
- [GET /register/export.xlsx](register/export.md) — Download register spreadsheet

## Security

The server accepts only localhost `Host` headers and local or same-origin `Origin`s (403 `FORBIDDEN_HOST` / `FORBIDDEN_ORIGIN`), requires `application/json` for request bodies (415 `UNSUPPORTED_MEDIA_TYPE`), and, when started with `--token`, requires `Authorization: Bearer <token>` on every `/api/` request (401 `UNAUTHORIZED`). The UI client sends the token when built with `VITE_API_TOKEN`.
