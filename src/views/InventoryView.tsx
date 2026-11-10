import React, { useCallback, useEffect, useState } from 'react';
import { Package, Plus, Trash2 } from 'lucide-react';
import { getApiClient } from '../api/factory.ts';
import { Client, StockItem, StockSource } from '../api/types.ts';
import { formatCents, parseRandsToCents } from '../domain/money.ts';

interface IntakeRow {
  description: string;
  quantity: string;
  unitCost: string;
  markupPercent: string;
}

const newIntakeRow = (): IntakeRow => ({
  description: '',
  quantity: '1',
  unitCost: '',
  markupPercent: '9',
});

function messageFromError(error: unknown): string {
  return error instanceof Error ? error.message : 'The stock operation failed.';
}

export const InventoryView: React.FC = () => {
  const api = getApiClient();
  const [items, setItems] = useState<StockItem[]>([]);
  const [clients, setClients] = useState<Client[]>([]);
  const [source, setSource] = useState<StockSource>('purchase');
  const [supplier, setSupplier] = useState('');
  const [couriered, setCouriered] = useState(false);
  const [courierCost, setCourierCost] = useState('');
  const [rows, setRows] = useState<IntakeRow[]>([newIntakeRow()]);
  const [selectedClients, setSelectedClients] = useState<Record<string, string>>({});
  const [assignmentQuantities, setAssignmentQuantities] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [stockItems, activeClients] = await Promise.all([api.listStockItems(), api.listClients()]);
      setItems(stockItems);
      setClients(activeClients);
      setError('');
    } catch (loadError) {
      setError(messageFromError(loadError));
    } finally {
      setLoading(false);
    }
  }, [api]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  const updateRow = (index: number, patch: Partial<IntakeRow>) => {
    setRows((current) => current.map((row, rowIndex) => rowIndex === index ? { ...row, ...patch } : row));
  };

  const handleCreateIntake = async (event: React.FormEvent) => {
    event.preventDefault();
    setError('');
    setNotice('');

    if (source === 'purchase' && !supplier.trim()) {
      setError('Enter the shop or supplier for purchased stock.');
      return;
    }

    const intakeItems = rows.map((row) => ({
      description: row.description.trim(),
      quantity: Number(row.quantity),
      unitCostCents: parseRandsToCents(row.unitCost),
      markupPercent: Number(row.markupPercent),
    }));
    if (intakeItems.some((item) =>
      !item.description || !Number.isInteger(item.quantity) || item.quantity < 1 ||
      item.unitCostCents <= 0 || ![8, 9, 10].includes(item.markupPercent)
    )) {
      setError('Each item needs a description, positive quantity and cost, and an 8%, 9%, or 10% markup.');
      return;
    }

    const courierCostInput = courierCost.trim();
    const courierCostValue = couriered ? Number(courierCostInput) : 0;
    const courierCostCents = Math.round(courierCostValue * 100);
    if (couriered && (
      !courierCostInput ||
      !Number.isFinite(courierCostValue) ||
      courierCostValue < 0 ||
      !Number.isSafeInteger(courierCostCents)
    )) {
      setError('Enter a valid courier cost in rand, or enter 0 when delivery was free.');
      return;
    }

    setSaving(true);
    try {
      const created = await api.createStockIntake({
        source,
        supplier: supplier.trim(),
        couriered: source === 'purchase' && couriered,
        courierCostCents: source === 'purchase' ? courierCostCents : 0,
        items: intakeItems,
      });
      setRows([newIntakeRow()]);
      setSupplier('');
      setCouriered(false);
      setCourierCost('');
      setNotice(`Recorded ${created.length} stock item${created.length === 1 ? '' : 's'}.`);
      await loadData();
    } catch (saveError) {
      setError(messageFromError(saveError));
    } finally {
      setSaving(false);
    }
  };

  const handleAssign = async (item: StockItem) => {
    setError('');
    setNotice('');
    const clientId = selectedClients[item.id] || '';
    const quantity = Number(assignmentQuantities[item.id] || '1');
    if (!clientId) {
      setError('Choose a client before assigning stock.');
      return;
    }
    if (!Number.isInteger(quantity) || quantity < 1 || quantity > item.quantityAvailable) {
      setError('Enter a quantity within the available stock.');
      return;
    }

    setSaving(true);
    try {
      const result = await api.allocateStock(item.id, clientId, quantity);
      const client = clients.find((candidate) => candidate.id === clientId);
      setNotice(`${quantity} × ${item.description} added to ${client?.name || 'the client'}'s draft invoice.`);
      setAssignmentQuantities((current) => ({ ...current, [item.id]: '1' }));
      await loadData();
      if (result.document.status !== 'draft') {
        setError('The stock was assigned, but its invoice is not in draft status.');
      }
    } catch (assignError) {
      setError(messageFromError(assignError));
    } finally {
      setSaving(false);
    }
  };

  const availableUnits = items.reduce((total, item) => total + item.quantityAvailable, 0);
  const allocatedUnits = items.reduce((total, item) => total + item.quantityReceived - item.quantityAvailable, 0);

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <header className="mb-6 flex items-start gap-3">
        <div className="rounded-lg bg-indigo-100 p-2 text-indigo-700">
          <Package className="h-6 w-6" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Stock &amp; Inventory</h1>
          <p className="mt-1 text-sm text-slate-600">Record purchases or stock on hand, then assign items to clients.</p>
        </div>
      </header>

      {(error || notice) && (
        <div
          role={error ? 'alert' : 'status'}
          className={`mb-5 rounded-md border px-4 py-3 text-sm ${
            error ? 'border-red-200 bg-red-50 text-red-800' : 'border-green-200 bg-green-50 text-green-800'
          }`}
        >
          {error || notice}
        </div>
      )}

      <section className="mb-6 grid gap-4 sm:grid-cols-3">
        <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Stock entries</p>
          <p className="mt-1 text-2xl font-bold text-slate-900">{items.length}</p>
        </div>
        <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Units available</p>
          <p className="mt-1 text-2xl font-bold text-slate-900">{availableUnits}</p>
        </div>
        <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Units assigned</p>
          <p className="mt-1 text-2xl font-bold text-slate-900">{allocatedUnits}</p>
        </div>
      </section>

      <section className="mb-8 rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
        <h2 className="mb-4 text-lg font-semibold text-slate-900">Record stock intake</h2>
        <form onSubmit={handleCreateIntake} className="space-y-5">
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block text-sm font-medium text-slate-700">
              Stock source
              <select
                value={source}
                onChange={(event) => {
                  const nextSource = event.target.value as StockSource;
                  setSource(nextSource);
                  if (nextSource === 'on_hand') {
                    setCouriered(false);
                    setCourierCost('');
                  }
                }}
                className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm"
              >
                <option value="purchase">New purchase</option>
                <option value="on_hand">Stock already on hand</option>
              </select>
            </label>
            <label className="block text-sm font-medium text-slate-700">
              Supplier or shop {source === 'purchase' ? <span className="text-red-600">*</span> : <span className="text-slate-400">(optional)</span>}
              <input
                value={supplier}
                onChange={(event) => setSupplier(event.target.value)}
                required={source === 'purchase'}
                className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm"
                placeholder="Where the stock was bought"
              />
            </label>
          </div>

          {source === 'purchase' && (
            <div className="grid gap-4 sm:grid-cols-2">
              <label className="flex items-center gap-2 text-sm font-medium text-slate-700">
                <input
                  type="checkbox"
                  checked={couriered}
                  onChange={(event) => {
                    setCouriered(event.target.checked);
                    if (!event.target.checked) setCourierCost('');
                  }}
                  className="h-4 w-4 rounded border-slate-300 text-indigo-600"
                />
                This purchase was couriered
              </label>
              {couriered && (
                <label className="block text-sm font-medium text-slate-700">
                  Courier cost (R)
                  <input
                    type="number"
                    min="0"
                    step="0.01"
                    value={courierCost}
                    onChange={(event) => setCourierCost(event.target.value)}
                    className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm"
                    placeholder="0.00"
                  />
                </label>
              )}
            </div>
          )}

          <div className="space-y-3">
            <div className="hidden grid-cols-[minmax(12rem,2fr)_6rem_8rem_8rem_auto] gap-3 text-xs font-semibold uppercase tracking-wide text-slate-500 md:grid">
              <span>Item</span><span>Quantity</span><span>Unit cost (R)</span><span>Markup</span><span />
            </div>
            {rows.map((row, index) => (
              <div key={index} className="grid gap-3 rounded-md border border-slate-200 p-3 md:grid-cols-[minmax(12rem,2fr)_6rem_8rem_8rem_auto] md:items-end md:border-0 md:p-0">
                <label className="block text-xs font-semibold text-slate-500 md:text-sm md:text-slate-700">
                  Item description
                  <input
                    value={row.description}
                    onChange={(event) => updateRow(index, { description: event.target.value })}
                    className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm font-normal"
                    placeholder="e.g. 1TB SSD"
                    required
                  />
                </label>
                <label className="block text-xs font-semibold text-slate-500 md:text-sm md:text-slate-700">
                  Quantity
                  <input
                    type="number"
                    min="1"
                    step="1"
                    value={row.quantity}
                    onChange={(event) => updateRow(index, { quantity: event.target.value })}
                    className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm font-normal"
                    required
                  />
                </label>
                <label className="block text-xs font-semibold text-slate-500 md:text-sm md:text-slate-700">
                  Unit cost (R)
                  <input
                    type="number"
                    min="0.01"
                    step="0.01"
                    value={row.unitCost}
                    onChange={(event) => updateRow(index, { unitCost: event.target.value })}
                    className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm font-normal"
                    placeholder="0.00"
                    required
                  />
                </label>
                <label className="block text-xs font-semibold text-slate-500 md:text-sm md:text-slate-700">
                  Markup
                  <select
                    value={row.markupPercent}
                    onChange={(event) => updateRow(index, { markupPercent: event.target.value })}
                    className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-normal"
                  >
                    <option value="8">8%</option>
                    <option value="9">9%</option>
                    <option value="10">10%</option>
                  </select>
                </label>
                <button
                  type="button"
                  onClick={() => setRows((current) => current.filter((_, rowIndex) => rowIndex !== index))}
                  disabled={rows.length === 1}
                  className="inline-flex h-9 items-center justify-center rounded-md border border-slate-300 px-3 text-slate-600 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-40"
                  aria-label="Remove item"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            ))}
            <button
              type="button"
              onClick={() => setRows((current) => [...current, newIntakeRow()])}
              className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-sm font-semibold text-indigo-700 hover:bg-indigo-50"
            >
              <Plus className="h-4 w-4" /> Add another item to this intake
            </button>
          </div>

          {couriered && (
            <p className="text-xs text-slate-500">
              Courier cost is split between these items according to their purchase cost and included in client prices.
            </p>
          )}
          <div className="flex justify-end">
            <button
              type="submit"
              disabled={saving}
              className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {saving ? 'Saving…' : 'Save stock intake'}
            </button>
          </div>
        </form>
      </section>

      <section className="rounded-lg border border-slate-200 bg-white shadow-sm">
        <div className="border-b border-slate-200 px-5 py-4">
          <h2 className="text-lg font-semibold text-slate-900">Inventory</h2>
          <p className="mt-1 text-sm text-slate-500">Assigning stock reduces its available quantity and updates a draft invoice for the client.</p>
        </div>
        {loading ? (
          <p className="px-5 py-8 text-sm text-slate-500">Loading inventory…</p>
        ) : items.length === 0 ? (
          <p className="px-5 py-8 text-sm text-slate-500">No stock recorded yet. Add a purchase or enter stock you already have.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-200 text-sm">
              <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                <tr>
                  <th className="px-4 py-3">Item</th>
                  <th className="px-4 py-3">Source / supplier</th>
                  <th className="px-4 py-3">Cost / courier share</th>
                  <th className="px-4 py-3">Markup</th>
                  <th className="px-4 py-3">Available</th>
                  <th className="px-4 py-3">Assign to client</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {items.map((item) => (
                  <tr key={item.id} className="align-top">
                    <td className="px-4 py-3">
                      <p className="font-semibold text-slate-900">{item.description}</p>
                      <p className="mt-1 text-xs text-slate-500">{item.quantityReceived} received · {item.createdAt.slice(0, 10)}</p>
                    </td>
                    <td className="px-4 py-3 text-slate-700">
                      <p>{item.source === 'purchase' ? 'Purchase' : 'On hand'}</p>
                      <p className="mt-1 text-xs text-slate-500">{item.supplier || 'Supplier not recorded'}</p>
                      <p className="text-xs text-slate-500">{item.couriered ? 'Couriered' : 'Not couriered'}</p>
                    </td>
                    <td className="px-4 py-3 text-slate-700">
                      <p>{formatCents(item.unitCostCents)} per unit</p>
                      <p className="mt-1 text-xs text-slate-500">Courier allocated: {formatCents(item.courierAllocatedCents)}</p>
                    </td>
                    <td className="px-4 py-3 text-slate-700">{item.markupPercent}%</td>
                    <td className="px-4 py-3">
                      <span className={`font-semibold ${item.quantityAvailable ? 'text-emerald-700' : 'text-slate-500'}`}>
                        {item.quantityAvailable} / {item.quantityReceived}
                      </span>
                    </td>
                    <td className="min-w-64 px-4 py-3">
                      {item.quantityAvailable > 0 ? (
                        <div className="flex flex-wrap gap-2">
                          <select
                            value={selectedClients[item.id] || ''}
                            onChange={(event) => setSelectedClients((current) => ({ ...current, [item.id]: event.target.value }))}
                            className="min-w-36 flex-1 rounded-md border border-slate-300 bg-white px-2 py-1.5 text-xs"
                            aria-label={`Client for ${item.description}`}
                          >
                            <option value="">Choose client</option>
                            {clients.map((client) => <option key={client.id} value={client.id}>{client.name}</option>)}
                          </select>
                          <input
                            type="number"
                            min="1"
                            max={item.quantityAvailable}
                            value={assignmentQuantities[item.id] || '1'}
                            onChange={(event) => setAssignmentQuantities((current) => ({ ...current, [item.id]: event.target.value }))}
                            className="w-16 rounded-md border border-slate-300 px-2 py-1.5 text-xs"
                            aria-label={`Quantity of ${item.description}`}
                          />
                          <button
                            type="button"
                            onClick={() => void handleAssign(item)}
                            disabled={saving || clients.length === 0}
                            className="rounded-md bg-slate-900 px-3 py-1.5 text-xs font-semibold text-white hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-50"
                          >
                            Assign
                          </button>
                        </div>
                      ) : (
                        <span className="text-xs text-slate-500">Out of stock</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
};
