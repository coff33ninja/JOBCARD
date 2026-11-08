import React, { useState } from 'react';
import {
  FileText,
  LayoutDashboard,
  Users,
  Settings as SettingsIcon,
  PlusCircle,
  Menu,
  X,
  Wrench,
} from 'lucide-react';

export type ActiveTab = 'dashboard' | 'register' | 'clients' | 'settings' | 'editor' | 'detail';

export interface NavbarProps {
  activeTab: ActiveTab;
  onNavigate: (tab: ActiveTab, docId?: string) => void;
  onNewDoc: (type?: 'invoice' | 'quotation' | 'jobcard') => void;
}

export const Navbar: React.FC<NavbarProps> = ({
  activeTab,
  onNavigate,
  onNewDoc,
}) => {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [newDocMenuOpen, setNewDocMenuOpen] = useState(false);

  const navItems: { tab: ActiveTab; label: string; icon: React.ReactNode }[] = [
    { tab: 'dashboard', label: 'Dashboard', icon: <LayoutDashboard className="w-4 h-4" /> },
    { tab: 'register', label: 'Register', icon: <FileText className="w-4 h-4" /> },
    { tab: 'clients', label: 'Clients', icon: <Users className="w-4 h-4" /> },
    { tab: 'settings', label: 'Settings', icon: <SettingsIcon className="w-4 h-4" /> },
  ];

  const handleNavClick = (tab: ActiveTab) => {
    onNavigate(tab);
    setMobileMenuOpen(false);
  };

  return (
    <header className="bg-slate-900 text-white border-b border-slate-800 sticky top-0 z-40 print:hidden">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-14 sm:h-16">
          {/* Brand */}
          <div className="flex items-center gap-3">
            <button
              onClick={() => handleNavClick('dashboard')}
              className="flex items-center gap-2 text-left hover:opacity-90 transition-opacity"
            >
              <div className="bg-indigo-600 text-white p-1.5 rounded-md shadow-xs">
                <Wrench className="w-5 h-5" />
              </div>
              <div>
                <span className="font-black text-lg tracking-tight uppercase">JobCard</span>
                <span className="text-[10px] text-slate-400 block -mt-1 font-mono tracking-wider">
                  IT REPAIR DESK
                </span>
              </div>
            </button>

            {/* Desktop Navigation */}
            <nav className="hidden md:flex items-center space-x-1 ml-8">
              {navItems.map((item) => {
                const isActive = activeTab === item.tab;
                return (
                  <button
                    key={item.tab}
                    onClick={() => handleNavClick(item.tab)}
                    className={`flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-semibold tracking-wide transition-colors ${
                      isActive
                        ? 'bg-slate-800 text-white'
                        : 'text-slate-300 hover:text-white hover:bg-slate-800/60'
                    }`}
                  >
                    {item.icon}
                    <span>{item.label}</span>
                  </button>
                );
              })}
            </nav>
          </div>

          {/* Quick Action Button: New Document */}
          <div className="hidden sm:flex items-center gap-2 relative">
            <div className="relative">
              <button
                onClick={() => setNewDocMenuOpen(!newDocMenuOpen)}
                className="flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-bold bg-indigo-600 hover:bg-indigo-500 active:bg-indigo-700 text-white rounded-md shadow-xs transition-colors"
              >
                <PlusCircle className="w-4 h-4" />
                <span>New Document</span>
              </button>

              {newDocMenuOpen && (
                <>
                  <div
                    className="fixed inset-0 z-10"
                    onClick={() => setNewDocMenuOpen(false)}
                  />
                  <div className="absolute right-0 mt-2 w-48 bg-white text-slate-800 rounded-md shadow-lg border border-slate-200 py-1 z-20 text-xs">
                    <button
                      onClick={() => {
                        onNewDoc('invoice');
                        setNewDocMenuOpen(false);
                      }}
                      className="w-full text-left px-4 py-2 hover:bg-slate-50 flex items-center justify-between"
                    >
                      <span className="font-semibold text-slate-800">New Invoice</span>
                      <span className="text-[10px] font-mono text-slate-400">INV</span>
                    </button>
                    <button
                      onClick={() => {
                        onNewDoc('quotation');
                        setNewDocMenuOpen(false);
                      }}
                      className="w-full text-left px-4 py-2 hover:bg-slate-50 flex items-center justify-between"
                    >
                      <span className="font-semibold text-slate-800">New Quotation</span>
                      <span className="text-[10px] font-mono text-slate-400">QUO</span>
                    </button>
                    <button
                      onClick={() => {
                        onNewDoc('jobcard');
                        setNewDocMenuOpen(false);
                      }}
                      className="w-full text-left px-4 py-2 hover:bg-slate-50 flex items-center justify-between"
                    >
                      <span className="font-semibold text-slate-800">New Job Card</span>
                      <span className="text-[10px] font-mono text-slate-400">JC</span>
                    </button>
                  </div>
                </>
              )}
            </div>
          </div>

          {/* Mobile hamburger menu */}
          <div className="flex md:hidden">
            <button
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
              className="text-slate-300 hover:text-white p-2 rounded-md"
              aria-label="Toggle navigation"
            >
              {mobileMenuOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
            </button>
          </div>
        </div>
      </div>

      {/* Mobile Menu Panel */}
      {mobileMenuOpen && (
        <div className="md:hidden border-t border-slate-800 px-4 pt-2 pb-3 space-y-1">
          {navItems.map((item) => (
            <button
              key={item.tab}
              onClick={() => handleNavClick(item.tab)}
              className={`w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium ${
                activeTab === item.tab
                  ? 'bg-slate-800 text-white'
                  : 'text-slate-300 hover:bg-slate-800/60 hover:text-white'
              }`}
            >
              {item.icon}
              <span>{item.label}</span>
            </button>
          ))}
          <div className="pt-2 border-t border-slate-800 flex flex-col gap-1">
            <button
              onClick={() => {
                onNewDoc('invoice');
                setMobileMenuOpen(false);
              }}
              className="w-full text-left px-3 py-2 text-xs font-semibold text-indigo-300 bg-slate-800 rounded-md"
            >
              + Create Invoice
            </button>
            <button
              onClick={() => {
                onNewDoc('quotation');
                setMobileMenuOpen(false);
              }}
              className="w-full text-left px-3 py-2 text-xs font-semibold text-indigo-300 bg-slate-800 rounded-md"
            >
              + Create Quotation
            </button>
            <button
              onClick={() => {
                onNewDoc('jobcard');
                setMobileMenuOpen(false);
              }}
              className="w-full text-left px-3 py-2 text-xs font-semibold text-indigo-300 bg-slate-800 rounded-md"
            >
              + Create Job Card
            </button>
          </div>
        </div>
      )}
    </header>
  );
};
