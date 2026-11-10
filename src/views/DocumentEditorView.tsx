import React, { useState, useEffect } from 'react';
import {
  ArrowLeft,
  Plus,
  Trash2,
  AlertCircle,
  CheckCircle,
  FileText,
  UserPlus,
  Send,
  Save,
  RotateCw,
} from 'lucide-react';
import {
  Authorization,
  Client,
  DiagnosticSection,
  Doc,
  DocType,
  IntakeItem,
  LineItem,
  Settings,
  WorkItem,
} from '../api/types.ts';
import { calculateTotals, calculateLineTotal } from '../domain/totals.ts';
import { formatCents, parseRandsToCents } from '../domain/money.ts';
import { ClientModal } from '../components/ClientModal.tsx';
import { ConfirmModal } from '../components/ConfirmModal.tsx';
import { getApiClient } from '../api/factory.ts';

export interface DocumentEditorViewProps {
  docId?: string | null;
  initialType?: DocType;
  onBack: () => void;
  onSaved: (docId: string) => void;
}

export const DocumentEditorView: React.FC<DocumentEditorViewProps> = ({
  docId,
  initialType = 'invoice',
  onBack,
  onSaved,
}) => {
  const [type, setType] = useState<DocType>(initialType);
  const [clientId, setClientId] = useState<string>('');
  const [reference, setReference] = useState<string>('');
  const [scope, setScope] = useState<string>('');
  const [jobCardId, setJobCardId] = useState<string | null>(null);
  const [quotationId, setQuotationId] = useState<string | null>(null);
  const [workItems, setWorkItems] = useState<WorkItem[]>([]);
  const [intakeItems, setIntakeItems] = useState<IntakeItem[]>([]);
  const [diagnosticSections, setDiagnosticSections] = useState<DiagnosticSection[]>([]);
  const [authorization, setAuthorization] = useState<Authorization>({
    technicianSignature: '',
    technicianDate: '',
    clientSignature: '',
    clientDate: '',
  });
  const [lines, setLines] = useState<LineItem[]>([]);
  const [depositRands, setDepositRands] = useState<string>('');
  const [terms, setTerms] = useState<string>('');
  const [exclusions, setExclusions] = useState<string>('');
  const [turnaround, setTurnaround] = useState<string>('');
  const [notes, setNotes] = useState<string>('');

  const [existingDoc, setExistingDoc] = useState<Doc | null>(null);
  const [clients, setClients] = useState<Client[]>([]);
  const [settings, setSettings] = useState<Settings | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [saving, setSaving] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  // Modals
  const [isNewClientModalOpen, setIsNewClientModalOpen] = useState(false);
  const [isConfirmIssueOpen, setIsConfirmIssueOpen] = useState(false);
  const [isConfirmConvertOpen, setIsConfirmConvertOpen] = useState(false);

  const api = getApiClient();

  useEffect(() => {
    let isMounted = true;
    const loadEditorData = async () => {
      try {
        setLoading(true);
        setError(null);
        const [activeClients, settingsData] = await Promise.all([
          api.listClients(),
          api.getSettings(),
        ]);

        if (!isMounted) return;
        setSettings(settingsData);

        if (docId) {
          const doc = await api.getDocument(docId);
          if (!isMounted) return;
          let clientsList = activeClients;
          if (!clientsList.some((client) => client.id === doc.clientId)) {
            const archivedClients = await api.listArchivedClients();
            clientsList = [
              ...clientsList,
              ...archivedClients.filter((client) => client.id === doc.clientId),
            ];
          }
          setClients(clientsList);
          setExistingDoc(doc);
          setType(doc.type);
          setClientId(doc.clientId);
          setReference(doc.reference);
          setScope(doc.scope);
          setJobCardId(doc.jobCardId);
          setQuotationId(doc.quotationId);
          setWorkItems(doc.workItems || []);
          setIntakeItems(doc.intakeItems || []);
          setDiagnosticSections(doc.diagnosticSections || []);
          setAuthorization(doc.authorization || {
            technicianSignature: '',
            technicianDate: '',
            clientSignature: '',
            clientDate: '',
          });
          setLines(doc.lines || []);
          setDepositRands(doc.depositCents ? (doc.depositCents / 100).toFixed(2) : '');
          setTerms(doc.terms);
          setExclusions(doc.exclusions);
          setTurnaround(doc.turnaround);
          setNotes(doc.notes);
        } else {
          setClients(activeClients);
          // Defaults for new draft
          if (activeClients.length > 0) {
            setClientId(activeClients[0].id);
          }
          if (initialType === 'invoice') {
            setTerms(`Payment due within ${settingsData.defaultTermsDays || 7} days.`);
          } else if (initialType === 'quotation') {
            setTerms(`Quotation valid for ${settingsData.quoteValidityDays || 14} days.`);
          }
          // Default initial empty line
          setLines([{ description: '', detail: '', qty: 1, unitPriceCents: 0, totalCents: 0 }]);
        }
      } catch (err: unknown) {
        if (isMounted) {
          setError(err instanceof Error ? err.message : 'Failed to load document');
        }
      } finally {
        if (isMounted) {
          setLoading(false);
        }
      }
    };

    loadEditorData();
    return () => {
      isMounted = false;
    };
  }, [docId, initialType]);

  // Live calculated totals using domain pure functions
  const taxRate = settings?.taxRate || 0;
  const calculated = calculateTotals(lines, taxRate, existingDoc?.payments || []);

  const handleAddLine = () => {
    setLines([...lines, { description: '', detail: '', qty: 1, unitPriceCents: 0, totalCents: 0 }]);
  };

  const handleUpdateLine = (index: number, field: keyof LineItem, val: string | number) => {
    const updated = [...lines];
    const current = { ...updated[index] };

    if (field === 'description') {
      current.description = String(val);
    } else if (field === 'detail') {
      current.detail = String(val);
    } else if (field === 'qty') {
      const q = typeof val === 'number' ? val : parseFloat(String(val)) || 0;
      current.qty = q > 0 ? q : 1;
    } else if (field === 'unitPriceCents') {
      current.unitPriceCents = parseRandsToCents(val);
    }

    current.totalCents = calculateLineTotal(current.qty, current.unitPriceCents);
    updated[index] = current;
    setLines(updated);
  };

  const handleRemoveLine = (index: number) => {
    setLines(lines.filter((_, i) => i !== index));
  };

  const handleAddWorkItem = () => {
    setWorkItems([...workItems, { title: '', detail: '' }]);
  };

  const handleUpdateWorkItem = (index: number, field: 'title' | 'detail', val: string) => {
    const updated = [...workItems];
    updated[index] = { ...updated[index], [field]: val };
    setWorkItems(updated);
  };

  const handleRemoveWorkItem = (index: number) => {
    setWorkItems(workItems.filter((_, i) => i !== index));
  };

  const handleAddIntakeItem = () => {
    setIntakeItems([...intakeItems, { label: '', value: '' }]);
  };

  const handleUpdateIntakeItem = (index: number, field: keyof IntakeItem, val: string) => {
    const updated = [...intakeItems];
    updated[index] = { ...updated[index], [field]: val };
    setIntakeItems(updated);
  };

  const handleRemoveIntakeItem = (index: number) => {
    setIntakeItems(intakeItems.filter((_, i) => i !== index));
  };

  const handleAddDiagnosticSection = () => {
    setDiagnosticSections([...diagnosticSections, { title: '', status: '', details: '' }]);
  };

  const handleUpdateDiagnosticSection = (
    index: number,
    field: keyof DiagnosticSection,
    val: string
  ) => {
    const updated = [...diagnosticSections];
    updated[index] = { ...updated[index], [field]: val };
    setDiagnosticSections(updated);
  };

  const handleRemoveDiagnosticSection = (index: number) => {
    setDiagnosticSections(diagnosticSections.filter((_, i) => i !== index));
  };

  const handleUpdateAuthorization = (field: keyof Authorization, val: string) => {
    setAuthorization({ ...authorization, [field]: val });
  };

  const validateForm = (): boolean => {
    if (!clientId) {
      setError('Please select a client');
      return false;
    }
    if (!reference.trim()) {
      setError('Please enter a Device / Reference');
      return false;
    }
    setError(null);
    return true;
  };

  const handleSaveDraft = async () => {
    if (!validateForm()) return;
    try {
      setSaving(true);
      setError(null);

      const depositCents = depositRands ? parseRandsToCents(depositRands) : null;
      const cleanLines = lines.filter((l) => l.description.trim() || l.unitPriceCents > 0);
      const cleanWorkItems = workItems.filter((w) => w.title.trim() || w.detail.trim());
      const cleanIntakeItems = intakeItems
        .filter((item) => item.label.trim() || item.value.trim())
        .map((item) => ({ label: item.label.trim(), value: item.value.trim() }));
      const cleanDiagnosticSections = diagnosticSections
        .filter((section) => section.title.trim() || section.status.trim() || section.details.trim())
        .map((section) => ({
          title: section.title.trim(),
          status: section.status.trim(),
          details: section.details.trim(),
        }));

      let savedDoc: Doc;
      if (existingDoc) {
        savedDoc = await api.updateDraft(existingDoc.id, {
          clientId,
          reference: reference.trim(),
          scope: scope.trim(),
          jobCardId,
          quotationId,
          workItems: cleanWorkItems,
          intakeItems: cleanIntakeItems,
          diagnosticSections: cleanDiagnosticSections,
          authorization,
          lines: cleanLines,
          depositCents,
          terms: terms.trim(),
          exclusions: exclusions.trim(),
          turnaround: turnaround.trim(),
          notes: notes.trim(),
        });
      } else {
        savedDoc = await api.createDraft({
          type,
          clientId,
          reference: reference.trim(),
          scope: scope.trim(),
          jobCardId,
          quotationId,
          workItems: cleanWorkItems,
          intakeItems: cleanIntakeItems,
          diagnosticSections: cleanDiagnosticSections,
          authorization,
          lines: cleanLines,
          depositCents,
          terms: terms.trim(),
          exclusions: exclusions.trim(),
          turnaround: turnaround.trim(),
          notes: notes.trim(),
        });
      }

      onSaved(savedDoc.id);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to save draft');
    } finally {
      setSaving(false);
    }
  };

  const handleConfirmIssue = async () => {
    if (!validateForm()) return;
    try {
      setSaving(true);
      setError(null);

      // Save draft first if new or modified
      const depositCents = depositRands ? parseRandsToCents(depositRands) : null;
      const cleanLines = lines.filter((l) => l.description.trim() || l.unitPriceCents > 0);
      const cleanWorkItems = workItems.filter((w) => w.title.trim() || w.detail.trim());

      let currentId = existingDoc?.id;
      if (!currentId) {
        const created = await api.createDraft({
          type,
          clientId,
          reference: reference.trim(),
          scope: scope.trim(),
          jobCardId,
          quotationId,
          workItems: cleanWorkItems,
          lines: cleanLines,
          depositCents,
          terms: terms.trim(),
          exclusions: exclusions.trim(),
          turnaround: turnaround.trim(),
          notes: notes.trim(),
        });
        currentId = created.id;
      } else {
        await api.updateDraft(currentId, {
          clientId,
          reference: reference.trim(),
          scope: scope.trim(),
          jobCardId,
          quotationId,
          workItems: cleanWorkItems,
          lines: cleanLines,
          depositCents,
          terms: terms.trim(),
          exclusions: exclusions.trim(),
          turnaround: turnaround.trim(),
          notes: notes.trim(),
        });
      }

      const issued = await api.issueDocument(currentId);
      setIsConfirmIssueOpen(false);
      onSaved(issued.id);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to issue document');
      setIsConfirmIssueOpen(false);
    } finally {
      setSaving(false);
    }
  };

  const handleConfirmConvert = async () => {
    if (!existingDoc) return;
    try {
      setSaving(true);
      const newInvoice = await api.convertQuotation(existingDoc.id);
      setIsConfirmConvertOpen(false);
      onSaved(newInvoice.id);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Conversion failed');
      setIsConfirmConvertOpen(false);
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-slate-900" />
      </div>
    );
  }

  const isIssued = existingDoc?.number !== null && existingDoc?.number !== undefined;

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 py-6 space-y-6">
      {/* Top Header */}
      <div className="flex items-center justify-between pb-4 border-b border-slate-200">
        <button
          onClick={onBack}
          className="flex items-center gap-1.5 text-xs font-semibold text-slate-600 hover:text-slate-900 transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          <span>Back to Register</span>
        </button>

        <div className="flex items-center gap-2">
          {existingDoc?.type === 'quotation' && (
            <button
              onClick={() => setIsConfirmConvertOpen(true)}
              className="px-3 py-1.5 text-xs font-semibold text-indigo-700 bg-indigo-50 border border-indigo-200 rounded-md hover:bg-indigo-100 transition-colors"
            >
              Convert to Invoice
            </button>
          )}

          {!isIssued && (
            <>
              <button
                onClick={handleSaveDraft}
                disabled={saving}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 shadow-xs"
              >
                <Save className="w-3.5 h-3.5 text-slate-600" />
                <span>Save Draft</span>
              </button>
              <button
                onClick={() => setIsConfirmIssueOpen(true)}
                disabled={saving}
                className="flex items-center gap-1.5 px-4 py-1.5 text-xs font-bold text-white bg-slate-900 hover:bg-slate-800 rounded-md shadow-xs transition-colors"
              >
                <Send className="w-3.5 h-3.5" />
                <span>Issue Document</span>
              </button>
            </>
          )}
        </div>
      </div>

      {isIssued && (
        <div className="bg-amber-50 border border-amber-200 text-amber-800 p-3 rounded-md text-xs flex items-center justify-between">
          <span>
            This document has been issued with number <strong>{existingDoc.number}</strong>. Issued documents cannot be edited.
          </span>
          <button
            onClick={() => onSaved(existingDoc.id)}
            className="font-bold underline ml-2"
          >
            View preview & actions
          </button>
        </div>
      )}

      {error && (
        <div className="bg-rose-50 border border-rose-200 text-rose-800 p-3.5 rounded-md text-xs flex items-center gap-2">
          <AlertCircle className="w-4 h-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      <div className="bg-white border border-slate-200 rounded-lg p-5 sm:p-6 shadow-xs space-y-5">
        {/* Document Type Selector (only for new draft) */}
        {!existingDoc && (
          <div>
            <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-2">
              Document Type
            </label>
            <div className="grid grid-cols-3 gap-2">
              {(['invoice', 'quotation', 'jobcard'] as const).map((t) => (
                <button
                  key={t}
                  type="button"
                  onClick={() => setType(t)}
                  className={`py-2 px-3 text-xs font-semibold uppercase tracking-wider rounded-md border text-center transition-colors ${
                    type === t
                      ? 'bg-slate-900 text-white border-slate-900 shadow-xs'
                      : 'bg-slate-50 text-slate-600 border-slate-200 hover:bg-slate-100'
                  }`}
                >
                  {t === 'jobcard' ? 'Job Card' : t}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Client Selector with inline "+ New Client" */}
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="block text-xs font-semibold text-slate-700">
              Client / Customer <span className="text-rose-500">*</span>
            </label>
            <button
              type="button"
              onClick={() => setIsNewClientModalOpen(true)}
              className="text-xs text-indigo-600 hover:text-indigo-800 font-semibold flex items-center gap-1"
            >
              <UserPlus className="w-3.5 h-3.5" />
              <span>+ New Client</span>
            </button>
          </div>
          <select
            value={clientId}
            onChange={(e) => setClientId(e.target.value)}
            disabled={isIssued}
            className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-sm"
          >
            {clients.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name} {c.contact ? `(${c.contact})` : ''}
              </option>
            ))}
          </select>
        </div>

        {/* Device Reference & Service Scope */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Device / Job Reference <span className="text-rose-500">*</span>
            </label>
            <input
              type="text"
              required
              placeholder="e.g. Dell Latitude 5420 (SN: 99182)"
              value={reference}
              onChange={(e) => setReference(e.target.value)}
              disabled={isIssued}
              className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-sm"
            />
          </div>
          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Service Scope
            </label>
            <input
              type="text"
              placeholder="e.g. SSD upgrade & thermal repaste"
              value={scope}
              onChange={(e) => setScope(e.target.value)}
              disabled={isIssued}
              className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-sm"
            />
          </div>
        </div>

        {/* Intake Inventory */}
        <div className="space-y-3 pt-3 border-t border-slate-100">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-xs font-bold text-slate-800 uppercase tracking-wider">
                Job Intake & Equipment Inventory
              </h3>
              <p className="text-[11px] text-slate-500 mt-1">
                Record the equipment and condition received from the client.
              </p>
            </div>
            {!isIssued && (
              <button
                type="button"
                onClick={handleAddIntakeItem}
                className="text-xs text-indigo-600 hover:text-indigo-800 font-medium flex items-center gap-1"
              >
                <Plus className="w-3.5 h-3.5" />
                <span>Add Intake Item</span>
              </button>
            )}
          </div>
          {intakeItems.length === 0 ? (
            <p className="text-xs text-slate-400 italic">No intake items recorded.</p>
          ) : (
            <div className="space-y-2">
              {intakeItems.map((item, idx) => (
                <div key={idx} className="flex items-start gap-2">
                  <input
                    type="text"
                    placeholder="Item (e.g. Laptop, charger)"
                    value={item.label}
                    onChange={(e) => handleUpdateIntakeItem(idx, 'label', e.target.value)}
                    disabled={isIssued}
                    className="w-1/3 px-3 py-1.5 bg-slate-50 border border-slate-300 rounded-md text-xs"
                  />
                  <input
                    type="text"
                    placeholder="Condition / accessories / notes"
                    value={item.value}
                    onChange={(e) => handleUpdateIntakeItem(idx, 'value', e.target.value)}
                    disabled={isIssued}
                    className="w-2/3 px-3 py-1.5 bg-slate-50 border border-slate-300 rounded-md text-xs"
                  />
                  {!isIssued && (
                    <button
                      type="button"
                      onClick={() => handleRemoveIntakeItem(idx)}
                      className="p-1.5 text-slate-400 hover:text-rose-600 rounded"
                      aria-label="Remove intake item"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Quotation specifics: exclusions & turnaround */}
        {type === 'quotation' && (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-2 border-t border-slate-100">
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">
                Estimated Turnaround
              </label>
              <input
                type="text"
                placeholder="e.g. 2-3 working days"
                value={turnaround}
                onChange={(e) => setTurnaround(e.target.value)}
                disabled={isIssued}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-sm"
              />
            </div>
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">
                Required Deposit (Rands)
              </label>
              <div className="relative">
                <span className="absolute left-3 top-2 text-slate-400 font-mono text-sm">R</span>
                <input
                  type="text"
                  placeholder="0.00"
                  value={depositRands}
                  onChange={(e) => setDepositRands(e.target.value)}
                  disabled={isIssued}
                  className="w-full pl-8 pr-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-sm font-mono"
                />
              </div>
            </div>
          </div>
        )}

        {/* Work Items Builder */}
        <div className="space-y-3 pt-3 border-t border-slate-100">
          <div className="flex items-center justify-between">
            <h3 className="text-xs font-bold text-slate-800 uppercase tracking-wider">
              {type === 'invoice'
                ? 'Completed Services & Work Items'
                : type === 'quotation'
                ? 'Proposed Services & Work Items'
                : 'Job Card Work Log'}
            </h3>
            {!isIssued && (
              <button
                type="button"
                onClick={handleAddWorkItem}
                className="text-xs text-indigo-600 hover:text-indigo-800 font-medium flex items-center gap-1"
              >
                <Plus className="w-3.5 h-3.5" />
                <span>Add Item</span>
              </button>
            )}
          </div>

          {workItems.length === 0 ? (
            <p className="text-xs text-slate-400 italic">No work items added yet.</p>
          ) : (
            <div className="space-y-2">
              {workItems.map((item, idx) => (
                <div key={idx} className="flex items-start gap-2">
                  <input
                    type="text"
                    placeholder="Title (e.g. Hardware swap)"
                    value={item.title}
                    onChange={(e) => handleUpdateWorkItem(idx, 'title', e.target.value)}
                    disabled={isIssued}
                    className="w-1/3 px-3 py-1.5 bg-slate-50 border border-slate-300 rounded-md text-xs"
                  />
                  <input
                    type="text"
                    placeholder="Details & outcomes"
                    value={item.detail}
                    onChange={(e) => handleUpdateWorkItem(idx, 'detail', e.target.value)}
                    disabled={isIssued}
                    className="w-2/3 px-3 py-1.5 bg-slate-50 border border-slate-300 rounded-md text-xs"
                  />
                  {!isIssued && (
                    <button
                      type="button"
                      onClick={() => handleRemoveWorkItem(idx)}
                      className="p-1.5 text-slate-400 hover:text-rose-600 rounded"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Line Items & Live Totals (Invoices & Quotations only) */}
        {type !== 'jobcard' && (
          <div className="space-y-3 pt-4 border-t border-slate-200">
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-bold text-slate-800 uppercase tracking-wider">
                Financial Breakdown & Line Items
              </h3>
              {!isIssued && (
                <button
                  type="button"
                  onClick={handleAddLine}
                  className="text-xs text-indigo-600 hover:text-indigo-800 font-medium flex items-center gap-1"
                >
                  <Plus className="w-3.5 h-3.5" />
                  <span>Add Line Item</span>
                </button>
              )}
            </div>

            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="border-b border-slate-200 text-slate-500 uppercase text-[10px] text-left">
                    <th className="py-2 pr-2 font-semibold">Description</th>
                    <th className="py-2 px-2 w-20 font-semibold text-center">Qty</th>
                    <th className="py-2 px-2 w-32 font-semibold text-right">Unit Price (R)</th>
                    <th className="py-2 pl-2 w-32 font-semibold text-right">Total (R)</th>
                    {!isIssued && <th className="w-8"></th>}
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {lines.map((line, idx) => (
                    <tr key={idx}>
                      <td className="py-2 pr-2">
                        <div className="space-y-1">
                          <input
                            type="text"
                            placeholder="e.g. 1TB SSD / Labour"
                            value={line.description}
                            onChange={(e) => handleUpdateLine(idx, 'description', e.target.value)}
                            disabled={isIssued}
                            className="w-full px-2.5 py-1.5 bg-slate-50 border border-slate-300 rounded text-xs"
                          />
                          <input
                            type="text"
                            placeholder="Additional item description (optional)"
                            value={line.detail || ''}
                            onChange={(e) => handleUpdateLine(idx, 'detail', e.target.value)}
                            disabled={isIssued}
                            className="w-full px-2.5 py-1.5 bg-slate-50 border border-slate-300 rounded text-xs text-slate-600"
                          />
                        </div>
                      </td>
                      <td className="py-2 px-2">
                        <input
                          type="number"
                          step="0.5"
                          min="0.5"
                          value={line.qty}
                          onChange={(e) => handleUpdateLine(idx, 'qty', e.target.value)}
                          disabled={isIssued}
                          className="w-full px-2 py-1.5 bg-slate-50 border border-slate-300 rounded text-xs text-center"
                        />
                      </td>
                      <td className="py-2 px-2">
                        <input
                          type="text"
                          placeholder="0.00"
                          value={(line.unitPriceCents / 100).toFixed(2)}
                          onChange={(e) => handleUpdateLine(idx, 'unitPriceCents', e.target.value)}
                          disabled={isIssued}
                          className="w-full px-2 py-1.5 bg-slate-50 border border-slate-300 rounded text-xs text-right font-mono"
                        />
                      </td>
                      <td className="py-2 pl-2 text-right font-mono font-semibold text-slate-800">
                        {formatCents(line.totalCents !== undefined ? line.totalCents : line.qty * line.unitPriceCents)}
                      </td>
                      {!isIssued && (
                        <td className="py-2 text-right">
                          <button
                            type="button"
                            onClick={() => handleRemoveLine(idx)}
                            className="p-1 text-slate-400 hover:text-rose-600 rounded"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Live Totals Card */}
            <div className="flex justify-end pt-3 border-t border-slate-200">
              <div className="w-64 bg-slate-50 p-3 rounded-md border border-slate-200 space-y-1.5 text-xs">
                <div className="flex justify-between text-slate-600">
                  <span>Subtotal:</span>
                  <span className="font-mono font-medium">{formatCents(calculated.subtotalCents)}</span>
                </div>
                <div className="flex justify-between text-slate-600">
                  <span>
                    {taxRate > 0 ? `VAT (${taxRate * 100}%):` : 'VAT (Non-VAT Vendor):'}
                  </span>
                  <span className="font-mono">{formatCents(calculated.taxCents)}</span>
                </div>
                <div className="flex justify-between text-slate-900 font-bold border-t border-slate-300 pt-1.5 text-sm">
                  <span>Total Amount:</span>
                  <span className="font-mono">{formatCents(calculated.totalCents)}</span>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Diagnostic Report */}
        {type !== 'quotation' && (
          <div className="space-y-3 pt-4 border-t border-slate-200">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-xs font-bold text-slate-800 uppercase tracking-wider">
                  Diagnostic Findings & Resolution
                </h3>
                <p className="text-[11px] text-slate-500 mt-1">
                  Add test results, diagnosis, and resolution details.
                </p>
              </div>
              {!isIssued && (
                <button
                  type="button"
                  onClick={handleAddDiagnosticSection}
                  className="text-xs text-indigo-600 hover:text-indigo-800 font-medium flex items-center gap-1"
                >
                  <Plus className="w-3.5 h-3.5" />
                  <span>Add Report Section</span>
                </button>
              )}
            </div>
            {diagnosticSections.length === 0 ? (
              <p className="text-xs text-slate-400 italic">No diagnostic findings recorded.</p>
            ) : (
              <div className="space-y-3">
                {diagnosticSections.map((section, idx) => (
                  <div key={idx} className="grid grid-cols-1 sm:grid-cols-[1fr_1fr_auto] gap-2">
                    <input
                      type="text"
                      placeholder="Section (e.g. Initial health audit)"
                      value={section.title}
                      onChange={(e) => handleUpdateDiagnosticSection(idx, 'title', e.target.value)}
                      disabled={isIssued}
                      className="px-3 py-1.5 bg-slate-50 border border-slate-300 rounded-md text-xs"
                    />
                    <input
                      type="text"
                      placeholder="Status (e.g. Passed, Issue found)"
                      value={section.status}
                      onChange={(e) => handleUpdateDiagnosticSection(idx, 'status', e.target.value)}
                      disabled={isIssued}
                      className="px-3 py-1.5 bg-slate-50 border border-slate-300 rounded-md text-xs"
                    />
                    {!isIssued && (
                      <button
                        type="button"
                        onClick={() => handleRemoveDiagnosticSection(idx)}
                        className="p-1.5 text-slate-400 hover:text-rose-600 rounded justify-self-end"
                        aria-label="Remove diagnostic section"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    )}
                    <textarea
                      rows={3}
                      placeholder="Tests performed, findings, diagnosis, or resolution"
                      value={section.details}
                      onChange={(e) => handleUpdateDiagnosticSection(idx, 'details', e.target.value)}
                      disabled={isIssued}
                      className="sm:col-span-2 px-3 py-2 bg-slate-50 border border-slate-300 rounded-md text-xs resize-y"
                    />
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Authorization & Sign-off */}
        {type !== 'quotation' && (
          <div className="space-y-3 pt-4 border-t border-slate-200">
            <h3 className="text-xs font-bold text-slate-800 uppercase tracking-wider">
              Authorization & Sign-off
            </h3>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div className="space-y-2">
                <label className="block text-xs font-semibold text-slate-700">Technician</label>
                <input
                  type="text"
                  placeholder="Technician signature / name"
                  value={authorization.technicianSignature}
                  onChange={(e) => handleUpdateAuthorization('technicianSignature', e.target.value)}
                  disabled={isIssued}
                  className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md text-xs"
                />
                <input
                  type="text"
                  placeholder="Date (YYYY-MM-DD)"
                  value={authorization.technicianDate}
                  onChange={(e) => handleUpdateAuthorization('technicianDate', e.target.value)}
                  disabled={isIssued}
                  className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md text-xs"
                />
              </div>
              <div className="space-y-2">
                <label className="block text-xs font-semibold text-slate-700">Client</label>
                <input
                  type="text"
                  placeholder="Client signature / name"
                  value={authorization.clientSignature}
                  onChange={(e) => handleUpdateAuthorization('clientSignature', e.target.value)}
                  disabled={isIssued}
                  className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md text-xs"
                />
                <input
                  type="text"
                  placeholder="Date (YYYY-MM-DD)"
                  value={authorization.clientDate}
                  onChange={(e) => handleUpdateAuthorization('clientDate', e.target.value)}
                  disabled={isIssued}
                  className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md text-xs"
                />
              </div>
            </div>
          </div>
        )}

        {/* Terms and Notes */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-4 border-t border-slate-200 text-xs">
          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Terms & Conditions
            </label>
            <textarea
              rows={2}
              value={terms}
              onChange={(e) => setTerms(e.target.value)}
              disabled={isIssued}
              className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md text-xs resize-none"
            />
          </div>
          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Internal & Client Notes
            </label>
            <textarea
              rows={2}
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              disabled={isIssued}
              className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md text-xs resize-none"
            />
          </div>
        </div>
      </div>

      {/* New Client Modal */}
      <ClientModal
        key={isNewClientModalOpen ? 'open' : 'closed'}
        isOpen={isNewClientModalOpen}
        onClose={() => setIsNewClientModalOpen(false)}
        onSubmit={async (newClientData) => {
          const created = await api.createClient(newClientData);
          setClients([...clients, created]);
          setClientId(created.id);
        }}
      />

      {/* Confirm Issue Modal */}
      <ConfirmModal
        isOpen={isConfirmIssueOpen}
        title="Issue Document"
        message="Number can't be changed after issuing. Once issued, a sequential number and issue date will be permanently locked."
        confirmLabel="Yes, Issue Document"
        cancelLabel="Keep as Draft"
        onConfirm={handleConfirmIssue}
        onCancel={() => setIsConfirmIssueOpen(false)}
        isLoading={saving}
      />

      {/* Confirm Convert Modal */}
      <ConfirmModal
        isOpen={isConfirmConvertOpen}
        title="Convert to Invoice"
        message="This will create a new draft invoice pre-populated with this quotation's line items and scope. The quotation will be marked as converted."
        confirmLabel="Convert to Invoice"
        onConfirm={handleConfirmConvert}
        onCancel={() => setIsConfirmConvertOpen(false)}
        isLoading={saving}
      />
    </div>
  );
};
