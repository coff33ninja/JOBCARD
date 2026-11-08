import { Cents } from '../api/types.ts';

/**
 * Formats an amount in integer cents as South African Rand (ZAR) in the style
 * required by the existing invoices: e.g. R1,000.00, R0.00, -R50.00.
 * Never uses locale-spaced formatting (e.g. "R 1 000,00").
 */
export function formatCents(cents: Cents): string {
  const isNegative = cents < 0;
  const absCents = Math.abs(Math.round(cents));
  const rands = Math.floor(absCents / 100);
  const remainingCents = absCents % 100;
  
  const randsFormatted = rands.toLocaleString('en-US'); // gives standard comma grouping e.g. 1,000
  const centsFormatted = remainingCents.toString().padStart(2, '0');
  
  const prefix = isNegative ? '-R' : 'R';
  return `${prefix}${randsFormatted}.${centsFormatted}`;
}

/**
 * Parses user input into integer cents. Accepts strings like "1000", "1000.50", "R1,000.50", etc.
 */
export function parseRandsToCents(val: string | number): Cents {
  if (typeof val === 'number') {
    return Math.round(val * 100);
  }
  const clean = val.replace(/[^0-9.-]/g, '');
  const parsed = parseFloat(clean);
  if (isNaN(parsed)) return 0;
  return Math.round(parsed * 100);
}
