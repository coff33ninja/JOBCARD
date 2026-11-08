import { Doc, ISODate, Payment } from '../api/types.ts';
import { daysBetween } from './dates.ts';

export function deriveInvoiceStatus(
  invoice: Pick<Doc, 'status' | 'number' | 'totalCents' | 'payments' | 'dueDate'>,
  today: ISODate
): string {
  // cancelled stays cancelled
  if (invoice.status === 'cancelled') return 'cancelled';
  // draft stays draft until issued (or if number is null)
  if (invoice.status === 'draft' || !invoice.number) return 'draft';

  const total = invoice.totalCents;
  const paid = (invoice.payments || []).reduce((acc: number, p: Payment) => acc + p.amountCents, 0);
  const balance = total - paid;
  const isPastDue = invoice.dueDate ? daysBetween(invoice.dueDate, today) > 0 : false;

  if (balance <= 0 && total > 0) {
    return 'paid';
  }

  if (paid > 0 && paid < total) {
    return isPastDue && balance > 0 ? 'overdue' : 'partially_paid';
  }

  if (paid === 0) {
    if (isPastDue && balance > 0) {
      return 'overdue';
    }
    return 'sent';
  }

  return invoice.status || 'sent';
}

export function deriveQuotationStatus(
  quote: Pick<Doc, 'status' | 'validUntil'>,
  today: ISODate
): string {
  if (['accepted', 'declined', 'converted', 'draft'].includes(quote.status)) {
    return quote.status;
  }
  if (quote.validUntil && daysBetween(quote.validUntil, today) > 0) {
    return 'expired';
  }
  return quote.status || 'sent';
}

export function convertQuotationToInvoice(
  quotation: Doc,
  newInvoiceId: string,
  nowIso: string = new Date().toISOString()
): { invoice: Doc; updatedQuotation: Doc } {
  const updatedQuotation: Doc = {
    ...quotation,
    status: 'converted',
    updatedAt: nowIso,
  };

  const invoice: Doc = {
    id: newInvoiceId,
    type: 'invoice',
    number: null, // draft until issued
    clientId: quotation.clientId,
    jobCardId: quotation.jobCardId,
    quotationId: quotation.id,
    issueDate: null,
    dueDate: null,
    validUntil: null,
    reference: quotation.reference,
    scope: quotation.scope,
    workItems: quotation.workItems ? JSON.parse(JSON.stringify(quotation.workItems)) : [],
    lines: quotation.lines ? JSON.parse(JSON.stringify(quotation.lines)) : [],
    taxCents: quotation.taxCents,
    subtotalCents: quotation.subtotalCents,
    totalCents: quotation.totalCents,
    depositCents: null,
    payments: [],
    status: 'draft',
    terms: quotation.terms,
    exclusions: '',
    turnaround: '',
    notes: quotation.notes,
    folderPath: quotation.folderPath,
    createdAt: nowIso,
    updatedAt: nowIso,
  };

  return { invoice, updatedQuotation };
}
