import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { ENDPOINTS } from '../../src/api/endpoints.ts';

const DOCS_DIR = path.resolve('docs/api');

test('every endpoint in registry has a corresponding doc file with required sections', () => {
  assert.ok(fs.existsSync(DOCS_DIR), 'docs/api directory must exist');

  const requiredSections = [
    '## ApiClient method',
    '## MCP tool',
    '## Request',
    '## Response',
    '## Errors',
    '## Rules and side effects',
    '## Example',
    '## Used by',
  ];

  for (const ep of ENDPOINTS) {
    const docPath = path.resolve(ep.doc);
    assert.ok(
      fs.existsSync(docPath),
      `Doc file missing for ${ep.method} ${ep.path}: expected ${ep.doc}`
    );

    const content = fs.readFileSync(docPath, 'utf-8');
    const lines = content.split('\n');

    // Section 1: Title
    const titleLine = lines.find((l) => l.startsWith('# '));
    assert.ok(
      titleLine,
      `Doc ${ep.doc} must have a # title with METHOD and path`
    );
    assert.ok(
      titleLine.includes(ep.method) && titleLine.includes(ep.path.replace(':id', '{id}')),
      `Doc ${ep.doc} title should match ${ep.method} ${ep.path}`
    );

    // Section 2: Exact client method and MCP tool
    assert.ok(
      content.includes(`\`${ep.clientMethod}\``) || content.includes(ep.clientMethod),
      `Doc ${ep.doc} must include clientMethod ${ep.clientMethod}`
    );
    assert.ok(
      content.includes(`\`${ep.mcpTool}\``) || content.includes(ep.mcpTool),
      `Doc ${ep.doc} must include mcpTool ${ep.mcpTool}`
    );

    // Check required sections in order
    let lastIndex = -1;
    for (const sec of requiredSections) {
      const idx = content.indexOf(sec);
      assert.ok(
        idx !== -1,
        `Doc ${ep.doc} is missing required section: ${sec}`
      );
      assert.ok(
        idx > lastIndex,
        `Doc ${ep.doc} section ${sec} is out of order`
      );
      lastIndex = idx;
    }
  }
});
