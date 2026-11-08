import React from 'react';
import { Doc, Client, Settings } from '../api/types.ts';
import { StampBadge } from './StampBadge.tsx';
import { stampFor } from '../domain/stamps.ts';
import { formatCents } from '../domain/money.ts';
import { getTodayISO } from '../domain/dates.ts';

export interface DocumentPreviewProps {
  doc: Doc;
  client?: Client | null;
  settings?: Settings | null;
  linkedQuote?: Doc | null;
  linkedInvoice?: Doc | null;
  linkedJobCard?: Doc | null;
  todayISO?: string;
}

export const DocumentPreview: React.FC<DocumentPreviewProps> = ({
  doc,
  client,
  settings,
  linkedQuote,
  linkedInvoice,
  linkedJobCard,
  todayISO = getTodayISO(),
}) => {
  const stamp = stampFor(doc, todayISO);

  const businessName = settings?.businessName || 'IT Services & Repairs';
  const person = settings?.person || '';
  const phone = settings?.phone || '';
  const email = settings?.email || '';
  const location = settings?.location || '';

  const clientName = client?.name || 'Client';
  const clientContact = client?.contact || '';
  const clientPhone = client?.phone || '';
  const clientEmail = client?.email || '';
  const clientAddress = client?.address || '';

  const totalPayments = (doc.payments || []).reduce((acc, p) => acc + p.amountCents, 0);
  const balanceCents = doc.totalCents - totalPayments;
  const isVatVendor = (settings?.taxRate || 0) > 0;

  return (
    <div
      id="document-preview-pane"
      className="bg-white text-slate-800 shadow-lg border border-slate-200 rounded-sm mx-auto p-8 sm:p-12 w-full max-w-[800px] min-h-[1050px] relative font-sans text-sm print:m-0 print:p-8 print:shadow-none print:border-none print:w-full print:max-w-none"
    >
      {/* Stamp Top-Right (rotates over header) */}
      {stamp && (
        <div className="absolute top-8 right-8 z-10">
          <StampBadge stamp={stamp} />
        </div>
      )}

      {/* Header */}
      <header className="border-b-2 border-slate-800 pb-5 mb-6">
        <h1 className="text-2xl sm:text-3xl font-black text-slate-900 tracking-tight uppercase">
          {businessName}
        </h1>
        <div className="text-xs text-slate-600 mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 font-medium">
          {person && <span>{person}</span>}
          {person && phone && <span>•</span>}
          {phone && <span>{phone}</span>}
          {phone && email && <span>•</span>}
          {email && <span>{email}</span>}
        </div>
        {location && <p className="text-xs text-slate-500 mt-0.5">{location}</p>}
      </header>

      {/* Document Title Banner */}
      <div className="bg-slate-900 text-white px-4 py-2 mb-6 flex justify-between items-center rounded-xs">
        <span className="font-bold tracking-wider text-xs sm:text-sm uppercase">
          {doc.type === 'invoice'
            ? 'INVOICE & SERVICE SUMMARY'
            : doc.type === 'quotation'
            ? 'QUOTATION & PROPOSED SERVICES'
            : 'INTERNAL JOB CARD & WORK LOG'}
        </span>
        <span className="font-mono text-xs sm:text-sm font-semibold text-slate-200">
          {doc.number || 'DRAFT (UNISSUED)'}
        </span>
      </div>

      {/* Top Meta Details Grid */}
      <section className="grid grid-cols-2 sm:grid-cols-4 gap-4 pb-6 mb-6 border-b border-slate-200 text-xs">
        <div>
          <span className="text-slate-500 font-semibold block uppercase text-[10px]">Document No</span>
          <span className="font-mono font-bold text-slate-800">{doc.number || 'Draft'}</span>
        </div>
        <div>
          <span className="text-slate-500 font-semibold block uppercase text-[10px]">
            {doc.type === 'quotation' ? 'Quote Date' : 'Issue Date'}
          </span>
          <span className="font-medium text-slate-800">{doc.issueDate || 'Pending issue'}</span>
        </div>
        <div>
          <span className="text-slate-500 font-semibold block uppercase text-[10px]">
            {doc.type === 'quotation' ? 'Valid Until' : 'Payment Due'}
          </span>
          <span className="font-medium text-slate-800">
            {doc.type === 'quotation' ? doc.validUntil || '14 days from issue' : doc.dueDate || '7 days from issue'}
          </span>
        </div>
        <div>
          <span className="text-slate-500 font-semibold block uppercase text-[10px]">
            {doc.type === 'invoice' ? 'Quotation Ref' : 'Job Card Ref'}
          </span>
          <span className="font-medium text-slate-800">
            {doc.type === 'invoice'
              ? linkedQuote?.number || (doc.quotationId ? 'Linked' : 'Not applicable')
              : linkedJobCard?.number || (doc.jobCardId ? 'Linked' : 'Direct')}
          </span>
        </div>
      </section>

      {/* Section 2: Client & Job Details */}
      <section className="mb-6">
        <h2 className="text-xs font-bold text-slate-900 uppercase tracking-wider bg-slate-100 px-3 py-1.5 border-l-4 border-slate-800 mb-3">
          CLIENT & JOB DETAILS
        </h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs px-1">
          <div className="space-y-1">
            <p>
              <span className="text-slate-500 font-medium">Client / Business:</span>{' '}
              <strong className="text-slate-900">{clientName}</strong>
            </p>
            {clientContact && (
              <p>
                <span className="text-slate-500 font-medium">Contact Person:</span> {clientContact}
              </p>
            )}
            {clientPhone && (
              <p>
                <span className="text-slate-500 font-medium">Phone:</span> {clientPhone}
              </p>
            )}
            {clientEmail && (
              <p>
                <span className="text-slate-500 font-medium">Email:</span> {clientEmail}
              </p>
            )}
            {clientAddress && (
              <p>
                <span className="text-slate-500 font-medium">Address:</span> {clientAddress}
              </p>
            )}
          </div>
          <div className="space-y-1 bg-slate-50/70 p-3 rounded border border-slate-100">
            <p>
              <span className="text-slate-500 font-medium">Device / Reference:</span>{' '}
              <strong className="text-slate-900">{doc.reference || '—'}</strong>
            </p>
            {doc.scope && (
              <p>
                <span className="text-slate-500 font-medium">Service Scope:</span> {doc.scope}
              </p>
            )}
            {doc.turnaround && (
              <p>
                <span className="text-slate-500 font-medium">Turnaround:</span> {doc.turnaround}
              </p>
            )}
          </div>
        </div>
      </section>

      {/* Section 3: Work Items (Completed Services or Proposed Work) */}
      {doc.workItems && doc.workItems.length > 0 && (
        <section className="mb-6">
          <h2 className="text-xs font-bold text-slate-900 uppercase tracking-wider bg-slate-100 px-3 py-1.5 border-l-4 border-slate-800 mb-3">
            {doc.type === 'invoice'
              ? 'COMPLETED SERVICES & WORK DONE'
              : doc.type === 'quotation'
              ? 'PROPOSED SERVICES & WORK TO BE DONE'
              : 'WORK LOG & BENCH NOTES'}
          </h2>
          <div className="divide-y divide-slate-100 border border-slate-200 rounded-sm">
            {doc.workItems.map((item, idx) => (
              <div key={idx} className="p-3 text-xs flex items-start gap-3">
                <span className="font-bold text-slate-800 shrink-0 w-36">{item.title}:</span>
                <span className="text-slate-600">{item.detail}</span>
              </div>
            ))}
          </div>
        </section>
      )}

      {/* Exclusions / Turnaround for Quotations */}
      {doc.type === 'quotation' && doc.exclusions && (
        <div className="mb-4 text-xs bg-amber-50/60 border border-amber-200/80 p-3 rounded">
          <strong className="text-amber-900">Assumptions & Exclusions: </strong>
          <span className="text-amber-800">{doc.exclusions}</span>
        </div>
      )}

      {/* Section 4: Financial Breakdown Table */}
      {doc.type !== 'jobcard' && (
        <section className="mb-6">
          <h2 className="text-xs font-bold text-slate-900 uppercase tracking-wider bg-slate-100 px-3 py-1.5 border-l-4 border-slate-800 mb-3">
            FINANCIAL BREAKDOWN
          </h2>
          <table className="w-full text-xs border-collapse">
            <thead>
              <tr className="border-b border-slate-300 text-slate-600">
                <th className="text-left py-2 font-semibold">Description</th>
                <th className="text-center py-2 w-16 font-semibold">Qty</th>
                <th className="text-right py-2 w-28 font-semibold">Unit Price</th>
                <th className="text-right py-2 w-28 font-semibold">Total</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {doc.lines && doc.lines.length > 0 ? (
                doc.lines.map((line, idx) => (
                  <tr key={idx} className="text-slate-800">
                    <td className="py-2.5 pr-2 font-medium">{line.description}</td>
                    <td className="py-2.5 text-center text-slate-600">{line.qty}</td>
                    <td className="py-2.5 text-right font-mono">{formatCents(line.unitPriceCents)}</td>
                    <td className="py-2.5 text-right font-mono font-semibold">
                      {formatCents(line.totalCents !== undefined ? line.totalCents : line.qty * line.unitPriceCents)}
                    </td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan={4} className="py-4 text-center text-slate-400 italic">
                    No line items specified
                  </td>
                </tr>
              )}
            </tbody>
          </table>

          {/* Totals Summary */}
          <div className="mt-4 border-t-2 border-slate-800 pt-3 flex justify-end">
            <div className="w-64 space-y-1.5 text-xs">
              <div className="flex justify-between text-slate-600">
                <span>Subtotal:</span>
                <span className="font-mono font-medium">{formatCents(doc.subtotalCents)}</span>
              </div>
              <div className="flex justify-between text-slate-600">
                <span>{isVatVendor ? `VAT (${(settings?.taxRate || 0) * 100}%):` : 'VAT (Non-VAT Vendor):'}</span>
                <span className="font-mono">{formatCents(doc.taxCents)}</span>
              </div>
              <div className="flex justify-between text-slate-900 font-bold border-t border-slate-200 pt-1.5 text-sm">
                <span>{doc.type === 'quotation' ? 'TOTAL QUOTED AMOUNT:' : 'TOTAL INVOICE AMOUNT:'}</span>
                <span className="font-mono">{formatCents(doc.totalCents)}</span>
              </div>

              {doc.type === 'quotation' && doc.depositCents !== null && (
                <div className="flex justify-between text-blue-800 font-medium border-t border-dashed border-slate-200 pt-1">
                  <span>Deposit Required:</span>
                  <span className="font-mono">{formatCents(doc.depositCents)}</span>
                </div>
              )}

              {doc.type === 'invoice' && (
                <>
                  <div className="flex justify-between text-emerald-700 font-medium border-t border-slate-200 pt-1">
                    <span>Payments Received:</span>
                    <span className="font-mono">-{formatCents(totalPayments)}</span>
                  </div>
                  <div
                    className={`flex justify-between font-bold border-t-2 border-slate-900 pt-1.5 text-sm ${
                      balanceCents <= 0 ? 'text-emerald-700' : 'text-slate-900'
                    }`}
                  >
                    <span>BALANCE DUE:</span>
                    <span className="font-mono">{formatCents(balanceCents)}</span>
                  </div>
                </>
              )}
            </div>
          </div>
        </section>
      )}

      {/* Job Card Linked Documents & Status */}
      {doc.type === 'jobcard' && (
        <section className="mb-6">
          <h2 className="text-xs font-bold text-slate-900 uppercase tracking-wider bg-slate-100 px-3 py-1.5 border-l-4 border-slate-800 mb-3">
            JOB CARD STATUS & LINKED DOCUMENTS
          </h2>
          <div className="grid grid-cols-2 gap-4 text-xs p-3 bg-slate-50 border border-slate-200 rounded">
            <div>
              <span className="text-slate-500 block">Job Status:</span>
              <span className="font-bold uppercase text-slate-800">{doc.status}</span>
            </div>
            <div>
              <span className="text-slate-500 block">Client Folder:</span>
              <span className="font-mono text-slate-700 text-[11px] truncate block">
                {doc.folderPath || 'Not configured'}
              </span>
            </div>
            {linkedQuote && (
              <div>
                <span className="text-slate-500 block">Quotation:</span>
                <span className="font-medium text-blue-700">{linkedQuote.number || linkedQuote.id}</span>
              </div>
            )}
            {linkedInvoice && (
              <div>
                <span className="text-slate-500 block">Invoice:</span>
                <span className="font-medium text-emerald-700">{linkedInvoice.number || linkedInvoice.id}</span>
              </div>
            )}
          </div>
        </section>
      )}

      {/* Section 5: Payment Details & Terms */}
      {doc.type !== 'jobcard' && (
        <section className="mb-6 pt-4 border-t border-slate-200 text-xs">
          <h2 className="text-xs font-bold text-slate-900 uppercase tracking-wider mb-2">
            PAYMENT DETAILS
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 bg-slate-50 p-3 rounded border border-slate-200">
            <div className="space-y-0.5">
              <p>
                <span className="text-slate-500">Bank:</span> <strong>{settings?.bankName || 'Example Bank'}</strong>
              </p>
              <p>
                <span className="text-slate-500">Account Holder:</span> {settings?.accountHolder || businessName}
              </p>
              <p>
                <span className="text-slate-500">Account Number:</span>{' '}
                <span className="font-mono font-medium">{settings?.accountNumber || '0000000000'}</span>
              </p>
              <p>
                <span className="text-slate-500">Branch Code:</span>{' '}
                <span className="font-mono">{settings?.branchCode || '000000'}</span>
              </p>
            </div>
            <div className="space-y-1">
              <p>
                <span className="text-slate-500">Payment Reference:</span>{' '}
                <strong className="font-mono text-slate-900">
                  {doc.number ? `${doc.number} / ${clientName}` : `DRAFT / ${clientName}`}
                </strong>
              </p>
              {doc.terms && (
                <p className="text-slate-600 text-[11px] mt-1">
                  <span className="font-semibold text-slate-700">Terms: </span>
                  {doc.terms}
                </p>
              )}
            </div>
          </div>
        </section>
      )}

      {/* Section 6: Terms & Client Acceptance for Quotations */}
      {doc.type === 'quotation' && (
        <section className="mb-6 pt-2 text-xs">
          <h2 className="text-xs font-bold text-slate-900 uppercase tracking-wider mb-2">
            TERMS & CLIENT ACCEPTANCE
          </h2>
          <div className="border border-slate-200 p-4 rounded bg-slate-50/50 space-y-4">
            <p className="text-[11px] text-slate-600 leading-relaxed">
              {doc.terms ||
                `By accepting this quotation, the client confirms approval for the services described above. Valid for ${
                  settings?.quoteValidityDays || 14
                } days from the quotation date.`}
            </p>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-6 pt-3 border-t border-slate-200 text-slate-600 text-[11px]">
              <div>
                <span className="block border-b border-slate-400 pb-4">Client Representative:</span>
                <span className="text-[10px] text-slate-400 mt-1 block">Name & Designation</span>
              </div>
              <div>
                <span className="block border-b border-slate-400 pb-4">Signature / PO Ref:</span>
                <span className="text-[10px] text-slate-400 mt-1 block">Authorised Signature</span>
              </div>
              <div>
                <span className="block border-b border-slate-400 pb-4">Date:</span>
                <span className="text-[10px] text-slate-400 mt-1 block">YYYY-MM-DD</span>
              </div>
            </div>
          </div>
        </section>
      )}

      {/* Section 7: Notes & Recommendations */}
      {doc.notes && (
        <section className="mt-4 pt-4 border-t border-slate-200 text-xs">
          <h3 className="font-bold text-slate-700 uppercase text-[10px] tracking-wider mb-1">
            NOTES & RECOMMENDATIONS
          </h3>
          <p className="text-slate-600 leading-relaxed italic bg-slate-50/50 p-2.5 rounded border border-slate-100">
            {doc.notes}
          </p>
        </section>
      )}
    </div>
  );
};
