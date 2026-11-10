export type Cents = number;
export type ISODate = string; // "YYYY-MM-DD"

export interface Client {
  id: string;
  name: string;
  contact: string;
  phone: string;
  email: string;
  address: string;
  notes: string;
}

export interface LineItem {
  description: string;
  detail?: string;
  qty: number;
  unitPriceCents: Cents;
  totalCents: Cents;
}

export interface Payment {
  id: string;
  date: ISODate;
  amountCents: Cents;
  method: string;
  reference: string;
}

export type DocType = "quotation" | "invoice" | "jobcard";

export interface WorkItem {
  title: string;
  detail: string;
}

export interface IntakeItem {
  label: string;
  value: string;
}

export interface DiagnosticSection {
  title: string;
  status: string;
  details: string;
}

export interface Authorization {
  technicianSignature: string;
  technicianDate: string;
  clientSignature: string;
  clientDate: string;
}

export interface Doc {
  id: string;
  type: DocType;
  number: string | null; // null while draft; server-assigned on issue
  clientId: string;
  jobCardId: string | null;
  quotationId: string | null;
  issueDate: ISODate | null;
  dueDate: ISODate | null;
  validUntil: ISODate | null;
  followUpDate: ISODate | null; // invoices: reminder date to chase payment, null when unset
  archivedAt?: string | null;
  revisionOfQuotationId?: string | null;
  reference: string; // device / serial / job reference text
  scope: string;
  workItems: WorkItem[]; // completed or proposed work
  intakeItems: IntakeItem[];
  diagnosticSections: DiagnosticSection[];
  authorization: Authorization;
  lines: LineItem[];
  taxCents: Cents;
  subtotalCents: Cents;
  totalCents: Cents;
  depositCents: Cents | null; // quotations
  payments: Payment[]; // invoices
  status: string; // stored status (see domain rules)
  terms: string;
  exclusions: string;
  turnaround: string;
  notes: string;
  folderPath: string | null; // client folder on disk, set by backend
  createdAt: string;
  updatedAt: string;
}

export interface DocumentRevision {
  id: string;
  documentId: string;
  recordedAt: string;
  snapshot: Doc;
}

export interface Settings {
  businessName: string;
  person: string;
  phone: string;
  email: string;
  location: string;
  bankName: string;
  accountHolder: string;
  accountNumber: string;
  branchCode: string;
  taxRate: number; // 0 = non-VAT vendor
  defaultTermsDays: number;
  quoteValidityDays: number;
}

export interface Storage {
  dataDir: string;
  dbPath: string;
  clientsDir: string;
}

export interface Capabilities {
  docx: boolean;
  pdf: boolean;
  revealInFolder: boolean;
}

export interface DashboardSummary {
  year: number;
  totals: {
    invoicedCents: Cents;
    paidCents: Cents;
    outstandingCents: Cents;
    overdueCents: Cents;
  };
  statusCounts: Record<string, number>; // draft, sent, partially_paid, paid, overdue, cancelled
  attention: {
    followUpsDue: number;
    dueNext7Count: number;
    dueNext7Cents: Cents;
  };
  ageing: {
    d1_30: Cents;
    d31_60: Cents;
    d61_90: Cents;
    d90plus: Cents;
  };
  monthly: {
    month: number; // 1-12
    jobs: number;
    invoicedCents: Cents;
    paidCents: Cents;
    outstandingCents: Cents;
  }[]; // 12 rows
  nextNumbers: {
    jobcard: string;
    quotation: string;
    invoice: string;
  };
}

export interface ApiErrorResponse {
  error: {
    code: string;
    message: string;
    fields?: Record<string, string>;
  };
}

export class ApiError extends Error {
  code: string;
  status: number;
  fields?: Record<string, string>;

  constructor(status: number, code: string, message: string, fields?: Record<string, string>) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.fields = fields;
  }
}
