import test from 'node:test';
import assert from 'node:assert/strict';
import { addDays, daysBetween, resolveRelativeDate, calculateDaysOverdue } from './dates.ts';

test('addDays adds positive and negative days', () => {
  assert.equal(addDays('2026-11-01', 7), '2026-11-08');
  assert.equal(addDays('2026-11-08', -8), '2026-10-31');
});

test('daysBetween computes day difference accurately', () => {
  assert.equal(daysBetween('2026-11-01', '2026-11-08'), 7);
  assert.equal(daysBetween('2026-11-08', '2026-11-01'), -7);
  assert.equal(daysBetween('2026-11-08', '2026-11-08'), 0);
});

test('resolveRelativeDate resolves seed dates correctly', () => {
  const today = '2026-11-08';
  assert.equal(resolveRelativeDate('today', today), '2026-11-08');
  assert.equal(resolveRelativeDate('today-45', today), '2026-09-24');
  assert.equal(resolveRelativeDate('today+7', today), '2026-11-15');
  assert.equal(resolveRelativeDate('2026-01-01', today), '2026-01-01');
});

test('calculateDaysOverdue returns correct number or null', () => {
  const today = '2026-11-08';
  // past due date with balance
  assert.equal(calculateDaysOverdue('2026-11-01', today, 5000), 7);
  // past due date with zero balance
  assert.equal(calculateDaysOverdue('2026-11-01', today, 0), null);
  // future due date with balance
  assert.equal(calculateDaysOverdue('2026-11-15', today, 5000), null);
  // null due date
  assert.equal(calculateDaysOverdue(null, today, 5000), null);
});
