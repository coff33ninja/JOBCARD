import React, { useState, useEffect } from 'react';
import {
  Search,
  Filter,
  Download,
  Plus,
  Inbox,
  FileSpreadsheet,
  AlertCircle,
  ChevronRight,
} from 'lucide-react';
import { Doc, DocType, Client } from '../api/types.ts';
import { StatusPill } from '../components/StatusPill.tsx';
import { formatCents } from '../domain/money.ts';
import { calculateDaysOverdue, getTodayISO } from '../domain/dates.ts';
import { deriveInvoiceStatus, deriveQuotationStatus } from '../domain/status.ts';
import { getApiClient } from '../api/factory.ts';

export interface RegisterViewProps {
  initialStatusFilter?: string;
  onOpenDocument: (docId: string) => void;
  onNewDocument: (type?: DocType) => void;
}

export const RegisterView: React.FC<RegisterViewProps> = ({
  initialStatusFilter,
  onOpenDocument,
  onNewDocument,
}) => {
  const [documents, setDocuments] = useState<Doc[]>([]);
  const [clients, setClients] = useState<Client[]>([]);
  const [typeFilter, setTypeFilter] = useState<'all' | 'invoice' | 'quotation' | 'jobcard'>('all');
  const [statusFilter, setStatusFilter] = useState<string>(initialStatusFilter || 'all');
  const [clientIdFilter, setClientIdFilter] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [dateFrom, setDateFrom] = useState<string>('');
  const [dateTo, setDateTo] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(true);
  const [exporting, setExporting] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  const api = getApiClient();
  const today = getTodayISO();

  useEffect(() => {
    if (initialStatusFilter !== undefined) {
      setStatusFilter(initialStatusFilter);
    }
  }, [initialStatusFilter]);

  useEffect(() => {
    let isMounted = true;
    const loadData = async () => {
      try {
        setLoading(true);
        setError(null);
        const [docsData, clientsData] = await Promise.all([
          api.listDocuments(),
          api.listClients(),
        ]);
        if (isMounted) {
          setDocuments(docsData);
          setClients(clientsData);
        }
      } catch (err: unknown) {
        if (isMounted) {
          setError(err instanceof Error ? err.message : 'Failed to load register');
        }
      } finally {
        if (isMounted) {
          setLoading(false);
        }
      }
    };
    loadData();
    return () => {
      isMounted = false;
    };
  }, []);

  const handleExportSpreadsheet = async () => {
    try {
      setExporting(true);
      const blob = await api.exportRegisterSpreadsheet();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `jobcard-register-${today}.xlsx`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Spreadsheet export failed');
    } finally {
      setExporting(false);
    }
  };

  const clientMap = new Map<string, string>();
  for (const c of clients) {
    clientMap.set(c.id, c.name);
  }

  // Filter in-memory for instant responsive UI
  const filteredDocs = documents.filter((doc) => {
    // Type
    if (typeFilter !== 'all' && doc.type !== typeFilter) {
      return false;
    }

    // Client
    if (clientIdFilter !== 'all' && doc.clientId !== clientIdFilter) {
      return false;
    }

    // Status
    const derivedStatus = doc.type === 'invoice'
      ? deriveInvoiceStatus(doc, today)
      : doc.type === 'quotation'
      ? deriveQuotationStatus(doc, today)
      : doc.status;

    if (statusFilter !== 'all' && derivedStatus !== statusFilter) {
      return false;
    }

    // Date range
    if (dateFrom && doc.issueDate && doc.issueDate < dateFrom) {
      return false;
    }
    if (dateTo && doc.issueDate && doc.issueDate > dateTo) {
      return false;
    }

    // Text search (number, reference, scope, client name)
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      const numMatch = doc.number?.toLowerCase().includes(q) ?? false;
      const refMatch = doc.reference.toLowerCase().includes(q);
      const scopeMatch = doc.scope.toLowerCase().includes(q);
      const clientName = clientMap.get(doc.clientId)?.toLowerCase() || '';
      const clientMatch = clientName.includes(q);

      if (!numMatch && !refMatch && !scopeMatch && !clientMatch) {
        return false;
      }
    }

    return true;
  });

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-5">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-slate-200">
        <div>
          <h1 className="text-xl sm:text-2xl font-black text-slate-900 tracking-tight">
            Document Register
          </h1>
          <p className="text-xs text-slate-500 mt-0.5">
            Master ledger of all job cards, quotations, and invoices with real-time payment balances.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={handleExportSpreadsheet}
            disabled={exporting}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 shadow-xs"
          >
            <FileSpreadsheet className="w-4 h-4 text-emerald-600" />
            <span>{exporting ? 'Exporting...' : 'Export (.xlsx)'}</span>
          </button>
          <button
            onClick={() => onNewDocument(typeFilter === 'all' ? 'invoice' : typeFilter)}
            className="flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-bold text-white bg-slate-900 hover:bg-slate-800 rounded-md shadow-xs transition-colors"
          >
            <Plus className="w-4 h-4" />
            <span>Create Draft</span>
          </button>
        </div>
      </div>

      {/* Filter & Search Bar */}
      <div className="bg-white border border-slate-200 rounded-lg p-4 shadow-xs space-y-3">
        {/* Document Type Toggle */}
        <div className="flex flex-wrap items-center gap-1 pb-3 border-b border-slate-100">
          {(['all', 'invoice', 'quotation', 'jobcard'] as const).map((t) => (
            <button
              key={t}
              onClick={() => setTypeFilter(t)}
              className={`px-3 py-1.5 rounded-md text-xs font-semibold uppercase tracking-wider transition-colors ${
                typeFilter === t
                  ? 'bg-slate-900 text-white shadow-xs'
                  : 'bg-slate-50 text-slate-600 hover:bg-slate-100'
              }`}
            >
              {t === 'all' ? 'All Documents' : t === 'jobcard' ? 'Job Cards' : `${t}s`}
            </button>
          ))}
        </div>

        {/* Search & Select Filters */}
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-5 gap-3 text-xs">
          {/* Text Search */}
          <div className="md:col-span-2 relative">
            <Search className="w-4 h-4 absolute left-3 top-2.5 text-slate-400" />
            <input
              type="text"
              placeholder="Search number, client, device, serial, scope..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-3 py-2 bg-slate-50 border border-slate-200 rounded-md focus:bg-white focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-xs"
            />
          </div>

          {/* Client Filter */}
          <div>
            <select
              value={clientIdFilter}
              onChange={(e) => setClientIdFilter(e.target.value)}
              className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-md focus:bg-white focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-xs text-slate-700"
            >
              <option value="all">All Clients</option>
              {clients.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>

          {/* Status Filter */}
          <div>
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-md focus:bg-white focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-xs text-slate-700"
            >
              <option value="all">All Statuses</option>
              <option value="draft">Draft</option>
              <option value="sent">Sent / Unpaid</option>
              <option value="partially_paid">Partially Paid</option>
              <option value="paid">Paid</option>
              <option value="overdue">Overdue</option>
              <option value="accepted">Accepted</option>
              <option value="declined">Declined</option>
              <option value="expired">Expired</option>
              <option value="converted">Converted</option>
              <option value="cancelled">Cancelled</option>
              <option value="open">Open (Job Card)</option>
              <option value="in_progress">In Progress</option>
              <option value="completed">Completed</option>
              <option value="closed">Closed</option>
            </select>
          </div>

          {/* Clear Filters */}
          <div className="flex items-center">
            {(typeFilter !== 'all' || statusFilter !== 'all' || clientIdFilter !== 'all' || searchQuery || dateFrom || dateTo) && (
              <button
                onClick={() => {
                  setTypeFilter('all');
                  setStatusFilter('all');
                  setClientIdFilter('all');
                  setSearchQuery('');
                  setDateFrom('');
                  setDateTo('');
                }}
                className="text-xs text-rose-600 hover:text-rose-800 font-medium underline"
              >
                Reset filters
              </button>
            )}
          </div>
        </div>

        {/* Date Range Row */}
        <div className="flex flex-wrap items-center gap-3 pt-2 text-xs text-slate-600 border-t border-slate-100">
          <span className="font-semibold text-slate-500">Issue Date Range:</span>
          <div className="flex items-center gap-2">
            <input
              type="date"
              value={dateFrom}
              onChange={(e) => setDateFrom(e.target.value)}
              className="px-2 py-1 bg-slate-50 border border-slate-200 rounded text-xs"
            />
            <span>to</span>
            <input
              type="date"
              value={dateTo}
              onChange={(e) => setDateTo(e.target.value)}
              className="px-2 py-1 bg-slate-50 border border-slate-200 rounded text-xs"
            />
          </div>
          <span className="text-[11px] text-slate-400 ml-auto">
            Showing {filteredDocs.length} of {documents.length} documents
          </span>
        </div>
      </div>

      {/* Main Register Table */}
      <div className="bg-white border border-slate-200 rounded-lg shadow-xs overflow-hidden">
        {loading ? (
          <div className="p-12 text-center text-slate-400">Loading register...</div>
        ) : filteredDocs.length === 0 ? (
          <div className="p-12 text-center space-y-3">
            <div className="w-10 h-10 bg-slate-100 text-slate-400 rounded-full flex items-center justify-center mx-auto">
              <Inbox className="w-5 h-5" />
            </div>
            <h3 className="text-sm font-bold text-slate-800">No matching documents found</h3>
            <p className="text-xs text-slate-500 max-w-sm mx-auto">
              Try adjusting your search terms or filter selections, or create a new document.
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs text-left">
              <thead>
                <tr className="border-b border-slate-200 bg-slate-50/70 text-slate-500 uppercase text-[10px] tracking-wider font-semibold">
                  <th className="py-3 px-4">Document</th>
                  <th className="py-3 px-4">Client</th>
                  <th className="py-3 px-4">Device & Scope</th>
                  <th className="py-3 px-4">Issue Date</th>
                  <th className="py-3 px-4">Due Date</th>
                  <th className="py-3 px-4 text-right">Total</th>
                  <th className="py-3 px-4 text-right">Paid</th>
                  <th className="py-3 px-4 text-right">Balance</th>
                  <th className="py-3 px-4 text-center">Status</th>
                  <th className="py-3 pr-4 text-right"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {filteredDocs.map((doc) => {
                  const clientName = clientMap.get(doc.clientId) || 'Unknown Client';
                  const totalPaid = (doc.payments || []).reduce((acc, p) => acc + p.amountCents, 0);
                  const balance = doc.totalCents - totalPaid;

                  const derivedStatus = doc.type === 'invoice'
                    ? deriveInvoiceStatus(doc, today)
                    : doc.type === 'quotation'
                    ? deriveQuotationStatus(doc, today)
                    : doc.status;

                  const daysOverdue = calculateDaysOverdue(doc.dueDate, today, balance);

                  return (
                    <tr
                      key={doc.id}
                      onClick={() => onOpenDocument(doc.id)}
                      className="hover:bg-indigo-50/40 cursor-pointer transition-colors group"
                    >
                      {/* Document Number & Type */}
                      <td className="py-3 px-4">
                        <span className="font-mono font-bold text-slate-900 group-hover:text-indigo-600">
                          {doc.number || <span className="text-slate-400 italic">Draft</span>}
                        </span>
                        <span className="text-[10px] text-slate-400 uppercase font-semibold block">
                          {doc.type}
                        </span>
                      </td>

                      {/* Client */}
                      <td className="py-3 px-4 font-medium text-slate-800">
                        {clientName}
                      </td>

                      {/* Device / Scope */}
                      <td className="py-3 px-4 max-w-xs">
                        <span className="font-medium text-slate-800 truncate block">
                          {doc.reference}
                        </span>
                        {doc.scope && (
                          <span className="text-[11px] text-slate-500 truncate block">
                            {doc.scope}
                          </span>
                        )}
                      </td>

                      {/* Issue Date */}
                      <td className="py-3 px-4 text-slate-600 whitespace-nowrap">
                        {doc.issueDate || '—'}
                      </td>

                      {/* Due Date / Validity */}
                      <td className="py-3 px-4 text-slate-600 whitespace-nowrap">
                        {doc.dueDate || doc.validUntil || '—'}
                      </td>

                      {/* Total */}
                      <td className="py-3 px-4 text-right font-mono font-semibold text-slate-900 whitespace-nowrap">
                        {doc.type === 'jobcard' ? '—' : formatCents(doc.totalCents)}
                      </td>

                      {/* Paid */}
                      <td className="py-3 px-4 text-right font-mono text-emerald-700 whitespace-nowrap">
                        {doc.type === 'invoice' ? formatCents(totalPaid) : '—'}
                      </td>

                      {/* Balance */}
                      <td className="py-3 px-4 text-right font-mono font-bold whitespace-nowrap">
                        {doc.type === 'invoice' ? (
                          <span className={balance > 0 ? (derivedStatus === 'overdue' ? 'text-rose-700' : 'text-slate-900') : 'text-slate-400'}>
                            {formatCents(balance)}
                          </span>
                        ) : (
                          '—'
                        )}
                      </td>

                      {/* Status */}
                      <td className="py-3 px-4 text-center whitespace-nowrap">
                        <StatusPill status={derivedStatus} daysOverdue={daysOverdue} />
                      </td>

                      {/* Action Arrow */}
                      <td className="py-3 pr-4 text-right text-slate-400 group-hover:text-slate-700">
                        <ChevronRight className="w-4 h-4 ml-auto" />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
};
