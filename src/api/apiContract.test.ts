import test from 'node:test';
import assert from 'node:assert/strict';
import { ApiClient } from './client.ts';
import { MockApi } from './mockApi.ts';
import { HttpApi } from './httpApi.ts';
import { RawSeedData } from './seedLoader.ts';
import rawSeedData from '../../seed/seed.json';
import { ApiError } from './types.ts';

function runApiContractSuite(name: string, createClient: () => ApiClient) {
  test(`[${name}] Client CRUD operations`, async () => {
    const api = createClient();
    const initialClients = await api.listClients();
    assert.ok(initialClients.length >= 6);

    // Create client
    const newClient = await api.createClient({
      name: 'Contract Test Corp',
      contact: 'Tester Alice',
      phone: '0829990000',
      email: 'alice@contract.test',
      address: '100 Test Blvd',
      notes: 'Automated contract test client',
    });
    assert.ok(newClient.id);
    assert.equal(newClient.name, 'Contract Test Corp');

    // Update client
    const updated = await api.updateClient(newClient.id, {
      contact: 'Tester Bob',
    });
    assert.equal(updated.contact, 'Tester Bob');
    assert.equal(updated.name, 'Contract Test Corp');

    // Validation: empty name throws 400
    await assert.rejects(
      async () => {
        await api.createClient({ name: '' });
      },
      (err: unknown) => {
        const apiErr = err as ApiError;
        assert.equal(apiErr.status, 400);
        return true;
      }
    );

    const history = await api.createDraft({
      type: 'jobcard',
      clientId: newClient.id,
      reference: 'Archive history test',
    });
    await api.archiveClient(newClient.id);
    assert.equal((await api.listClients()).some((client) => client.id === newClient.id), false);
    assert.equal((await api.listArchivedClients()).some((client) => client.id === newClient.id), true);
    assert.equal(
      (await api.updateDraft(history.id, { scope: 'Existing draft remains editable' })).scope,
      'Existing draft remains editable'
    );
    assert.equal((await api.getDocument(history.id)).clientId, newClient.id);
    await assert.rejects(
      async () => await api.createDraft({ type: 'jobcard', clientId: newClient.id, reference: 'Archived client' }),
      (err: unknown) => err instanceof ApiError && err.status === 400
    );

    await api.restoreClient(newClient.id);
    assert.equal((await api.listClients()).some((client) => client.id === newClient.id), true);
    assert.equal((await api.listArchivedClients()).some((client) => client.id === newClient.id), false);
  });

  test(`[${name}] Document draft creation, updating, and issuing sequence`, async () => {
    const api = createClient();
    const clients = await api.listClients();
    const clientId = clients[0].id;

    // 1. Create draft invoice
    const draft = await api.createDraft({
      type: 'invoice',
      clientId,
      reference: 'Contract Test Device',
      scope: 'Diagnostic check',
      lines: [
        { description: 'Testing fee', qty: 1, unitPriceCents: 50000, totalCents: 50000 },
      ],
    });

    assert.equal(draft.type, 'invoice');
    assert.equal(draft.number, null);
    assert.equal(draft.status, 'draft');
    assert.equal(draft.totalCents, 50000);

    // 2. Update draft
    const updatedDraft = await api.updateDraft(draft.id, {
      scope: 'Diagnostic check + thermal paste',
      lines: [
        { description: 'Testing fee', qty: 1, unitPriceCents: 50000, totalCents: 50000 },
        { description: 'Thermal paste', qty: 1, unitPriceCents: 20000, totalCents: 20000 },
      ],
    });
    assert.equal(updatedDraft.totalCents, 70000);

    // 3. Issue document assigns sequential number
    const issued = await api.issueDocument(draft.id, '2026-11-08');
    assert.ok(issued.number);
    assert.match(issued.number, /^INV-2026-\d{3}$/);
    assert.equal(issued.status, 'sent');
    assert.equal(issued.issueDate, '2026-11-08');
    assert.ok(issued.dueDate);
    const settings = await api.getSettings();
    assert.equal(issued.authorization.technicianSignature, settings.person);
    assert.equal(issued.authorization.technicianDate, '2026-11-08');

    const reportFields = {
      intakeItems: [{ label: 'Primary storage', value: '1TB HDD' }],
      diagnosticSections: [{ title: 'Drive health', status: 'PASSED', details: '100% health' }],
      authorization: {
        technicianSignature: 'Tech',
        technicianDate: '2026-10-12',
        clientSignature: 'Client',
        clientDate: '2026-10-12',
      },
    };
    const reportDraft = await api.createDraft(Object.assign({
      type: 'invoice' as const,
      clientId,
      reference: 'Report contract device',
      lines: [{ description: 'Report work', qty: 1, unitPriceCents: 1000, totalCents: 1000 }],
    }, reportFields));
    assert.deepEqual(JSON.parse(JSON.stringify(reportDraft)).intakeItems, reportFields.intakeItems);
    assert.deepEqual(JSON.parse(JSON.stringify(reportDraft)).diagnosticSections, reportFields.diagnosticSections);
    assert.deepEqual(JSON.parse(JSON.stringify(reportDraft)).authorization, reportFields.authorization);

    const reportUpdate = {
      ...reportFields,
      intakeItems: [{ label: 'Accessories', value: 'Carrying bag' }],
    };
    const updatedReport = await api.updateDraft(reportDraft.id, reportUpdate);
    assert.deepEqual(JSON.parse(JSON.stringify(updatedReport)).intakeItems, reportUpdate.intakeItems);
    const issuedReport = await api.issueDocument(reportDraft.id, '2026-11-09');
    assert.equal(issuedReport.authorization.technicianSignature, 'Tech');
    assert.equal(issuedReport.authorization.technicianDate, '2026-10-12');

    // 4. Immutability of issued document: cannot re-issue or edit
    await assert.rejects(
      async () => {
        await api.issueDocument(draft.id);
      },
      (err: unknown) => {
        const apiErr = err as ApiError;
        assert.equal(apiErr.status, 409);
        return true;
      }
    );

    await assert.rejects(
      async () => {
        await api.updateDraft(draft.id, { scope: 'Attempted edit after issue' });
      },
      (err: unknown) => {
        const apiErr = err as ApiError;
        assert.equal(apiErr.status, 409);
        return true;
      }
    );
  });

  test(`[${name}] Payment effects and overpayment rejection (422)`, async () => {
    const api = createClient();
    const clients = await api.listClients();

    const draft = await api.createDraft({
      type: 'invoice',
      clientId: clients[0].id,
      reference: 'Payment Test Rig',
      lines: [{ description: 'Motherboard Repair', qty: 1, unitPriceCents: 100000, totalCents: 100000 }],
    });
    const issued = await api.issueDocument(draft.id, '2026-11-08');

    // Partial payment: 40000 cents
    const partPaid = await api.addPayment(issued.id, {
      date: '2026-11-08',
      amountCents: 40000,
      method: 'EFT',
      reference: 'PART-1',
    });
    assert.equal(partPaid.payments.length, 1);
    assert.equal(partPaid.status, 'partially_paid');

    // Overpayment rejection: remaining balance is 60000, paying 70000 must reject with 422
    await assert.rejects(
      async () => {
        await api.addPayment(issued.id, {
          date: '2026-11-08',
          amountCents: 70000,
          method: 'EFT',
          reference: 'OVERPAY',
        });
      },
      (err: unknown) => {
        const apiErr = err as ApiError;
        assert.equal(apiErr.status, 422);
        assert.equal(apiErr.code, 'OVERPAYMENT_REJECTED');
        return true;
      }
    );

    // Final payment of remaining 60000 marks status paid
    const fullPaid = await api.addPayment(issued.id, {
      date: '2026-11-08',
      amountCents: 60000,
      method: 'EFT',
      reference: 'FULL-2',
    });
    assert.equal(fullPaid.payments.length, 2);
    assert.equal(fullPaid.status, 'paid');
  });

  test(`[${name}] Quotation conversion to invoice`, async () => {
    const api = createClient();
    const clients = await api.listClients();

    const quoDraft = await api.createDraft({
      type: 'quotation',
      clientId: clients[0].id,
      reference: 'Quote convert test',
      scope: 'Network installation',
      lines: [{ description: 'Cable spool', qty: 1, unitPriceCents: 120000, totalCents: 120000 }],
    });
    const quoIssued = await api.issueDocument(quoDraft.id, '2026-11-08');
    assert.match(quoIssued.number!, /^QUO-2026-\d{3}$/);

    // Convert quotation
    const newInvoice = await api.convertQuotation(quoIssued.id);
    assert.equal(newInvoice.type, 'invoice');
    assert.equal(newInvoice.number, null); // draft until issued
    assert.equal(newInvoice.quotationId, quoIssued.id);
    assert.equal(newInvoice.totalCents, 120000);

    // Source quotation must now be 'converted'
    const updatedQuo = await api.getDocument(quoIssued.id);
    assert.equal(updatedQuo.status, 'converted');

    // Cannot convert an already converted quotation
    await assert.rejects(
      async () => {
        await api.convertQuotation(quoIssued.id);
      },
      (err: unknown) => {
        const apiErr = err as ApiError;
        assert.equal(apiErr.status, 409);
        return true;
      }
    );
  });

  test(`[${name}] Draft archive/delete and quotation revision lifecycle`, async () => {
    const api = createClient();
    const clients = await api.listClients();
    const clientId = clients[0].id;
    const draft = await api.createDraft({ type: 'jobcard', clientId, reference: 'Archive lifecycle' });

    await api.archiveDraft(draft.id);
    assert.equal((await api.listDocuments()).some((doc) => doc.id === draft.id), false);
    assert.equal((await api.listArchivedDocuments()).some((doc) => doc.id === draft.id), true);
    await api.restoreDraft(draft.id);
    await assert.rejects(() => api.deleteDraft(draft.id, false), (err: unknown) =>
      err instanceof ApiError && err.code === 'CONFIRMATION_REQUIRED'
    );
    await api.deleteDraft(draft.id, true);
    await assert.rejects(() => api.getDocument(draft.id), (err: unknown) =>
      err instanceof ApiError && err.status === 404
    );

    const quoteDraft = await api.createDraft({
      type: 'quotation',
      clientId,
      reference: 'Quotation lifecycle',
      lines: [{ description: 'Repair', qty: 1, unitPriceCents: 12500, totalCents: 12500 }],
    });
    const quote = await api.issueDocument(quoteDraft.id, '2026-11-08');
    await assert.rejects(() => api.updateIssuedQuotation(quote.id, { scope: 'Revised' }, false), (err: unknown) =>
      err instanceof ApiError && err.code === 'CONFIRMATION_REQUIRED'
    );
    const revised = await api.updateIssuedQuotation(quote.id, { scope: 'Revised' }, true);
    assert.equal(revised.number, quote.number);
    assert.equal(revised.status, quote.status);
    assert.equal((await api.listQuotationRevisions(quote.id))[0].snapshot.scope, quote.scope);
    const revisionCount = (await api.listQuotationRevisions(quote.id)).length;
    await assert.rejects(
      () => api.updateIssuedQuotation(quote.id, { type: 'invoice' }, true),
      (err: unknown) => err instanceof ApiError && err.code === 'VALIDATION_FAILED'
    );
    await assert.rejects(
      () => api.updateIssuedQuotation(quote.id, {
        lines: [{ description: '', qty: 1, unitPriceCents: 0, totalCents: 0 }],
      }, true),
      (err: unknown) => err instanceof ApiError && err.code === 'VALIDATION_FAILED'
    );
    assert.equal((await api.getDocument(quote.id)).type, 'quotation');
    assert.equal((await api.listQuotationRevisions(quote.id)).length, revisionCount);

    const replacement = await api.createReplacementQuotation(quote.id);
    assert.equal(replacement.revisionOfQuotationId, quote.id);
    assert.equal(replacement.totalCents, revised.totalCents);
    await api.issueDocument(replacement.id, '2026-11-09');
    assert.equal((await api.getDocument(quote.id)).status, 'cancelled');
  });

  test(`[${name}] Document export constraints (drafts blocked, issued allowed)`, async () => {
    const api = createClient();
    const clients = await api.listClients();

    const draft = await api.createDraft({
      type: 'invoice',
      clientId: clients[0].id,
      reference: 'Export Test',
      lines: [{ description: 'Part', qty: 1, unitPriceCents: 10000, totalCents: 10000 }],
    });

    // Drafts must not be exported
    await assert.rejects(
      async () => {
        await api.exportDocument(draft.id, 'pdf');
      },
      (err: unknown) => {
        const apiErr = err as ApiError;
        assert.equal(apiErr.status, 400);
        return true;
      }
    );

    // Issue then export
    const issued = await api.issueDocument(draft.id, '2026-11-08');
    const blob = await api.exportDocument(issued.id, 'pdf');
    assert.ok(blob);
    assert.ok(blob.size > 0);
  });

  test(`[${name}] Follow-up date: rules, set, clear and dashboard count`, async () => {
    const api = createClient();
    const clients = await api.listClients();
    const dueCount = async () => (await api.getDashboard()).attention.followUpsDue;
    const before = await dueCount();
    const expectError = (status: number, code: string) => (err: unknown) => {
      const apiErr = err as ApiError;
      assert.equal(apiErr.status, status);
      assert.equal(apiErr.code, code);
      return true;
    };

    const draft = await api.createDraft({
      type: 'invoice',
      clientId: clients[0].id,
      reference: 'Follow-up Test',
      lines: [{ description: 'Service', qty: 1, unitPriceCents: 50000, totalCents: 50000 }],
    });
    assert.equal(draft.followUpDate, null);

    // only issued invoices can carry a follow-up
    await assert.rejects(() => api.setFollowUp(draft.id, '2000-01-01'), expectError(409, 'INVALID_STATE'));
    const issued = await api.issueDocument(draft.id, '2026-11-08');
    assert.equal(issued.followUpDate, null);
    await assert.rejects(() => api.setFollowUp(issued.id, '08/11/2026'), expectError(400, 'VALIDATION_FAILED'));
    await assert.rejects(() => api.setFollowUp('missing-id', '2000-01-01'), expectError(404, 'NOT_FOUND'));

    // a past follow-up date on an unpaid invoice is due now
    const set = await api.setFollowUp(issued.id, '2000-01-01');
    assert.equal(set.followUpDate, '2000-01-01');
    assert.equal(await dueCount(), before + 1);

    // a future one is not due yet
    await api.setFollowUp(issued.id, '2999-01-01');
    assert.equal(await dueCount(), before);

    // paying in full removes it from the due list even if the date has passed
    await api.setFollowUp(issued.id, '2000-01-01');
    assert.equal(await dueCount(), before + 1);
    await api.addPayment(issued.id, { date: '2026-11-08', amountCents: 50000, method: 'EFT' });
    assert.equal(await dueCount(), before);

    // null clears it
    const cleared = await api.setFollowUp(issued.id, null);
    assert.equal(cleared.followUpDate, null);
    assert.equal((await api.getDocument(issued.id)).followUpDate, null);

    // quotations and job cards do not take follow-ups
    const quote = await api.createDraft({
      type: 'quotation',
      clientId: clients[0].id,
      reference: 'Follow-up Quote',
      lines: [{ description: 'Quote', qty: 1, unitPriceCents: 1000, totalCents: 1000 }],
    });
    await assert.rejects(() => api.setFollowUp(quote.id, '2000-01-01'), expectError(400, 'INVALID_DOCUMENT_TYPE'));
  });
  test(`[${name}] Settings and Dashboard queries`, async () => {
    const api = createClient();
    const settings = await api.getSettings();
    assert.ok(settings.businessName);

    const dashboard = await api.getDashboard(2026);
    assert.equal(dashboard.year, 2026);
    assert.ok(dashboard.totals.invoicedCents >= 0);
    assert.ok(dashboard.monthly.length === 12);
    assert.ok(dashboard.nextNumbers.invoice);
    assert.ok(dashboard.nextNumbers.quotation);
    assert.ok(dashboard.nextNumbers.jobcard);
  });
}

// 1. Run against MockApi
runApiContractSuite('MockApi', () => {
  return new MockApi(rawSeedData as unknown as RawSeedData, '2026-11-08');
});

// 2. Run against HttpApi if VITE_API_URL or API_URL is set in environment
if (process.env.VITE_API_URL || process.env.API_URL) {
  const url = process.env.VITE_API_URL || process.env.API_URL;
  runApiContractSuite('HttpApi', () => new HttpApi(url, process.env.VITE_API_TOKEN || ''));
}
