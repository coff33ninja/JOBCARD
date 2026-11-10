import React, { useState, useEffect } from 'react';
import {
  Building2,
  Landmark,
  FileCheck2,
  FolderOpen,
  Copy,
  Check,
  Download,
  Save,
  CheckCircle2,
  AlertCircle,
} from 'lucide-react';
import type { Settings, Storage } from '../api/types.ts';
import { getApiClient } from '../api/factory.ts';
import { getTodayISO } from '../domain/dates.ts';

const MAX_BRANDING_IMAGE_BYTES = 256 * 1024;

export const SettingsView: React.FC = () => {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [storage, setStorage] = useState<Storage | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [statusMsg, setStatusMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [downloading, setDownloading] = useState(false);

  const api = getApiClient();

  const handleBrandingImageSelect = (
    event: React.ChangeEvent<HTMLInputElement>,
    field: keyof Pick<Settings, 'companyLogo' | 'technicianSignatureImage'>,
    label: string,
  ) => {
    const files = event.currentTarget.files;
    event.target.value = '';
    if (files === null) return;
    const file = files.item(0);
    if (file === null) return;

    if (file.type !== 'image/png' && file.type !== 'image/jpeg') {
      setStatusMsg({ type: 'error', text: `${label} must be a PNG or JPEG image.` });
      return;
    }
    if (file.size < 1 || file.size > MAX_BRANDING_IMAGE_BYTES) {
      setStatusMsg({ type: 'error', text: `${label} must be between 1 byte and 256 KiB.` });
      return;
    }

    const reader = new FileReader();
    reader.onerror = () => {
      setStatusMsg({ type: 'error', text: `Could not read the selected ${label.toLowerCase()} file.` });
    };
    reader.onload = () => {
      const dataUrl = reader.result;
      if (typeof dataUrl !== 'string' || !dataUrl.startsWith(`data:${file.type};base64,`)) {
        setStatusMsg({ type: 'error', text: `Could not read the selected ${label.toLowerCase()} file.` });
        return;
      }

      setSettings((current) => {
        if (current === null) return current;
        return field === 'companyLogo'
          ? { ...current, companyLogo: dataUrl }
          : { ...current, technicianSignatureImage: dataUrl };
      });
      setStatusMsg({ type: 'success', text: `${label} selected. Save Settings to persist the change.` });
    };
    reader.readAsDataURL(file);
  };

  const handleBrandingImageRemove = (
    field: keyof Pick<Settings, 'companyLogo' | 'technicianSignatureImage'>,
    label: string,
  ) => {
    setSettings((current) => {
      if (current === null) return current;
      return field === 'companyLogo'
        ? { ...current, companyLogo: undefined }
        : { ...current, technicianSignatureImage: undefined };
    });
    setStatusMsg({ type: 'success', text: `${label} will be removed when you save settings.` });
  };

  useEffect(() => {
    let isMounted = true;
    const loadSettings = async () => {
      try {
        setLoading(true);
        const [settingsData, storageData] = await Promise.all([
          api.getSettings(),
          api.getStorage(),
        ]);
        if (isMounted) {
          setSettings(settingsData);
          setStorage(storageData);
        }
      } catch (err: unknown) {
        if (isMounted) {
          setStatusMsg({
            type: 'error',
            text: err instanceof Error ? err.message : 'Failed to load configuration',
          });
        }
      } finally {
        if (isMounted) {
          setLoading(false);
        }
      }
    };

    loadSettings();
    return () => {
      isMounted = false;
    };
  }, []);

  const handleCopy = (key: string, text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => {
      setCopiedKey(null);
    }, 2000);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!settings) return;

    try {
      setSaving(true);
      setStatusMsg(null);
      const updated = await api.updateSettings(settings);
      setSettings(updated);
      setStatusMsg({ type: 'success', text: 'Settings updated successfully.' });
    } catch (err: unknown) {
      setStatusMsg({
        type: 'error',
        text: err instanceof Error ? err.message : 'Failed to update settings',
      });
    } finally {
      setSaving(false);
    }
  };

  const handleDownloadRegister = async () => {
    try {
      setDownloading(true);
      const blob = await api.exportRegisterSpreadsheet();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `jobcard-register-${getTodayISO()}.xlsx`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Spreadsheet export failed');
    } finally {
      setDownloading(false);
    }
  };

  if (loading || !settings) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-slate-900" />
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 py-6 space-y-6">
      <div className="pb-4 border-b border-slate-200">
        <h1 className="text-xl sm:text-2xl font-black text-slate-900 tracking-tight">
          Application Settings
        </h1>
        <p className="text-xs text-slate-500 mt-0.5">
          Configure business identity, banking payment details, and inspect local storage directories.
        </p>
      </div>

      {statusMsg && (
        <div
          className={`flex items-center gap-2 p-3 text-xs rounded-md ${
            statusMsg.type === 'error'
              ? 'bg-rose-50 text-rose-800 border border-rose-200'
              : 'bg-emerald-50 text-emerald-800 border border-emerald-200'
          }`}
        >
          {statusMsg.type === 'error' ? (
            <AlertCircle className="w-4 h-4 shrink-0" />
          ) : (
            <CheckCircle2 className="w-4 h-4 shrink-0" />
          )}
          <span>{statusMsg.text}</span>
        </div>
      )}

      <form onSubmit={handleSave} className="space-y-6">
        {/* Business Profile */}
        <div className="bg-white border border-slate-200 rounded-lg p-5 shadow-xs space-y-4">
          <div className="flex items-center gap-2 pb-2 border-b border-slate-100">
            <Building2 className="w-4 h-4 text-indigo-600" />
            <h2 className="text-sm font-bold text-slate-800">Business Identity & Contact</h2>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
            <div>
              <label className="block font-semibold text-slate-700 mb-1">
                Business Name <span className="text-rose-500">*</span>
              </label>
              <input
                type="text"
                required
                value={settings.businessName}
                onChange={(e) => setSettings({ ...settings, businessName: e.target.value })}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white text-xs"
              />
            </div>
            <div>
              <label className="block font-semibold text-slate-700 mb-1">
                Proprietor / Contact Person <span className="text-rose-500">*</span>
              </label>
              <input
                type="text"
                required
                value={settings.person}
                onChange={(e) => setSettings({ ...settings, person: e.target.value })}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white text-xs"
              />
            </div>
            <div>
              <label className="block font-semibold text-slate-700 mb-1">
                Phone Number <span className="text-rose-500">*</span>
              </label>
              <input
                type="text"
                required
                value={settings.phone}
                onChange={(e) => setSettings({ ...settings, phone: e.target.value })}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white text-xs font-mono"
              />
            </div>
            <div>
              <label className="block font-semibold text-slate-700 mb-1">
                Email Address <span className="text-rose-500">*</span>
              </label>
              <input
                type="email"
                required
                value={settings.email}
                onChange={(e) => setSettings({ ...settings, email: e.target.value })}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white text-xs"
              />
            </div>
            <div className="sm:col-span-2">
              <label className="block font-semibold text-slate-700 mb-1">
                Business Location / City <span className="text-rose-500">*</span>
              </label>
              <input
                type="text"
                required
                value={settings.location}
                onChange={(e) => setSettings({ ...settings, location: e.target.value })}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white text-xs"
              />
            </div>
          </div>
        </div>

        {/* Branding Assets */}
        <div className="bg-white border border-slate-200 rounded-lg p-5 shadow-xs space-y-4">
          <div className="flex items-center gap-2 pb-2 border-b border-slate-100">
            <Building2 className="w-4 h-4 text-indigo-600" />
            <h2 className="text-sm font-bold text-slate-800">Client Document Branding</h2>
          </div>
          <p className="text-xs text-slate-500">
            Add images used in job cards, quotations, and invoices. PNG and JPEG files up to 256 KiB each are supported, with dimensions up to 4096 × 4096 pixels and 16 megapixels.
          </p>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
            {([
              { field: 'companyLogo', label: 'Company Logo', id: 'company-logo' },
              { field: 'technicianSignatureImage', label: 'Technician Signature', id: 'technician-signature' },
            ] as const).map(({ field, label, id }) => (
              <div key={field} className="space-y-2">
                <label htmlFor={id} className="block font-semibold text-slate-700">{label}</label>
                <input
                  id={id}
                  type="file"
                  accept="image/png,image/jpeg"
                  aria-label={label}
                  onChange={(event) => { handleBrandingImageSelect(event, field, label); }}
                  className="block w-full text-xs text-slate-600 file:mr-3 file:rounded-md file:border file:border-slate-300 file:bg-slate-50 file:px-3 file:py-1.5 file:text-xs file:font-semibold"
                />
                {settings[field] ? (
                  <div className="flex items-center gap-3">
                    <img
                      src={settings[field]}
                      alt={`${label} preview`}
                      className="max-h-20 max-w-48 rounded border border-slate-200 object-contain"
                    />
                    <button
                      type="button"
                      onClick={() => handleBrandingImageRemove(field, label)}
                      className="text-rose-700 hover:text-rose-900 font-semibold"
                    >
                      Remove
                    </button>
                  </div>
                ) : null}
              </div>
            ))}
          </div>
        </div>

        {/* Banking Details */}
        <div className="bg-white border border-slate-200 rounded-lg p-5 shadow-xs space-y-4">
          <div className="flex items-center gap-2 pb-2 border-b border-slate-100">
            <Landmark className="w-4 h-4 text-emerald-600" />
            <h2 className="text-sm font-bold text-slate-800">Banking & Settlement Details</h2>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
            <div>
              <label className="block font-semibold text-slate-700 mb-1">
                Bank Name <span className="text-rose-500">*</span>
              </label>
              <input
                type="text"
                required
                value={settings.bankName}
                onChange={(e) => setSettings({ ...settings, bankName: e.target.value })}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white text-xs"
              />
            </div>
            <div>
              <label className="block font-semibold text-slate-700 mb-1">
                Account Holder <span className="text-rose-500">*</span>
              </label>
              <input
                type="text"
                required
                value={settings.accountHolder}
                onChange={(e) => setSettings({ ...settings, accountHolder: e.target.value })}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white text-xs"
              />
            </div>
            <div>
              <label className="block font-semibold text-slate-700 mb-1">
                Account Number <span className="text-rose-500">*</span>
              </label>
              <input
                type="text"
                required
                value={settings.accountNumber}
                onChange={(e) => setSettings({ ...settings, accountNumber: e.target.value })}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white text-xs font-mono"
              />
            </div>
            <div>
              <label className="block font-semibold text-slate-700 mb-1">
                Branch Code <span className="text-rose-500">*</span>
              </label>
              <input
                type="text"
                required
                value={settings.branchCode}
                onChange={(e) => setSettings({ ...settings, branchCode: e.target.value })}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white text-xs font-mono"
              />
            </div>
          </div>
        </div>

        {/* Invoice & Quotation Defaults */}
        <div className="bg-white border border-slate-200 rounded-lg p-5 shadow-xs space-y-4">
          <div className="flex items-center gap-2 pb-2 border-b border-slate-100">
            <FileCheck2 className="w-4 h-4 text-blue-600" />
            <h2 className="text-sm font-bold text-slate-800">Tax & Default Document Terms</h2>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 text-xs">
            <div>
              <label className="block font-semibold text-slate-700 mb-1">
                VAT Tax Rate
              </label>
              <select
                value={settings.taxRate}
                onChange={(e) => setSettings({ ...settings, taxRate: parseFloat(e.target.value) })}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white text-xs"
              >
                <option value="0">0% (Non-VAT Vendor)</option>
                <option value="0.15">15% (Standard South Africa VAT)</option>
              </select>
              <span className="text-[10px] text-slate-400 mt-1 block">
                Non-VAT vendors display "VAT (Non-VAT Vendor): R0.00"
              </span>
            </div>
            <div>
              <label className="block font-semibold text-slate-700 mb-1">
                Default Payment Terms (Days)
              </label>
              <input
                type="number"
                min="1"
                required
                value={settings.defaultTermsDays}
                onChange={(e) => setSettings({ ...settings, defaultTermsDays: parseInt(e.target.value, 10) || 7 })}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white text-xs"
              />
            </div>
            <div>
              <label className="block font-semibold text-slate-700 mb-1">
                Quote Validity (Days)
              </label>
              <input
                type="number"
                min="1"
                required
                value={settings.quoteValidityDays}
                onChange={(e) => setSettings({ ...settings, quoteValidityDays: parseInt(e.target.value, 10) || 14 })}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-md focus:bg-white text-xs"
              />
            </div>
          </div>
        </div>

        {/* Save Settings Button */}
        <div className="flex justify-end">
          <button
            type="submit"
            disabled={saving}
            className="flex items-center gap-2 px-5 py-2 text-xs font-bold text-white bg-slate-900 hover:bg-slate-800 rounded-md shadow-xs transition-colors"
          >
            <Save className="w-4 h-4" />
            <span>{saving ? 'Saving Changes...' : 'Save Settings'}</span>
          </button>
        </div>
      </form>

      {/* Read-Only Data Location Card */}
      {storage && (
        <div className="bg-white border border-slate-200 rounded-lg p-5 shadow-xs space-y-4">
          <div className="flex items-center justify-between pb-2 border-b border-slate-100">
            <div className="flex items-center gap-2">
              <FolderOpen className="w-4 h-4 text-amber-600" />
              <h2 className="text-sm font-bold text-slate-800">Data Storage Location (Read-Only)</h2>
            </div>
            <button
              type="button"
              onClick={handleDownloadRegister}
              disabled={downloading}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 shadow-xs"
            >
              <Download className="w-3.5 h-3.5 text-emerald-600" />
              <span>{downloading ? 'Downloading...' : 'Download Register (.xlsx)'}</span>
            </button>
          </div>

          <p className="text-xs text-slate-500 leading-relaxed">
            The Go backend stores persistent data and SQLite databases under the host user's local application profile.
            These paths are controlled by the host system and cannot be edited via the browser UI.
          </p>

          <div className="space-y-2.5 text-xs font-mono">
            <div className="p-2.5 bg-slate-50 rounded border border-slate-200 flex items-center justify-between">
              <div className="truncate mr-2">
                <span className="text-[10px] text-slate-400 block uppercase font-sans font-semibold">
                  Application Data Directory
                </span>
                <span className="text-slate-800 font-bold text-[11px] truncate">{storage.dataDir}</span>
              </div>
              <button
                type="button"
                onClick={() => handleCopy('dataDir', storage.dataDir)}
                className="p-1.5 text-slate-500 hover:text-slate-900 rounded shrink-0 bg-white border border-slate-200"
                title="Copy to clipboard"
              >
                {copiedKey === 'dataDir' ? <Check className="w-3.5 h-3.5 text-emerald-600" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
            </div>

            <div className="p-2.5 bg-slate-50 rounded border border-slate-200 flex items-center justify-between">
              <div className="truncate mr-2">
                <span className="text-[10px] text-slate-400 block uppercase font-sans font-semibold">
                  SQLite Database Path
                </span>
                <span className="text-slate-800 font-bold text-[11px] truncate">{storage.dbPath}</span>
              </div>
              <button
                type="button"
                onClick={() => handleCopy('dbPath', storage.dbPath)}
                className="p-1.5 text-slate-500 hover:text-slate-900 rounded shrink-0 bg-white border border-slate-200"
                title="Copy to clipboard"
              >
                {copiedKey === 'dbPath' ? <Check className="w-3.5 h-3.5 text-emerald-600" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
            </div>

            <div className="p-2.5 bg-slate-50 rounded border border-slate-200 flex items-center justify-between">
              <div className="truncate mr-2">
                <span className="text-[10px] text-slate-400 block uppercase font-sans font-semibold">
                  Client Documents Export Directory
                </span>
                <span className="text-slate-800 font-bold text-[11px] truncate">{storage.clientsDir}</span>
              </div>
              <button
                type="button"
                onClick={() => handleCopy('clientsDir', storage.clientsDir)}
                className="p-1.5 text-slate-500 hover:text-slate-900 rounded shrink-0 bg-white border border-slate-200"
                title="Copy to clipboard"
              >
                {copiedKey === 'clientsDir' ? <Check className="w-3.5 h-3.5 text-emerald-600" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
