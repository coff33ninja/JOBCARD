# POST /clients/check-duplicates

Find active or archived clients that may match the supplied client details.

## ApiClient method

`findPotentialDuplicateClients`

## MCP tool

`find_duplicate_clients`

## Request

Body:

- `name` (string, required)
- `contact` (string, optional)
- `phone` (string, optional)
- `email` (string, optional)
- `address` and `notes` (string, optional; not used for matching)

## Response

- Status: `200 OK`
- Type: `ClientDuplicateMatch[]`
- Each match contains the existing `client`, an `archived` flag, and matching `reasons` (`name/contact`, `phone`, or `email`).
- Returns an empty array when no possible matches are found.

## Errors

| Status | Code | Condition |
|---|---|---|
| 400 | `VALIDATION_FAILED` | Missing client name or invalid JSON |

## Rules and side effects

Names are matched exactly after case folding and whitespace normalization across the company/client name and contact-person fields. Phone numbers match after formatting characters are removed; email addresses are matched case-insensitively. Matching is a warning only and does not create, merge, or block client records.

The MCP workflow should call this tool before `create_client` or `create_draft` with `newClient`, then tell the user about any matches. Archived clients are included so prior customer records remain discoverable.

## Example

Request: `POST /api/v1/clients/check-duplicates`
```json
{
  "name": "ACME Holdings",
  "contact": "Jane Doe"
}
```

Response:
```json
[
  {
    "client": {
      "id": "cli-2",
      "name": "Acme Holdings",
      "contact": "Jane Doe",
      "phone": "082 123 4567",
      "email": "accounts@acme.co.za",
      "address": "",
      "notes": ""
    },
    "archived": false,
    "reasons": ["name/contact"]
  }
]
```

## Used by

Client creation in the Clients screen, the document editor, and the MCP client-creation workflow.
