/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState } from 'react';
import { Navbar, ActiveTab } from './components/Navbar.tsx';
import { DashboardView } from './views/DashboardView.tsx';
import { RegisterView } from './views/RegisterView.tsx';
import { DocumentEditorView } from './views/DocumentEditorView.tsx';
import { DocumentDetailView } from './views/DocumentDetailView.tsx';
import { ClientsView } from './views/ClientsView.tsx';
import { SettingsView } from './views/SettingsView.tsx';
import { DocType } from './api/types.ts';

export default function App() {
  const [activeTab, setActiveTab] = useState<ActiveTab>('dashboard');
  const [currentDocId, setCurrentDocId] = useState<string | null>(null);
  const [editorInitialType, setEditorInitialType] = useState<DocType>('invoice');
  const [registerStatusFilter, setRegisterStatusFilter] = useState<string | undefined>(undefined);

  const handleNavigate = (tab: ActiveTab, docId?: string) => {
    if (docId) {
      setCurrentDocId(docId);
    }
    setActiveTab(tab);
  };

  const handleNewDoc = (type: DocType = 'invoice') => {
    setCurrentDocId(null);
    setEditorInitialType(type);
    setActiveTab('editor');
  };

  const handleOpenDoc = (docId: string) => {
    setCurrentDocId(docId);
    setActiveTab('detail');
  };

  const handleEditDoc = (docId: string) => {
    setCurrentDocId(docId);
    setActiveTab('editor');
  };

  const handleDashboardNavigateToRegister = (filter?: { status?: string }) => {
    setRegisterStatusFilter(filter?.status || 'all');
    setActiveTab('register');
  };

  return (
    <div className="min-h-screen bg-slate-100/70 text-slate-900 flex flex-col font-sans selection:bg-indigo-100 selection:text-indigo-900">
      <Navbar
        activeTab={activeTab}
        onNavigate={handleNavigate}
        onNewDoc={handleNewDoc}
      />

      <main className="flex-1 pb-16">
        {activeTab === 'dashboard' && (
          <DashboardView
            onNavigateToRegister={handleDashboardNavigateToRegister}
            onNavigateToNewDoc={handleNewDoc}
          />
        )}

        {activeTab === 'register' && (
          <RegisterView
            initialStatusFilter={registerStatusFilter}
            onOpenDocument={handleOpenDoc}
            onNewDocument={handleNewDoc}
          />
        )}

        {activeTab === 'clients' && (
          <ClientsView
            onOpenDocument={handleOpenDoc}
            onNewDocumentForClient={(clientId) => {
              setCurrentDocId(null);
              setEditorInitialType('invoice');
              setActiveTab('editor');
            }}
          />
        )}

        {activeTab === 'settings' && <SettingsView />}

        {activeTab === 'editor' && (
          <DocumentEditorView
            docId={currentDocId}
            initialType={editorInitialType}
            onBack={() => setActiveTab('register')}
            onSaved={(savedId) => {
              setCurrentDocId(savedId);
              setActiveTab('detail');
            }}
          />
        )}

        {activeTab === 'detail' && currentDocId && (
          <DocumentDetailView
            docId={currentDocId}
            onBack={() => setActiveTab('register')}
            onEdit={handleEditDoc}
            onNavigateToInvoice={(invId) => {
              setCurrentDocId(invId);
              setActiveTab('detail');
            }}
          />
        )}
      </main>
    </div>
  );
}
