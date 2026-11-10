import {
  Client,
  Doc,
  Settings,
  Storage,
  Capabilities,
  ISODate,
} from './types.ts';
import { resolveRelativeDate, getTodayISO } from '../domain/dates.ts';
import { calculateTotals } from '../domain/totals.ts';

export interface RawSeedDoc extends Omit<Doc, 'subtotalCents' | 'taxCents' | 'totalCents' | 'createdAt' | 'updatedAt'> {
  subtotalCents?: number;
  taxCents?: number;
  totalCents?: number;
  createdAt: string;
  updatedAt: string;
}

export interface RawSeedData {
  settings: Settings;
  storage: Storage;
  capabilities: Capabilities;
  clients: Client[];
  documents: RawSeedDoc[];
}

export interface LoadedSeedData {
  settings: Settings;
  storage: Storage;
  capabilities: Capabilities;
  clients: Client[];
  documents: Doc[];
}

export function loadSeed(raw: RawSeedData, todayISO: ISODate = getTodayISO()): LoadedSeedData {
  const settings: Settings = { ...raw.settings };
  const storage: Storage = { ...raw.storage };
  const capabilities: Capabilities = { ...raw.capabilities };
  const clients: Client[] = raw.clients.map((c) => ({ ...c }));

  const taxRate = settings.taxRate || 0;

  const documents: Doc[] = raw.documents.map((rawDoc) => {
    // Resolve relative dates
    const issueDate = rawDoc.issueDate ? resolveRelativeDate(rawDoc.issueDate, todayISO) : null;
    const dueDate = rawDoc.dueDate ? resolveRelativeDate(rawDoc.dueDate, todayISO) : null;
    const validUntil = rawDoc.validUntil ? resolveRelativeDate(rawDoc.validUntil, todayISO) : null;
    const followUpDate = rawDoc.followUpDate ? resolveRelativeDate(rawDoc.followUpDate, todayISO) : null;
    const createdAt = rawDoc.createdAt ? resolveRelativeDate(rawDoc.createdAt, todayISO) : new Date().toISOString();
    const updatedAt = rawDoc.updatedAt ? resolveRelativeDate(rawDoc.updatedAt, todayISO) : new Date().toISOString();

    const payments = (rawDoc.payments || []).map((p) => ({
      ...p,
      date: resolveRelativeDate(p.date, todayISO),
    }));

    const lines = (rawDoc.lines || []).map((l) => ({
      ...l,
      totalCents: l.totalCents !== undefined ? l.totalCents : Math.round(l.qty * l.unitPriceCents),
    }));

    const totals = calculateTotals(lines, taxRate, payments);

    return {
      ...rawDoc,
      issueDate,
      dueDate,
      validUntil,
      followUpDate,
      createdAt,
      updatedAt,
      lines,
      payments,
      subtotalCents: totals.subtotalCents,
      taxCents: totals.taxCents,
      totalCents: totals.totalCents,
      depositCents: rawDoc.depositCents ?? null,
      folderPath: rawDoc.folderPath ?? null,
    };
  });

  return {
    settings,
    storage,
    capabilities,
    clients,
    documents,
  };
}
