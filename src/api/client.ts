import {
  Client,
  Doc,
  DocType,
  ISODate,
  Cents,
  LineItem,
  WorkItem,
  IntakeItem,
  DiagnosticSection,
  Authorization,
  Settings,
  Storage,
  Capabilities,
  DashboardSummary,
  DocumentRevision,
  ClientDuplicateMatch,
  StockAllocationResult,
  StockIntakeInput,
  StockItem,
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
  intakeItems?: IntakeItem[];
  diagnosticSections?: DiagnosticSection[];
  authorization?: Authorization;
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
  listStockItems(): Promise<StockItem[]>;
  createStockIntake(input: StockIntakeInput): Promise<StockItem[]>;
  allocateStock(itemId: string, clientId: string, quantity: number): Promise<StockAllocationResult>;
  listArchivedClients(): Promise<Client[]>;
  findPotentialDuplicateClients(input: CreateClientInput): Promise<ClientDuplicateMatch[]>;
  createClient(input: CreateClientInput): Promise<Client>;
  updateClient(id: string, input: Partial<CreateClientInput>): Promise<Client>;
  archiveClient(id: string): Promise<Client>;
  restoreClient(id: string): Promise<Client>;

  listDocuments(filters?: DocumentFilters): Promise<Doc[]>;
  listArchivedDocuments(): Promise<Doc[]>;
  getDocument(id: string): Promise<Doc>;
  createDraft(input: CreateDraftInput): Promise<Doc>;
  updateDraft(id: string, input: Partial<CreateDraftInput>): Promise<Doc>;
  archiveDraft(id: string): Promise<Doc>;
  restoreDraft(id: string): Promise<Doc>;
  deleteDraft(id: string, confirm: boolean): Promise<Doc>;
  createReplacementQuotation(id: string): Promise<Doc>;
  updateIssuedQuotation(id: string, input: Partial<CreateDraftInput>, confirm: boolean): Promise<Doc>;
  listQuotationRevisions(id: string): Promise<DocumentRevision[]>;
  issueDocument(id: string, issueDate?: ISODate): Promise<Doc>;
  addPayment(id: string, input: AddPaymentInput): Promise<Doc>;
  updateStatus(id: string, status: string): Promise<Doc>;
  setFollowUp(id: string, followUpDate: ISODate | null): Promise<Doc>;
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
