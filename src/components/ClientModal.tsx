import React, { useState } from 'react';
import { X, UserPlus, AlertCircle } from 'lucide-react';
import { Client } from '../api/types.ts';
import { CreateClientInput } from '../api/client.ts';

export interface ClientModalProps {
  isOpen: boolean;
  onClose: () => void;
  client?: Client | null;
  onSubmit: (clientData: CreateClientInput) => Promise<void>;
}

export const ClientModal: React.FC<ClientModalProps> = ({
  isOpen,
  onClose,
  client,
  onSubmit,
}) => {
  const [name, setName] = useState(client?.name || '');
  const [contact, setContact] = useState(client?.contact || '');
  const [phone, setPhone] = useState(client?.phone || '');
  const [email, setEmail] = useState(client?.email || '');
  const [address, setAddress] = useState(client?.address || '');
  const [notes, setNotes] = useState(client?.notes || '');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      setError('Client or Business Name is required');
      return;
    }
    setError(null);
    try {
      setIsSubmitting(true);
      await onSubmit({
        name: name.trim(),
        contact: contact.trim(),
        phone: phone.trim(),
        email: email.trim(),
        address: address.trim(),
        notes: notes.trim(),
      });
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
              value={name}
              onChange={(e) => setName(e.target.value)}
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
                value={contact}
                onChange={(e) => setContact(e.target.value)}
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
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
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
              value={email}
              onChange={(e) => setEmail(e.target.value)}
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
              value={address}
              onChange={(e) => setAddress(e.target.value)}
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
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-sm"
            />
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
              {isSubmitting ? 'Saving...' : client ? 'Update Client' : 'Create Client'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
