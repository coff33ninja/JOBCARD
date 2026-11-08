# JobCard — Project Architecture & Design Guidelines Reference

This reference document contains the full architecture, API contract, domain rules, and design specifications for **JobCard** — a single-user job card, quotation, and invoice manager for an IT repair business in South Africa (ZAR).

---

## 1. Core Architecture Principles

- **UI Scope Only:** The frontend interfaces against a fixed `/api/v1` REST contract. The application switches between `MockApi` (in-memory, seeded) and `HttpApi` (real Go backend serving SQLite) via `VITE_API=mock|http` and `VITE_API_URL`.
- **Zero LocalStorage as Data Store:** All application data flows through the `ApiClient` interface. The mock keeps data in memory only.
- **Strict Money Representation:** All monetary values are represented as integer cents (`Cents = number`). Display formatting is applied only at the edge as `R1,000.00` (South African Rand).
- **Server-Authoritative Numbering:** Document numbers (`JC-YYYY-NNN`, `QUO-YYYY-NNN`, `INV-YYYY-NNN`) are assigned exclusively by the server on issue. Drafts have `number: null`. Once issued, numbers are immutable.
- **Zero Hard-Coded Business Data:** Business details (name, bank, contact, terms) come from `GET /settings`.
- **Zero Inline Sample Data in Code:** All mock data is loaded from `seed/seed.json` via a single `loadSeed()` function.

---

## 2. Domain Rules (`src/domain/`)

- **Document Types:** `quotation`, `invoice`, `jobcard`.
- **Line Items & Totals:**
  - `totalCents = round(qty * unitPriceCents)`
  - `subtotal = sum(lines)`
  - `taxCents = round(subtotal * taxRate)` (default `taxRate = 0` for non-VAT vendor: "VAT (Non-VAT Vendor): R0.00")
  - `total = subtotal + tax`
  - `paid = sum(payments)`
  - `balance = total - paid`
- **Invoice Status Derivation (`deriveInvoiceStatus(invoice, today)`):**
  - `cancelled` stays cancelled; `draft` stays draft until issued.
  - `balance <= 0` and total > 0 &rarr; `paid`
  - `0 < paid < total` &rarr; `partially_paid` (or `overdue` if past `dueDate` and balance > 0)
  - `paid == 0`, issued, past `dueDate`, balance > 0 &rarr; `overdue`; otherwise `sent`
  - `daysOverdue = today - dueDate` when balance > 0 and dueDate < today, else `null`
- **Quotation Status & Conversion:**
  - `draft | sent | accepted | declined | expired | converted`
  - `expired` derived when past `validUntil` and not accepted/declined/converted.
  - Quote-to-invoice conversion creates a new draft invoice linked via `quotationId` and sets the quote status to `converted`.
- **Stamps (`stampFor(document, today)`):**
  - Invoices: `PAID` (green), `PARTIALLY PAID` (amber), `UNPAID` (red), `OVERDUE` (red, bold), `DRAFT` (grey), `CANCELLED` (grey, struck).
  - Quotations: `DRAFT`, `ACCEPTED` (green), `DECLINED` (red), `EXPIRED` (grey).
  - Job cards: No stamp.

---

## 3. Document Layouts (A4 Portrait, Navy/Grey)

- **Invoice:** Header &rarr; Number & Dates &rarr; Client & Job Details &rarr; Completed Services & Work Done &rarr; Financial Breakdown &rarr; Payment Details & Terms &rarr; Notes & Recommendations.
- **Quotation:** Header &rarr; Number & Validity &rarr; Client & Job Details &rarr; Proposed Services & Work to be Done &rarr; Financial Breakdown & Deposit &rarr; Payment Details &rarr; Terms & Client Acceptance &rarr; Notes.
- **Job Card:** Internal view with job card number, client, device/serial, scope, work log notes, linked quotation(s) and invoice(s), status (`open | in_progress | completed | invoiced | closed`).

---

## 4. API Endpoints (`/api/v1`)

| Method + Path | Purpose | Client Method |
|---|---|---|
| `GET /clients` | List clients | `listClients` |
| `POST /clients` | Create client | `createClient` |
| `PUT /clients/{id}` | Update client | `updateClient` |
| `GET /documents` | Query documents (type, status, client, search, dates) | `listDocuments` |
| `GET /documents/{id}` | Fetch document | `getDocument` |
| `POST /documents` | Create draft document | `createDraft` |
| `PUT /documents/{id}` | Update draft document | `updateDraft` |
| `POST /documents/{id}/issue` | Issue document (assigns number & dates) | `issueDocument` |
| `POST /documents/{id}/payments` | Record payment (rejects overpayment: 422) | `addPayment` |
| `POST /documents/{id}/status` | Update status (cancel, accept, decline) | `updateStatus` |
| `POST /quotations/{id}/convert` | Convert quotation to draft invoice | `convertQuotation` |
| `POST /documents/{id}/export` | Export PDF or DOCX file bytes | `exportDocument` |
| `POST /documents/{id}/reveal` | Reveal client folder on disk | `revealFolder` |
| `GET /dashboard` | Get dashboard financial metrics & summaries | `getDashboard` |
| `GET /settings` | Get business & bank settings | `getSettings` |
| `PUT /settings` | Update settings | `updateSettings` |
| `GET /storage` | Get storage paths (dataDir, dbPath, clientsDir) | `getStorage` |
| `GET /capabilities` | Get backend capabilities (`docx`, `pdf`, `revealInFolder`) | `getCapabilities` |
| `GET /register/export.xlsx` | Export register spreadsheet | `exportRegisterSpreadsheet` |

---

## 5. Export Decision Hierarchy

1. **Download PDF** (`export?format=pdf` or fallback to browser print with "Save as PDF" notice)
2. **Download DOCX** (`export?format=docx` when `capabilities.docx` is true)
3. **Print** (`window.print()` with `@media print` isolation of A4 document)
4. **Share...** (when `navigator.canShare({ files })` is supported)
5. **Email** (`mailto:` with prefilled subject/body + reminder to attach downloaded PDF)
6. **WhatsApp** (`https://wa.me/<27...>?text=...` with normalised SA phone number)
7. **Show in folder** (`POST /documents/{id}/reveal` when supported)
