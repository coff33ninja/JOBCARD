import React, { useState } from 'react';
import { X, UserPlus, AlertCircle } from 'lucide-react';
import { Client, ClientDuplicateMatch } from '../api/types.ts';
import { CreateClientInput } from '../api/client.ts';

export interface ClientModalProps {
  isOpen: boolean;
  onClose: () => void;
  client?: Client | null;
  findDuplicates: (clientData: CreateClientInput) => Promise<ClientDuplicateMatch[]>;
  onSubmit: (clientData: CreateClientInput) => Promise<void>;
}

export const ClientModal: React.FC<ClientModalProps> = ({
  isOpen,
  onClose,
  client,
  findDuplicates,
  onSubmit,
}) => {
  const [form, setForm] = useState<Partial<CreateClientInput>>({});
  const [error, setError] = useState<string | null>(null);
  const [duplicateMatches, setDuplicateMatches] = useState<ClientDuplicateMatch[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const fieldValue = (field: keyof CreateClientInput): string =>
    form[field] ?? client?.[field] ?? '';

  const updateField = (field: keyof CreateClientInput, value: string) => {
    setForm((current) => ({ ...current, [field]: value }));
    setDuplicateMatches([]);
  };

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!fieldValue('name').trim()) {
      setError('Client or Business Name is required');
      return;
    }
    const input: CreateClientInput = {
      name: fieldValue('name').trim(),
      contact: fieldValue('contact').trim(),
      phone: fieldValue('phone').trim(),
      email: fieldValue('email').trim(),
      address: fieldValue('address').trim(),
      notes: fieldValue('notes').trim(),
    };
    setError(null);
    try {
      setIsSubmitting(true);
      if (!client && duplicateMatches.length === 0) {
        const matches = await findDuplicates(input);
        if (matches.length > 0) {
          setDuplicateMatches(matches);
          return;
        }
      }
      await onSubmit(input);
      onClose();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to save client');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 backdrop-blur-xs p-4">
      <div className="bg-white rounded-lg shadow-xl border border-slate-200 w-full max-w-lg overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        <div className="flex items-center justify-between px-5 py-4 border-b border-slate-100 bg-slate-50/50">
          <div className="flex items-center gap-2">
            <UserPlus className="w-5 h-5 text-indigo-600" />
            <h2 className="text-base font-semibold text-slate-800">
              {client ? 'Edit Client Profile' : 'New Client Profile'}
            </h2>
          </div>
          <button
            onClick={onClose}
            disabled={isSubmitting}
            aria-label="Close client modal"
            className="text-slate-400 hover:text-slate-600 p-1 rounded-md"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-5 space-y-4 text-sm">
          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Business / Client Name <span className="text-rose-500">*</span>
            </label>
            <input
              type="text"
              required
              placeholder="e.g. Acme Corporation"
              value={fieldValue('name')}
              onChange={(e) => updateField('name', e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-sm"
            />
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">
                Contact Person
              </label>
              <input
                type="text"
                placeholder="e.g. Jane Doe"
                value={fieldValue('contact')}
                onChange={(e) => updateField('contact', e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-sm"
              />
            </div>
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">
                Phone Number
              </label>
              <input
                type="text"
                placeholder="e.g. 082 123 4567"
                value={fieldValue('phone')}
                onChange={(e) => updateField('phone', e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-sm"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Email Address
            </label>
            <input
              type="email"
              placeholder="e.g. accounts@acme.co.za"
              value={fieldValue('email')}
              onChange={(e) => updateField('email', e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-sm"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Physical / Billing Address
            </label>
            <textarea
              rows={2}
              placeholder="e.g. 10 Main Road, Cape Town"
              value={fieldValue('address')}
              onChange={(e) => updateField('address', e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-sm resize-none"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Client Notes
            </label>
            <input
              type="text"
              placeholder="e.g. Server room access code 1234"
              value={fieldValue('notes')}
              onChange={(e) => updateField('notes', e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-sm"
            />
          </div>

          {duplicateMatches.length > 0 && (
            <div role="alert" className="flex items-start gap-2 p-3 text-xs bg-amber-50 text-amber-900 border border-amber-200 rounded-md">
              <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
              <div>
                <p className="font-semibold">Possible existing clients found. Review before continuing.</p>
                <ul className="mt-1 list-disc pl-4">
                  {duplicateMatches.map((match) => (
                    <li key={match.client.id}>
                      {match.client.name}{match.client.contact ? ` — ${match.client.contact}` : ''}
                      {match.archived ? ' (archived)' : ''} — matched by {match.reasons.join(', ')}
                    </li>
                  ))}
                </ul>
                <p className="mt-1">If these are different customers, you can still create this client.</p>
              </div>
            </div>
          )}

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
              {isSubmitting ? 'Saving...' : duplicateMatches.length > 0 ? 'Create Anyway' : client ? 'Update Client' : 'Create Client'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
