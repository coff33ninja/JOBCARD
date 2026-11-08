import test from 'node:test';
import assert from 'node:assert/strict';
import { exportOptions, toWhatsAppNumber } from './export.ts';

test('toWhatsAppNumber normalises South African numbers', () => {
  assert.equal(toWhatsAppNumber('082 123 4567'), '27821234567');
  assert.equal(toWhatsAppNumber('+27 82 123 4567'), '27821234567');
  assert.equal(toWhatsAppNumber('(011) 555-0199'), '27115550199');
});

test('exportOptions when all capabilities are true and canShare is true', () => {
  const caps = { docx: true, pdf: true, revealInFolder: true };
  const options = exportOptions(caps, { canShare: true, isDraft: false });

  assert.equal(options.length, 5);
  assert.equal(options[0].type, 'pdf');
  assert.equal(options[1].type, 'docx');
  assert.equal(options[2].type, 'print');
  assert.equal(options[3].type, 'share');
  assert.equal(options[4].type, 'reveal');
});

test('exportOptions when canShare is false offers email and whatsapp', () => {
  const caps = { docx: false, pdf: true, revealInFolder: false };
  const options = exportOptions(caps, { canShare: false, isDraft: false });

  assert.equal(options.length, 4);
  assert.equal(options[0].type, 'pdf');
  assert.equal(options[1].type, 'print');
  assert.equal(options[2].type, 'email');
  assert.equal(options[3].type, 'whatsapp');
});

test('exportOptions when pdf capability is false provides print fallback', () => {
  const caps = { docx: false, pdf: false, revealInFolder: false };
  const options = exportOptions(caps, { canShare: false, isDraft: false });

  assert.equal(options[0].isFallback, true);
  assert.equal(options[0].type, 'pdf');
});

test('exportOptions disables non-print options for drafts', () => {
  const caps = { docx: true, pdf: true, revealInFolder: true };
  const options = exportOptions(caps, { canShare: true, isDraft: true });

  const pdf = options.find(o => o.type === 'pdf')!;
  const docx = options.find(o => o.type === 'docx')!;
  const print = options.find(o => o.type === 'print')!;
  const share = options.find(o => o.type === 'share')!;

  assert.equal(pdf.disabled, true);
  assert.equal(docx.disabled, true);
  assert.equal(print.disabled, false);
  assert.equal(share.disabled, true);
});
