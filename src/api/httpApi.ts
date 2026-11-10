import { z } from 'zod';
import {
  ApiClient,
  CreateClientInput,
  CreateDraftInput,
  AddPaymentInput,
  DocumentFilters,
} from './client.ts';
import {
  Client,
  Doc,
  Settings,
  Storage,
  Capabilities,
  DashboardSummary,
  ApiError,
  ISODate,
  DocumentRevision,
} from './types.ts';

// Zod schemas for runtime response validation
const ClientSchema = z.object({
  id: z.string(),
  name: z.string(),
  contact: z.string(),
  phone: z.string(),
  email: z.string(),
  address: z.string(),
  notes: z.string(),
});

const LineItemSchema = z.object({
  description: z.string(),
  detail: z.string().optional(),
  qty: z.number(),
  unitPriceCents: z.number(),
  totalCents: z.number(),
});

const PaymentSchema = z.object({
  id: z.string(),
  date: z.string(),
  amountCents: z.number(),
  method: z.string(),
  reference: z.string(),
});

const WorkItemSchema = z.object({
  title: z.string(),
  detail: z.string(),
});

const IntakeItemSchema = z.object({
  label: z.string(),
  value: z.string(),
});

const DiagnosticSectionSchema = z.object({
  title: z.string(),
  status: z.string(),
  details: z.string(),
});

const AuthorizationSchema = z.object({
  technicianSignature: z.string(),
  technicianDate: z.string(),
  clientSignature: z.string(),
  clientDate: z.string(),
});

const DocSchema = z.object({
  id: z.string(),
  type: z.enum(['quotation', 'invoice', 'jobcard']),
  number: z.string().nullable(),
  clientId: z.string(),
  jobCardId: z.string().nullable(),
  quotationId: z.string().nullable(),
  issueDate: z.string().nullable(),
  dueDate: z.string().nullable(),
  validUntil: z.string().nullable(),
  followUpDate: z.string().nullable(),
  archivedAt: z.string().nullable().optional(),
  revisionOfQuotationId: z.string().nullable().optional(),
  reference: z.string(),
  scope: z.string(),
  workItems: z.array(WorkItemSchema),
  intakeItems: z.array(IntakeItemSchema),
  diagnosticSections: z.array(DiagnosticSectionSchema),
  authorization: AuthorizationSchema,
  lines: z.array(LineItemSchema),
  taxCents: z.number(),
  subtotalCents: z.number(),
  totalCents: z.number(),
  depositCents: z.number().nullable(),
  payments: z.array(PaymentSchema),
  status: z.string(),
  terms: z.string(),
  exclusions: z.string(),
  turnaround: z.string(),
  notes: z.string(),
  folderPath: z.string().nullable(),
  createdAt: z.string(),
  updatedAt: z.string(),
});

const DocumentRevisionSchema = z.object({
  id: z.string(),
  documentId: z.string(),
  recordedAt: z.string(),
  snapshot: DocSchema,
});

const SettingsSchema = z.object({
  businessName: z.string(),
  person: z.string(),
  phone: z.string(),
  email: z.string(),
  location: z.string(),
  bankName: z.string(),
  accountHolder: z.string(),
  accountNumber: z.string(),
  branchCode: z.string(),
  taxRate: z.number(),
  defaultTermsDays: z.number(),
  quoteValidityDays: z.number(),
});

const StorageSchema = z.object({
  dataDir: z.string(),
  dbPath: z.string(),
  clientsDir: z.string(),
});

const CapabilitiesSchema = z.object({
  docx: z.boolean(),
  pdf: z.boolean(),
  revealInFolder: z.boolean(),
});

const DashboardSummarySchema = z.object({
  year: z.number(),
  totals: z.object({
    invoicedCents: z.number(),
    paidCents: z.number(),
    outstandingCents: z.number(),
    overdueCents: z.number(),
  }),
  statusCounts: z.record(z.string(), z.number()),
  attention: z.object({
    followUpsDue: z.number(),
    dueNext7Count: z.number(),
    dueNext7Cents: z.number(),
  }),
  ageing: z.object({
    d1_30: z.number(),
    d31_60: z.number(),
    d61_90: z.number(),
    d90plus: z.number(),
  }),
  monthly: z.array(
    z.object({
      month: z.number(),
      jobs: z.number(),
      invoicedCents: z.number(),
      paidCents: z.number(),
      outstandingCents: z.number(),
    })
  ),
  nextNumbers: z.object({
    jobcard: z.string(),
    quotation: z.string(),
    invoice: z.string(),
  }),
});

export class HttpApi implements ApiClient {
  private baseUrl: string;
  private token: string;

  constructor(baseUrl: string = '', token: string = '') {
    this.token = token;
    // Normalise base url to remove trailing slash
    const base = baseUrl ? baseUrl.replace(/\/$/, '') : '';
    this.baseUrl = `${base}/api/v1`;
  }

  private async request<T>(
    path: string,
    options: RequestInit = {},
    schema?: z.ZodType<T>
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      ...(this.token ? { Authorization: `Bearer ${this.token}` } : {}),
      ...((options.headers as Record<string, string>) || {}),
    };

    const res = await fetch(url, { ...options, headers });

    if (!res.ok) {
      let errorBody: { error?: { code?: string; message?: string; fields?: Record<string, string> } } = {};
      try {
        errorBody = await res.json();
      } catch {
        // Ignored if non-json
      }
      const code = errorBody.error?.code || 'HTTP_ERROR';
      const message = errorBody.error?.message || `Request failed with status ${res.status}`;
      throw new ApiError(res.status, code, message, errorBody.error?.fields);
    }

    const data = await res.json();
    if (schema) {
      return schema.parse(data);
    }
    return data as T;
  }

  async listClients(): Promise<Client[]> {
    return this.request('/clients', { method: 'GET' }, z.array(ClientSchema));
  }

  async listArchivedClients(): Promise<Client[]> {
    return this.request('/clients/archived', { method: 'GET' }, z.array(ClientSchema));
  }

  async createClient(input: CreateClientInput): Promise<Client> {
    return this.request('/clients', {
      method: 'POST',
      body: JSON.stringify(input),
    }, ClientSchema);
  }

  async archiveClient(id: string): Promise<Client> {
    return this.request(`/clients/${id}`, { method: 'DELETE' }, ClientSchema);
  }

  async restoreClient(id: string): Promise<Client> {
    return this.request(`/clients/${id}/restore`, { method: 'POST' }, ClientSchema);
  }

  async updateClient(id: string, input: Partial<CreateClientInput>): Promise<Client> {
    return this.request(`/clients/${id}`, {
      method: 'PUT',
      body: JSON.stringify(input),
    }, ClientSchema);
  }

  async listDocuments(filters?: DocumentFilters): Promise<Doc[]> {
    const params = new URLSearchParams();
    if (filters?.type) params.set('type', filters.type);
    if (filters?.status) params.set('status', filters.status);
    if (filters?.clientId) params.set('clientId', filters.clientId);
    if (filters?.q) params.set('q', filters.q);
    if (filters?.from) params.set('from', filters.from);
    if (filters?.to) params.set('to', filters.to);

    const query = params.toString() ? `?${params.toString()}` : '';
    return this.request(`/documents${query}`, { method: 'GET' }, z.array(DocSchema));
  }

  async listArchivedDocuments(): Promise<Doc[]> {
    return this.request('/documents/archived', { method: 'GET' }, z.array(DocSchema));
  }

  async getDocument(id: string): Promise<Doc> {
    return this.request(`/documents/${id}`, { method: 'GET' }, DocSchema);
  }

  async createDraft(input: CreateDraftInput): Promise<Doc> {
    return this.request('/documents', {
      method: 'POST',
      body: JSON.stringify(input),
    }, DocSchema);
  }

  async updateDraft(id: string, input: Partial<CreateDraftInput>): Promise<Doc> {
    return this.request(`/documents/${id}`, {
      method: 'PUT',
      body: JSON.stringify(input),
    }, DocSchema);
  }

  async archiveDraft(id: string): Promise<Doc> {
    return this.request(`/documents/${id}/archive`, { method: 'POST' }, DocSchema);
  }

  async restoreDraft(id: string): Promise<Doc> {
    return this.request(`/documents/${id}/restore`, { method: 'POST' }, DocSchema);
  }

  async deleteDraft(id: string, confirm: boolean): Promise<Doc> {
    return this.request(`/documents/${id}`, {
      method: 'DELETE',
      body: JSON.stringify({ confirm }),
    }, DocSchema);
  }

  async createReplacementQuotation(id: string): Promise<Doc> {
    return this.request(`/quotations/${id}/replacement`, { method: 'POST' }, DocSchema);
  }

  async updateIssuedQuotation(
    id: string,
    input: Partial<CreateDraftInput>,
    confirm: boolean
  ): Promise<Doc> {
    return this.request(`/quotations/${id}/revision`, {
      method: 'PUT',
      body: JSON.stringify({ ...input, confirm }),
    }, DocSchema);
  }

  async listQuotationRevisions(id: string): Promise<DocumentRevision[]> {
    return this.request(`/quotations/${id}/revisions`, { method: 'GET' }, z.array(DocumentRevisionSchema));
  }

  async issueDocument(id: string, issueDate?: ISODate): Promise<Doc> {
    return this.request(`/documents/${id}/issue`, {
      method: 'POST',
      body: issueDate ? JSON.stringify({ issueDate }) : undefined,
    }, DocSchema);
  }

  async addPayment(id: string, input: AddPaymentInput): Promise<Doc> {
    return this.request(`/documents/${id}/payments`, {
      method: 'POST',
      body: JSON.stringify(input),
    }, DocSchema);
  }

  async updateStatus(id: string, status: string): Promise<Doc> {
    return this.request(`/documents/${id}/status`, {
      method: 'POST',
      body: JSON.stringify({ status }),
    }, DocSchema);
  }

  async setFollowUp(id: string, followUpDate: ISODate | null): Promise<Doc> {
    return this.request(`/documents/${id}/follow-up`, {
      method: 'POST',
      body: JSON.stringify({ followUpDate }),
    }, DocSchema);
  }

  async convertQuotation(id: string): Promise<Doc> {
    return this.request(`/quotations/${id}/convert`, {
      method: 'POST',
    }, DocSchema);
  }

  async exportDocument(id: string, format: 'docx' | 'pdf'): Promise<Blob> {
    const url = `${this.baseUrl}/documents/${id}/export?format=${format}`;
    const res = await fetch(url, { method: 'POST', headers: this.token ? { Authorization: `Bearer ${this.token}` } : {} });
    if (!res.ok) {
      let code = 'EXPORT_FAILED';
      let message = 'Export failed';
      try {
        const err = await res.json();
        code = err.error?.code || code;
        message = err.error?.message || message;
      } catch {
        // Non-JSON
      }
      throw new ApiError(res.status, code, message);
    }
    return res.blob();
  }

  async revealFolder(id: string): Promise<{ success: boolean; path: string }> {
    return this.request(`/documents/${id}/reveal`, { method: 'POST' });
  }

  async getDashboard(year?: number): Promise<DashboardSummary> {
    const query = year ? `?year=${year}` : '';
    return this.request(`/dashboard${query}`, { method: 'GET' }, DashboardSummarySchema);
  }

  async getSettings(): Promise<Settings> {
    return this.request('/settings', { method: 'GET' }, SettingsSchema);
  }

  async updateSettings(settings: Settings): Promise<Settings> {
    return this.request('/settings', {
      method: 'PUT',
      body: JSON.stringify(settings),
    }, SettingsSchema);
  }

  async getStorage(): Promise<Storage> {
    return this.request('/storage', { method: 'GET' }, StorageSchema);
  }

  async getCapabilities(): Promise<Capabilities> {
    return this.request('/capabilities', { method: 'GET' }, CapabilitiesSchema);
  }

  async exportRegisterSpreadsheet(): Promise<Blob> {
    const url = `${this.baseUrl}/register/export.xlsx`;
    const res = await fetch(url, { method: 'GET', headers: this.token ? { Authorization: `Bearer ${this.token}` } : {} });
    if (!res.ok) {
      throw new ApiError(res.status, 'EXPORT_FAILED', 'Failed to export register spreadsheet');
    }
    return res.blob();
  }
}
