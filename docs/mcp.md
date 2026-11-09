# JobCard MCP server

`jobcard.exe --mcp` speaks the Model Context Protocol over stdin/stdout, so Claude Desktop (or any MCP client) can work with the same database as the UI. It is a thin layer over the same code the HTTP API uses: one operation, one rule set.

## Register it in Claude Desktop

Add to `claude_desktop_config.json` (use the same `--data-dir`, `--clients-dir` and `--templates-dir` as the UI server; the defaults match):

```json
{
  "mcpServers": {
    "jobcard": {
      "command": "E:\\SCRIPTS\\Operations\\JOBCARD\\JOBCARD\\backend\\jobcard.exe",
      "args": ["--mcp", "--clients-dir", "G:\\My Drive\\BUSINESS\\Clients", "--templates-dir", "G:\\My Drive\\BUSINESS\\Templates"]
    }
  }
}
```

Restart Claude Desktop. Logs go to stderr, stdout carries only JSON-RPC. The HTTP server and the MCP process can run at the same time (SQLite WAL, immediate transactions).

## Tools

Names match `mcpTool` in `src/api/endpoints.ts`. Money is always integer cents; the server computes totals.

| Tool | What it does | Needs `confirm=true` |
|---|---|---|
| `list_clients`, `create_client`, `update_client` | Clients | no |
| `list_documents`, `get_document` | Read quotations, invoices, job cards | no |
| `create_draft`, `update_draft` | Create and edit drafts (no number yet) | no |
| `issue_document` | Assign the permanent number and lock the document | **yes** |
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
- **No deletes.** There is no delete tool anywhere.
- **Audit log.** Every write through MCP (and every POST/PUT through the API) is recorded with time, source, action and arguments. Ask for `get_audit_log` to see what changed.
- **Errors are readable.** Failures come back as `{"error":{"code","message","fields"}}` so the model can correct itself.
