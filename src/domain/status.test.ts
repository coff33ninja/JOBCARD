import test from 'node:test';
import assert from 'node:assert/strict';
import { deriveInvoiceStatus, deriveQuotationStatus, convertQuotationToInvoice } from './status.ts';
import { Doc } from '../api/types.ts';

test('deriveInvoiceStatus respects cancelled and draft', () => {
  const today = '2026-11-08';
  assert.equal(
    deriveInvoiceStatus({ status: 'cancelled', number: 'INV-2026-001', totalCents: 1000, payments: [], dueDate: '2026-10-01' }, today),
    'cancelled'
  );
  assert.equal(
    deriveInvoiceStatus({ status: 'draft', number: null, totalCents: 1000, payments: [], dueDate: null }, today),
    'draft'
  );
});

test('deriveInvoiceStatus marks paid when balance <= 0', () => {
  const today = '2026-11-08';
  const payments = [{ id: 'p1', date: '2026-11-05', amountCents: 5000, method: 'EFT', reference: 'ref' }];
  assert.equal(
    deriveInvoiceStatus({ status: 'sent', number: 'INV-2026-001', totalCents: 5000, payments, dueDate: '2026-11-01' }, today),
    'paid'
  );
});

test('deriveInvoiceStatus marks partially_paid or overdue', () => {
  const today = '2026-11-08';
  const partialPayment = [{ id: 'p1', date: '2026-11-05', amountCents: 2000, method: 'EFT', reference: 'ref' }];
  // before due date -> partially_paid
  assert.equal(
    deriveInvoiceStatus({ status: 'sent', number: 'INV-2026-001', totalCents: 5000, payments: partialPayment, dueDate: '2026-11-15' }, today),
    'partially_paid'
  );
  // past due date -> overdue
  assert.equal(
    deriveInvoiceStatus({ status: 'sent', number: 'INV-2026-001', totalCents: 5000, payments: partialPayment, dueDate: '2026-11-01' }, today),
    'overdue'
  );
});

test('deriveInvoiceStatus marks unpaid as sent or overdue', () => {
  const today = '2026-11-08';
  // before due date -> sent
  assert.equal(
    deriveInvoiceStatus({ status: 'sent', number: 'INV-2026-001', totalCents: 5000, payments: [], dueDate: '2026-11-15' }, today),
    'sent'
  );
  // past due date -> overdue
  assert.equal(
    deriveInvoiceStatus({ status: 'sent', number: 'INV-2026-001', totalCents: 5000, payments: [], dueDate: '2026-11-01' }, today),
    'overdue'
  );
});

test('deriveQuotationStatus derives expired correctly', () => {
  const today = '2026-11-08';
  assert.equal(deriveQuotationStatus({ status: 'sent', validUntil: '2026-11-01' }, today), 'expired');
  assert.equal(deriveQuotationStatus({ status: 'sent', validUntil: '2026-11-15' }, today), 'sent');
  assert.equal(deriveQuotationStatus({ status: 'accepted', validUntil: '2026-11-01' }, today), 'accepted');
  assert.equal(deriveQuotationStatus({ status: 'converted', validUntil: '2026-11-01' }, today), 'converted');
});

test('convertQuotationToInvoice creates draft invoice and marks quotation converted', () => {
  const quotation: Doc = {
    id: 'quo-1',
    type: 'quotation',
    number: 'QUO-2026-001',
    clientId: 'cli-1',
    jobCardId: 'jc-1',
    quotationId: null,
    issueDate: '2026-11-01',
    dueDate: null,
    validUntil: '2026-11-15',
    followUpDate: null,
    reference: 'Dell XPS 15',
    scope: 'Screen replacement',
    workItems: [{ title: 'Screen replace', detail: 'OLED panel' }],
    lines: [{ description: 'Panel', qty: 1, unitPriceCents: 450000, totalCents: 450000 }],
    taxCents: 0,
    subtotalCents: 450000,
    totalCents: 450000,
    depositCents: 200000,
    payments: [],
    status: 'accepted',
    terms: '7 days',
    exclusions: 'liquid damage',
    turnaround: '2 days',
    notes: 'Urgent',
    folderPath: '/clients/cli-1',
    createdAt: '2026-11-01T10:00:00Z',
    updatedAt: '2026-11-01T10:00:00Z',
  };

  const { invoice, updatedQuotation } = convertQuotationToInvoice(quotation, 'inv-new', '2026-11-08T12:00:00Z');
  assert.equal(updatedQuotation.status, 'converted');
  assert.equal(invoice.id, 'inv-new');
  assert.equal(invoice.type, 'invoice');
  assert.equal(invoice.number, null);
  assert.equal(invoice.quotationId, 'quo-1');
  assert.equal(invoice.clientId, 'cli-1');
  assert.equal(invoice.followUpDate, null);
  assert.equal(invoice.status, 'draft');
  assert.equal(invoice.lines.length, 1);
  assert.equal(invoice.lines[0].totalCents, 450000);
});
