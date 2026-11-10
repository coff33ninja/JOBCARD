# GET /settings

Retrieve business identity, bank details, invoice terms, and optional document branding images.

## ApiClient method
`getSettings`

## MCP tool
`get_settings`

## Request
No query parameters or body required.

## Response
- Status: `200 OK`
- Type: `Settings`
- Returns business profile, banking details, VAT tax rate, default terms days, and optional `companyLogo` and `technicianSignatureImage` PNG/JPEG data URLs. Image fields are omitted when unset; each image is limited to 256 KiB, 4096 pixels per side, and 16 megapixels.

## Errors
| Status | Code | Condition |
|---|---|---|
| 500 | `INTERNAL_ERROR` | Unable to read settings |

## Rules and side effects
Read-only.

## Example
Request: `GET /api/v1/settings`

Response:
```json
{
  "businessName": "Example IT Repairs",
  "person": "Alex Morgan",
  "phone": "082 000 0000",
  "email": "info@example.co.za",
  "location": "Cape Town, South Africa",
  "bankName": "Example Bank",
  "accountHolder": "Example IT Repairs",
  "accountNumber": "0000000000",
  "branchCode": "250655",
  "taxRate": 0,
  "defaultTermsDays": 7,
  "quoteValidityDays": 14,
  "companyLogo": "data:image/png;base64,...",
  "technicianSignatureImage": "data:image/jpeg;base64,..."
}
```

## Used by
Settings screen, Document headers and preview, New document default terms.
