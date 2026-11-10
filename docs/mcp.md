# JobCard MCP server

`jobcard.exe --mcp` speaks the Model Context Protocol over stdin/stdout, so Claude Desktop (or any MCP client) can work with the same database as the UI. It is a thin layer over the same code the HTTP API uses: one operation, one rule set.

## Register it in Claude Desktop

Add to `claude_desktop_config.json`. The MCP process reads the same remembered folders as the tray app from `%APPDATA%\JobCardLauncher\config.json`, so it sees the same database and templates:

```json
{
  "mcpServers": {
    "jobcard": {
      "command": "E:\\SCRIPTS\\Operations\\JOBCARD\\JOBCARD\\backend\\jobcard.exe",
      "args": ["--mcp"]
    }
  }
}
```

Restart Claude Desktop. Logs are written to `%APPDATA%\JobCardLauncher\jobcard.log`; stdout carries only JSON-RPC. The HTTP server and the MCP process can run at the same time (SQLite WAL, immediate transactions).

## Tools

Names match `mcpTool` in `src/api/endpoints.ts`. Money is always integer cents; the server computes totals.

`create_draft` accepts either an existing `clientId` or `newClient` details (`name` required; `contact`, `phone`, `email`, `address`, and `notes` optional). It creates the new client and draft in one database transaction, so an invalid draft does not leave an orphan client. The in-app editor also supports adding a client from its document form using “+ New Client”.

`create_draft` and `update_draft` support the same report fields as the editor: `intakeItems` (`label`, `value`), `diagnosticSections` (`title`, `status`, `details`), `authorization` (`technicianSignature`, `technicianDate`, `clientSignature`, `clientDate`), and the optional `lines[].detail`. When updating, send complete replacement arrays and a complete authorization object. `get_document` returns these fields, and `export_document` includes them using the built-in template or matching placeholders in a custom template.

Example:

```json
{
  "type": "invoice",
  "reference": "Example laptop repair",
  "newClient": {
    "name": "Example Client",
    "phone": "000 000 0000",
    "email": "client@example.test"
  },
  "lines": [
    { "description": "Replacement part", "detail": "Example specification", "qty": 1, "unitPriceCents": 100000 }
  ]
}
```

| Tool | What it does | Needs `confirm=true` |
|---|---|---|
| `list_clients`, `list_archived_clients` | List active or archived clients | no |
| `create_client`, `update_client` | Create or edit client details | no |
| `archive_client`, `restore_client` | Archive a client without deleting its documents, or restore it | no |
| `list_documents`, `get_document` | Read quotations, invoices, job cards | no |
| `create_draft`, `update_draft` | Create and edit drafts (no number yet) | no |
| `issue_document` | Assign the permanent number and lock the document | **yes** |
| `set_follow_up` | Set or clear a reminder date on an issued invoice | no |
| `add_payment` | Record a payment on an issued invoice (cannot be deleted) | **yes** |
| `convert_quotation` | Draft invoice from an issued quotation | **yes** |
| `update_status` | Accept/decline, job card progress, cancel | only when cancelling |
| `export_document` | Render docx or pdf from the template, save into the client folder, return the path | no |
| `reveal_folder` | Open the client folder in Explorer | no |
| `get_dashboard`, `get_settings`, `get_storage`, `get_capabilities` | Read-only information | no |
| `export_register_spreadsheet` | Write the register `.xlsx` to `<data-dir>\Exports` | no |
| `get_audit_log` | Recent changes made through the API and MCP | no |

## Safety design

- **Drafts first.** The AI fills in drafts; permanent steps (issue, payment, convert, cancel) are refused unless the call carries `confirm=true`, and the server instructions tell the model to get your approval for that exact document first. The check is enforced in the server, not only in the prompt.
- **No settings access.** Business and bank details can be read but not changed over MCP (`update_settings` is deliberately not exposed). Change them in the app.
- **No permanent client deletes.** Archiving hides a client from active lists and new documents; existing documents stay linked, and `restore_client` reverses the archive.
- **Audit log.** Every write through MCP (and every POST/PUT/DELETE through the API) is recorded with time, source, action and arguments. Ask for `get_audit_log` to see what changed.
- **Errors are readable.** Failures come back as `{"error":{"code","message","fields"}}` so the model can correct itself.
