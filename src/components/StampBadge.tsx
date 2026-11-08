import React from 'react';
import { Stamp } from '../domain/stamps.ts';

export interface StampBadgeProps {
  stamp: Stamp | null;
}

export const StampBadge: React.FC<StampBadgeProps> = ({ stamp }) => {
  if (!stamp) return null;

  let colorClasses = 'border-slate-400 text-slate-500';
  if (stamp.color === 'green') {
    colorClasses = 'border-emerald-600 text-emerald-700 bg-emerald-50/40';
  } else if (stamp.color === 'amber') {
    colorClasses = 'border-amber-600 text-amber-700 bg-amber-50/40';
  } else if (stamp.color === 'red') {
    colorClasses = stamp.isStrong
      ? 'border-red-600 text-red-600 bg-red-50/50 shadow-sm font-black'
      : 'border-rose-600 text-rose-600 bg-rose-50/30';
  } else if (stamp.color === 'grey') {
    colorClasses = 'border-slate-400 text-slate-500 bg-slate-50/50';
  }

  return (
    <div
      data-testid="document-stamp"
      className={`inline-block select-none transform rotate-[-8deg] border-4 border-dashed rounded-md px-4 py-1.5 text-center font-extrabold uppercase tracking-widest text-base sm:text-lg transition-transform duration-200 pointer-events-none ${colorClasses} ${
        stamp.isStruck ? 'line-through decoration-slate-600 decoration-2' : ''
      }`}
      style={{
        letterSpacing: '0.18em',
      }}
    >
      {stamp.text}
    </div>
  );
};
