# GET /dashboard

Retrieve consolidated business dashboard metrics and financial totals for a given calendar year.

## ApiClient method
`getDashboard`

## MCP tool
`get_dashboard`

## Request
Query parameter:
- `year` (number, optional): 4-digit calendar year (e.g. `2026`). Defaults to current year.

## Response
- Status: `200 OK`
- Type: `DashboardSummary`
- Provides invoiced, paid, outstanding, and overdue totals, status breakdown counts, needs attention metrics, aging buckets, 12-month series, and next sequence numbers.

## Errors
| Status | Code | Condition |
|---|---|---|
| 400 | `INVALID_YEAR` | Invalid year format |
| 500 | `INTERNAL_ERROR` | Database query failure |

## Rules and side effects
Read-only metrics calculation aggregated from SQLite records.

## Example
Request: `GET /api/v1/dashboard?year=2026`

Response:
```json
{
  "year": 2026,
  "totals": {
    "invoicedCents": 12500000,
    "paidCents": 9500000,
    "outstandingCents": 3000000,
    "overdueCents": 1200000
  },
  "statusCounts": {
    "draft": 2,
    "sent": 3,
    "partially_paid": 1,
    "paid": 6,
    "overdue": 2,
    "cancelled": 1
  },
  "attention": {
    "followUpsDue": 2,
    "dueNext7Count": 3,
    "dueNext7Cents": 450000
  },
  "ageing": {
    "d1_30": 300000,
    "d31_60": 400000,
    "d61_90": 200000,
    "d90plus": 300000
  },
  "monthly": [
    { "month": 1, "jobs": 4, "invoicedCents": 2000000, "paidCents": 1500000, "outstandingCents": 500000 }
  ],
  "nextNumbers": {
    "jobcard": "JC-2026-009",
    "quotation": "QUO-2026-006",
    "invoice": "INV-2026-008"
  }
}
```

## Used by
Dashboard screen.
