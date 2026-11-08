import { Capabilities } from '../api/types.ts';

export type ExportActionType = 'pdf' | 'docx' | 'print' | 'share' | 'email' | 'whatsapp' | 'reveal';

export interface ExportOption {
  type: ExportActionType;
  label: string;
  isFallback?: boolean;
  disabled?: boolean;
  disabledReason?: string;
}

export interface ExportEnv {
  canShare?: boolean;
  isDraft?: boolean;
}

/**
 * Normalises South African phone numbers for WhatsApp:
 * strips non-digits, converts leading '0' to '27'. E.g. "082 123 4567" -> "27821234567".
 */
export function toWhatsAppNumber(phone: string): string {
  const digits = phone.replace(/\D/g, '');
  if (digits.startsWith('0')) {
    return '27' + digits.slice(1);
  }
  return digits;
}

/**
 * Decision logic for available export actions based on backend capabilities and environment.
 */
export function exportOptions(
  capabilities: Capabilities,
  env: ExportEnv = {}
): ExportOption[] {
  const isDraft = !!env.isDraft;
  const options: ExportOption[] = [];

  // 1. Download PDF
  if (capabilities.pdf) {
    options.push({
      type: 'pdf',
      label: 'Download PDF',
      disabled: isDraft,
      disabledReason: isDraft ? 'Document must be issued to export PDF' : undefined,
    });
  } else {
    // If backend cannot generate PDF, fall back to Print / Save as PDF
    options.push({
      type: 'pdf',
      label: 'Save as PDF (via Print)',
      isFallback: true,
      disabled: false,
    });
  }

  // 2. Download DOCX
  if (capabilities.docx) {
    options.push({
      type: 'docx',
      label: 'Download DOCX',
      disabled: isDraft,
      disabledReason: isDraft ? 'Document must be issued to export DOCX' : undefined,
    });
  }

  // 3. Print (always available, even for drafts)
  options.push({
    type: 'print',
    label: 'Print',
    disabled: false,
  });

  // 4. Share... or Fallbacks (Email & WhatsApp)
  if (env.canShare) {
    options.push({
      type: 'share',
      label: 'Share...',
      disabled: isDraft,
      disabledReason: isDraft ? 'Document must be issued to share' : undefined,
    });
  } else {
    // Fallbacks: Email and WhatsApp
    options.push({
      type: 'email',
      label: 'Email client',
      disabled: isDraft,
      disabledReason: isDraft ? 'Document must be issued to email' : undefined,
    });
    options.push({
      type: 'whatsapp',
      label: 'WhatsApp client',
      disabled: isDraft,
      disabledReason: isDraft ? 'Document must be issued to message on WhatsApp' : undefined,
    });
  }

  // 5. Show in folder
  if (capabilities.revealInFolder) {
    options.push({
      type: 'reveal',
      label: 'Show in folder',
      disabled: isDraft,
      disabledReason: isDraft ? 'Document must be issued to reveal in folder' : undefined,
    });
  }

  return options;
}
