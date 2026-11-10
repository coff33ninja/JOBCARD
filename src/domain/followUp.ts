import { Doc, ISODate } from '../api/types.ts';

export type FollowUpState = 'none' | 'scheduled' | 'due';

type FollowUpDoc = Pick<Doc, 'type' | 'number' | 'status' | 'followUpDate' | 'totalCents' | 'payments'>;

/** Only issued, non-cancelled invoices can carry a follow-up reminder. */
export function canSetFollowUp(doc: Pick<Doc, 'type' | 'number' | 'status'>): boolean {
  return doc.type === 'invoice' && doc.number !== null && doc.status !== 'cancelled';
}

/**
 * 'due' when the reminder date has been reached and money is still owed (same rule as the
 * dashboard's "follow-ups due"), 'scheduled' when it is in the future, otherwise 'none'.
 */
export function followUpState(doc: FollowUpDoc, today: ISODate): FollowUpState {
  if (!canSetFollowUp(doc) || !doc.followUpDate) return 'none';
  const paid = (doc.payments || []).reduce((sum, p) => sum + p.amountCents, 0);
  if (doc.totalCents - paid <= 0) return 'none';
  return doc.followUpDate <= today ? 'due' : 'scheduled';
}
