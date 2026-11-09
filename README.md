# JobCard — IT Repair & Document Management Desk

A single-user job card, quotation, and invoice management web application built for sole-proprietor IT repair businesses in South Africa (currency: ZAR).

Designed to run locally on a single machine, interfacing with either an in-memory development mock (`MockApi`) or a native Go backend (`HttpApi`) storing persistent data in SQLite.

---

## Architecture & Design Rules

- **Zero LocalStorage as Data Store:** All application data flows exclusively through the `ApiClient` interface (`src/api/client.ts`).
- **Strict Money in Integer Cents:** Money values (`Cents = number`) are stored and computed in integer cents throughout code and API payloads to prevent floating-point rounding errors. Currency is formatted only at display boundaries as `R1,000.00`.
- **Server-Authoritative Numbering:** Document sequential numbers (`INV-YYYY-NNN`, `QUO-YYYY-NNN`, `JC-YYYY-NNN`) are assigned by the server upon issuing and are immutable once issued. Drafts have `number: null`.
- **Pure Domain Logic:** Totals, tax calculations, invoice and quotation status derivations, stamps, days overdue, and export capability options are isolated in tested pure functions under `src/domain/`.
- **Externalized Seed Data:** All mock test and demonstration data lives strictly in `seed/seed.json` and is imported via `loadSeed()`.

---

## Available Scripts

- `npm run dev`: Starts the Vite development server on port 3000.
- `npm run build`: Builds the production bundle to `dist/`.
- `npm test`: Runs the complete unit, contract, and documentation test suite via `tsx --test`.
- `npm run test:ci`: Runs the full test suite and exits non-zero on any failure.
- `npm run typecheck`: Runs TypeScript compiler (`tsc --noEmit`) to verify strict typing.
- `npm run lint`: Runs syntax and type validation.

---

## How to Switch to the Real Backend

By default, JobCard runs against `MockApi`, which loads `seed/seed.json` in memory and resets on page reload.

To connect JobCard to the Go SQLite backend:

1. Create or update `.env.local` (or set environment variables):
   ```bash
   VITE_API=http
   VITE_API_URL=http://localhost:8080
   ```
2. Start the Go backend on `http://localhost:8080`.
3. Restart or rebuild the frontend:
   ```bash
   npm run dev
   ```

When `VITE_API=http`, all calls are routed through `HttpApi`, which validates all responses with `zod` and throws structured `ApiError` instances. The contract test suite (`src/api/apiContract.test.ts`) can also be run against the live backend by executing:
```bash
VITE_API_URL=http://localhost:8080 npm test
```

---

## API Reference Documentation

Complete endpoint documentation is maintained under `docs/api/`:
- Index & Type Glossary: [`docs/api/README.md`](docs/api/README.md)
- Machine-readable registry: [`src/api/endpoints.ts`](src/api/endpoints.ts)
- Automated synchronization test: [`docs/api/docs.test.ts`](docs/api/docs.test.ts)

---

## List of Assumptions Made

1. **South African VAT vendor status:** Default settings specify `taxRate: 0` representing a non-VAT vendor. The invoice preview explicitly renders `"VAT (Non-VAT Vendor): R0.00"`. If `taxRate` is set to `0.15`, standard 15% VAT is calculated.
2. **Relative seed date resolution:** Dates in `seed/seed.json` use `"today-X"` and `"today+X"` relative strings. `loadSeed()` resolves these to the current UTC date so overdue and ageing buckets are always active and accurate whenever loaded.
3. **Sequential numbering resets:** Numbering resets annually based on the issue date year (`YYYY`).
4. **Export fallbacks:** When `capabilities.pdf` is false, the export menu seamlessly falls back to print mode with instructions for the user to select "Save as PDF".
5. **Spreadsheet register export:** `GET /register/export.xlsx` downloads the full documents register as an Excel-compatible file.

## Go backend (`backend/`)

One Go binary: HTTP API under `/api/v1` plus SQLite storage. It is built to pass `src/api/apiContract.test.ts` unchanged.

```
cd backend
go build -o jobcard.exe ./cmd/jobcard
.\jobcard.exe --seed ..\seed\seed.json          # development: sample data into an EMPTY database
.\jobcard.exe --clients-dir "G:\My Drive\BUSINESS\Clients"   # real use: clean database, client folders on Drive
```

| Flag | Default | Purpose |
|---|---|---|
| `--addr` | `127.0.0.1:8080` | Listen address. Keep it on localhost, there is no authentication. |
| `--data-dir` | `%APPDATA%\JobCard` | Database folder (`jobcard.sqlite`), per user. |
| `--clients-dir` | `<data-dir>\Clients` | Client folders and exported docx/pdf copies. |
| `--seed` | none | Development only. Loads `seed/seed.json`, and only into an empty database. |
| `--today` | system date | Fixed date for testing. |
| `--static` | none | Serve a built UI (`dist`) at `/`. |

Run the UI against it: `VITE_API=http VITE_API_URL=http://127.0.0.1:8080 npm run dev`.
Verify the backend: start it with `--seed`, then `VITE_API_URL=http://127.0.0.1:8080 npm run test:ci`. Go unit tests: `cd backend && go test ./...`.

Exports: DOCX is generated in Go. PDF is made by converting that DOCX with headless Microsoft Word, so `capabilities.pdf` is true only where Word is installed. Each export also saves a copy into the client folder.

## Templates, MCP and security

**Templates.** Exports are filled from editable Word templates: `invoice.docx`, `quotation.docx`, `jobcard.docx` in `--templates-dir` (default `<data-dir>\Templates`, e.g. `--templates-dir "G:\My Drive\BUSINESS\Templates"`). Missing files are created from the built-in layout, existing files are never overwritten, and a `README.txt` next to them lists every `{{placeholder}}`. Open a template in Word, change fonts/colours/logo/wording, keep the placeholders. A table row containing `{{line.*}}` repeats per line item; a paragraph with `{{item.text}}` repeats per work item. Placeholders split across runs by Word are handled. To reset a layout, delete the file. An unrecognised placeholder stays visible in the output and is logged.

**MCP.** `jobcard.exe --mcp` runs the MCP server for Claude Desktop. See `docs/mcp.md` for the config snippet, the tool list, and the safety rules (drafts first, `confirm=true` for permanent steps, no settings access, audit log).

**Security.** The API has no login by default and refuses to leave localhost: it only accepts `localhost`/`127.0.0.1` Host headers (DNS rebinding), rejects cross-site `Origin`s (browser CSRF), and requires `application/json` bodies. To listen on another address you must set `--token` (or `JOBCARD_TOKEN`); every `/api/` request then needs `Authorization: Bearer <token>`, and the UI sends it when built with `VITE_API_TOKEN`. MCP runs as your own local process and needs no token.

| Extra flag | Default | Purpose |
|---|---|---|
| `--templates-dir` | `<data-dir>\Templates` | Editable `.docx` templates |
| `--token` | none (or `JOBCARD_TOKEN`) | Bearer token; mandatory when `--addr` is not localhost |
| `--mcp` | off | Run as an MCP server on stdio instead of HTTP |
