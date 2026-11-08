# JobCard — UI build brief for AI Studio

Paste everything below the line into AI Studio as the build instructions.

---

## 0. What you are building

A small, single-user **job card, quotation and invoice manager** for a sole-proprietor IT repair business in South Africa (currency ZAR). It runs locally on one Windows machine; a separate Go backend (written by someone else) serves `/api/v1` and stores everything in SQLite.

**Your job is the UI only.** You build it against a fixed API contract (section 4) using a mock implementation. The real backend plugs in later by switching one environment variable. Do not build a backend, do not invent API endpoints or fields, and do not add features that are not listed here.

Hard rules:
- **Minimal and literal.** Build exactly what is listed. No extra pages, no auth, no multi-user, no POS, no inventory, no WhatsApp bridge, no charts library beyond what the Dashboard needs.
- **Stack:** React + TypeScript (strict) + Vite + Tailwind. Tests: Vitest + React Testing Library. No state-management library, no UI kit. Allowed extras: `zod` for validating API responses, nothing else without a stated reason.
- **No `localStorage`/`sessionStorage` as a data store.** All data goes through the API client. The mock keeps data in memory only and resets on reload.
- **No hard-coded business details** (name, phone, bank, terms) in components. They come from `GET /settings`. The placeholder settings live in the seed file (section 8). Never use real banking details.
- **Sample data is never written in code.** No hand-written arrays or objects of clients, documents, amounts, names or settings in any `.ts`/`.tsx` file. All sample data lives in `seed/seed.json` (section 8).
- **MCP is not your job.** Do not build MCP servers, tool definitions or AI-agent features. The Go backend exposes the same operations as MCP tools later, so keep every UI action a single, coarse, named `ApiClient` method (issue, convert, addPayment, export...) that maps to one endpoint. Never call anything except `ApiClient`.
- Money is **integer cents** everywhere in code and in the API. Format only at the display edge as `R1,000.00` (the style of the existing invoices), not locale-spaced.

## 1. Test-driven by default (mandatory process)

For **every** feature below: write the failing test first, run it and show it fail (red), implement the minimum code, run it and show it pass (green), then refactor. Do not move to the next feature until `npm test` is fully green and `npm run typecheck` is clean. If a requirement is ambiguous, write the test that encodes your reading of it and note the assumption in the reply.

Required test layers:
1. **Pure logic unit tests** (no React): totals/tax/balance maths, status and stamp derivation, days overdue, due-date calculation, money formatting, quote-to-invoice conversion, export-capability decision (section 6).
2. **Component tests:** document form, line-item editor, payment dialog, stamp display, status pill, dashboard cards, export menu.
3. **API contract suite** `src/api/apiContract.test.ts`: one test suite parametrised over an `ApiClient` factory. It runs against `MockApi` now, and must run unchanged against `HttpApi` when `VITE_API_URL` is set (skipped otherwise). This suite is how the real backend will be verified, so make it thorough: numbering, validation errors, payment effects, status transitions, immutability of issued numbers.
4. A `npm run test:ci` script that runs everything once and exits non-zero on failure.
5. **API docs sync test** `docs/api/docs.test.ts` (section 9): fails if any endpoint lacks a doc file, any doc describes an endpoint that does not exist, or a doc is missing a required section. Write this test first, before the first endpoint.

Definition of done for the whole project: all tests green, typecheck clean, every screen in section 5 reachable and working against `MockApi`, one API doc per endpoint in `docs/api/` (section 9), and `README.md` explaining how to switch to the real backend.

## 2. Domain rules (encode these as tested pure functions in `src/domain/`)

**Document types:** `quotation`, `invoice`, `jobcard`. A job card is the internal record of a job (reference number, client, device/reference, scope, status of work); quotations and invoices are linked to it.

**Numbers are assigned by the server, never by the UI.** Formats: `JC-YYYY-NNN`, `QUO-YYYY-NNN`, `INV-YYYY-NNN` (NNN zero-padded, restarts each year, year from the issue date). A draft has `number: null` until issued. Once issued, a number never changes. `MockApi` must follow the same rules so the UI cannot depend on anything else.

**Line item:** `description`, `qty` (number, >0), `unitPriceCents`, `totalCents = round(qty * unitPriceCents)`.

**Totals:** `subtotal = sum(lines)`, `taxCents = round(subtotal * taxRate)` where `taxRate` comes from settings (default `0`, the business is a non-VAT vendor and the document then shows "VAT (Non-VAT Vendor): R0.00"), `total = subtotal + tax`, `paid = sum(payments)`, `balance = total - paid`.

**Dates:** `issueDate`, `dueDate` (invoice) or `validUntil` (quotation). Default terms days come from settings (`defaultTermsDays`, default 7). Due date = issue date + terms days unless edited. Quotation validity default from settings (`quoteValidityDays`, default 14).

**Invoice status** (stored values): `draft | sent | partially_paid | paid | overdue | cancelled`. Derived display rules, implemented as one pure function `deriveInvoiceStatus(invoice, today)` and tested:
- cancelled stays cancelled; draft stays draft until issued.
- `balance <= 0` and total > 0 → `paid`.
- `0 < paid < total` → `partially_paid` (becomes `overdue` if past due date and balance > 0).
- `paid == 0`, issued, past due date, balance > 0 → `overdue`; otherwise `sent`.
- `daysOverdue = today - dueDate` when balance > 0 and dueDate < today, else `null`.

**Quotation status:** `draft | sent | accepted | declined | expired | converted`. `expired` is derived when past `validUntil` and not accepted/declined/converted. **Convert to invoice:** pure function that copies client, job card reference and line items into a new draft invoice, links `quotationId`, never mutates the quotation except setting it to `converted`.

**Stamp** (pure function `stampFor(document, today)`; text exactly as shown): invoices → `PAID` (green), `PARTIALLY PAID` (amber), `UNPAID` (red; issued, not past due), `OVERDUE` (red, stronger), `DRAFT` (grey), `CANCELLED` (grey, struck). Quotations → `DRAFT`, `ACCEPTED` (green), `DECLINED` (red), `EXPIRED` (grey). Job cards have no stamp. The stamp appears on the on-screen preview, in print, and is also requested from the backend export (the backend renders it on the file; the UI only shows it in preview/print).

## 3. Document layout the preview must match

The client receives documents that look like the business's existing Word templates. The in-app preview is an HTML approximation of the same sections, in this order, A4 portrait, calm navy/grey styling, with the stamp as a rotated outlined label top-right of the page. Final PDF/DOCX files are produced by the backend from the real templates; the preview only needs to be faithful in section order and wording.

**Header (from settings):** business name (large), line with `person | phone | email`, location.

**Invoice** — title `INVOICE & SERVICE SUMMARY`. Sections and fields:
1. Number, invoice date, payment due date, related quotation number (or "Not applicable").
2. `CLIENT & JOB DETAILS`: client name/business, contact person, phone, billing/delivery email, billing address, job/device reference (the job card number plus device/serial text), service scope.
3. `COMPLETED SERVICES & WORK DONE`: list of `Item: what was done and outcome`.
4. `FINANCIAL BREAKDOWN`: table `Description | Qty | Unit price (R) | Total (R)`, then right-aligned Subtotal, VAT line, **TOTAL INVOICE AMOUNT**, Payments received, **BALANCE DUE**.
5. `PAYMENT DETAILS` (from settings): bank name, account holder, account number, branch code, payment reference `INV-YYYY-NNN / Client Name`, payment terms.
6. `NOTES & RECOMMENDATIONS`.

**Quotation** — title `QUOTATION & PROPOSED SERVICES`. Same header and client section, with: quotation number, date, validity days and valid-until date; `PROPOSED SERVICES & WORK TO BE DONE` (items, exclusions/assumptions, estimated turnaround); financial table with **TOTAL QUOTED AMOUNT** and "Deposit, if agreed"; payment details with reference `QUO-YYYY-NNN / Client Name` and payment/deposit terms; `TERMS & CLIENT ACCEPTANCE` (terms text, client representative, acceptance signature/approval reference, acceptance date); notes.

**Job card** — internal page: job card number, client, device/reference, scope, work log notes, linked quotation(s) and invoice(s), job status (`open | in_progress | completed | invoiced | closed`).

## 4. API contract (the only things you may call)

Base path `/api/v1`, JSON, no auth. Define these exact types in `src/api/types.ts` and the interface `ApiClient` in `src/api/client.ts`; implement `MockApi` (in memory, seeded, same validation and numbering rules as the real server) and `HttpApi` (`fetch`). Select with `VITE_API=mock|http` (default `mock`) and `VITE_API_URL` (default same origin). Validate every `HttpApi` response with zod.

```ts
type Cents = number;
type ISODate = string;            // "2026-11-08"

interface Client { id: string; name: string; contact: string; phone: string; email: string; address: string; notes: string; }

interface LineItem { description: string; qty: number; unitPriceCents: Cents; totalCents: Cents; }
interface Payment  { id: string; date: ISODate; amountCents: Cents; method: string; reference: string; }

type DocType = "quotation" | "invoice" | "jobcard";
interface Doc {
  id: string; type: DocType;
  number: string | null;            // null while draft; server-assigned on issue
  clientId: string; jobCardId: string | null; quotationId: string | null;
  issueDate: ISODate | null; dueDate: ISODate | null; validUntil: ISODate | null;
  reference: string;                // device / serial / job reference text
  scope: string;
  workItems: { title: string; detail: string }[];   // completed or proposed work
  lines: LineItem[];
  taxCents: Cents; subtotalCents: Cents; totalCents: Cents;
  depositCents: Cents | null;       // quotations
  payments: Payment[];              // invoices
  status: string;                   // stored status (see domain rules)
  terms: string; exclusions: string; turnaround: string; notes: string;
  folderPath: string | null;        // client folder on disk, set by backend
  createdAt: string; updatedAt: string;
}

interface Settings {
  businessName: string; person: string; phone: string; email: string; location: string;
  bankName: string; accountHolder: string; accountNumber: string; branchCode: string;
  taxRate: number;                  // 0 = non-VAT vendor
  defaultTermsDays: number; quoteValidityDays: number;
}
interface Storage { dataDir: string; dbPath: string; clientsDir: string; }
interface Capabilities { docx: boolean; pdf: boolean; revealInFolder: boolean; }
interface DashboardSummary {
  year: number;
  totals: { invoicedCents: Cents; paidCents: Cents; outstandingCents: Cents; overdueCents: Cents };
  statusCounts: Record<string, number>;       // draft, sent, partially_paid, paid, overdue, cancelled
  attention: { followUpsDue: number; dueNext7Count: number; dueNext7Cents: Cents };
  ageing: { d1_30: Cents; d31_60: Cents; d61_90: Cents; d90plus: Cents };
  monthly: { month: number; jobs: number; invoicedCents: Cents; paidCents: Cents; outstandingCents: Cents }[]; // 12 rows
  nextNumbers: { jobcard: string; quotation: string; invoice: string };
}
```

| Method + path | Purpose |
|---|---|
| `GET /clients`, `POST /clients`, `PUT /clients/{id}` | client CRUD (no delete in v1) |
| `GET /documents?type=&status=&clientId=&q=&from=&to=` | list/filter (register) |
| `GET /documents/{id}`, `POST /documents`, `PUT /documents/{id}` | read, create draft, edit draft |
| `POST /documents/{id}/issue` | assigns number + dates, status → sent; 409 if already issued |
| `POST /documents/{id}/payments` | body `{date, amountCents, method, reference}`; rejects overpayment (422) |
| `POST /documents/{id}/status` | body `{status}`; for cancel, accept/decline |
| `POST /quotations/{id}/convert` | returns new draft invoice |
| `POST /documents/{id}/export?format=docx\|pdf` | returns file bytes; also saves a copy into the client folder; `501` if format unsupported |
| `POST /documents/{id}/reveal` | opens the client folder in the file manager |
| `GET /dashboard?year=YYYY` | `DashboardSummary` |
| `GET /settings`, `PUT /settings` | business settings |
| `GET /storage` | read-only data locations |
| `GET /capabilities` | what the backend can export |
| `GET /register/export.xlsx` | register as a spreadsheet download |

Error shape for all 4xx: `{ "error": { "code": string, "message": string, "fields"?: Record<string,string> } }`. `MockApi` throws the same.

## 5. Screens (v1 — nothing else)

1. **Dashboard:** four total cards (invoiced, paid, outstanding, overdue), status counts, "Needs attention" (follow-ups due, due in next 7 days count and amount), overdue ageing buckets, monthly table for a year selector, and "next numbers" (job card, quotation, invoice). Every figure comes from `GET /dashboard`; the UI does no money maths here. Renders from day 1 against the mock.
2. **Register:** table of invoices and quotations (toggle) with columns number, client, service, issue date, due date, total, paid, balance, status pill, days overdue. Filters by status, client, date range, text search. Status colours: overdue red, partially paid amber, paid green. Row opens the document.
3. **Document editor:** create/edit drafts for all three types, client picker with inline "new client", reference, scope, work items, line-item editor with live totals (using the pure functions), dates, terms, notes. Buttons: Save draft, Issue (confirms: "Number can't be changed after issuing"), Convert to invoice (quotations).
4. **Document view:** A4 preview with stamp, a payments panel (add payment dialog, list), status actions, and the export menu (section 6).
5. **Clients:** list, add/edit, shows the client's documents and balance.
6. **Settings:** business details, bank details, tax rate, default terms days, quote validity days. A read-only **Data location** card from `GET /storage` showing data folder, database file and clients folder, with copy-to-clipboard buttons and a "Download register (.xlsx)" button. (The backend stores data under the current user's profile, e.g. `%APPDATA%\JobCard`, with clients folders configurable on the backend side. The UI only displays this; it never changes it.)

Keyboard/UX minimums: usable on a phone width, forms validate inline, destructive or irreversible actions (issue, cancel) ask for confirmation.

## 6. Export, print and share (TDD this logic first)

Single component `ExportMenu` on the document view. Decision logic lives in a pure function `exportOptions(capabilities, env)` with unit tests; the menu just renders its result.

Actions, in this order:
1. **Download PDF** — `POST /documents/{id}/export?format=pdf`; if `capabilities.pdf` is false, fall back to **Print** and tell the user to choose "Save as PDF".
2. **Download DOCX** — `export?format=docx` when `capabilities.docx`.
3. **Print** — `window.print()` with a print stylesheet that hides all app chrome and prints only the A4 preview including the stamp.
4. **Share…** — shown only when `navigator.canShare({ files: [file] })` is true: fetch the PDF as a `File` and call `navigator.share({ files, title, text })` so the operating system's share sheet offers whatever is registered on the machine (mail, WhatsApp, etc.).
5. **Email** — fallback when Share isn't available: download the PDF, then open a `mailto:` link with the client's email, subject (`Invoice INV-2026-001 from <business>`) and a short prefilled body. Show a one-line notice: "Attach the downloaded PDF". (`mailto:` cannot attach files; do not pretend otherwise.)
6. **WhatsApp** — fallback: download the PDF, then open `https://wa.me/<digits-only phone>?text=<prefilled message>`. Same one-line notice. Normalise South African numbers (leading `0` → `27`) in a tested pure function `toWhatsAppNumber`.
7. **Show in folder** — only when `capabilities.revealInFolder`: `POST /documents/{id}/reveal`.
8. **Download register (.xlsx)** is on Register and Settings, not here.

Rules: each action shows loading and failure states, never silently fails, and never sends anything automatically. Drafts can be previewed and printed with a DRAFT stamp but the export buttons (except Print) require an issued document.

## 7. Order of work

1. Project scaffold, test tooling, `types.ts`, `ApiClient`, `MockApi`, contract suite (red to green). `seed/seed.json` plus its single `loadSeed()` come in this step. Each endpoint gets its doc file in the same step it is implemented (section 9), not at the end.
2. `src/domain/` pure functions with tests (section 2).
3. Dashboard, Register, Clients, Settings.
4. Document editor and document view with preview and stamps.
5. Export menu (section 6).
6. README: scripts, how to run against the real backend (`VITE_API=http`, `VITE_API_URL`), and a list of every assumption you made.

Reply at the end of each step with: tests added, tests passing count, anything you deviated from, and anything you could not do.

## 8. Seed data (all mock data lives in a data file, never in code)

- **One file:** `seed/seed.json`. It holds the placeholder settings (`Example Business`, `000 000 0000`, `Example Bank`, `0000000000` — never real details), about 6 clients, and about 14 documents. Use field names exactly as in `types.ts`.
- **Coverage (so every screen has something to show):** invoices that are paid, partially paid, unpaid and not yet due, overdue in each ageing bucket (1-30, 31-60, 61-90, 90+), draft and cancelled; quotations that are draft, sent, accepted, expired and converted; job cards in every status; payments on the paid and partially paid invoices.
- **Relative dates:** dates in the seed are strings like `"today-45"` or `"today+7"`, resolved when loaded, so overdue and ageing figures stay correct on any day.
- **Stored values only:** the seed omits derived fields (`subtotalCents`, `taxCents`, `totalCents`). The loader computes them with the domain functions from section 2. Issued documents carry explicit numbers.
- **Single entry point:** `MockApi` fills its in-memory store from the seed through one `loadSeed()` function at startup. No other code path puts data in. Do not add a browser SQL engine (no sql.js or similar).
- **Backend contract:** the Go backend imports this same file into SQLite only when started with an explicit `--seed` flag for development. A real database always starts empty, so the UI must never assume seed data exists. Every screen needs a tested empty state.
- **Numbering after seeding:** next numbers continue after the highest seeded number per type and year. This is tested.
- **Tests:** tests create what they need through `ApiClient` calls or a purpose-built `seed/test-*.json` loaded with `loadSeed()`. Tests must not contain inline business-data literals in components or domain code. Small numeric inputs to pure-function tests are fine.
## 9. API reference docs (one per endpoint, kept in sync)

Every endpoint in section 4 gets its own reference document. These are the long-term reference for anyone (human or AI) who works on the UI or the backend later, so they must be accurate, short and uniform.

**Files**
- `src/api/endpoints.ts`: the single machine-readable registry. One entry per endpoint: `{ method, path, clientMethod, mcpTool, doc }`. `HttpApi` builds its requests from this registry, and the docs test checks against it. `mcpTool` is a reserved snake_case name in `verb_noun` form (for example `list_clients`, `issue_document`, `add_payment`); you do not implement it.
- `docs/api/README.md`: index table of all endpoints (method, path, one-line purpose, link), grouped by resource, plus the shared error shape and the shared type glossary (money in integer cents, `ISODate`, server-assigned numbers).
- `docs/api/<resource>/<operation>.md`: one file per endpoint, for example `docs/api/documents/issue.md`.

**Required sections in every endpoint doc, in this order** (the docs test checks the headings):
1. `# METHOD /path` as the title, followed by a one-line summary.
2. `## ApiClient method` and `## MCP tool`: the exact `clientMethod` and `mcpTool` names from the registry.
3. `## Request`: path params, query params and body, each field with type, required or optional, and constraints (min, max, allowed values).
4. `## Response`: success status code, response type, and what each field means; for file responses, the content type and headers.
5. `## Errors`: a table of every status code, error `code`, and the exact condition that causes it (validation, 404, 409, 422, 501...).
6. `## Rules and side effects`: everything the call changes or enforces, such as number assignment, status changes, files written to the client folder, totals recalculation, immutability.
7. `## Example`: one request and one response as JSON. Use placeholder values only (`Example Client`, `R1,000.00` as `100000` cents), never real business data.
8. `## Used by`: the screens and components that call it.

**Rules**
- A doc is written in the same step as its endpoint is added to the registry and the contract suite, and updated in the same step whenever behaviour changes. Docs are never left for the end.
- Docs describe the contract, not the mock. If `MockApi` behaves differently from the doc, that is a bug in `MockApi`.
- Keep each doc under about 80 lines. No marketing text, no duplicated type definitions (link to `types.ts` by name instead of copying it, except for the example).
- The README index and the registry must list exactly the same endpoints.