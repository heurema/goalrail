const MAX_BODY_BYTES = 8192;
const MAX_QUESTION_LENGTH = 1000;
const DEFAULT_TIMEOUT_MS = 12000;
const DEFAULT_MAX_RESULTS = 8;
const DISCLAIMER = 'Answers use public Goalrail materials. This page cannot scan repos or execute code.';

const SECURITY_SOURCES = [
  {
    title: 'Start Assistant Security and Privacy Boundary',
    path: 'docs/ops/START_ASSISTANT_SECURITY_AND_PRIVACY.md',
    section: 'Hard boundaries',
  },
];

const DEFAULT_SOURCES = [
  {
    title: 'Goalrail Global Start Assistant',
    path: 'docs/product/GOALRAIL_GLOBAL_START_ASSISTANT.md',
    section: 'Assistant behavior',
  },
];

const SYSTEM_INSTRUCTIONS = [
  'You are the public Goalrail /start assistant.',
  'Answer only from approved public Goalrail knowledge sources retrieved through file_search.',
  'If the sources do not answer the question, say that the public knowledge base does not answer it yet.',
  'Do not invent product maturity, repo scanning, code execution, autonomous delivery, customer results, or integrations.',
  'Keep answers short, concrete, and calm.',
  'Prefer Goalrail terms: goal intake, clarification, contract, bounded execution, checks, proof, approval, repo readiness, AI delivery drift.',
  'Do not hard sell and do not ask the user to book a demo.',
].join('\n');

export default {
  fetch(request, env, ctx) {
    return handleStartAssistantRequest(request, env, { ctx });
  },
};

export async function handleStartAssistantRequest(request, env = {}, runtime = {}) {
  const url = new URL(request.url);
  if (url.pathname !== '/api/start-chat') {
    return jsonError('not_found', 'Use POST /api/start-chat.', 404);
  }

  if (request.method !== 'POST') {
    return jsonError('method_not_allowed', 'Use POST with an application/json body.', 405, { Allow: 'POST' });
  }

  const contentType = request.headers.get('content-type') || '';
  if (!contentType.toLowerCase().includes('application/json')) {
    return jsonError('unsupported_media_type', 'Content-Type must be application/json.', 415);
  }

  const contentLength = Number(request.headers.get('content-length') || '0');
  if (Number.isFinite(contentLength) && contentLength > MAX_BODY_BYTES) {
    return jsonError('invalid_request', 'Request body is too large.', 413);
  }

  const bodyRead = await readLimitedBody(request, MAX_BODY_BYTES);
  if (!bodyRead.ok) {
    return jsonError('invalid_request', bodyRead.message, bodyRead.status);
  }

  const bodyText = bodyRead.text;
  let payload;
  try {
    payload = JSON.parse(bodyText);
  } catch {
    return jsonError('invalid_request', 'Request body must be valid JSON.', 400);
  }

  const validation = validatePayload(payload);
  if (!validation.ok) {
    return jsonError('invalid_request', validation.message, 400);
  }

  const question = validation.question;
  const refusal = refusalForQuestion(question);
  if (refusal) {
    return jsonResponse(answerEnvelope(refusal.answer, SECURITY_SOURCES, refusal.suggested_questions, env), 200);
  }

  if (env.START_ASSISTANT_PROVIDER_MODE === 'mock') {
    return jsonResponse(answerEnvelope(mockAnswer(question), DEFAULT_SOURCES, suggestedQuestions(question), env), 200);
  }

  const providerConfig = readProviderConfig(env);
  if (!providerConfig.ok) {
    return unavailable(providerConfig.message);
  }

  try {
    const openaiResponse = await callOpenAIResponses(question, providerConfig, runtime);
    const shaped = shapeOpenAIResponse(openaiResponse, question);
    if (!shaped.answer) {
      return unavailable('The assistant did not return an answer.');
    }
    if (shaped.sources.length === 0) {
      return unavailable('The assistant did not return a sourced answer.');
    }

    return jsonResponse(answerEnvelope(shaped.answer, shaped.sources, shaped.suggested_questions, env), 200);
  } catch {
    return unavailable('The public Goalrail assistant is temporarily unavailable. Static overview and artifacts are still available.');
  }
}

export async function callOpenAIResponses(question, config, runtime = {}) {
  const fetchImpl = runtime.fetch || fetch;
  const controller = new AbortController();
  const setTimer = runtime.setTimeout || setTimeout;
  const clearTimer = runtime.clearTimeout || clearTimeout;
  const timer = setTimer(() => controller.abort(), config.timeoutMs);

  const body = {
    model: config.model,
    instructions: SYSTEM_INSTRUCTIONS,
    input: [
      {
        role: 'user',
        content: [{ type: 'input_text', text: question }],
      },
    ],
    tools: [
      {
        type: 'file_search',
        vector_store_ids: [config.vectorStoreId],
        max_num_results: config.maxResults,
      },
    ],
    tool_choice: 'required',
    max_output_tokens: 450,
    store: false,
    metadata: {
      surface: 'goalrail-start',
      kb_revision: config.kbRevision || 'unknown',
    },
  };

  try {
    const response = await fetchImpl('https://api.openai.com/v1/responses', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${config.apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
      signal: controller.signal,
    });

    if (!response.ok) {
      throw new Error(`OpenAI request failed with status ${response.status}`);
    }

    return response.json();
  } finally {
    clearTimer(timer);
  }
}

export function validatePayload(payload) {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
    return { ok: false, message: 'Question must be a non-empty string under 1000 characters.' };
  }

  if (typeof payload.question !== 'string') {
    return { ok: false, message: 'Question must be a non-empty string under 1000 characters.' };
  }

  const question = payload.question.trim();
  if (question.length === 0 || question.length > MAX_QUESTION_LENGTH) {
    return { ok: false, message: 'Question must be a non-empty string under 1000 characters.' };
  }

  return { ok: true, question };
}

export function shapeOpenAIResponse(response, question) {
  const output = Array.isArray(response?.output) ? response.output : [];
  const textParts = [];
  const sources = [];

  for (const item of output) {
    if (item?.type !== 'message' || !Array.isArray(item.content)) {
      continue;
    }

    for (const part of item.content) {
      if (part?.type !== 'output_text' || typeof part.text !== 'string') {
        continue;
      }

      textParts.push(part.text.trim());

      if (Array.isArray(part.annotations)) {
        for (const annotation of part.annotations) {
          const source = sourceFromAnnotation(annotation);
          if (source) {
            sources.push(source);
          }
        }
      }
    }
  }

  return {
    answer: textParts.filter(Boolean).join('\n\n').trim(),
    sources: dedupeSources(sources),
    suggested_questions: suggestedQuestions(question),
  };
}

function readProviderConfig(env) {
  const apiKey = stringValue(env.OPENAI_API_KEY);
  const model = stringValue(env.OPENAI_START_MODEL);
  const vectorStoreId = stringValue(env.OPENAI_START_VECTOR_STORE_ID);

  if (!apiKey || !model || !vectorStoreId) {
    return {
      ok: false,
      message: 'The public Goalrail assistant is temporarily unavailable. Static overview and artifacts are still available.',
    };
  }

  return {
    ok: true,
    apiKey,
    model,
    vectorStoreId,
    kbRevision: stringValue(env.START_ASSISTANT_KB_REVISION),
    timeoutMs: positiveInteger(env.START_ASSISTANT_PROVIDER_TIMEOUT_MS, DEFAULT_TIMEOUT_MS),
    maxResults: positiveInteger(env.START_ASSISTANT_MAX_RESULTS, DEFAULT_MAX_RESULTS),
  };
}

function answerEnvelope(answer, sources, suggestedQuestionsValue, env) {
  return {
    answer,
    sources: dedupeSources(sources),
    suggested_questions: suggestedQuestionsValue,
    knowledge: {
      updated_at: stringValue(env.START_ASSISTANT_KB_UPDATED_AT) || null,
      commit_sha: stringValue(env.START_ASSISTANT_KB_REVISION) || null,
    },
    disclaimer: DISCLAIMER,
  };
}

function unavailable(message) {
  return jsonError('assistant_unavailable', message, 503);
}

function containsCredentialMaterial(question) {
  return [
    /-----BEGIN [A-Z ]*PRIVATE KEY-----/i,
    /\bsk-(?:proj-|svcacct-)?[A-Za-z0-9_-]{16,}\b/,
    /\bgithub_pat_[A-Za-z0-9_]{20,}\b/,
    /\bgh[pousr]_[A-Za-z0-9_]{20,}\b/,
    /\bxox[baprs]-[A-Za-z0-9-]{20,}\b/,
    /\b(?:AKIA|ASIA)[A-Z0-9]{16}\b/,
    /\bAIza[0-9A-Za-z_-]{25,}\b/,
    /\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b/,
    /\bAuthorization\s*:\s*Bearer\s+[A-Za-z0-9._~+/=-]{12,}/i,
    /\bBearer\s+[A-Za-z0-9._~+/=-]{30,}\b/i,
    /\b(?:API_KEY|OPENAI_API_KEY|TOKEN|ACCESS_TOKEN|SECRET|PASSWORD|PRIVATE_KEY|CLIENT_SECRET)\s*[:=]\s*["']?[A-Za-z0-9._~+/=-]{12,}/i,
  ].some((pattern) => pattern.test(question));
}

function containsCredentialSharingIntent(question) {
  return [
    /\b(?:here is|this is|my|our|pasted|paste|leaked|exposed|shared|sending|send)\b.{0,60}\b(?:api[_\s-]?key|secret|password|credential|private token|access token|bearer token)\b/i,
    /\b(?:api[_\s-]?key|secret|password|credential|private token|access token|bearer token)\b.{0,30}\b(?:is|=|:)\b/i,
    /-----begin/i,
  ].some((pattern) => pattern.test(question));
}

function containsPastedCode(question) {
  return [
    /```|~~~/,
    /^#!\//m,
    /^\s*#include\s+</m,
    /^\s*(?:const|let|var)\s+[A-Za-z_$][\w$]*\s*=/m,
    /^\s*function\s+[A-Za-z_$][\w$]*\s*\(/m,
    /^\s*class\s+[A-Za-z_$][\w$]*(?:\s+extends|\s*\{)/m,
    /^\s*(?:import|export)\s+.+\s+from\s+['"][^'"]+['"]/m,
    /^\s*def\s+[A-Za-z_]\w*\s*\(/m,
    /^\s*package\s+main\b/m,
    /^\s*func\s+[A-Za-z_]\w*\s*\(/m,
    /^\s*(?:if|for|while|switch)\s*\(.+\)\s*\{/m,
    /^\s*SELECT\b[\s\S]{0,160}\bFROM\b/m,
  ].some((pattern) => pattern.test(question));
}

function containsFileUploadIntent(question) {
  return [
    /\b(?:upload|attach|send|drop|submit|share)\b.{0,40}\b(?:file|files|zip|archive|pdf|docx?|spreadsheet|screenshot|folder)\b/i,
    /\b(?:file|zip|archive|pdf|docx?|spreadsheet|screenshot|folder)\b.{0,40}\b(?:upload|attachment|attach|send|submit)\b/i,
    /\bcan i upload\b/i,
    /\bfile upload\b/i,
  ].some((pattern) => pattern.test(question));
}

function refusalForQuestion(question) {
  const normalized = question.toLowerCase();

  if (
    containsCredentialMaterial(question) ||
    containsCredentialSharingIntent(question)
  ) {
    return {
      answer:
        'Please do not share secrets here. This public assistant cannot process private credentials. Rotate any secret that may have been exposed.',
      suggested_questions: ['What is Goalrail?', 'What does proof before approval mean?'],
    };
  }

  if (containsPastedCode(question)) {
    return {
      answer:
        'I cannot process pasted code snippets from this public page. Describe the workflow, boundary, check, or review problem without including private code.',
      suggested_questions: ['Is my repo ready for coding agents?', 'How should a team review AI-generated changes?'],
    };
  }

  if (containsFileUploadIntent(question)) {
    return {
      answer:
        'I cannot accept file uploads from this public page. Describe the workflow or review problem without attaching files, private code, secrets, or customer data.',
      suggested_questions: ['What would a pilot fit check look like?', 'Is my repo ready for coding agents?'],
    };
  }

  if (
    /(scan|analyze|inspect|clone|connect).{0,40}(my|our|this).{0,40}(repo|repository|github|codebase)/i.test(question) ||
    /(private repo|private repository|repo url|repository url|github\.com\/)/i.test(question)
  ) {
    return {
      answer:
        'I cannot scan repositories from this page. For a pilot fit check, describe your team size, current AI tools, repo or workflow shape, and one review or proof problem you are seeing.',
      suggested_questions: ['Is my repo ready for coding agents?', 'What would a pilot fit check look like?'],
    };
  }

  if (
    /(run|execute).{0,40}(code|script|test|command|npm|pytest|cargo|go test)/i.test(question) ||
    /(can you run|please run|execute this|run this)/i.test(normalized)
  ) {
    return {
      answer:
        'I cannot execute code from this page. I can explain how Goalrail thinks about checks, proof, and approval boundaries using public materials.',
      suggested_questions: ['What does proof before approval mean?', 'How should a team review AI-generated changes?'],
    };
  }

  return null;
}

function mockAnswer(question) {
  if (/proof/i.test(question)) {
    return 'Proof before approval means reviewers compare the contract, diff, checks, artifacts, and remaining risk before accepting AI-assisted work.';
  }

  if (/repo|repository/i.test(question)) {
    return 'Repo readiness means the repository exposes enough working signals for an agent to operate safely: run commands, checks, ownership, boundaries, proof expectations, and rollback paths.';
  }

  return 'Goalrail is a control layer for AI-assisted software delivery: from business goal to verified code change through contracts, checks, proof, and human approval.';
}

function suggestedQuestions(question) {
  if (/proof|approval/i.test(question)) {
    return ['What is contract-first execution?', 'How should a team review AI-generated changes?'];
  }

  if (/repo|repository|readiness/i.test(question)) {
    return ['What is AI delivery drift?', 'What does proof before approval mean?'];
  }

  return ['What is contract-first execution?', 'What would a pilot fit check look like?'];
}

function sourceFromAnnotation(annotation) {
  if (annotation?.type === 'file_citation') {
    const filename = stringValue(annotation.filename) || 'Public Goalrail knowledge file';
    return {
      title: filename,
      path: filename,
      section: null,
    };
  }

  return null;
}

function dedupeSources(sources) {
  const seen = new Set();
  const deduped = [];

  for (const source of sources) {
    const key = `${source.title}|${source.path}|${source.section || ''}`;
    if (seen.has(key)) {
      continue;
    }

    seen.add(key);
    deduped.push(source);
  }

  return deduped;
}

function jsonError(error, message, status, headers = {}) {
  return jsonResponse({ error, message }, status, headers);
}

function jsonResponse(body, status = 200, headers = {}) {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
      'Cache-Control': 'no-store',
      'X-Content-Type-Options': 'nosniff',
      'Referrer-Policy': 'no-referrer',
      ...headers,
    },
  });
}

function stringValue(value) {
  return typeof value === 'string' && value.trim() ? value.trim() : '';
}

function positiveInteger(value, fallback) {
  const parsed = Number(value);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback;
}

async function readLimitedBody(request, maxBytes) {
  if (!request.body) {
    return { ok: true, text: '' };
  }

  const reader = request.body.getReader();
  const decoder = new TextDecoder();
  let bytesRead = 0;
  let text = '';

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }

      bytesRead += value.byteLength;
      if (bytesRead > maxBytes) {
        try {
          await reader.cancel();
        } catch {
          // Best-effort cancellation after the hard byte cap is reached.
        }

        return { ok: false, status: 413, message: 'Request body is too large.' };
      }

      text += decoder.decode(value, { stream: true });
    }

    text += decoder.decode();
  } catch {
    return { ok: false, status: 400, message: 'Could not read request body.' };
  } finally {
    try {
      reader.releaseLock();
    } catch {
      // Some runtimes release the lock when the stream closes or is cancelled.
    }
  }

  return { ok: true, text };
}
