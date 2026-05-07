#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';
import { fileURLToPath } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(scriptDir, '..', '..');
const defaultDistDir = '.goalrail/public-kb/dist';
const openaiBaseUrl = 'https://api.openai.com/v1';

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const distDir = normalizeOutputPath(args.dist ?? defaultDistDir);
  const distAbs = path.join(repoRoot, distDir);
  const manifestPath = path.join(distAbs, 'public-manifest.json');
  const publicKbPath = path.join(distAbs, 'public-kb.md');

  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  const publicKb = fs.readFileSync(publicKbPath);

  if (!args.execute) {
    console.log(
      [
        'Dry run only. No OpenAI request was sent.',
        `Would upload: ${path.relative(repoRoot, publicKbPath)}`,
        `Sources: ${manifest.sources_count}`,
        `Chunks: ${manifest.chunks_count}`,
        `Commit: ${manifest.commit_sha}`,
        'Re-run with --execute and OPENAI_API_KEY set to create a vector store.',
      ].join('\n'),
    );
    return;
  }

  const apiKey = process.env.OPENAI_API_KEY;

  if (!apiKey) {
    throw new Error('OPENAI_API_KEY is required for --execute.');
  }

  if (typeof fetch !== 'function' || typeof FormData !== 'function' || typeof Blob !== 'function') {
    throw new Error('Node.js with fetch, FormData, and Blob support is required.');
  }

  const uploadedFile = await uploadFile(apiKey, publicKb, 'goalrail-public-kb.md');
  const vectorStore = await createVectorStore(apiKey, manifest);
  const batch = await createFileBatch(apiKey, vectorStore.id, uploadedFile.id);
  const finalBatch = args.noWait
    ? batch
    : await waitForFileBatch(apiKey, vectorStore.id, batch.id);

  const runtimeManifest = {
    ...manifest,
    retrieval: {
      provider: 'openai',
      kind: 'file_search',
      index_id: vectorStore.id,
      file_id: uploadedFile.id,
      file_batch_id: batch.id,
      file_batch_status: finalBatch.status,
    },
    worker_config: {
      OPENAI_START_VECTOR_STORE_ID: vectorStore.id,
      START_ASSISTANT_KB_REVISION: manifest.commit_sha,
      START_ASSISTANT_KB_UPDATED_AT: manifest.updated_at,
    },
    uploaded_at: new Date().toISOString(),
  };

  fs.writeFileSync(
    path.join(distAbs, 'runtime-manifest.json'),
    `${JSON.stringify(runtimeManifest, null, 2)}\n`,
  );

  const runtimeManifestPath = path.join(distDir, 'runtime-manifest.json');

  if (args.quiet) {
    console.log(
      [
        'Uploaded Goalrail public KB to OpenAI file_search.',
        `Runtime manifest: ${runtimeManifestPath}`,
        'Worker runtime config values were written to the runtime manifest and were not printed.',
      ].join('\n'),
    );
    return;
  }

  console.log(
    [
      'Uploaded Goalrail public KB to OpenAI file_search.',
      `Vector store: ${vectorStore.id}`,
      `File: ${uploadedFile.id}`,
      `Batch: ${batch.id} (${finalBatch.status})`,
      `Runtime manifest: ${runtimeManifestPath}`,
      '',
      'Configure the Worker deployment with:',
      `OPENAI_START_VECTOR_STORE_ID=${vectorStore.id}`,
      `START_ASSISTANT_KB_REVISION=${manifest.commit_sha}`,
      `START_ASSISTANT_KB_UPDATED_AT=${manifest.updated_at}`,
    ].join('\n'),
  );
}

function parseArgs(argv) {
  const args = { execute: false, noWait: false, quiet: false };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];

    if (arg === '--execute') {
      args.execute = true;
      continue;
    }

    if (arg === '--no-wait') {
      args.noWait = true;
      continue;
    }

    if (arg === '--quiet') {
      args.quiet = true;
      continue;
    }

    if (arg === '--dist') {
      args.dist = argv[index + 1];
      index += 1;
      continue;
    }

    if (arg === '--help' || arg === '-h') {
      printHelp();
      process.exit(0);
    }

    throw new Error(`Unknown argument: ${arg}`);
  }

  return args;
}

function printHelp() {
  console.log(`Usage: node scripts/start-assistant/upload-public-kb-openai.mjs [options]

Options:
  --execute      Send OpenAI requests. Omitted by default for a safe dry run.
  --no-wait      Do not poll the vector store file batch.
  --quiet        Do not print provider IDs or Worker config values after upload.
  --dist <path>  Generated KB artifact directory. Default: ${defaultDistDir}
`);
}

function normalizeOutputPath(rawPath) {
  if (!rawPath || typeof rawPath !== 'string') {
    throw new Error('Expected a non-empty repository-relative path.');
  }

  const cleaned = rawPath.trim();

  if (path.isAbsolute(cleaned)) {
    throw new Error(`Absolute paths are not allowed: ${cleaned}`);
  }

  const normalized = path.posix.normalize(cleaned.replaceAll(path.sep, '/'));

  if (normalized === '.' || normalized.startsWith('../') || normalized === '..') {
    throw new Error(`Path must stay inside the repository: ${cleaned}`);
  }

  return normalized;
}

async function uploadFile(apiKey, content, filename) {
  const form = new FormData();
  form.append('purpose', 'assistants');
  form.append('file', new Blob([content], { type: 'text/markdown' }), filename);

  const response = await fetch(`${openaiBaseUrl}/files`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${apiKey}`,
    },
    body: form,
  });

  return parseOpenAIResponse(response, 'upload file');
}

async function createVectorStore(apiKey, manifest) {
  const nameSuffix = String(manifest.commit_sha ?? 'unknown').slice(0, 12);

  const response = await fetch(`${openaiBaseUrl}/vector_stores`, {
    method: 'POST',
    headers: jsonHeaders(apiKey),
    body: JSON.stringify({
      name: `goalrail-start-assistant-${nameSuffix}`,
      metadata: {
        project: 'goalrail',
        repository: String(manifest.repository ?? 'heurema/goalrail'),
        commit_sha: String(manifest.commit_sha ?? 'unknown'),
        source_manifest_sha: String(manifest.source_manifest_sha ?? 'unknown'),
        purpose: 'public_start_assistant',
      },
    }),
  });

  return parseOpenAIResponse(response, 'create vector store');
}

async function createFileBatch(apiKey, vectorStoreId, fileId) {
  const response = await fetch(
    `${openaiBaseUrl}/vector_stores/${encodeURIComponent(vectorStoreId)}/file_batches`,
    {
      method: 'POST',
      headers: jsonHeaders(apiKey),
      body: JSON.stringify({ file_ids: [fileId] }),
    },
  );

  return parseOpenAIResponse(response, 'create vector store file batch');
}

async function waitForFileBatch(apiKey, vectorStoreId, batchId) {
  const terminalStatuses = new Set(['completed', 'failed', 'cancelled', 'expired']);
  const maxAttempts = 60;

  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    const response = await fetch(
      `${openaiBaseUrl}/vector_stores/${encodeURIComponent(vectorStoreId)}/file_batches/${encodeURIComponent(batchId)}`,
      {
        headers: jsonHeaders(apiKey),
      },
    );
    const batch = await parseOpenAIResponse(response, 'retrieve vector store file batch');

    if (terminalStatuses.has(batch.status)) {
      if (batch.status !== 'completed') {
        throw new Error(`Vector store file batch ended with status: ${batch.status}`);
      }

      return batch;
    }

    await delay(2000);
  }

  throw new Error(`Timed out waiting for vector store file batch: ${batchId}`);
}

function jsonHeaders(apiKey) {
  return {
    Authorization: `Bearer ${apiKey}`,
    'Content-Type': 'application/json',
    'OpenAI-Beta': 'assistants=v2',
  };
}

async function parseOpenAIResponse(response, action) {
  const text = await response.text();
  let payload = null;

  if (text) {
    try {
      payload = JSON.parse(text);
    } catch {
      payload = { raw: text };
    }
  }

  if (!response.ok) {
    const message = payload?.error?.message ?? text ?? response.statusText;
    throw new Error(`OpenAI ${action} failed (${response.status}): ${message}`);
  }

  return payload;
}

try {
  await main();
} catch (error) {
  console.error(error instanceof Error ? error.message : String(error));
  process.exit(1);
}
