import React from 'react';

export interface StatusPillProps {
  status: string;
  daysOverdue?: number | null;
}

export const StatusPill: React.FC<StatusPillProps> = ({ status, daysOverdue }) => {
  const norm = status.toLowerCase();

  let bg = 'bg-slate-100 text-slate-700 border-slate-200';
  let label = status.replace('_', ' ');

  if (norm === 'paid' || norm === 'accepted') {
    bg = 'bg-emerald-50 text-emerald-700 border-emerald-200';
  } else if (norm === 'overdue') {
    bg = 'bg-rose-50 text-rose-700 border-rose-200 font-semibold';
  } else if (norm === 'partially_paid') {
    bg = 'bg-amber-50 text-amber-700 border-amber-200';
  } else if (norm === 'sent' || norm === 'open' || norm === 'in_progress') {
    bg = 'bg-blue-50 text-blue-700 border-blue-200';
  } else if (norm === 'converted' || norm === 'completed') {
    bg = 'bg-teal-50 text-teal-700 border-teal-200';
  } else if (norm === 'cancelled' || norm === 'declined' || norm === 'closed') {
    bg = 'bg-stone-100 text-stone-600 border-stone-200';
  } else if (norm === 'expired') {
    bg = 'bg-orange-50 text-orange-700 border-orange-200';
  }

  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs border font-medium uppercase tracking-wide ${bg}`}
    >
      <span className="capitalize">{label}</span>
      {norm === 'overdue' && daysOverdue && daysOverdue > 0 && (
        <span className="text-[10px] font-bold text-rose-800 bg-rose-200/70 px-1 rounded">
          +{daysOverdue}d
        </span>
      )}
    </span>
  );
};
