import { Doc, ISODate } from '../api/types.ts';
import { deriveInvoiceStatus, deriveQuotationStatus } from './status.ts';

export type StampColor = 'green' | 'amber' | 'red' | 'grey';

export interface Stamp {
  text: string;
  color: StampColor;
  isStruck?: boolean;
  isStrong?: boolean;
}

export function stampFor(document: Doc, today: ISODate): Stamp | null {
  if (document.type === 'jobcard') {
    return null;
  }

  if (document.type === 'invoice') {
    if (document.status === 'cancelled') {
      return { text: 'CANCELLED', color: 'grey', isStruck: true };
    }
    if (document.status === 'draft' || !document.number) {
      return { text: 'DRAFT', color: 'grey' };
    }

    const derived = deriveInvoiceStatus(document, today);
    if (derived === 'paid') {
      return { text: 'PAID', color: 'green' };
    }
    if (derived === 'partially_paid') {
      return { text: 'PARTIALLY PAID', color: 'amber' };
    }
    if (derived === 'overdue') {
      return { text: 'OVERDUE', color: 'red', isStrong: true };
    }
    // Issued, not past due, unpaid:
    return { text: 'UNPAID', color: 'red' };
  }

  if (document.type === 'quotation') {
    if (document.status === 'draft' || !document.number) {
      return { text: 'DRAFT', color: 'grey' };
    }
    const derived = deriveQuotationStatus(document, today);
    if (derived === 'accepted') {
      return { text: 'ACCEPTED', color: 'green' };
    }
    if (derived === 'declined') {
      return { text: 'DECLINED', color: 'red' };
    }
    if (derived === 'expired') {
      return { text: 'EXPIRED', color: 'grey' };
    }
    if (derived === 'converted') {
      return { text: 'ACCEPTED', color: 'green' };
    }
    return { text: 'DRAFT', color: 'grey' };
  }

  return null;
}
