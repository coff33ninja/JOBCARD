import { Cents, LineItem, Payment } from '../api/types.ts';

export function calculateLineTotal(qty: number, unitPriceCents: Cents): Cents {
  return Math.round(qty * unitPriceCents);
}

export interface CalculatedTotals {
  subtotalCents: Cents;
  taxCents: Cents;
  totalCents: Cents;
  paidCents: Cents;
  balanceCents: Cents;
}

export function calculateTotals(
  lines: LineItem[],
  taxRate: number = 0,
  payments: Payment[] = []
): CalculatedTotals {
  const subtotalCents = lines.reduce((acc, line) => {
    const lineTotal = line.totalCents !== undefined ? line.totalCents : calculateLineTotal(line.qty, line.unitPriceCents);
    return acc + lineTotal;
  }, 0);

  const taxCents = Math.round(subtotalCents * taxRate);
  const totalCents = subtotalCents + taxCents;
  const paidCents = payments.reduce((acc, p) => acc + p.amountCents, 0);
  const balanceCents = totalCents - paidCents;

  return {
    subtotalCents,
    taxCents,
    totalCents,
    paidCents,
    balanceCents,
  };
}
