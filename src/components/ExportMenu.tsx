import React, { useState } from 'react';
import {
  FileText,
  Printer,
  Share2,
  Mail,
  MessageCircle,
  FolderOpen,
  Download,
  AlertCircle,
  CheckCircle2,
  Loader2,
} from 'lucide-react';
import { Doc, Capabilities, Client, Settings } from '../api/types.ts';
import { exportOptions, toWhatsAppNumber, ExportOption } from '../domain/export.ts';
import { getApiClient } from '../api/factory.ts';

export interface ExportMenuProps {
  doc: Doc;
  client?: Client | null;
  settings?: Settings | null;
  capabilities: Capabilities;
}

export const ExportMenu: React.FC<ExportMenuProps> = ({
  doc,
  client,
  settings,
  capabilities,
}) => {
  const [loadingAction, setLoadingAction] = useState<string | null>(null);
  const [message, setMessage] = useState<{ type: 'info' | 'error' | 'success'; text: string } | null>(null);

  const canShare = typeof navigator !== 'undefined' && typeof navigator.canShare === 'function';
  const isDraft = doc.number === null || doc.status === 'draft';

  const options = exportOptions(capabilities, { canShare, isDraft });
  const api = getApiClient();

  const handleDownloadFile = async (format: 'pdf' | 'docx') => {
    try {
      setLoadingAction(format);
      setMessage(null);
      const blob = await api.exportDocument(doc.id, format);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${doc.number || 'document'}.${format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      setMessage({ type: 'success', text: `Downloaded ${format.toUpperCase()} successfully.` });
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Export failed';
      setMessage({ type: 'error', text: msg });
    } finally {
      setLoadingAction(null);
    }
  };

  const handlePrint = () => {
    window.print();
  };

  const handleShare = async () => {
    try {
      setLoadingAction('share');
      setMessage(null);
      const blob = await api.exportDocument(doc.id, 'pdf');
      const file = new File([blob], `${doc.number || 'document'}.pdf`, { type: 'application/pdf' });

      if (navigator.canShare && navigator.canShare({ files: [file] })) {
        await navigator.share({
          files: [file],
          title: `${doc.number || 'Document'} - ${settings?.businessName || 'JobCard'}`,
          text: `Please find attached ${doc.type} ${doc.number || ''} from ${settings?.businessName || 'our business'}.`,
        });
        setMessage({ type: 'success', text: 'Document shared successfully.' });
      } else {
        setMessage({ type: 'info', text: 'Native sharing not supported on this device.' });
      }
    } catch (err: unknown) {
      if ((err as Error)?.name !== 'AbortError') {
        const msg = err instanceof Error ? err.message : 'Sharing failed';
        setMessage({ type: 'error', text: msg });
      }
    } finally {
      setLoadingAction(null);
    }
  };

  const handleEmail = async () => {
    try {
      setLoadingAction('email');
      setMessage(null);

      // Download the PDF first as instructed
      await handleDownloadFile('pdf');

      const emailTo = client?.email || '';
      const subject = encodeURIComponent(`${doc.type === 'invoice' ? 'Invoice' : 'Quotation'} ${doc.number || ''} from ${settings?.businessName || ''}`);
      const body = encodeURIComponent(
        `Dear ${client?.contact || client?.name || 'Customer'},\n\nPlease find attached ${doc.type} ${doc.number || ''} for your attention.\n\nKind regards,\n${settings?.person || settings?.businessName || ''}`
      );

      window.location.href = `mailto:${emailTo}?subject=${subject}&body=${body}`;
      setMessage({
        type: 'info',
        text: 'Attach the downloaded PDF to your email draft before sending.',
      });
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Email preparation failed';
      setMessage({ type: 'error', text: msg });
    } finally {
      setLoadingAction(null);
    }
  };

  const handleWhatsApp = async () => {
    try {
      setLoadingAction('whatsapp');
      setMessage(null);

      // Download the PDF first as instructed
      await handleDownloadFile('pdf');

      const phoneRaw = client?.phone || '';
      const phoneFormatted = toWhatsAppNumber(phoneRaw);
      const text = encodeURIComponent(
        `Hi ${client?.contact || client?.name || ''}, here is your ${doc.type} ${doc.number || ''} from ${settings?.businessName || ''}. Please see attached document.`
      );

      window.open(`https://wa.me/${phoneFormatted}?text=${text}`, '_blank');
      setMessage({
        type: 'info',
        text: 'Attach the downloaded PDF to your WhatsApp message.',
      });
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'WhatsApp preparation failed';
      setMessage({ type: 'error', text: msg });
    } finally {
      setLoadingAction(null);
    }
  };

  const handleReveal = async () => {
    try {
      setLoadingAction('reveal');
      setMessage(null);
      const res = await api.revealFolder(doc.id);
      setMessage({ type: 'success', text: `Opened folder: ${res.path}` });
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Could not open folder';
      setMessage({ type: 'error', text: msg });
    } finally {
      setLoadingAction(null);
    }
  };

  const executeAction = (opt: ExportOption) => {
    if (opt.disabled) return;
    if (opt.type === 'pdf') {
      if (opt.isFallback) {
        handlePrint();
      } else {
        handleDownloadFile('pdf');
      }
    } else if (opt.type === 'docx') {
      handleDownloadFile('docx');
    } else if (opt.type === 'print') {
      handlePrint();
    } else if (opt.type === 'share') {
      handleShare();
    } else if (opt.type === 'email') {
      handleEmail();
    } else if (opt.type === 'whatsapp') {
      handleWhatsApp();
    } else if (opt.type === 'reveal') {
      handleReveal();
    }
  };

  const getIcon = (type: string) => {
    switch (type) {
      case 'pdf':
        return <Download className="w-4 h-4 text-rose-600" />;
      case 'docx':
        return <FileText className="w-4 h-4 text-blue-600" />;
      case 'print':
        return <Printer className="w-4 h-4 text-slate-700" />;
      case 'share':
        return <Share2 className="w-4 h-4 text-indigo-600" />;
      case 'email':
        return <Mail className="w-4 h-4 text-emerald-600" />;
      case 'whatsapp':
        return <MessageCircle className="w-4 h-4 text-green-600" />;
      case 'reveal':
        return <FolderOpen className="w-4 h-4 text-amber-600" />;
      default:
        return <Download className="w-4 h-4" />;
    }
  };

  return (
    <div className="bg-white border border-slate-200 rounded-lg p-4 shadow-sm space-y-3">
      <div className="flex items-center justify-between pb-2 border-b border-slate-100">
        <h3 className="text-sm font-semibold text-slate-800">Export & Actions</h3>
        {isDraft && (
          <span className="text-[11px] font-medium text-amber-700 bg-amber-50 px-2 py-0.5 rounded border border-amber-200">
            Draft (Print only)
          </span>
        )}
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
        {options.map((opt) => {
          const isLoading = loadingAction === opt.type;
          return (
            <button
              key={opt.type}
              onClick={() => executeAction(opt)}
              disabled={opt.disabled || isLoading}
              title={opt.disabledReason || opt.label}
              className={`flex items-center gap-2 px-3 py-2 text-xs font-medium rounded-md border text-left transition-colors ${
                opt.disabled
                  ? 'bg-slate-50 text-slate-400 border-slate-200 cursor-not-allowed opacity-60'
                  : 'bg-white hover:bg-slate-50 text-slate-700 border-slate-200 shadow-xs active:bg-slate-100'
              }`}
            >
              {isLoading ? (
                <Loader2 className="w-4 h-4 animate-spin text-slate-500" />
              ) : (
                getIcon(opt.type)
              )}
              <span className="truncate">{opt.label}</span>
            </button>
          );
        })}
      </div>

      {message && (
        <div
          className={`flex items-start gap-2 p-2.5 rounded text-xs ${
            message.type === 'error'
              ? 'bg-rose-50 text-rose-800 border border-rose-200'
              : message.type === 'success'
              ? 'bg-emerald-50 text-emerald-800 border border-emerald-200'
              : 'bg-blue-50 text-blue-800 border border-blue-200'
          }`}
        >
          {message.type === 'error' ? (
            <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
          ) : (
            <CheckCircle2 className="w-4 h-4 shrink-0 mt-0.5" />
          )}
          <span className="leading-snug">{message.text}</span>
        </div>
      )}
    </div>
  );
};
