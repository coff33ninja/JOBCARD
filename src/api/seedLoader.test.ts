import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { loadSeed, RawSeedData } from './seedLoader.ts';

test('loadSeed correctly parses seed.json and calculates derived values', () => {
  const seedPath = path.resolve('seed/seed.json');
  assert.ok(fs.existsSync(seedPath), 'seed.json must exist');

  const rawSeed = JSON.parse(fs.readFileSync(seedPath, 'utf-8')) as RawSeedData;
  const today = '2026-11-08';
  const loaded = loadSeed(rawSeed, today);

  assert.equal(loaded.clients.length, 6);
  assert.equal(loaded.documents.length, 19);

  // Check that relative dates are resolved to concrete ISO dates
  const inv1 = loaded.documents.find((d) => d.id === 'inv-001')!;
  assert.equal(inv1.issueDate, '2026-10-09'); // today-30 from 2026-11-08
  assert.equal(inv1.dueDate, '2026-10-16'); // today-23

  // Check derived financial totals
  assert.equal(inv1.subtotalCents, 295000); // 165000 + 2*65000
  assert.equal(inv1.taxCents, 0); // 0% tax
  assert.equal(inv1.totalCents, 295000);
  assert.equal(inv1.payments[0].amountCents, 295000);
});
