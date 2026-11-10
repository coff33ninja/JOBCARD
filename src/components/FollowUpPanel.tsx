import React, { useEffect, useState } from 'react';
import { BellRing } from 'lucide-react';
import { Doc } from '../api/types.ts';
import { canSetFollowUp, followUpState } from '../domain/followUp.ts';
import { getTodayISO } from '../domain/dates.ts';

export interface FollowUpPanelProps {
  doc: Doc;
  /** Saves the date (or null to clear) and reloads the document. */
  onSave: (followUpDate: string | null) => Promise<void>;
}

export const FollowUpPanel: React.FC<FollowUpPanelProps> = ({ doc, onSave }) => {
  const [value, setValue] = useState<string>(doc.followUpDate ?? '');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setValue(doc.followUpDate ?? '');
  }, [doc.followUpDate]);

  if (!canSetFollowUp(doc)) return null;

  const state = followUpState(doc, getTodayISO());
  const save = async (next: string | null) => {
    setSaving(true);
    setError(null);
    try {
      await onSave(next);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not save the follow-up date');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="bg-white border border-slate-200 rounded-lg p-4 shadow-sm space-y-3" data-testid="follow-up-panel">
      <div className="flex items-center justify-between pb-2 border-b border-slate-100">
        <div className="flex items-center gap-2">
          <BellRing className="w-4 h-4 text-amber-600" />
          <h3 className="text-sm font-semibold text-slate-800">Payment Follow-up</h3>
        </div>
        {state === 'due' && (
          <span className="text-[10px] font-bold uppercase tracking-wide text-red-700 bg-red-50 border border-red-200 rounded px-2 py-0.5">
            Follow up now
          </span>
        )}
        {state === 'scheduled' && (
          <span className="text-[10px] font-bold uppercase tracking-wide text-amber-700 bg-amber-50 border border-amber-200 rounded px-2 py-0.5">
            Scheduled
          </span>
        )}
      </div>
      <div className="flex items-center gap-2">
        <input
          type="date"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          disabled={saving}
          aria-label="Follow-up date"
          className="flex-1 border border-slate-300 rounded px-2 py-1.5 text-xs text-slate-800"
        />
        <button
          onClick={() => save(value || null)}
          disabled={saving || value === (doc.followUpDate ?? '')}
          className="text-xs font-semibold text-indigo-600 hover:text-indigo-800 disabled:opacity-40"
        >
          Save
        </button>
        <button
          onClick={() => save(null)}
          disabled={saving || !doc.followUpDate}
          className="text-xs font-semibold text-slate-500 hover:text-slate-700 disabled:opacity-40"
        >
          Clear
        </button>
      </div>
      {error && <p className="text-xs text-red-600">{error}</p>}
      <p className="text-[11px] text-slate-400">
        A reminder to chase payment. It shows in the dashboard's follow-ups due once the date is reached and money is still owed.
      </p>
    </div>
  );
};
