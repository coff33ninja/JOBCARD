import test from 'node:test';
import assert from 'node:assert/strict';
import { followUpState, canSetFollowUp } from './followUp.ts';

const pay = (amountCents: number) => [{ id: 'p', date: '2026-11-01', amountCents, method: 'EFT', reference: '' }];
const inv = (over: Record<string, unknown> = {}) => ({
  type: 'invoice' as const,
  number: 'INV-2026-001' as string | null,
  status: 'sent',
  followUpDate: '2026-11-08' as string | null,
  totalCents: 10000,
  payments: [] as ReturnType<typeof pay>,
  ...over,
});
const today = '2026-11-08';

test('followUpState: due on or after the reminder date while money is owed', () => {
  assert.equal(followUpState(inv(), today), 'due');
  assert.equal(followUpState(inv({ followUpDate: '2026-11-01' }), today), 'due');
  assert.equal(followUpState(inv({ payments: pay(4000) }), today), 'due');
});

test('followUpState: scheduled before the date', () => {
  assert.equal(followUpState(inv({ followUpDate: '2026-11-09' }), today), 'scheduled');
});

test('followUpState: none when nothing is owed, unset, or not an issued live invoice', () => {
  assert.equal(followUpState(inv({ payments: pay(10000) }), today), 'none');
  assert.equal(followUpState(inv({ followUpDate: null }), today), 'none');
  assert.equal(followUpState(inv({ status: 'cancelled' }), today), 'none');
  assert.equal(followUpState(inv({ number: null, status: 'draft' }), today), 'none');
  assert.equal(followUpState(inv({ type: 'quotation' }), today), 'none');
});

test('canSetFollowUp: only issued, non-cancelled invoices', () => {
  assert.equal(canSetFollowUp(inv()), true);
  assert.equal(canSetFollowUp(inv({ payments: pay(10000) })), true); // can still be cleared
  assert.equal(canSetFollowUp(inv({ status: 'cancelled' })), false);
  assert.equal(canSetFollowUp(inv({ number: null, status: 'draft' })), false);
  assert.equal(canSetFollowUp(inv({ type: 'jobcard' })), false);
});
