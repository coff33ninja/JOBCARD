import React, { useState, useEffect } from 'react';
import {
  TrendingUp,
  CheckCircle2,
  Clock,
  AlertTriangle,
  Calendar,
  Layers,
  ChevronRight,
  ArrowUpRight,
  Inbox,
  Hash,
} from 'lucide-react';
import { DashboardSummary } from '../api/types.ts';
import { formatCents } from '../domain/money.ts';
import { getApiClient } from '../api/factory.ts';

export interface DashboardViewProps {
  onNavigateToRegister: (filter?: { status?: string }) => void;
  onNavigateToNewDoc: (type?: 'invoice' | 'quotation' | 'jobcard') => void;
}

export const DashboardView: React.FC<DashboardViewProps> = ({
  onNavigateToRegister,
  onNavigateToNewDoc,
}) => {
  const currentYear = new Date().getFullYear();
  const [selectedYear, setSelectedYear] = useState<number>(currentYear);
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const api = getApiClient();

  useEffect(() => {
    let isMounted = true;
    const fetchDashboard = async () => {
      try {
        setLoading(true);
        setError(null);
        const data = await api.getDashboard(selectedYear);
        if (isMounted) {
          setSummary(data);
        }
      } catch (err: unknown) {
        if (isMounted) {
          setError(err instanceof Error ? err.message : 'Failed to load dashboard metrics');
        }
      } finally {
        if (isMounted) {
          setLoading(false);
        }
      }
    };

    fetchDashboard();
    return () => {
      isMounted = false;
    };
  }, [selectedYear]);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-slate-900" />
      </div>
    );
  }

  if (error || !summary) {
    return (
      <div className="p-8 max-w-4xl mx-auto">
        <div className="bg-rose-50 border border-rose-200 text-rose-800 p-4 rounded-lg flex items-center gap-3">
          <AlertTriangle className="w-5 h-5 shrink-0" />
          <div>
            <h4 className="font-semibold text-sm">Dashboard Error</h4>
            <p className="text-xs">{error || 'Unable to retrieve dashboard information.'}</p>
          </div>
        </div>
      </div>
    );
  }

  const isEmpty =
    summary.totals.invoicedCents === 0 &&
    summary.totals.paidCents === 0 &&
    summary.totals.outstandingCents === 0 &&
    summary.totals.overdueCents === 0 &&
    Object.values(summary.statusCounts).reduce((a, b) => a + b, 0) === 0;

  const monthNames = [
    'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
    'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'
  ];

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6">
      {/* Top Header & Year Selector */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-slate-200">
        <div>
          <h1 className="text-xl sm:text-2xl font-black text-slate-900 tracking-tight">
            Business Dashboard
          </h1>
          <p className="text-xs text-slate-500 mt-0.5">
            Real-time financial summary, attention items, and sequence numbers for South African IT repair desk.
          </p>
        </div>

        <div className="flex items-center gap-2 self-start sm:self-auto">
          <label htmlFor="dashboard-year-select" className="text-xs font-semibold text-slate-600 flex items-center gap-1">
            <Calendar className="w-3.5 h-3.5" />
            Year:
          </label>
          <select
            id="dashboard-year-select"
            value={selectedYear}
            onChange={(e) => setSelectedYear(Number(e.target.value))}
            className="px-3 py-1.5 bg-white border border-slate-300 rounded-md text-xs font-medium text-slate-800 shadow-xs focus:outline-hidden focus:ring-2 focus:ring-slate-900"
          >
            {[currentYear - 2, currentYear - 1, currentYear, currentYear + 1].map((y) => (
              <option key={y} value={y}>
                {y}
              </option>
            ))}
          </select>
        </div>
      </div>

      {isEmpty && (
        <div className="bg-slate-50 border border-dashed border-slate-300 rounded-lg p-10 text-center space-y-3">
          <div className="w-12 h-12 bg-slate-200 text-slate-500 rounded-full flex items-center justify-center mx-auto">
            <Inbox className="w-6 h-6" />
          </div>
          <h3 className="text-sm font-bold text-slate-800">No Document Activity Found</h3>
          <p className="text-xs text-slate-500 max-w-sm mx-auto">
            There are no invoices, quotations or job cards recorded yet for {selectedYear}. Create your first document to see live totals.
          </p>
          <button
            onClick={() => onNavigateToNewDoc('invoice')}
            className="mt-2 inline-flex items-center gap-1.5 px-4 py-2 text-xs font-semibold text-white bg-slate-900 rounded-md hover:bg-slate-800 shadow-xs"
          >
            Create First Invoice
          </button>
        </div>
      )}

      {/* 4 Total Metric Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Invoiced */}
        <div className="bg-white border border-slate-200 rounded-lg p-4 shadow-xs">
          <div className="flex items-center justify-between text-slate-500 mb-1">
            <span className="text-xs font-semibold uppercase tracking-wider">Invoiced ({selectedYear})</span>
            <div className="p-1.5 bg-blue-50 text-blue-600 rounded">
              <TrendingUp className="w-4 h-4" />
            </div>
          </div>
          <div className="text-xl sm:text-2xl font-black text-slate-900 font-mono">
            {formatCents(summary.totals.invoicedCents)}
          </div>
          <span className="text-[11px] text-slate-400 mt-1 block">Total billed across issued invoices</span>
        </div>

        {/* Paid */}
        <div className="bg-white border border-slate-200 rounded-lg p-4 shadow-xs">
          <div className="flex items-center justify-between text-slate-500 mb-1">
            <span className="text-xs font-semibold uppercase tracking-wider">Paid Received</span>
            <div className="p-1.5 bg-emerald-50 text-emerald-600 rounded">
              <CheckCircle2 className="w-4 h-4" />
            </div>
          </div>
          <div className="text-xl sm:text-2xl font-black text-emerald-700 font-mono">
            {formatCents(summary.totals.paidCents)}
          </div>
          <span className="text-[11px] text-slate-400 mt-1 block">Total payments collected</span>
        </div>

        {/* Outstanding */}
        <div className="bg-white border border-slate-200 rounded-lg p-4 shadow-xs">
          <div className="flex items-center justify-between text-slate-500 mb-1">
            <span className="text-xs font-semibold uppercase tracking-wider">Outstanding</span>
            <div className="p-1.5 bg-amber-50 text-amber-600 rounded">
              <Clock className="w-4 h-4" />
            </div>
          </div>
          <div className="text-xl sm:text-2xl font-black text-amber-700 font-mono">
            {formatCents(summary.totals.outstandingCents)}
          </div>
          <span className="text-[11px] text-slate-400 mt-1 block">Awaiting payment</span>
        </div>

        {/* Overdue */}
        <div className="bg-white border border-slate-200 rounded-lg p-4 shadow-xs">
          <div className="flex items-center justify-between text-slate-500 mb-1">
            <span className="text-xs font-semibold uppercase tracking-wider">Overdue Balance</span>
            <div className="p-1.5 bg-rose-50 text-rose-600 rounded">
              <AlertTriangle className="w-4 h-4" />
            </div>
          </div>
          <div className="text-xl sm:text-2xl font-black text-rose-700 font-mono">
            {formatCents(summary.totals.overdueCents)}
          </div>
          <button
            onClick={() => onNavigateToRegister({ status: 'overdue' })}
            className="text-[11px] text-rose-600 hover:text-rose-800 font-medium mt-1 flex items-center gap-1"
          >
            <span>View overdue documents</span>
            <ChevronRight className="w-3 h-3" />
          </button>
        </div>
      </div>

      {/* Middle Row: Needs Attention + Ageing Buckets + Next Numbers */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Needs Attention */}
        <div className="bg-white border border-slate-200 rounded-lg p-5 shadow-xs space-y-4">
          <div className="flex items-center justify-between pb-2 border-b border-slate-100">
            <div className="flex items-center gap-2">
              <AlertTriangle className="w-4 h-4 text-rose-600" />
              <h3 className="text-sm font-bold text-slate-800">Needs Attention</h3>
            </div>
            <span className="text-[10px] font-semibold text-rose-700 bg-rose-50 px-2 py-0.5 rounded border border-rose-200">
              Action Required
            </span>
          </div>

          <div className="space-y-3 text-xs">
            <div className="p-3 bg-slate-50 rounded-md border border-slate-200 flex justify-between items-center">
              <div>
                <span className="font-semibold text-slate-800 block">Follow-ups Due</span>
                <span className="text-[11px] text-slate-500">Overdue invoices requiring reminders</span>
              </div>
              <span className="text-base font-bold font-mono text-rose-700 bg-white px-2.5 py-1 rounded border border-slate-200">
                {summary.attention.followUpsDue}
              </span>
            </div>

            <div className="p-3 bg-slate-50 rounded-md border border-slate-200 flex justify-between items-center">
              <div>
                <span className="font-semibold text-slate-800 block">Due in Next 7 Days</span>
                <span className="text-[11px] text-slate-500">
                  {summary.attention.dueNext7Count} invoices approaching due date
                </span>
              </div>
              <span className="text-sm font-bold font-mono text-amber-700 bg-white px-2 py-1 rounded border border-slate-200">
                {formatCents(summary.attention.dueNext7Cents)}
              </span>
            </div>
          </div>
        </div>

        {/* Overdue Ageing Buckets */}
        <div className="bg-white border border-slate-200 rounded-lg p-5 shadow-xs space-y-3">
          <div className="flex items-center justify-between pb-2 border-b border-slate-100">
            <div className="flex items-center gap-2">
              <Layers className="w-4 h-4 text-indigo-600" />
              <h3 className="text-sm font-bold text-slate-800">Overdue Ageing Buckets</h3>
            </div>
            <span className="text-[11px] text-slate-400 font-mono">ZAR</span>
          </div>

          <div className="grid grid-cols-2 gap-2 text-xs">
            <div className="p-3 bg-slate-50 rounded border border-slate-200">
              <span className="text-[10px] font-semibold text-slate-500 block uppercase">1 - 30 Days</span>
              <span className="text-sm font-bold font-mono text-slate-800 block mt-0.5">
                {formatCents(summary.ageing.d1_30)}
              </span>
            </div>
            <div className="p-3 bg-slate-50 rounded border border-slate-200">
              <span className="text-[10px] font-semibold text-slate-500 block uppercase">31 - 60 Days</span>
              <span className="text-sm font-bold font-mono text-amber-800 block mt-0.5">
                {formatCents(summary.ageing.d31_60)}
              </span>
            </div>
            <div className="p-3 bg-slate-50 rounded border border-slate-200">
              <span className="text-[10px] font-semibold text-slate-500 block uppercase">61 - 90 Days</span>
              <span className="text-sm font-bold font-mono text-rose-700 block mt-0.5">
                {formatCents(summary.ageing.d61_90)}
              </span>
            </div>
            <div className="p-3 bg-slate-50 rounded border border-slate-200">
              <span className="text-[10px] font-semibold text-slate-500 block uppercase">90+ Days</span>
              <span className="text-sm font-bold font-mono text-rose-900 block mt-0.5">
                {formatCents(summary.ageing.d90plus)}
              </span>
            </div>
          </div>
        </div>

        {/* Next Server-Assigned Numbers */}
        <div className="bg-white border border-slate-200 rounded-lg p-5 shadow-xs space-y-3">
          <div className="flex items-center justify-between pb-2 border-b border-slate-100">
            <div className="flex items-center gap-2">
              <Hash className="w-4 h-4 text-emerald-600" />
              <h3 className="text-sm font-bold text-slate-800">Next Document Sequences</h3>
            </div>
            <span className="text-[10px] font-medium text-slate-400">Server Controlled</span>
          </div>

          <div className="space-y-2 text-xs">
            <div className="flex items-center justify-between p-2.5 bg-slate-50 rounded border border-slate-200">
              <span className="text-slate-600 font-medium">Job Card</span>
              <span className="font-mono font-bold text-slate-900 bg-white px-2 py-0.5 rounded border border-slate-200">
                {summary.nextNumbers.jobcard}
              </span>
            </div>
            <div className="flex items-center justify-between p-2.5 bg-slate-50 rounded border border-slate-200">
              <span className="text-slate-600 font-medium">Quotation</span>
              <span className="font-mono font-bold text-blue-700 bg-white px-2 py-0.5 rounded border border-slate-200">
                {summary.nextNumbers.quotation}
              </span>
            </div>
            <div className="flex items-center justify-between p-2.5 bg-slate-50 rounded border border-slate-200">
              <span className="text-slate-600 font-medium">Invoice</span>
              <span className="font-mono font-bold text-emerald-700 bg-white px-2 py-0.5 rounded border border-slate-200">
                {summary.nextNumbers.invoice}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Invoice Status Counts Strip */}
      <div className="bg-white border border-slate-200 rounded-lg p-4 shadow-xs">
        <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
          Invoice Status Distribution
        </h3>
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-3 text-center">
          {Object.entries(summary.statusCounts).map(([status, count]) => {
            const formattedName = status.replace('_', ' ');
            return (
              <button
                key={status}
                onClick={() => onNavigateToRegister({ status })}
                className="p-2.5 rounded-md border border-slate-200 hover:border-slate-300 hover:bg-slate-50 transition-colors text-left sm:text-center"
              >
                <span className="text-lg font-black font-mono text-slate-800 block">{count}</span>
                <span className="text-[10px] uppercase font-semibold text-slate-500 tracking-wider">
                  {formattedName}
                </span>
              </button>
            );
          })}
        </div>
      </div>

      {/* 12-Month Calendar Breakdown Table */}
      <div className="bg-white border border-slate-200 rounded-lg p-5 shadow-xs space-y-3">
        <div className="flex items-center justify-between pb-2 border-b border-slate-100">
          <h3 className="text-sm font-bold text-slate-800">Monthly Revenue & Jobs ({selectedYear})</h3>
          <span className="text-xs text-slate-400">12 Months</span>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-xs text-left">
            <thead>
              <tr className="border-b border-slate-200 text-slate-500 uppercase text-[10px] tracking-wider">
                <th className="py-2.5 pr-4 font-semibold">Month</th>
                <th className="py-2.5 px-4 font-semibold text-center">Jobs Logged</th>
                <th className="py-2.5 px-4 font-semibold text-right">Invoiced</th>
                <th className="py-2.5 px-4 font-semibold text-right">Collected</th>
                <th className="py-2.5 pl-4 font-semibold text-right">Outstanding</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {summary.monthly.map((row) => (
                <tr key={row.month} className="hover:bg-slate-50/70 transition-colors">
                  <td className="py-2.5 pr-4 font-semibold text-slate-800">
                    {monthNames[row.month - 1]} {selectedYear}
                  </td>
                  <td className="py-2.5 px-4 text-center text-slate-600 font-mono">
                    {row.jobs}
                  </td>
                  <td className="py-2.5 px-4 text-right font-mono text-slate-800 font-medium">
                    {formatCents(row.invoicedCents)}
                  </td>
                  <td className="py-2.5 px-4 text-right font-mono text-emerald-700 font-medium">
                    {formatCents(row.paidCents)}
                  </td>
                  <td className="py-2.5 pl-4 text-right font-mono text-slate-600">
                    {formatCents(row.outstandingCents)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
