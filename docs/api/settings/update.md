# PUT /settings

Update business identity, banking details, and invoice terms configuration.

## ApiClient method
`updateSettings`

## MCP tool
`update_settings`

## Request
Body:
- `businessName` (string, required)
- `person` (string, required)
- `phone` (string, required)
- `email` (string, required)
- `location` (string, required)
- `bankName` (string, required)
- `accountHolder` (string, required)
- `accountNumber` (string, required)
- `branchCode` (string, required)
- `taxRate` (number, required, e.g. 0 or 0.15)
- `defaultTermsDays` (number, required, >= 1)
- `quoteValidityDays` (number, required, >= 1)

## Response
- Status: `200 OK`
- Type: `Settings`
- Returns updated settings.

## Errors
| Status | Code | Condition |
|---|---|---|
| 400 | `VALIDATION_FAILED` | Missing or invalid configuration fields |

## Rules and side effects
Saves new settings in configuration storage. Future issued documents and preview templates use these updated values.

## Example
Request: `PUT /api/v1/settings`
```json
{
  "businessName": "Precision Tech Solutions",
  "person": "Alex Morgan",
  "phone": "082 111 2233",
  "email": "alex@precisiontech.co.za",
  "location": "Durbanville, Cape Town",
  "bankName": "First National Bank",
  "accountHolder": "Precision Tech",
  "accountNumber": "62000000000",
  "branchCode": "250655",
  "taxRate": 0,
  "defaultTermsDays": 7,
  "quoteValidityDays": 14
}
```

Response:
```json
{
  "businessName": "Precision Tech Solutions",
  "person": "Alex Morgan",
  "phone": "082 111 2233",
  "email": "alex@precisiontech.co.za",
  "location": "Durbanville, Cape Town",
  "bankName": "First National Bank",
  "accountHolder": "Precision Tech",
  "accountNumber": "62000000000",
  "branchCode": "250655",
  "taxRate": 0,
  "defaultTermsDays": 7,
  "quoteValidityDays": 14
}
```

## Used by
Settings screen save action.
