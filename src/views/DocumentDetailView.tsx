import React, { useState, useEffect } from 'react';
import {
  ArrowLeft,
  Edit3,
  Send,
  CreditCard,
  Ban,
  Check,
  XCircle,
  FileCheck,
  AlertCircle,
  Clock,
  Printer,
} from 'lucide-react';
import { Doc, Client, Settings, Capabilities } from '../api/types.ts';
import { DocumentPreview } from '../components/DocumentPreview.tsx';
import { ExportMenu } from '../components/ExportMenu.tsx';
import { PaymentModal } from '../components/PaymentModal.tsx';
import { ConfirmModal } from '../components/ConfirmModal.tsx';
import { StatusPill } from '../components/StatusPill.tsx';
import { FollowUpPanel } from '../components/FollowUpPanel.tsx';
import { formatCents } from '../domain/money.ts';
import { calculateDaysOverdue, getTodayISO } from '../domain/dates.ts';
import { deriveInvoiceStatus, deriveQuotationStatus } from '../domain/status.ts';
import { getApiClient } from '../api/factory.ts';

export interface DocumentDetailViewProps {
  docId: string;
  onBack: () => void;
  onEdit: (docId: string) => void;
  onNavigateToInvoice: (invoiceId: string) => void;
}

export const DocumentDetailView: React.FC<DocumentDetailViewProps> = ({
  docId,
  onBack,
  onEdit,
  onNavigateToInvoice,
}) => {
  const [doc, setDoc] = useState<Doc | null>(null);
  const [client, setClient] = useState<Client | null>(null);
  const [settings, setSettings] = useState<Settings | null>(null);
  const [capabilities, setCapabilities] = useState<Capabilities>({
    docx: true,
    pdf: true,
    revealInFolder: true,
  });
  const [linkedQuote, setLinkedQuote] = useState<Doc | null>(null);
  const [linkedJobCard, setLinkedJobCard] = useState<Doc | null>(null);
  const [linkedInvoice, setLinkedInvoice] = useState<Doc | null>(null);

  const [loading, setLoading] = useState<boolean>(true);
  const [actionLoading, setActionLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  // Modals
  const [isPaymentModalOpen, setIsPaymentModalOpen] = useState(false);
  const [confirmModal, setConfirmModal] = useState<{
    isOpen: boolean;
    title: string;
    message: string;
    action: () => Promise<void>;
    isDestructive?: boolean;
    confirmLabel?: string;
  }>({
    isOpen: false,
    title: '',
    message: '',
    action: async () => {},
  });

  const api = getApiClient();
  const today = getTodayISO();

  const loadDocumentData = async () => {
    try {
      setLoading(true);
      setError(null);
      const [fetchedDoc, fetchedSettings, fetchedCaps] = await Promise.all([
        api.getDocument(docId),
        api.getSettings(),
        api.getCapabilities(),
      ]);

      setDoc(fetchedDoc);
      setSettings(fetchedSettings);
      setCapabilities(fetchedCaps);

      if (fetchedDoc.clientId) {
        const clients = await api.listClients();
        const found = clients.find((c) => c.id === fetchedDoc.clientId);
        setClient(found || null);
      }

      if (fetchedDoc.quotationId) {
        try {
          const q = await api.getDocument(fetchedDoc.quotationId);
          setLinkedQuote(q);
        } catch {
          // ignore
        }
      }

      if (fetchedDoc.jobCardId) {
        try {
          const jc = await api.getDocument(fetchedDoc.jobCardId);
          setLinkedJobCard(jc);
        } catch {
          // ignore
        }
      }

      // If quotation, look for converted invoice
      if (fetchedDoc.type === 'quotation') {
        const allDocs = await api.listDocuments({ type: 'invoice' });
        const linkedInv = allDocs.find((d) => d.quotationId === fetchedDoc.id);
        if (linkedInv) setLinkedInvoice(linkedInv);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load document');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadDocumentData();
  }, [docId]);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-slate-900" />
      </div>
    );
  }

  if (error || !doc) {
    return (
      <div className="p-8 max-w-4xl mx-auto space-y-4">
        <button onClick={onBack} className="text-xs font-semibold text-slate-600 flex items-center gap-1">
          <ArrowLeft className="w-4 h-4" /> Back to Register
        </button>
        <div className="bg-rose-50 border border-rose-200 text-rose-800 p-4 rounded-lg">
          <h4 className="font-semibold text-sm">Error Loading Document</h4>
          <p className="text-xs">{error || 'Document not found'}</p>
        </div>
      </div>
    );
  }

  const isDraft = doc.number === null || doc.status === 'draft';
  const totalPaid = (doc.payments || []).reduce((acc, p) => acc + p.amountCents, 0);
  const balanceCents = doc.totalCents - totalPaid;

  const derivedStatus = doc.type === 'invoice'
    ? deriveInvoiceStatus(doc, today)
    : doc.type === 'quotation'
    ? deriveQuotationStatus(doc, today)
    : doc.status;

  const daysOverdue = calculateDaysOverdue(doc.dueDate, today, balanceCents);

  // Status Actions
  const handleIssue = () => {
    setConfirmModal({
      isOpen: true,
      title: 'Issue Document',
      message: 'Number can\'t be changed after issuing. Once issued, a sequential number and issue date will be permanently locked.',
      confirmLabel: 'Issue Document',
      action: async () => {
        setActionLoading(true);
        try {
          await api.issueDocument(doc.id);
          await loadDocumentData();
        } finally {
          setActionLoading(false);
          setConfirmModal((prev) => ({ ...prev, isOpen: false }));
        }
      },
    });
  };

  const handleCancel = () => {
    setConfirmModal({
      isOpen: true,
      title: 'Cancel Document',
      message: 'Are you sure you want to cancel this document? This status is permanent.',
      confirmLabel: 'Yes, Cancel Document',
      isDestructive: true,
      action: async () => {
        setActionLoading(true);
        try {
          await api.updateStatus(doc.id, 'cancelled');
          await loadDocumentData();
        } finally {
          setActionLoading(false);
          setConfirmModal((prev) => ({ ...prev, isOpen: false }));
        }
      },
    });
  };

  const handleQuotationStatus = (newStatus: 'accepted' | 'declined') => {
    setConfirmModal({
      isOpen: true,
      title: `Mark Quotation as ${newStatus}`,
      message: `Are you sure you want to mark this quotation as ${newStatus}?`,
      confirmLabel: `Mark as ${newStatus}`,
      action: async () => {
        setActionLoading(true);
        try {
          await api.updateStatus(doc.id, newStatus);
          await loadDocumentData();
        } finally {
          setActionLoading(false);
          setConfirmModal((prev) => ({ ...prev, isOpen: false }));
        }
      },
    });
  };

  const handleConvertToInvoice = () => {
    setConfirmModal({
      isOpen: true,
      title: 'Convert Quotation to Invoice',
      message: 'This will create a new draft invoice populated with the client, scope, and line items from this quotation.',
      confirmLabel: 'Convert to Invoice',
      action: async () => {
        setActionLoading(true);
        try {
          const newInvoice = await api.convertQuotation(doc.id);
          onNavigateToInvoice(newInvoice.id);
        } finally {
          setActionLoading(false);
          setConfirmModal((prev) => ({ ...prev, isOpen: false }));
        }
      },
    });
  };

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6">
      {/* Top Action Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-slate-200 print:hidden">
        <div className="flex items-center gap-3">
          <button
            onClick={onBack}
            className="flex items-center gap-1.5 text-xs font-semibold text-slate-600 hover:text-slate-900 transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>Register</span>
          </button>
          <span className="text-slate-300">/</span>
          <span className="font-mono font-bold text-slate-800 text-sm">
            {doc.number || 'Draft'}
          </span>
          <StatusPill status={derivedStatus} daysOverdue={daysOverdue} />
        </div>

        {/* Action Buttons */}
        <div className="flex flex-wrap items-center gap-2">
          {isDraft && (
            <>
              <button
                onClick={() => onEdit(doc.id)}
                className="flex items-center gap-1 px-3 py-1.5 text-xs font-semibold text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 shadow-xs"
              >
                <Edit3 className="w-3.5 h-3.5" />
                <span>Edit Draft</span>
              </button>
              <button
                onClick={handleIssue}
                className="flex items-center gap-1 px-3.5 py-1.5 text-xs font-bold text-white bg-slate-900 hover:bg-slate-800 rounded-md shadow-xs transition-colors"
              >
                <Send className="w-3.5 h-3.5" />
                <span>Issue</span>
              </button>
            </>
          )}

          {doc.type === 'quotation' && !isDraft && (
            <>
              {doc.status !== 'converted' && (
                <button
                  onClick={handleConvertToInvoice}
                  className="flex items-center gap-1 px-3 py-1.5 text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-700 rounded-md shadow-xs transition-colors"
                >
                  <FileCheck className="w-3.5 h-3.5" />
                  <span>Convert to Invoice</span>
                </button>
              )}
              {doc.status === 'sent' && (
                <>
                  <button
                    onClick={() => handleQuotationStatus('accepted')}
                    className="flex items-center gap-1 px-3 py-1.5 text-xs font-semibold text-emerald-700 bg-emerald-50 border border-emerald-200 rounded-md hover:bg-emerald-100"
                  >
                    <Check className="w-3.5 h-3.5" />
                    <span>Accept</span>
                  </button>
                  <button
                    onClick={() => handleQuotationStatus('declined')}
                    className="flex items-center gap-1 px-3 py-1.5 text-xs font-semibold text-rose-700 bg-rose-50 border border-rose-200 rounded-md hover:bg-rose-100"
                  >
                    <XCircle className="w-3.5 h-3.5" />
                    <span>Decline</span>
                  </button>
                </>
              )}
            </>
          )}

          {doc.type === 'invoice' && !isDraft && doc.status !== 'cancelled' && (
            <>
              {balanceCents > 0 && (
                <button
                  onClick={() => setIsPaymentModalOpen(true)}
                  className="flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-bold text-white bg-emerald-700 hover:bg-emerald-800 rounded-md shadow-xs transition-colors"
                >
                  <CreditCard className="w-3.5 h-3.5" />
                  <span>Record Payment</span>
                </button>
              )}
              <button
                onClick={handleCancel}
                className="flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium text-slate-500 hover:text-rose-700 hover:bg-rose-50 rounded-md border border-slate-200"
              >
                <Ban className="w-3.5 h-3.5" />
                <span>Cancel</span>
              </button>
            </>
          )}

          {doc.type === 'jobcard' && !isDraft && (
            <div className="flex items-center gap-1.5">
              {doc.status === 'open' && (
                <button
                  onClick={async () => {
                    await api.updateStatus(doc.id, 'in_progress');
                    await loadDocumentData();
                  }}
                  className="px-3 py-1.5 text-xs font-semibold text-white bg-blue-600 hover:bg-blue-700 rounded-md shadow-xs transition-colors"
                >
                  Start Work (In Progress)
                </button>
              )}
              {doc.status === 'in_progress' && (
                <button
                  onClick={async () => {
                    await api.updateStatus(doc.id, 'completed');
                    await loadDocumentData();
                  }}
                  className="px-3 py-1.5 text-xs font-semibold text-white bg-teal-600 hover:bg-teal-700 rounded-md shadow-xs transition-colors"
                >
                  Mark Work Completed
                </button>
              )}
              {doc.status === 'completed' && (
                <button
                  onClick={async () => {
                    await api.updateStatus(doc.id, 'closed');
                    await loadDocumentData();
                  }}
                  className="px-3 py-1.5 text-xs font-semibold text-slate-700 bg-white border border-slate-300 hover:bg-slate-50 rounded-md shadow-xs transition-colors"
                >
                  Close Job Card
                </button>
              )}
              {doc.status === 'closed' && (
                <button
                  onClick={async () => {
                    await api.updateStatus(doc.id, 'open');
                    await loadDocumentData();
                  }}
                  className="px-2.5 py-1.5 text-xs font-medium text-slate-500 hover:text-slate-800 rounded-md border border-slate-200"
                >
                  Reopen
                </button>
              )}
            </div>
          )}

          <button
            onClick={() => window.print()}
            className="flex items-center gap-1 px-3 py-1.5 text-xs font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 shadow-xs"
          >
            <Printer className="w-3.5 h-3.5" />
            <span>Print</span>
          </button>
        </div>
      </div>

      {/* Main Content Layout: Left Document Preview, Right Sidebar */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
        {/* Document Preview (Main) */}
        <div className="lg:col-span-8 flex justify-center">
          <DocumentPreview
            doc={doc}
            client={client}
            settings={settings}
            linkedQuote={linkedQuote}
            linkedInvoice={linkedInvoice}
            linkedJobCard={linkedJobCard}
            todayISO={today}
          />
        </div>

        {/* Sidebar Controls (Export Menu, Payments Panel) */}
        <div className="lg:col-span-4 space-y-6 print:hidden sticky top-20">
          {/* Export Menu */}
          <ExportMenu
            doc={doc}
            client={client}
            settings={settings}
            capabilities={capabilities}
          />

          {/* Payments Panel (Invoices Only) */}
          {doc.type === 'invoice' && (
            <div className="bg-white border border-slate-200 rounded-lg p-4 shadow-sm space-y-3">
              <div className="flex items-center justify-between pb-2 border-b border-slate-100">
                <div className="flex items-center gap-2">
                  <CreditCard className="w-4 h-4 text-emerald-600" />
                  <h3 className="text-sm font-semibold text-slate-800">Payments Recorded</h3>
                </div>
                {!isDraft && doc.status !== 'cancelled' && balanceCents > 0 && (
                  <button
                    onClick={() => setIsPaymentModalOpen(true)}
                    className="text-xs text-indigo-600 hover:text-indigo-800 font-semibold"
                  >
                    + Add Payment
                  </button>
                )}
              </div>

              {/* Balance Summary Box */}
              <div className="bg-slate-50 p-3 rounded border border-slate-200 text-xs flex justify-between items-center">
                <div>
                  <span className="text-slate-500 block">Total Invoiced:</span>
                  <span className="font-mono font-medium text-slate-800">{formatCents(doc.totalCents)}</span>
                </div>
                <div>
                  <span className="text-slate-500 block">Total Paid:</span>
                  <span className="font-mono font-semibold text-emerald-700">{formatCents(totalPaid)}</span>
                </div>
                <div>
                  <span className="text-slate-500 block">Balance Due:</span>
                  <span className={`font-mono font-bold ${balanceCents <= 0 ? 'text-emerald-700' : 'text-slate-900'}`}>
                    {formatCents(balanceCents)}
                  </span>
                </div>
              </div>

              {/* Payments List */}
              {doc.payments && doc.payments.length > 0 ? (
                <div className="divide-y divide-slate-100 text-xs">
                  {doc.payments.map((p) => (
                    <div key={p.id} className="py-2 flex items-center justify-between">
                      <div>
                        <div className="font-medium text-slate-800">{p.method} Payment</div>
                        <div className="text-[11px] text-slate-400">
                          {p.date} {p.reference ? `• Ref: ${p.reference}` : ''}
                        </div>
                      </div>
                      <span className="font-mono font-bold text-emerald-700">
                        {formatCents(p.amountCents)}
                      </span>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-xs text-slate-400 italic py-2 text-center">
                  No payments recorded for this invoice.
                </p>
              )}
            </div>
          )}

          {/* Follow-up reminder (issued invoices only) */}
          {doc.type === 'invoice' && (
            <FollowUpPanel
              doc={doc}
              onSave={async (date) => {
                await api.setFollowUp(doc.id, date);
                await loadDocumentData();
              }}
            />
          )}

          {/* Linked Document Links */}
          {(linkedQuote || linkedInvoice || linkedJobCard) && (
            <div className="bg-white border border-slate-200 rounded-lg p-4 shadow-sm text-xs space-y-2">
              <h4 className="font-semibold text-slate-700 uppercase tracking-wider text-[10px]">
                Related Documents
              </h4>
              {linkedQuote && (
                <div className="flex justify-between items-center py-1">
                  <span className="text-slate-500">Quotation:</span>
                  <button
                    onClick={() => onNavigateToInvoice(linkedQuote.id)}
                    className="font-mono font-bold text-blue-600 hover:underline"
                  >
                    {linkedQuote.number || linkedQuote.id}
                  </button>
                </div>
              )}
              {linkedInvoice && (
                <div className="flex justify-between items-center py-1">
                  <span className="text-slate-500">Invoice:</span>
                  <button
                    onClick={() => onNavigateToInvoice(linkedInvoice.id)}
                    className="font-mono font-bold text-emerald-600 hover:underline"
                  >
                    {linkedInvoice.number || linkedInvoice.id}
                  </button>
                </div>
              )}
              {linkedJobCard && (
                <div className="flex justify-between items-center py-1">
                  <span className="text-slate-500">Job Card:</span>
                  <button
                    onClick={() => onNavigateToInvoice(linkedJobCard.id)}
                    className="font-mono font-bold text-slate-800 hover:underline"
                  >
                    {linkedJobCard.number || linkedJobCard.id}
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Payment Modal */}
      <PaymentModal
        isOpen={isPaymentModalOpen}
        onClose={() => setIsPaymentModalOpen(false)}
        doc={doc}
        onSubmit={async (payInput) => {
          await api.addPayment(doc.id, payInput);
          await loadDocumentData();
        }}
      />

      {/* Confirmation Modal */}
      <ConfirmModal
        isOpen={confirmModal.isOpen}
        title={confirmModal.title}
        message={confirmModal.message}
        confirmLabel={confirmModal.confirmLabel}
        isDestructive={confirmModal.isDestructive}
        onConfirm={confirmModal.action}
        onCancel={() => setConfirmModal((prev) => ({ ...prev, isOpen: false }))}
        isLoading={actionLoading}
      />
    </div>
  );
};
