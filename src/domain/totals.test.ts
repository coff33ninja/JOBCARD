import test from 'node:test';
import assert from 'node:assert/strict';
import { calculateLineTotal, calculateTotals } from './totals.ts';

test('calculateLineTotal rounds correctly', () => {
  assert.equal(calculateLineTotal(1, 5000), 5000);
  assert.equal(calculateLineTotal(2.5, 1000), 2500);
  assert.equal(calculateLineTotal(3, 3333), 9999);
});

test('calculateTotals computes subtotal, zero tax, payments, and balance', () => {
  const lines = [
    { description: 'Service A', qty: 2, unitPriceCents: 5000, totalCents: 10000 },
    { description: 'Part B', qty: 1, unitPriceCents: 15000, totalCents: 15000 },
  ];
  const payments = [
    { id: 'p1', date: '2026-11-01', amountCents: 10000, method: 'EFT', reference: 'INV-1' }
  ];

  const result = calculateTotals(lines, 0, payments);
  assert.equal(result.subtotalCents, 25000);
  assert.equal(result.taxCents, 0);
  assert.equal(result.totalCents, 25000);
  assert.equal(result.paidCents, 10000);
  assert.equal(result.balanceCents, 15000);
});

test('calculateTotals handles tax rate > 0', () => {
  const lines = [
    { description: 'Service', qty: 1, unitPriceCents: 10000, totalCents: 10000 },
  ];
  const result = calculateTotals(lines, 0.15, []);
  assert.equal(result.subtotalCents, 10000);
  assert.equal(result.taxCents, 1500);
  assert.equal(result.totalCents, 11500);
  assert.equal(result.paidCents, 0);
  assert.equal(result.balanceCents, 11500);
});
