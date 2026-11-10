import React, { useState, useEffect } from 'react';
import {
  Users,
  UserPlus,
  Search,
  Phone,
  Mail,
  MapPin,
  FileText,
  CreditCard,
  Edit2,
  ChevronRight,
  Inbox,
  AlertCircle,
  Archive,
  RotateCcw,
} from 'lucide-react';
import { Client, Doc } from '../api/types.ts';
import { ClientModal } from '../components/ClientModal.tsx';
import { formatCents } from '../domain/money.ts';
import { getApiClient } from '../api/factory.ts';

export interface ClientsViewProps {
  onOpenDocument: (docId: string) => void;
  onNewDocumentForClient: (clientId: string) => void;
}

export const ClientsView: React.FC<ClientsViewProps> = ({
  onOpenDocument,
  onNewDocumentForClient,
}) => {
  const [clients, setClients] = useState<Client[]>([]);
  const [documents, setDocuments] = useState<Doc[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedClient, setSelectedClient] = useState<Client | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [clientToEdit, setClientToEdit] = useState<Client | null>(null);
  const [showArchived, setShowArchived] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const api = getApiClient();

  const loadData = async (archived = showArchived, preferredClientId?: string): Promise<void> => {
    try {
      setLoading(true);
      setError(null);
      const [clientList, docList] = await Promise.all([
        archived ? api.listArchivedClients() : api.listClients(),
        api.listDocuments(),
      ]);
      setClients(clientList);
      setDocuments(docList);
      setSelectedClient((current) =>
        clientList.find((client) => client.id === preferredClientId) ??
        clientList.find((client) => client.id === current?.id) ??
        clientList[0] ??
        null
      );
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load clients');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadData(showArchived);
  }, [showArchived]);

  const archiveSelectedClient = async (): Promise<void> => {
    if (!selectedClient || !window.confirm(
      `Archive ${selectedClient.name}? Its documents will be kept, and you can restore the client later.`
    )) return;
    try {
      await api.archiveClient(selectedClient.id);
      setShowArchived(true);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to archive client');
    }
  };

  const restoreSelectedClient = async (): Promise<void> => {
    if (!selectedClient) return;
    try {
      await api.restoreClient(selectedClient.id);
      setShowArchived(false);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to restore client');
    }
  };

  const filteredClients = clients.filter((c) => {
    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    return (
      c.name.toLowerCase().includes(q) ||
      c.contact.toLowerCase().includes(q) ||
      c.phone.toLowerCase().includes(q) ||
      c.email.toLowerCase().includes(q)
    );
  });

  const getClientMetrics = (clientId: string) => {
    const clientDocs = documents.filter((d) => d.clientId === clientId);
    const invoices = clientDocs.filter((d) => d.type === 'invoice');
    let balance = 0;
    for (const inv of invoices) {
      if (inv.status === 'cancelled') continue;
      const paid = (inv.payments || []).reduce((acc, p) => acc + p.amountCents, 0);
      balance += Math.max(0, inv.totalCents - paid);
    }
    return {
      docCount: clientDocs.length,
      balanceCents: balance,
      docs: clientDocs,
    };
  };

  const activeMetrics = selectedClient ? getClientMetrics(selectedClient.id) : null;

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-slate-200">
        <div>
          <h1 className="text-xl sm:text-2xl font-black text-slate-900 tracking-tight">
            {showArchived ? 'Archived Clients' : 'Client Directory'}
          </h1>
          <p className="text-xs text-slate-500 mt-0.5">
            Manage customer contact records, outstanding balances, and associated repair jobs.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowArchived((archived) => !archived)}
            className="px-3 py-1.5 text-xs font-semibold text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50"
          >
            {showArchived ? 'Show active clients' : 'View archived clients'}
          </button>
          {!showArchived && (
            <button
              onClick={() => {
                setClientToEdit(null);
                setIsModalOpen(true);
              }}
              className="flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-bold text-white bg-slate-900 hover:bg-slate-800 rounded-md shadow-xs transition-colors"
            >
              <UserPlus className="w-4 h-4" />
              <span>New Client</span>
            </button>
          )}
        </div>
      </div>

      {error && (
        <div className="bg-rose-50 border border-rose-200 text-rose-800 p-3.5 rounded-md text-xs flex items-center gap-2">
          <AlertCircle className="w-4 h-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {/* Main 2-Column Split: Client List & Detail View */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Left Column: Client List */}
        <div className="lg:col-span-5 space-y-3">
          {/* Search */}
          <div className="relative">
            <Search className="w-4 h-4 absolute left-3 top-2.5 text-slate-400" />
            <input
              type="text"
              placeholder="Search clients by name, contact, phone..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-3 py-2 bg-white border border-slate-200 rounded-md focus:outline-hidden focus:ring-2 focus:ring-slate-900 text-xs shadow-xs"
            />
          </div>

          <div className="bg-white border border-slate-200 rounded-lg shadow-xs overflow-hidden divide-y divide-slate-100">
            {loading ? (
              <div className="p-8 text-center text-xs text-slate-400">Loading clients...</div>
            ) : filteredClients.length === 0 ? (
              <div className="p-8 text-center text-xs text-slate-400">
                {showArchived ? 'No archived clients found' : 'No clients found'}
              </div>
            ) : (
              filteredClients.map((client) => {
                const metrics = getClientMetrics(client.id);
                const isSelected = selectedClient?.id === client.id;
                return (
                  <button
                    key={client.id}
                    onClick={() => setSelectedClient(client)}
                    className={`w-full text-left p-4 hover:bg-slate-50 transition-colors flex items-center justify-between ${
                      isSelected ? 'bg-indigo-50/50 border-l-4 border-indigo-600' : ''
                    }`}
                  >
                    <div className="space-y-1 pr-2">
                      <h3 className="font-bold text-xs text-slate-900">{client.name}</h3>
                      {client.contact && (
                        <p className="text-[11px] text-slate-500">{client.contact}</p>
                      )}
                      <div className="flex items-center gap-3 text-[10px] text-slate-400 font-mono">
                        <span>{metrics.docCount} docs</span>
                        {metrics.balanceCents > 0 && (
                          <span className="font-bold text-rose-600">
                            Due: {formatCents(metrics.balanceCents)}
                          </span>
                        )}
                      </div>
                    </div>
                    <ChevronRight className="w-4 h-4 text-slate-400 shrink-0" />
                  </button>
                );
              })
            )}
          </div>
        </div>

        {/* Right Column: Selected Client Details & Linked Docs */}
        <div className="lg:col-span-7 space-y-4">
          {selectedClient ? (
            <>
              {/* Profile Card */}
              <div className="bg-white border border-slate-200 rounded-lg p-5 shadow-xs space-y-4">
                <div className="flex items-start justify-between pb-3 border-b border-slate-100">
                  <div>
                    <h2 className="text-base font-bold text-slate-900">{selectedClient.name}</h2>
                    <span className="text-[11px] text-slate-400 font-mono">ID: {selectedClient.id}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => {
                        setClientToEdit(selectedClient);
                        setIsModalOpen(true);
                      }}
                      className="flex items-center gap-1 px-2.5 py-1 text-xs font-semibold text-slate-700 bg-white border border-slate-300 rounded hover:bg-slate-50 shadow-xs"
                    >
                      <Edit2 className="w-3 h-3" />
                      <span>Edit</span>
                    </button>
                    {showArchived ? (
                      <button
                        onClick={restoreSelectedClient}
                        className="flex items-center gap-1 px-2.5 py-1 text-xs font-semibold text-emerald-800 bg-emerald-50 border border-emerald-200 rounded hover:bg-emerald-100"
                      >
                        <RotateCcw className="w-3 h-3" />
                        <span>Restore</span>
                      </button>
                    ) : (
                      <>
                        <button
                          onClick={() => onNewDocumentForClient(selectedClient.id)}
                          className="flex items-center gap-1 px-3 py-1 text-xs font-semibold text-white bg-slate-900 hover:bg-slate-800 rounded shadow-xs"
                        >
                          <span>+ Create Document</span>
                        </button>
                        <button
                          onClick={archiveSelectedClient}
                          className="flex items-center gap-1 px-2.5 py-1 text-xs font-semibold text-rose-700 bg-white border border-rose-200 rounded hover:bg-rose-50"
                        >
                          <Archive className="w-3 h-3" />
                          <span>Archive</span>
                        </button>
                      </>
                    )}
                  </div>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
                  {selectedClient.contact && (
                    <div className="flex items-center gap-2 text-slate-700">
                      <Users className="w-4 h-4 text-slate-400 shrink-0" />
                      <span>{selectedClient.contact}</span>
                    </div>
                  )}
                  {selectedClient.phone && (
                    <div className="flex items-center gap-2 text-slate-700">
                      <Phone className="w-4 h-4 text-slate-400 shrink-0" />
                      <span className="font-mono">{selectedClient.phone}</span>
                    </div>
                  )}
                  {selectedClient.email && (
                    <div className="flex items-center gap-2 text-slate-700">
                      <Mail className="w-4 h-4 text-slate-400 shrink-0" />
                      <span>{selectedClient.email}</span>
                    </div>
                  )}
                  {selectedClient.address && (
                    <div className="flex items-center gap-2 text-slate-700 sm:col-span-2">
                      <MapPin className="w-4 h-4 text-slate-400 shrink-0" />
                      <span>{selectedClient.address}</span>
                    </div>
                  )}
                  {selectedClient.notes && (
                    <div className="p-2.5 bg-slate-50 rounded border border-slate-100 text-slate-600 sm:col-span-2 italic text-[11px]">
                      {selectedClient.notes}
                    </div>
                  )}
                </div>

                {/* Financial Balance Strip */}
                <div className="pt-3 border-t border-slate-100 flex items-center justify-between text-xs">
                  <span className="text-slate-500">Outstanding Balance:</span>
                  <span className={`font-mono text-base font-bold ${
                    (activeMetrics?.balanceCents || 0) > 0 ? 'text-rose-700' : 'text-emerald-700'
                  }`}>
                    {formatCents(activeMetrics?.balanceCents || 0)}
                  </span>
                </div>
              </div>

              {/* Linked Documents List */}
              <div className="bg-white border border-slate-200 rounded-lg p-5 shadow-xs space-y-3">
                <h3 className="text-xs font-bold text-slate-800 uppercase tracking-wider">
                  Documents History ({activeMetrics?.docs.length || 0})
                </h3>

                {activeMetrics && activeMetrics.docs.length > 0 ? (
                  <div className="divide-y divide-slate-100 text-xs">
                    {activeMetrics.docs.map((d) => (
                      <div
                        key={d.id}
                        onClick={() => onOpenDocument(d.id)}
                        className="py-2.5 flex items-center justify-between hover:bg-slate-50 cursor-pointer px-2 rounded -mx-2 transition-colors"
                      >
                        <div className="space-y-0.5">
                          <div className="flex items-center gap-2">
                            <span className="font-mono font-bold text-slate-900">
                              {d.number || 'Draft'}
                            </span>
                            <span className="text-[10px] uppercase font-semibold text-slate-400">
                              {d.type}
                            </span>
                          </div>
                          <p className="text-[11px] text-slate-500 truncate max-w-sm">
                            {d.reference} {d.scope ? `— ${d.scope}` : ''}
                          </p>
                        </div>
                        <div className="text-right">
                          <span className="font-mono font-semibold text-slate-800 block">
                            {formatCents(d.totalCents)}
                          </span>
                          <span className="text-[10px] text-slate-400 capitalize block">
                            {d.status.replace('_', ' ')}
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-xs text-slate-400 italic py-4 text-center">
                    No documents created for this client yet.
                  </p>
                )}
              </div>
            </>
          ) : (
            <div className="bg-white border border-slate-200 rounded-lg p-12 text-center text-slate-400 text-xs">
              Select a client to view details.
            </div>
          )}
        </div>
      </div>

      {/* Client Modal (Add/Edit) */}
      <ClientModal
        key={`${isModalOpen ? 'open' : 'closed'}-${clientToEdit?.id ?? 'new'}`}
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        client={clientToEdit}
        onSubmit={async (input) => {
          const savedClient = clientToEdit
            ? await api.updateClient(clientToEdit.id, input)
            : await api.createClient(input);
          const archived = Boolean(clientToEdit && showArchived);
          setSelectedClient(savedClient);
          setShowArchived(archived);
          await loadData(archived, savedClient.id);
        }}
      />
    </div>
  );
};
