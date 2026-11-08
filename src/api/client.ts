import {
  Client,
  Doc,
  DocType,
  ISODate,
  Cents,
  LineItem,
  WorkItem,
  Settings,
  Storage,
  Capabilities,
  DashboardSummary,
} from './types.ts';

export interface DocumentFilters {
  type?: DocType;
  status?: string;
  clientId?: string;
  q?: string;
  from?: ISODate;
  to?: ISODate;
}

export interface CreateClientInput {
  name: string;
  contact?: string;
  phone?: string;
  email?: string;
  address?: string;
  notes?: string;
}

export interface CreateDraftInput {
  type: DocType;
  clientId: string;
  reference: string;
  scope?: string;
  jobCardId?: string | null;
  quotationId?: string | null;
  workItems?: WorkItem[];
  lines?: LineItem[];
  depositCents?: Cents | null;
  terms?: string;
  exclusions?: string;
  turnaround?: string;
  notes?: string;
}

export interface AddPaymentInput {
  date: ISODate;
  amountCents: Cents;
  method: string;
  reference?: string;
}

export interface ApiClient {
  listClients(): Promise<Client[]>;
  createClient(input: CreateClientInput): Promise<Client>;
  updateClient(id: string, input: Partial<CreateClientInput>): Promise<Client>;

  listDocuments(filters?: DocumentFilters): Promise<Doc[]>;
  getDocument(id: string): Promise<Doc>;
  createDraft(input: CreateDraftInput): Promise<Doc>;
  updateDraft(id: string, input: Partial<CreateDraftInput>): Promise<Doc>;
  issueDocument(id: string, issueDate?: ISODate): Promise<Doc>;
  addPayment(id: string, input: AddPaymentInput): Promise<Doc>;
  updateStatus(id: string, status: string): Promise<Doc>;
  convertQuotation(id: string): Promise<Doc>;
  exportDocument(id: string, format: 'docx' | 'pdf'): Promise<Blob>;
  revealFolder(id: string): Promise<{ success: boolean; path: string }>;

  getDashboard(year?: number): Promise<DashboardSummary>;
  getSettings(): Promise<Settings>;
  updateSettings(settings: Settings): Promise<Settings>;
  getStorage(): Promise<Storage>;
  getCapabilities(): Promise<Capabilities>;
  exportRegisterSpreadsheet(): Promise<Blob>;
}
