import { ISODate } from '../api/types.ts';

export function parseISODate(dateStr: ISODate): Date {
  const [y, m, d] = dateStr.split('-').map(Number);
  return new Date(Date.UTC(y, m - 1, d));
}

export function formatISODate(date: Date): ISODate {
  const y = date.getUTCFullYear();
  const m = String(date.getUTCMonth() + 1).padStart(2, '0');
  const d = String(date.getUTCDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

export function getTodayISO(): ISODate {
  return formatISODate(new Date());
}

export function addDays(baseDateISO: ISODate, days: number): ISODate {
  const date = parseISODate(baseDateISO);
  date.setUTCDate(date.getUTCDate() + days);
  return formatISODate(date);
}

export function daysBetween(fromISO: ISODate, toISO: ISODate): number {
  const from = parseISODate(fromISO).getTime();
  const to = parseISODate(toISO).getTime();
  const diffMs = to - from;
  return Math.round(diffMs / (1000 * 60 * 60 * 24));
}

/**
 * Resolves relative date strings like "today-45", "today+7", "today"
 * into a concrete ISODate based on the provided today reference.
 */
export function resolveRelativeDate(dateStr: string, todayISO: ISODate): ISODate {
  if (!dateStr.startsWith('today')) {
    return dateStr;
  }
  if (dateStr === 'today') {
    return todayISO;
  }
  const match = dateStr.match(/^today([+-]\d+)$/);
  if (!match) {
    return todayISO;
  }
  const offset = parseInt(match[1], 10);
  return addDays(todayISO, offset);
}

/**
 * daysOverdue = today - dueDate when balance > 0 and dueDate < today, else null.
 */
export function calculateDaysOverdue(dueDate: ISODate | null, today: ISODate, balanceCents: number): number | null {
  if (!dueDate || balanceCents <= 0) return null;
  const days = daysBetween(dueDate, today);
  return days > 0 ? days : null;
}
