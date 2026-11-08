import test from 'node:test';
import assert from 'node:assert/strict';
import { formatCents, parseRandsToCents } from './money.ts';

test('formatCents formats correctly without locale spaces', () => {
  assert.equal(formatCents(0), 'R0.00');
  assert.equal(formatCents(50), 'R0.50');
  assert.equal(formatCents(100), 'R1.00');
  assert.equal(formatCents(100000), 'R1,000.00');
  assert.equal(formatCents(123456789), 'R1,234,567.89');
  assert.equal(formatCents(-5000), '-R50.00');
});

test('parseRandsToCents parses numbers and strings cleanly', () => {
  assert.equal(parseRandsToCents(10), 1000);
  assert.equal(parseRandsToCents('10.50'), 1050);
  assert.equal(parseRandsToCents('R1,000.00'), 100000);
  assert.equal(parseRandsToCents(''), 0);
  assert.equal(parseRandsToCents('invalid'), 0);
});
