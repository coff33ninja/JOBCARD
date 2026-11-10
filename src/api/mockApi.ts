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
  DocType,
  Settings,
  Storage,
  Capabilities,
  DashboardSummary,
  ApiError,
  ISODate,
} from './types.ts';
import { loadSeed, RawSeedData } from './seedLoader.ts';
import { getTodayISO, addDays, calculateDaysOverdue, parseISODate } from '../domain/dates.ts';
import { calculateTotals } from '../domain/totals.ts';
import { deriveInvoiceStatus, deriveQuotationStatus, convertQuotationToInvoice } from '../domain/status.ts';

export class MockApi implements ApiClient {
  private clients: Client[] = [];
  private readonly archivedClientIds = new Set<string>();
  private documents: Doc[] = [];
  private settings: Settings;
  private storage: Storage;
  private capabilities: Capabilities;
  private today: ISODate;

  constructor(seedRaw?: RawSeedData, today: ISODate = getTodayISO()) {
    this.today = today;
    if (seedRaw) {
      const loaded = loadSeed(seedRaw, today);
      this.clients = loaded.clients;
      this.documents = loaded.documents;
      this.settings = loaded.settings;
      this.storage = loaded.storage;
      this.capabilities = loaded.capabilities;
    } else {
      this.settings = {
        businessName: 'Example IT Repairs',
        person: 'Alex Morgan',
        phone: '000 000 0000',
        email: 'info@example.co.za',
        location: 'Cape Town, South Africa',
        bankName: 'Example Bank',
        accountHolder: 'Example IT Repairs',
        accountNumber: '0000000000',
        branchCode: '000000',
        taxRate: 0,
        defaultTermsDays: 7,
        quoteValidityDays: 14,
      };
      this.storage = {
        dataDir: '%APPDATA%\\JobCard',
        dbPath: '%APPDATA%\\JobCard\\jobcard.sqlite',
        clientsDir: '%APPDATA%\\JobCard\\Clients',
      };
      this.capabilities = {
        docx: true,
        pdf: true,
        revealInFolder: true,
      };
    }
  }

  // --- Clients ---
  async listClients(): Promise<Client[]> {
    return this.clients
      .filter((client) => !this.archivedClientIds.has(client.id))
      .sort((a, b) => a.name.localeCompare(b.name));
  }

  async listArchivedClients(): Promise<Client[]> {
    return this.clients
      .filter((client) => this.archivedClientIds.has(client.id))
      .sort((a, b) => a.name.localeCompare(b.name));
  }

  async createClient(input: CreateClientInput): Promise<Client> {
    if (!input.name || !input.name.trim()) {
      throw new ApiError(400, 'VALIDATION_FAILED', 'Client name is required', { name: 'Name is required' });
    }
    const newClient: Client = {
      id: `cli-${Date.now()}-${Math.floor(Math.random() * 1000)}`,
      name: input.name.trim(),
      contact: input.contact?.trim() || '',
      phone: input.phone?.trim() || '',
      email: input.email?.trim() || '',
      address: input.address?.trim() || '',
      notes: input.notes?.trim() || '',
    };
    this.clients.push(newClient);
    return { ...newClient };
  }

  async updateClient(id: string, input: Partial<CreateClientInput>): Promise<Client> {
    const idx = this.clients.findIndex((c) => c.id === id);
    if (idx === -1) {
      throw new ApiError(404, 'NOT_FOUND', `Client ${id} not found`);
    }
    if (input.name !== undefined && !input.name.trim()) {
      throw new ApiError(400, 'VALIDATION_FAILED', 'Client name cannot be empty', { name: 'Name is required' });
    }
    this.clients[idx] = {
      ...this.clients[idx],
      ...input,
      name: input.name !== undefined ? input.name.trim() : this.clients[idx].name,
    };
    return { ...this.clients[idx] };
  }

  async archiveClient(id: string): Promise<Client> {
    const client = this.clients.find((item) => item.id === id);
    if (!client) {
      throw new ApiError(404, 'NOT_FOUND', `Client ${id} not found`);
    }
    this.archivedClientIds.add(id);
    return { ...client };
  }

  async restoreClient(id: string): Promise<Client> {
    const client = this.clients.find((item) => item.id === id);
    if (!client) {
      throw new ApiError(404, 'NOT_FOUND', `Client ${id} not found`);
    }
    this.archivedClientIds.delete(id);
    return { ...client };
  }

  // --- Documents ---
  async listDocuments(filters?: DocumentFilters): Promise<Doc[]> {
    let result = [...this.documents];

    if (filters?.type) {
      result = result.filter((d) => d.type === filters.type);
    }
    if (filters?.status) {
      result = result.filter((d) => {
        const derived = d.type === 'invoice'
          ? deriveInvoiceStatus(d, this.today)
          : d.type === 'quotation'
          ? deriveQuotationStatus(d, this.today)
          : d.status;
        return derived === filters.status;
      });
    }
    if (filters?.clientId) {
      result = result.filter((d) => d.clientId === filters.clientId);
    }
    if (filters?.q) {
      const q = filters.q.toLowerCase();
      result = result.filter(
        (d) =>
          (d.number && d.number.toLowerCase().includes(q)) ||
          d.reference.toLowerCase().includes(q) ||
          d.scope.toLowerCase().includes(q)
      );
    }
    if (filters?.from) {
      result = result.filter((d) => (d.issueDate ? d.issueDate >= filters.from! : true));
    }
    if (filters?.to) {
      result = result.filter((d) => (d.issueDate ? d.issueDate <= filters.to! : true));
    }

    return result.sort((a, b) => {
      const dateA = a.issueDate || a.createdAt;
      const dateB = b.issueDate || b.createdAt;
      return dateB.localeCompare(dateA);
    });
  }

  async getDocument(id: string): Promise<Doc> {
    const doc = this.documents.find((d) => d.id === id);
    if (!doc) {
      throw new ApiError(404, 'NOT_FOUND', `Document ${id} not found`);
    }
    return JSON.parse(JSON.stringify(doc));
  }

  async createDraft(input: CreateDraftInput): Promise<Doc> {
    if (!input.type || !input.clientId || !input.reference) {
      throw new ApiError(400, 'VALIDATION_FAILED', 'type, clientId, and reference are required');
    }
    const client = this.clients.find((item) => item.id === input.clientId);
    if (!client || this.archivedClientIds.has(input.clientId)) {
      throw new ApiError(400, 'VALIDATION_FAILED', 'Client not found');
    }

    const lines = (input.lines || []).map((l) => ({
      ...l,
      totalCents: l.totalCents !== undefined ? l.totalCents : Math.round(l.qty * l.unitPriceCents),
    }));

    const totals = calculateTotals(lines, this.settings.taxRate, []);
    const nowIso = new Date().toISOString();

    const folderPath = client ? `C:\\JobCard\\Clients\\${client.name}` : null;

    const newDoc: Doc = {
      id: `doc-${Date.now()}-${Math.floor(Math.random() * 1000)}`,
      type: input.type,
      number: null,
      clientId: input.clientId,
      jobCardId: input.jobCardId ?? null,
      quotationId: input.quotationId ?? null,
      issueDate: null,
      dueDate: null,
      validUntil: null,
      followUpDate: null,
      reference: input.reference.trim(),
      scope: input.scope?.trim() || '',
      workItems: input.workItems || [],
      intakeItems: input.intakeItems || [],
      diagnosticSections: input.diagnosticSections || [],
      authorization: input.authorization || {
        technicianSignature: '',
        technicianDate: '',
        clientSignature: '',
        clientDate: '',
      },
      lines,
      taxCents: totals.taxCents,
      subtotalCents: totals.subtotalCents,
      totalCents: totals.totalCents,
      depositCents: input.depositCents ?? null,
      payments: [],
      status: 'draft',
      terms: input.terms || '',
      exclusions: input.exclusions || '',
      turnaround: input.turnaround || '',
      notes: input.notes || '',
      folderPath,
      createdAt: nowIso,
      updatedAt: nowIso,
    };

    this.documents.push(newDoc);
    return JSON.parse(JSON.stringify(newDoc));
  }

  async updateDraft(id: string, input: Partial<CreateDraftInput>): Promise<Doc> {
    const idx = this.documents.findIndex((d) => d.id === id);
    if (idx === -1) {
      throw new ApiError(404, 'NOT_FOUND', `Document ${id} not found`);
    }
    const current = this.documents[idx];
    if (current.number !== null) {
      throw new ApiError(409, 'CANNOT_EDIT_ISSUED', 'Cannot edit an already issued document');
    }
    if (
      input.clientId !== undefined &&
      input.clientId !== current.clientId &&
      (!this.clients.some((client) => client.id === input.clientId) ||
        this.archivedClientIds.has(input.clientId))
    ) {
      throw new ApiError(400, 'VALIDATION_FAILED', 'Client not found');
    }

    const lines = input.lines !== undefined ? input.lines.map((l) => ({
      ...l,
      totalCents: l.totalCents !== undefined ? l.totalCents : Math.round(l.qty * l.unitPriceCents),
    })) : current.lines;

    const totals = calculateTotals(lines, this.settings.taxRate, current.payments);
    const nowIso = new Date().toISOString();

    const updated: Doc = {
      ...current,
      clientId: input.clientId !== undefined ? input.clientId : current.clientId,
      reference: input.reference !== undefined ? input.reference.trim() : current.reference,
      scope: input.scope !== undefined ? input.scope.trim() : current.scope,
      jobCardId: input.jobCardId !== undefined ? input.jobCardId : current.jobCardId,
      quotationId: input.quotationId !== undefined ? input.quotationId : current.quotationId,
      workItems: input.workItems !== undefined ? input.workItems : current.workItems,
      intakeItems: input.intakeItems !== undefined ? input.intakeItems : current.intakeItems,
      diagnosticSections:
        input.diagnosticSections !== undefined ? input.diagnosticSections : current.diagnosticSections,
      authorization: input.authorization !== undefined ? input.authorization : current.authorization,
      lines,
      depositCents: input.depositCents !== undefined ? input.depositCents : current.depositCents,
      terms: input.terms !== undefined ? input.terms : current.terms,
      exclusions: input.exclusions !== undefined ? input.exclusions : current.exclusions,
      turnaround: input.turnaround !== undefined ? input.turnaround : current.turnaround,
      notes: input.notes !== undefined ? input.notes : current.notes,
      subtotalCents: totals.subtotalCents,
      taxCents: totals.taxCents,
      totalCents: totals.totalCents,
      updatedAt: nowIso,
    };

    this.documents[idx] = updated;
    return JSON.parse(JSON.stringify(updated));
  }

  async issueDocument(id: string, issueDate?: ISODate): Promise<Doc> {
    const idx = this.documents.findIndex((d) => d.id === id);
    if (idx === -1) {
      throw new ApiError(404, 'NOT_FOUND', `Document ${id} not found`);
    }
    const doc = this.documents[idx];
    if (doc.number !== null) {
      throw new ApiError(409, 'ALREADY_ISSUED', `Document ${id} has already been issued with number ${doc.number}`);
    }

    const dateToUse = issueDate || this.today;
    const year = parseISODate(dateToUse).getUTCFullYear();
    const assignedNumber = this.getNextNumber(doc.type, year);

    let dueDate = doc.dueDate;
    let validUntil = doc.validUntil;

    if (doc.type === 'invoice') {
      dueDate = addDays(dateToUse, this.settings.defaultTermsDays || 7);
    } else if (doc.type === 'quotation') {
      validUntil = addDays(dateToUse, this.settings.quoteValidityDays || 14);
    }

    const status = doc.type === 'jobcard' ? 'open' : 'sent';
    const nowIso = new Date().toISOString();

    const issuedDoc: Doc = {
      ...doc,
      number: assignedNumber,
      issueDate: dateToUse,
      dueDate,
      validUntil,
      status,
      updatedAt: nowIso,
    };

    this.documents[idx] = issuedDoc;
    return JSON.parse(JSON.stringify(issuedDoc));
  }

  async addPayment(id: string, input: AddPaymentInput): Promise<Doc> {
    const idx = this.documents.findIndex((d) => d.id === id);
    if (idx === -1) {
      throw new ApiError(404, 'NOT_FOUND', `Document ${id} not found`);
    }
    const doc = this.documents[idx];
    if (doc.type !== 'invoice') {
      throw new ApiError(400, 'INVALID_DOCUMENT_TYPE', 'Payments can only be added to invoices');
    }

    if (!input.date || !input.amountCents || input.amountCents <= 0) {
      throw new ApiError(400, 'VALIDATION_FAILED', 'Valid payment date and positive amount in cents are required');
    }

    const currentPaid = (doc.payments || []).reduce((acc, p) => acc + p.amountCents, 0);
    const balance = doc.totalCents - currentPaid;

    if (input.amountCents > balance) {
      throw new ApiError(
        422,
        'OVERPAYMENT_REJECTED',
        `Payment of ${input.amountCents} cents exceeds outstanding balance of ${balance} cents`
      );
    }

    const payment = {
      id: `pay-${Date.now()}-${Math.floor(Math.random() * 1000)}`,
      date: input.date,
      amountCents: input.amountCents,
      method: input.method || 'EFT',
      reference: input.reference || '',
    };

    const updatedPayments = [...(doc.payments || []), payment];
    const newPaid = currentPaid + input.amountCents;
    const newBalance = doc.totalCents - newPaid;

    let newStatus = doc.status;
    if (newBalance <= 0) {
      newStatus = 'paid';
    } else if (newPaid > 0) {
      newStatus = 'partially_paid';
    }

    const updatedDoc: Doc = {
      ...doc,
      payments: updatedPayments,
      status: newStatus,
      updatedAt: new Date().toISOString(),
    };

    this.documents[idx] = updatedDoc;
    return JSON.parse(JSON.stringify(updatedDoc));
  }

  async updateStatus(id: string, status: string): Promise<Doc> {
    const idx = this.documents.findIndex((d) => d.id === id);
    if (idx === -1) {
      throw new ApiError(404, 'NOT_FOUND', `Document ${id} not found`);
    }
    const updated: Doc = {
      ...this.documents[idx],
      status,
      updatedAt: new Date().toISOString(),
    };
    this.documents[idx] = updated;
    return JSON.parse(JSON.stringify(updated));
  }

  async setFollowUp(id: string, followUpDate: ISODate | null): Promise<Doc> {
    const idx = this.documents.findIndex((d) => d.id === id);
    if (idx === -1) {
      throw new ApiError(404, 'NOT_FOUND', `Document ${id} not found`);
    }
    const doc = this.documents[idx];
    if (doc.type !== 'invoice') {
      throw new ApiError(400, 'INVALID_DOCUMENT_TYPE', 'Follow-up dates apply to invoices only');
    }
    if (followUpDate !== null && (!/^\d{4}-\d{2}-\d{2}$/.test(followUpDate) || Number.isNaN(parseISODate(followUpDate).getTime()))) {
      throw new ApiError(400, 'VALIDATION_FAILED', 'followUpDate must be YYYY-MM-DD or null');
    }
    if (!doc.number || doc.status === 'cancelled') {
      throw new ApiError(409, 'INVALID_STATE', 'Follow-up dates can only be set on issued, non-cancelled invoices');
    }
    const updated: Doc = { ...doc, followUpDate, updatedAt: new Date().toISOString() };
    this.documents[idx] = updated;
    return JSON.parse(JSON.stringify(updated));
  }

  async convertQuotation(id: string): Promise<Doc> {
    const idx = this.documents.findIndex((d) => d.id === id);
    if (idx === -1) {
      throw new ApiError(404, 'NOT_FOUND', `Quotation ${id} not found`);
    }
    const quo = this.documents[idx];
    if (quo.type !== 'quotation') {
      throw new ApiError(409, 'INVALID_STATE', 'Only quotations can be converted to invoices');
    }
    if (quo.status === 'converted') {
      throw new ApiError(409, 'INVALID_STATE', 'Quotation is already converted');
    }

    const newInvoiceId = `inv-conv-${Date.now()}-${Math.floor(Math.random() * 1000)}`;
    const { invoice, updatedQuotation } = convertQuotationToInvoice(quo, newInvoiceId);

    this.documents[idx] = updatedQuotation;
    this.documents.push(invoice);
    return JSON.parse(JSON.stringify(invoice));
  }

  async exportDocument(id: string, format: 'docx' | 'pdf'): Promise<Blob> {
    const doc = await this.getDocument(id);
    if (doc.number === null) {
      throw new ApiError(400, 'DRAFT_CANNOT_EXPORT', 'Drafts cannot be exported. Only issued documents can be exported.');
    }
    if (format === 'pdf' && !this.capabilities.pdf) {
      throw new ApiError(501, 'FORMAT_NOT_SUPPORTED', 'PDF export capability is not supported on backend');
    }
    if (format === 'docx' && !this.capabilities.docx) {
      throw new ApiError(501, 'FORMAT_NOT_SUPPORTED', 'DOCX export capability is not supported on backend');
    }

    const mime = format === 'pdf' ? 'application/pdf' : 'application/vnd.openxmlformats-officedocument.wordprocessingml.document';
    const content = `Mock binary content for ${doc.number} (${format.toUpperCase()})`;
    return new Blob([content], { type: mime });
  }

  async revealFolder(id: string): Promise<{ success: boolean; path: string }> {
    const doc = await this.getDocument(id);
    if (!this.capabilities.revealInFolder) {
      throw new ApiError(501, 'NOT_SUPPORTED', 'Reveal folder capability is not supported');
    }
    const path = doc.folderPath || `C:\\JobCard\\Clients\\${doc.clientId}`;
    return { success: true, path };
  }

  // --- Dashboard ---
  async getDashboard(year?: number): Promise<DashboardSummary> {
    const targetYear = year || parseISODate(this.today).getUTCFullYear();
    const invoices = this.documents.filter((d) => d.type === 'invoice');

    let invoicedCents = 0;
    let paidCents = 0;
    let outstandingCents = 0;
    let overdueCents = 0;

    const statusCounts: Record<string, number> = {
      draft: 0,
      sent: 0,
      partially_paid: 0,
      paid: 0,
      overdue: 0,
      cancelled: 0,
    };

    let followUpsDue = 0;
    let dueNext7Count = 0;
    let dueNext7Cents = 0;

    const ageing = {
      d1_30: 0,
      d31_60: 0,
      d61_90: 0,
      d90plus: 0,
    };

    const monthlyMap: Record<number, { jobs: number; invoicedCents: number; paidCents: number; outstandingCents: number }> = {};
    for (let m = 1; m <= 12; m++) {
      monthlyMap[m] = { jobs: 0, invoicedCents: 0, paidCents: 0, outstandingCents: 0 };
    }

    // Also count job cards per month
    for (const jc of this.documents.filter((d) => d.type === 'jobcard')) {
      if (jc.issueDate) {
        const d = parseISODate(jc.issueDate);
        if (d.getUTCFullYear() === targetYear) {
          const m = d.getUTCMonth() + 1;
          monthlyMap[m].jobs += 1;
        }
      }
    }

    for (const inv of invoices) {
      const derivedStatus = deriveInvoiceStatus(inv, this.today);
      if (statusCounts[derivedStatus] !== undefined) {
        statusCounts[derivedStatus] += 1;
      }

      if (derivedStatus === 'cancelled') continue;

      const invPaid = (inv.payments || []).reduce((acc, p) => acc + p.amountCents, 0);
      const invBalance = inv.totalCents - invPaid;

      // Filter by issueDate year if issued
      if (inv.issueDate) {
        const d = parseISODate(inv.issueDate);
        if (d.getUTCFullYear() === targetYear) {
          invoicedCents += inv.totalCents;
          paidCents += invPaid;
          outstandingCents += invBalance;

          const m = d.getUTCMonth() + 1;
          monthlyMap[m].invoicedCents += inv.totalCents;
          monthlyMap[m].paidCents += invPaid;
          monthlyMap[m].outstandingCents += invBalance;
        }
      }

      // Overdue & Ageing buckets
      const daysOverdue = calculateDaysOverdue(inv.dueDate, this.today, invBalance);
      if (daysOverdue !== null && daysOverdue > 0) {
        overdueCents += invBalance;
        if (daysOverdue <= 30) {
          ageing.d1_30 += invBalance;
        } else if (daysOverdue <= 60) {
          ageing.d31_60 += invBalance;
        } else if (daysOverdue <= 90) {
          ageing.d61_90 += invBalance;
        } else {
          ageing.d90plus += invBalance;
        }
      }

      // Follow-ups: reminder date reached on an unpaid invoice
      if (inv.followUpDate && invBalance > 0 && inv.followUpDate <= this.today) {
        followUpsDue += 1;
      }

      // Attention: due in next 7 days
      if (inv.dueDate && invBalance > 0) {
        const todayMs = parseISODate(this.today).getTime();
        const dueMs = parseISODate(inv.dueDate).getTime();
        const diffDays = Math.round((dueMs - todayMs) / (1000 * 60 * 60 * 24));
        if (diffDays >= 0 && diffDays <= 7) {
          dueNext7Count += 1;
          dueNext7Cents += invBalance;
        }
      }
    }

    const monthly = Object.keys(monthlyMap).map((mKey) => {
      const m = Number(mKey);
      return {
        month: m,
        ...monthlyMap[m],
      };
    });

    return {
      year: targetYear,
      totals: {
        invoicedCents,
        paidCents,
        outstandingCents,
        overdueCents,
      },
      statusCounts,
      attention: {
        followUpsDue,
        dueNext7Count,
        dueNext7Cents,
      },
      ageing,
      monthly,
      nextNumbers: {
        jobcard: this.getNextNumber('jobcard', targetYear),
        quotation: this.getNextNumber('quotation', targetYear),
        invoice: this.getNextNumber('invoice', targetYear),
      },
    };
  }

  // --- Settings & Storage ---
  async getSettings(): Promise<Settings> {
    return { ...this.settings };
  }

  async updateSettings(settings: Settings): Promise<Settings> {
    this.settings = { ...settings };
    return { ...this.settings };
  }

  async getStorage(): Promise<Storage> {
    return { ...this.storage };
  }

  async getCapabilities(): Promise<Capabilities> {
    return { ...this.capabilities };
  }

  async exportRegisterSpreadsheet(): Promise<Blob> {
    const csvOrXlsx = `Document Register\nTotal Documents: ${this.documents.length}\n`;
    return new Blob([csvOrXlsx], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    });
  }

  // Helper for sequential numbering continuation
  public getNextNumber(type: DocType, year: number): string {
    const prefix = type === 'jobcard' ? 'JC' : type === 'quotation' ? 'QUO' : 'INV';
    const regex = new RegExp(`^${prefix}-${year}-(\\d+)$`);

    let maxNum = 0;
    for (const d of this.documents) {
      if (d.type === type && d.number) {
        const match = d.number.match(regex);
        if (match) {
          const num = parseInt(match[1], 10);
          if (num > maxNum) maxNum = num;
        }
      }
    }

    const nextSeq = String(maxNum + 1).padStart(3, '0');
    return `${prefix}-${year}-${nextSeq}`;
  }
}
