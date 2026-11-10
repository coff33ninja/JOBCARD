import assert from 'node:assert/strict';
import test from 'node:test';
import { MockApi } from './mockApi';

test('MockApi tracks couriered stock and charges allocated items on one draft invoice', async () => {
  const api = new MockApi(undefined, '2026-11-08');
  const client = await api.createClient({ name: 'Stock Client' });
  const items = await api.createStockIntake({
    source: 'purchase',
    supplier: 'Parts Shop',
    couriered: true,
    courierCostCents: 1000,
    items: [
      { description: 'SSD', quantity: 2, unitCostCents: 10000, markupPercent: 10 },
      { description: 'RAM', quantity: 1, unitCostCents: 10000, markupPercent: 8 },
    ],
  });

  assert.equal(items[0].courierAllocatedCents, 667);
  assert.equal(items[1].courierAllocatedCents, 333);
  const first = await api.allocateStock(items[0].id, client.id, 1);
  assert.equal(first.item.quantityAvailable, 1);
  assert.equal(first.document.type, 'invoice');
  assert.equal(first.document.lines[0].unitPriceCents, 11367);
  assert.equal(first.document.lines[0].description, 'SSD');

  const second = await api.allocateStock(items[0].id, client.id, 1);
  assert.equal(second.document.id, first.document.id);
  assert.equal(second.document.lines[1].unitPriceCents, 11366);

  await assert.rejects(api.allocateStock(items[0].id, client.id, 1), /available stock/i);
  const inventory = await api.listStockItems();
  assert.equal(inventory[0].quantityAvailable, 0);
});

test('MockApi accepts existing stock without courier details and validates markup', async () => {
  const api = new MockApi(undefined, '2026-11-08');
  const [item] = await api.createStockIntake({
    source: 'on_hand',
    supplier: '',
    couriered: false,
    courierCostCents: 0,
    items: [{ description: 'Mouse', quantity: 1, unitCostCents: 1000, markupPercent: 8 }],
  });
  assert.equal(item.source, 'on_hand');
  assert.equal(item.courierAllocatedCents, 0);

  await assert.rejects(api.createStockIntake({
    source: 'on_hand',
    supplier: '',
    couriered: false,
    courierCostCents: 0,
    items: [{ description: 'Bad markup', quantity: 1, unitCostCents: 1000, markupPercent: 11 }],
  }), /markup/i);
});
