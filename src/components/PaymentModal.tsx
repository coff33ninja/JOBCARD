import React, { useState } from 'react';
import { X, CreditCard, AlertCircle } from 'lucide-react';
import { Doc, Cents } from '../api/types.ts';
import { formatCents, parseRandsToCents } from '../domain/money.ts';
import { getTodayISO } from '../domain/dates.ts';

export interface PaymentModalProps {
  isOpen: boolean;
  onClose: () => void;
  doc: Doc;
  onSubmit: (payment: { date: string; amountCents: Cents; method: string; reference: string }) => Promise<void>;
}

export const PaymentModal: React.FC<PaymentModalProps> = ({
  isOpen,
  onClose,
  doc,
  onSubmit,
}) => {
  const [date, setDate] = useState(getTodayISO());
  const [amountStr, setAmountStr] = useState('');
  const [method, setMethod] = useState('EFT');
  const [reference, setReference] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  if (!isOpen) return null;

  const currentPaid = (doc.payments || []).reduce((acc, p) => acc + p.amountCents, 0);
  const balanceCents = doc.totalCents - currentPaid;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    const enteredCents = parseRandsToCents(amountStr);
    if (enteredCents <= 0) {
      setError('Please enter a payment amount greater than R0.00');
      return;
    }

    if (enteredCents > balanceCents) {
      setError(`Payment (${formatCents(enteredCents)}) cannot exceed the outstanding balance of ${formatCents(balanceCents)}`);
      return;
    }

    try {
      setIsSubmitting(true);
      await onSubmit({
        date,
        amountCents: enteredCents,
        method,
        reference: reference.trim(),
      });
      onClose();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to record payment');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 backdrop-blur-xs p-4">
      <div className="bg-white rounded-lg shadow-xl border border-slate-200 w-full max-w-md overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        <div className="flex items-center justify-between px-5 py-4 border-b border-slate-100 bg-slate-50/50">
          <div className="flex items-center gap-2">
            <CreditCard className="w-5 h-5 text-indigo-600" />
            <h2 className="text-base font-semibold text-slate-800">Record Payment</h2>
          </div>
          <button
            onClick={onClose}
            aria-label="Close payment dialog"
            className="text-slate-400 hover:text-slate-600 p-1 rounded-md"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-5 space-y-4 text-sm">
          <div className="bg-slate-50 p-3 rounded-md border border-slate-200 flex justify-between items-center text-xs">
            <div>
              <span className="text-slate-500 block">Invoice Total:</span>
              <span className="font-mono font-semibold text-slate-800">{formatCents(doc.totalCents)}</span>
            </div>
            <div>
              <span className="text-slate-500 block">Already Paid:</span>
              <span className="font-mono font-medium text-emerald-700">{formatCents(currentPaid)}</span>
            </div>
            <div>
              <span className="text-slate-500 block font-semibold text-slate-700">Balance Due:</span>
              <span className="font-mono font-bold text-slate-900">{formatCents(balanceCents)}</span>
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Payment Date
            </label>
            <input
              type="date"
              required
              value={date}
              onChange={(e) => setDate(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-sm"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Amount (Rands)
            </label>
            <div className="relative">
              <span className="absolute left-3 top-2 text-slate-400 font-mono text-sm">R</span>
              <input
                type="text"
                required
                placeholder="0.00"
                value={amountStr}
                onChange={(e) => setAmountStr(e.target.value)}
                className="w-full pl-8 pr-3 py-2 border border-slate-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-sm font-mono"
              />
            </div>
            <button
              type="button"
              onClick={() => setAmountStr((balanceCents / 100).toFixed(2))}
              className="text-xs text-indigo-600 hover:text-indigo-800 font-medium mt-1 inline-block"
            >
              Pay full balance ({formatCents(balanceCents)})
            </button>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">
                Payment Method
              </label>
              <select
                value={method}
                onChange={(e) => setMethod(e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-sm"
              >
                <option value="EFT">EFT / Bank Transfer</option>
                <option value="Cash">Cash</option>
                <option value="Card">Card</option>
                <option value="SnapScan">SnapScan</option>
                <option value="Zapper">Zapper</option>
                <option value="Other">Other</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">
                Bank Reference
              </label>
              <input
                type="text"
                placeholder="e.g. EFT-9821"
                value={reference}
                onChange={(e) => setReference(e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-sm"
              />
            </div>
          </div>

          {error && (
            <div className="flex items-center gap-2 p-3 text-xs bg-rose-50 text-rose-800 border border-rose-200 rounded-md">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <div className="flex justify-end gap-2 pt-3 border-t border-slate-100">
            <button
              type="button"
              onClick={onClose}
              disabled={isSubmitting}
              className="px-4 py-2 text-xs font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="px-4 py-2 text-xs font-semibold text-white bg-slate-900 rounded-md hover:bg-slate-800 transition-colors shadow-xs"
            >
              {isSubmitting ? 'Recording...' : 'Record Payment'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
