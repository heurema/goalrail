#!/usr/bin/env node

import { createHash } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(scriptDir, '..', '..');
const defaultManifestPath = '.goalrail/public-kb/manifest.yaml';
const defaultOutDir = '.goalrail/public-kb/dist';

function main() {
  const args = parseArgs(process.argv.slice(2));
  const manifestPath = normalizeRepoPath(args.manifest ?? defaultManifestPath);
  const outDir = normalizeOutputPath(args.out ?? defaultOutDir);
  const now = new Date().toISOString();
  const commitSha = readGit(['rev-parse', 'HEAD']) ?? 'unknown';

  const manifestAbs = path.join(repoRoot, manifestPath);
  const manifestText = fs.readFileSync(manifestAbs, 'utf8');
  const manifest = parseManifest(manifestText);
  validateManifest(manifest, manifestPath);

  const sourceManifestSha = sha256(manifestText);
  const chunks = [];
  const sourceRows = [];

  for (const source of manifest.sources) {
    validateSourceRow(source);
    const sourcePath = normalizeRepoPath(source.path);
    const sourceAbs = path.join(repoRoot, sourcePath);

    if (!fs.existsSync(sourceAbs)) {
      if (source.optional) {
        continue;
      }

      throw new Error(`Whitelisted source does not exist: ${sourcePath}`);
    }

    const text = fs.readFileSync(sourceAbs, 'utf8');
    scanForSecretMarkers(text, sourcePath);
    scanForCyrillic(text, sourcePath);

    const sourceChunks = splitMarkdown(text, {
      path: sourcePath,
      title: source.title,
      priority: source.priority,
      commitSha,
      updatedAt: now,
    });

    sourceRows.push({
      path: sourcePath,
      title: source.title,
      priority: source.priority,
      public: true,
      optional: Boolean(source.optional),
      sha256: sha256(text),
      bytes: Buffer.byteLength(text),
      chunks_count: sourceChunks.length,
    });

    chunks.push(...sourceChunks);
  }

  if (chunks.length === 0) {
    throw new Error('No public KB chunks were generated.');
  }

  ensureUniqueChunkIds(chunks);
  const publicDocument = renderPublicDocument(chunks, {
    repository: manifest.repository ?? 'heurema/goalrail',
    commitSha,
    updatedAt: now,
    sourceManifestPath: manifestPath,
    sourceManifestSha,
  });

  scanForSecretMarkers(publicDocument, 'generated public-kb.md');
  scanForCyrillic(publicDocument, 'generated public-kb.md');

  const compiledManifest = {
    schema_version: 'goalrail.public_kb_compiled.v1',
    project: manifest.project ?? 'goalrail',
    repository: manifest.repository ?? 'heurema/goalrail',
    purpose: manifest.purpose ?? 'public_start_assistant',
    status: 'compiled',
    commit_sha: commitSha,
    updated_at: now,
    source_manifest_path: manifestPath,
    source_manifest_sha: sourceManifestSha,
    sources_count: sourceRows.length,
    chunks_count: chunks.length,
    sources: sourceRows,
    artifacts: {
      public_manifest: 'public-manifest.json',
      chunks: 'chunks.ndjson',
      public_document: 'public-kb.md',
    },
  };

  const outAbs = path.join(repoRoot, outDir);
  fs.mkdirSync(outAbs, { recursive: true });
  fs.writeFileSync(
    path.join(outAbs, 'public-manifest.json'),
    `${JSON.stringify(compiledManifest, null, 2)}\n`,
  );
  fs.writeFileSync(
    path.join(outAbs, 'chunks.ndjson'),
    `${chunks.map((chunk) => JSON.stringify(chunk)).join('\n')}\n`,
  );
  fs.writeFileSync(path.join(outAbs, 'public-kb.md'), publicDocument);

  console.log(
    [
      `Built Goalrail public KB from ${sourceRows.length} sources and ${chunks.length} chunks.`,
      `Output: ${outDir}`,
      `Commit: ${commitSha}`,
      `Manifest SHA-256: ${sourceManifestSha}`,
    ].join('\n'),
  );
}

function parseArgs(argv) {
  const args = {};

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];

    if (arg === '--manifest') {
      args.manifest = argv[index + 1];
      index += 1;
      continue;
    }

    if (arg === '--out') {
      args.out = argv[index + 1];
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
  console.log(`Usage: node scripts/start-assistant/build-public-kb.mjs [options]

Options:
  --manifest <path>  Source whitelist manifest. Default: ${defaultManifestPath}
  --out <path>       Generated artifact directory. Default: ${defaultOutDir}
`);
}

function normalizeRepoPath(rawPath) {
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

  if (isForbiddenPath(normalized)) {
    throw new Error(`Forbidden KB path class: ${normalized}`);
  }

  return normalized;
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

  const lower = normalized.toLowerCase();

  if (
    lower.includes('/.git/') ||
    lower.startsWith('.git/') ||
    lower.includes('/node_modules/') ||
    lower.startsWith('node_modules/') ||
    lower.includes('/.env') ||
    lower.startsWith('.env') ||
    lower.includes('secret') ||
    lower.includes('credential')
  ) {
    throw new Error(`Forbidden generated output path class: ${normalized}`);
  }

  return normalized;
}

function isForbiddenPath(repoPath) {
  const lower = repoPath.toLowerCase();

  return (
    lower.includes('/.git/') ||
    lower.startsWith('.git/') ||
    lower.includes('/node_modules/') ||
    lower.startsWith('node_modules/') ||
    lower.includes('/dist/') ||
    lower.endsWith('/dist') ||
    lower.includes('/build/') ||
    lower.endsWith('/build') ||
    lower.includes('.local.') ||
    lower.includes('/.env') ||
    lower.startsWith('.env') ||
    lower.includes('secret') ||
    lower.includes('credential')
  );
}

function parseManifest(text) {
  const manifest = {
    sources: [],
    exclude_patterns: [],
  };
  let section = null;
  let currentSource = null;

  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine.replace(/\s+#.*$/, '');

    if (/^\s*$/.test(line)) {
      continue;
    }

    const topLevel = line.match(/^([A-Za-z_][A-Za-z0-9_]*):\s*(.*)$/);

    if (topLevel) {
      const [, key, value] = topLevel;
      section = key;
      currentSource = null;

      if (value !== '') {
        manifest[key] = parseScalar(value);
      }

      continue;
    }

    if (section === 'sources') {
      const sourceStart = line.match(/^ {2}- path:\s*(.+)$/);

      if (sourceStart) {
        currentSource = { path: parseScalar(sourceStart[1]) };
        manifest.sources.push(currentSource);
        continue;
      }

      const sourceField = line.match(/^ {4}([A-Za-z_][A-Za-z0-9_]*):\s*(.*)$/);

      if (sourceField && currentSource) {
        const [, key, value] = sourceField;
        currentSource[key] = parseScalar(value);
        continue;
      }
    }

    if (section === 'exclude_patterns') {
      const exclude = line.match(/^ {2}-\s*(.+)$/);

      if (exclude) {
        manifest.exclude_patterns.push(parseScalar(exclude[1]));
      }
    }
  }

  return manifest;
}

function parseScalar(value) {
  const trimmed = value.trim();

  if (trimmed === 'true') {
    return true;
  }

  if (trimmed === 'false') {
    return false;
  }

  if (
    (trimmed.startsWith('"') && trimmed.endsWith('"')) ||
    (trimmed.startsWith("'") && trimmed.endsWith("'"))
  ) {
    return trimmed.slice(1, -1);
  }

  return trimmed;
}

function validateManifest(manifest, manifestPath) {
  if (!manifest.schema_version) {
    throw new Error(`${manifestPath} is missing schema_version.`);
  }

  if (!Array.isArray(manifest.sources) || manifest.sources.length === 0) {
    throw new Error(`${manifestPath} must include a non-empty sources list.`);
  }

  if (!Array.isArray(manifest.exclude_patterns)) {
    throw new Error(`${manifestPath} must include exclude_patterns.`);
  }
}

function validateSourceRow(source) {
  if (!source.path) {
    throw new Error('Manifest source is missing path.');
  }

  if (source.public !== true) {
    throw new Error(`Manifest source must be explicitly public: ${source.path}`);
  }

  if (!source.title) {
    throw new Error(`Manifest source is missing title: ${source.path}`);
  }

  if (source.language !== 'en') {
    throw new Error(`Manifest source must declare language: en: ${source.path}`);
  }

  normalizeRepoPath(source.path);
}

function splitMarkdown(text, source) {
  const withoutFrontmatter = stripFrontmatter(text);
  const lines = withoutFrontmatter.split(/\r?\n/);
  const chunks = [];
  const headingStack = [];
  let currentHeading = 'Overview';
  let currentLines = [];

  const flush = () => {
    const body = currentLines.join('\n').trim();

    if (!body) {
      return;
    }

    for (const chunkText of splitLongText(body)) {
      chunks.push({
        id: makeChunkId(source.path, currentHeading, chunks.length + 1),
        path: source.path,
        title: source.title,
        heading: currentHeading,
        priority: source.priority ?? 'reference',
        commit_sha: source.commitSha,
        updated_at: source.updatedAt,
        public: true,
        text: chunkText,
      });
    }
  };

  for (const line of lines) {
    const headingMatch = line.match(/^(#{1,4})\s+(.+?)\s*#*$/);

    if (headingMatch) {
      flush();
      const level = headingMatch[1].length;
      const title = headingMatch[2].trim();
      headingStack.splice(level - 1);
      headingStack[level - 1] = title;
      currentHeading = headingStack.filter(Boolean).join(' > ');
      currentLines = [line];
      continue;
    }

    currentLines.push(line);
  }

  flush();
  return chunks;
}

function stripFrontmatter(text) {
  if (!text.startsWith('---\n')) {
    return text;
  }

  const end = text.indexOf('\n---\n', 4);

  if (end === -1) {
    return text;
  }

  return text.slice(end + 5);
}

function splitLongText(text) {
  const maxChars = 6000;

  if (text.length <= maxChars) {
    return [text];
  }

  const paragraphs = text.split(/\n{2,}/);
  const chunks = [];
  let current = '';

  for (const paragraph of paragraphs) {
    const candidate = current ? `${current}\n\n${paragraph}` : paragraph;

    if (candidate.length > maxChars && current) {
      chunks.push(current.trim());
      current = paragraph;
      continue;
    }

    current = candidate;
  }

  if (current.trim()) {
    chunks.push(current.trim());
  }

  return chunks;
}

function makeChunkId(sourcePath, heading, offset) {
  const base = `${sourcePath}-${heading}-${offset}`
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');

  return base || `chunk-${offset}`;
}

function ensureUniqueChunkIds(chunks) {
  const seen = new Set();

  for (const chunk of chunks) {
    if (seen.has(chunk.id)) {
      throw new Error(`Duplicate chunk id: ${chunk.id}`);
    }

    seen.add(chunk.id);
  }
}

function renderPublicDocument(chunks, metadata) {
  const lines = [
    '# Goalrail Public Start Assistant Knowledge Base',
    '',
    'Generated artifact for Goalrail `/start` assistant retrieval.',
    'GitHub remains the source of truth. This file is not canonical source material.',
    '',
    `Repository: ${metadata.repository}`,
    `Commit: ${metadata.commitSha}`,
    `Updated at: ${metadata.updatedAt}`,
    `Source manifest: ${metadata.sourceManifestPath}`,
    `Source manifest SHA-256: ${metadata.sourceManifestSha}`,
    '',
  ];

  for (const chunk of chunks) {
    lines.push(`## ${chunk.title}`);
    lines.push('');
    lines.push(`Path: ${chunk.path}`);
    lines.push(`Heading: ${chunk.heading}`);
    lines.push(`Priority: ${chunk.priority}`);
    lines.push('');
    lines.push(chunk.text);
    lines.push('');
  }

  return `${lines.join('\n').trim()}\n`;
}

function scanForSecretMarkers(text, label) {
  const markers = [
    [/-----BEGIN [A-Z ]*PRIVATE KEY-----/, 'private key block'],
    [/\bsk-[A-Za-z0-9_-]{20,}\b/, 'OpenAI-style API key'],
    [/\bghp_[A-Za-z0-9_]{20,}\b/, 'GitHub personal access token'],
    [/\bgithub_pat_[A-Za-z0-9_]{20,}\b/, 'GitHub fine-grained token'],
    [/\bxox[baprs]-[A-Za-z0-9-]{10,}\b/, 'Slack token'],
    [/\b[A-Z0-9_]*API_KEY\s*=\s*['"]?[A-Za-z0-9_-]{12,}/, 'API key assignment'],
  ];

  for (const [pattern, description] of markers) {
    if (pattern.test(text)) {
      throw new Error(`Potential ${description} found in ${label}.`);
    }
  }
}

function scanForCyrillic(text, label) {
  if (/[А-Яа-яЁё]/.test(text)) {
    throw new Error(`Cyrillic text found in English public KB source: ${label}.`);
  }
}

function sha256(text) {
  return createHash('sha256').update(text).digest('hex');
}

function readGit(args) {
  try {
    return execFileSync('git', args, {
      cwd: repoRoot,
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'ignore'],
    }).trim();
  } catch {
    return null;
  }
}

try {
  main();
} catch (error) {
  console.error(error instanceof Error ? error.message : String(error));
  process.exit(1);
}
