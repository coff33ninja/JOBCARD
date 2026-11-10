import test from 'node:test';
import assert from 'node:assert/strict';
import { stampFor } from './stamps.ts';
import { Doc } from '../api/types.ts';

const baseDoc: Doc = {
  id: '1',
  type: 'invoice',
  number: 'INV-2026-001',
  clientId: 'c1',
  jobCardId: null,
  quotationId: null,
  issueDate: '2026-11-01',
  dueDate: '2026-11-08',
  validUntil: null,
  followUpDate: null,
  reference: 'Device',
  scope: 'Repair',
  workItems: [],
  lines: [{ description: 'Item', qty: 1, unitPriceCents: 10000, totalCents: 10000 }],
  taxCents: 0,
  subtotalCents: 10000,
  totalCents: 10000,
  depositCents: null,
  payments: [],
  status: 'sent',
  terms: '',
  exclusions: '',
  turnaround: '',
  notes: '',
  folderPath: null,
  createdAt: '',
  updatedAt: '',
};

test('stampFor handles job cards (returns null)', () => {
  const jc = { ...baseDoc, type: 'jobcard' as const };
  assert.equal(stampFor(jc, '2026-11-08'), null);
});

test('stampFor invoice states', () => {
  const today = '2026-11-08';
  // DRAFT
  assert.deepEqual(stampFor({ ...baseDoc, number: null, status: 'draft' }, today), { text: 'DRAFT', color: 'grey' });
  // CANCELLED
  assert.deepEqual(stampFor({ ...baseDoc, status: 'cancelled' }, today), { text: 'CANCELLED', color: 'grey', isStruck: true });
  // PAID
  assert.deepEqual(
    stampFor({ ...baseDoc, payments: [{ id: 'p', date: today, amountCents: 10000, method: 'EFT', reference: '' }] }, today),
    { text: 'PAID', color: 'green' }
  );
  // PARTIALLY PAID
  assert.deepEqual(
    stampFor({ ...baseDoc, dueDate: '2026-11-15', payments: [{ id: 'p', date: today, amountCents: 5000, method: 'EFT', reference: '' }] }, today),
    { text: 'PARTIALLY PAID', color: 'amber' }
  );
  // OVERDUE
  assert.deepEqual(
    stampFor({ ...baseDoc, dueDate: '2026-11-01', payments: [] }, today),
    { text: 'OVERDUE', color: 'red', isStrong: true }
  );
  // UNPAID (issued, not past due)
  assert.deepEqual(
    stampFor({ ...baseDoc, dueDate: '2026-11-15', payments: [] }, today),
    { text: 'UNPAID', color: 'red' }
  );
});

test('stampFor quotation states', () => {
  const today = '2026-11-08';
  const baseQuo = { ...baseDoc, type: 'quotation' as const, number: 'QUO-2026-001' };
  assert.deepEqual(stampFor({ ...baseQuo, status: 'accepted' }, today), { text: 'ACCEPTED', color: 'green' });
  assert.deepEqual(stampFor({ ...baseQuo, status: 'declined' }, today), { text: 'DECLINED', color: 'red' });
  assert.deepEqual(stampFor({ ...baseQuo, status: 'sent', validUntil: '2026-11-01' }, today), { text: 'EXPIRED', color: 'grey' });
  assert.deepEqual(stampFor({ ...baseQuo, number: null, status: 'draft' }, today), { text: 'DRAFT', color: 'grey' });
});
