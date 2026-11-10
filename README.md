# JobCard

JobCard is a single-user job card, quotation, and invoice manager for a sole-proprietor IT repair business in South Africa. It is designed to run locally on one machine and can use either:

- an in-memory mock API for UI development and demos, or
- a native Go backend with SQLite persistence for real use. The Windows desktop EXE embeds the frontend and connects it to this backend.

The app uses ZAR, keeps money in integer cents, and keeps business logic isolated in tested pure domain functions so formatting and totals remain consistent.

---

## Quick start

### Option 1: run the UI with the built-in mock API

```bash
npm install
npm run dev
```

This starts Vite on port 3000 and uses `MockApi`, which loads `seed/seed.json` into memory. The mock resets on page reload and is intended for local development.

### Option 2: run the real backend

1. Build the Go backend:

   ```bash
   cd backend
   go build -o jobcard.exe ./cmd/jobcard
   ```

2. Start the API with seeded sample data for development:

   ```bash
   .\jobcard.exe --seed ..\seed\seed.json
   ```

3. In another terminal, run the frontend against the backend:

   ```bash
   cd ..
   set VITE_API=http
   set VITE_API_URL=http://127.0.0.1:8080
   npm run dev
   ```

   On Linux/macOS use `export` instead of `set`.

4. Optional: include a bearer token if you expose the API beyond localhost:

   ```bash
   set VITE_API_TOKEN=your-token-here
   ```

---

## Architecture and design rules

- Zero local storage for business data: all data flows through the `ApiClient` interface in [`src/api/client.ts`](src/api/client.ts).
- Strict integer-cent money model: values are stored and calculated as cents to avoid floating-point rounding errors.
- Server-authoritative numbering: document numbers such as `INV-YYYY-NNN`, `QUO-YYYY-NNN`, and `JC-YYYY-NNN` are assigned by the backend when a document is issued and never changed after that.
- Pure domain logic: totals, tax, status derivation, overdue calculations, stamps, and export capability decisions live under [`src/domain/`](src/domain/).
- Externalized seed data: only [`seed/seed.json`](seed/seed.json) contains demo business data.

---

## Available scripts

```bash
npm run dev        # Vite dev server on port 3000
npm run build      # production bundle to dist/
npm run build:desktop # frontend bundle embedded by the Go desktop EXE
npm test          # unit + contract + API docs tests
npm run test:ci    # same as npm test but exits non-zero on failure
npm run typecheck # TypeScript compiler check
npm run lint      # TypeScript validation
```

---

## Local development workflow

### Mock mode

By default JobCard uses `MockApi` and reads from `seed/seed.json` in memory. This is the fastest way to work on the UI without needing the Go service.

### Backend mode

When `VITE_API=http`, the frontend routes requests through `HttpApi`, which validates responses with `zod` and throws structured API errors when something is wrong. The contract suite in [`src/api/apiContract.test.ts`](src/api/apiContract.test.ts) is designed to work against both the mock and the real backend.

The Windows desktop build uses Vite only at build time. `npm run build:desktop` loads `.env.desktop` to select `HttpApi`, writes the static bundle to `backend/cmd/jobcard/dist`, and the Go build embeds that bundle in `backend/jobcard.exe`. At runtime, the EXE serves both the UI and `/api/v1` on the same origin; no Vite process or separate frontend port is used.

To verify the live API contract:

```bash
set VITE_API_URL=http://127.0.0.1:8080
npm test
```

---

## API documentation

The API contract and endpoint docs live under [`docs/api/`](docs/api/):

- Index and glossary: [`docs/api/README.md`](docs/api/README.md)
- Machine-readable endpoint list: [`src/api/endpoints.ts`](src/api/endpoints.ts)
- Sync test: [`docs/api/docs.test.ts`](docs/api/docs.test.ts)

These files are intended to stay aligned with the backend contract.

---

## Go backend (`backend/`)

The Go project is a single binary that exposes the JobCard HTTP API under `/api/v1` and stores data in SQLite. On Windows, build the tray version from the repository root with either script:

```powershell
.\scripts\build.ps1
# Or from Command Prompt:
scripts\build.bat
```

### Common startup examples

```bash
# development: seed sample data into an empty database
.\backend\jobcard.exe --seed .\seed\seed.json

# double-click this single-exe tray app to launch JobCard
.\scripts\build.ps1
.\backend\jobcard.exe
```

For real use, select **Move data to folder...** from the tray and choose an empty folder such as `G:\My Drive\BUSINESS\JobCard`. Use **Choose client files folder...** if client exports should live elsewhere.

The Windows build script embeds the frontend and marks the EXE as a GUI application, so it starts in the system tray without opening a terminal. The tray menu can open the web UI, open or change the data/client/templates folders, or exit JobCard. Changing the data folder moves the current data only when the destination is empty; it never merges or overwrites, and the app restarts to use the new location. Selecting a different client-files or templates folder switches to that folder without moving the old contents. The selected folders are saved in `%APPDATA%\JobCardLauncher\config.json`, and logs are written to `%APPDATA%\JobCardLauncher\jobcard.log`.

### Backend flags

| Flag | Default | Purpose |
|---|---|---|
| `--addr` | `127.0.0.1:8080` | Advanced listen-address override. Keep it on localhost unless you explicitly want to secure it with a token. |
| `--data-dir` | Remembered folder; initially `%APPDATA%\JobCard` | Advanced override. In desktop use, choose/move the data folder from the tray. |
| `--clients-dir` | Remembered folder; initially `<data-dir>\Clients` | Advanced override. In desktop use, choose the client files folder from the tray. |
| `--templates-dir` | Remembered folder; initially `<data-dir>\Templates` | Advanced override. In desktop use, choose the editable templates folder from the tray. |
| `--seed` | none | Loads `seed/seed.json` into an empty database. Development only. |
| `--today` | system date | Fixed date for deterministic testing. |
| `--static` | embedded UI | Development override for serving a UI directory from disk. |
| `--token` | none | Advanced override. Requires a valid bearer token in the Authorization header on every `/api/` request when the server is not localhost only. |
| `--mcp` | off | Runs the MCP server over stdio instead of the HTTP API; uses the remembered folders unless overridden. |
| `--tray` | on for Windows builds | Advanced override for tray mode. |
| `--open` | on | Opens the web UI at startup; the tray menu can open it again later. |

The tray replaces everyday folder switches with remembered settings. `--seed`, `--today`, and `--mcp` remain command-line-only development/integration modes; `--addr` and `--token` stay advanced security-sensitive overrides, and `--static` is for frontend development.

### Running the app against the backend

```bash
VITE_API=http VITE_API_URL=http://127.0.0.1:8080 npm run dev
```

### Verification

```bash
# seed the database and test live API behavior
cd backend
.\jobcard.exe --seed ..\seed\seed.json

# in another terminal
cd ..
VITE_API_URL=http://127.0.0.1:8080 npm run test:ci

# Go unit tests
cd backend
go test ./...
```

### Export behavior

DOCX files are generated in Go. PDF output is produced by converting the DOCX with headless Microsoft Word, so `capabilities.pdf` is only true where Word is installed. Each export also saves a copy inside the client folder.

---

## Templates, MCP, and security

### Templates

Exports are filled from editable Word templates named `invoice.docx`, `quotation.docx`, and `jobcard.docx` in the configured templates directory. Missing templates are created automatically, existing files are never overwritten, and a `README.txt` next to them lists all supported placeholders such as `{{placeholder}}`.

Open a template in Word, update the layout, fonts, colours, logo, wording, and keep the placeholders in place. A table row containing `{{line.*}}` repeats for each line item; a paragraph with `{{item.text}}` repeats for each work item. Placeholder splits caused by Word line breaks are handled automatically. To reset a template, delete the file and let the app recreate it.

### MCP

`jobcard.exe --mcp` runs the MCP server for Claude Desktop. See [`docs/mcp.md`](docs/mcp.md) for the config snippet, tool list, and safety rules. Drafts are created before permanent actions, `confirm=true` is required for irreversible operations, settings are not exposed through MCP, and an audit trail is kept.

### Security

The API is intentionally local-first and secure by default:

- it only accepts localhost/127.0.0.1/::1 hosts when bound to loopback addresses,
- it rejects cross-site browser requests that do not originate from a trusted local origin,
- it requires JSON request bodies for write operations,
- and, when the server is exposed beyond localhost, it requires a bearer token.

To listen on another address, set `--token` or `JOBCARD_TOKEN` and make every `/api/` request with a valid bearer token in the Authorization header. The UI sends that token automatically when it is built with `VITE_API_TOKEN` set.

---

## Assumptions and business rules encoded in the app

1. South African VAT status: the default `taxRate` is `0`, which represents a non-VAT vendor. The preview renders `VAT (Non-VAT Vendor): R0.00` in that case.
2. Relative seed dates: `seed/seed.json` uses values such as `today-X` and `today+X`, and the loader converts them to the current UTC date so overdue buckets remain accurate when the app loads.
3. Sequential numbering resets each year based on the issue date year.
4. Export fallback: when PDF export is unavailable, the app falls back to print mode and tells the user to select "Save as PDF".
5. Spreadsheet register export: `GET /register/export.xlsx` downloads the full document register as an Excel-compatible file.

---

## Project layout

- [`src/`](src/) — React + TypeScript frontend and business logic
- [`seed/`](seed/) — seed data for mock/development runs
- [`docs/`](docs/) — documentation and API reference files
- [`backend/`](backend/) — Go backend and SQLite persistence layer
- [`public/`](public/) — public assets and onboarding briefs

---

## Summary

If you want the fastest path:

```bash
npm install
npm run dev
```

If you want the full local business workflow with real persistence:

```bash
cd backend
go build -o jobcard.exe ./cmd/jobcard
.\jobcard.exe --seed ..\seed\seed.json
cd ..
set VITE_API=http
set VITE_API_URL=http://127.0.0.1:8080
npm run dev
```

This gives you the UI, the real backend, SQLite storage, export templates, and the same API contract the frontend is built against.
