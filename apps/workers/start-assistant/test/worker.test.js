import assert from 'node:assert/strict';
import test from 'node:test';

import { handleStartAssistantRequest } from '../src/worker.js';

const MOCK_ENV = {
  START_ASSISTANT_PROVIDER_MODE: 'mock',
  START_ASSISTANT_KB_UPDATED_AT: '2026-05-07T12:00:00Z',
  START_ASSISTANT_KB_REVISION: 'abc123',
};

function request(body, init = {}) {
  return new Request('https://goalrail.dev/api/start-chat', {
    method: init.method || 'POST',
    headers: init.headers || { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : typeof body === 'string' ? body : JSON.stringify(body),
  });
}

async function json(response) {
  return response.json();
}

test('returns a shaped mock answer for a valid public question', async () => {
  const response = await handleStartAssistantRequest(request({ question: 'What is Goalrail?' }), MOCK_ENV);
  const body = await json(response);

  assert.equal(response.status, 200);
  assert.match(body.answer, /control layer/i);
  assert.equal(body.knowledge.updated_at, '2026-05-07T12:00:00Z');
  assert.equal(body.knowledge.commit_sha, 'abc123');
  assert.equal(body.disclaimer, 'Answers use public Goalrail materials. This page cannot scan repos or execute code.');
  assert.ok(body.sources.length > 0);
  assert.ok(body.suggested_questions.length > 0);
});

test('rejects unsupported methods', async () => {
  const response = await handleStartAssistantRequest(request(undefined, { method: 'GET' }), MOCK_ENV);
  const body = await json(response);

  assert.equal(response.status, 405);
  assert.equal(response.headers.get('allow'), 'POST');
  assert.equal(body.error, 'method_not_allowed');
});

test('rejects off-path requests', async () => {
  const response = await handleStartAssistantRequest(
    new Request('https://goalrail.dev/api/other', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ question: 'What is Goalrail?' }),
    }),
    MOCK_ENV
  );
  const body = await json(response);

  assert.equal(response.status, 404);
  assert.equal(body.error, 'not_found');
});

test('rejects non-json content', async () => {
  const response = await handleStartAssistantRequest(
    request('question=hello', { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } }),
    MOCK_ENV
  );
  const body = await json(response);

  assert.equal(response.status, 415);
  assert.equal(body.error, 'unsupported_media_type');
});

test('rejects multipart uploads', async () => {
  const response = await handleStartAssistantRequest(
    request('fake multipart body', { headers: { 'Content-Type': 'multipart/form-data; boundary=x' } }),
    MOCK_ENV
  );
  const body = await json(response);

  assert.equal(response.status, 415);
  assert.equal(body.error, 'unsupported_media_type');
});

test('rejects empty questions', async () => {
  const response = await handleStartAssistantRequest(request({ question: '   ' }), MOCK_ENV);
  const body = await json(response);

  assert.equal(response.status, 400);
  assert.equal(body.error, 'invalid_request');
});

test('rejects over-limit questions', async () => {
  const response = await handleStartAssistantRequest(request({ question: 'x'.repeat(1001) }), MOCK_ENV);
  const body = await json(response);

  assert.equal(response.status, 400);
  assert.equal(body.error, 'invalid_request');
});

test('rejects streaming request bodies once the byte limit is exceeded', async () => {
  const encoder = new TextEncoder();
  let pulls = 0;
  const stream = new ReadableStream({
    pull(controller) {
      pulls += 1;
      controller.enqueue(encoder.encode('x'.repeat(9000)));
      controller.close();
    },
  });

  const response = await handleStartAssistantRequest(
    new Request('https://goalrail.dev/api/start-chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: stream,
      duplex: 'half',
    }),
    MOCK_ENV
  );
  const body = await json(response);

  assert.equal(response.status, 413);
  assert.equal(body.error, 'invalid_request');
  assert.equal(pulls, 1);
});

test('returns unavailable when provider config is missing', async () => {
  const response = await handleStartAssistantRequest(request({ question: 'What is Goalrail?' }), {});
  const body = await json(response);

  assert.equal(response.status, 503);
  assert.equal(body.error, 'assistant_unavailable');
});

test('refuses repo scan requests before provider calls', async () => {
  let called = false;
  const response = await handleStartAssistantRequest(
    request({ question: 'Can you scan my private repo?' }),
    {
      OPENAI_API_KEY: 'test-key',
      OPENAI_START_MODEL: 'test-model',
      OPENAI_START_VECTOR_STORE_ID: 'vs_test',
    },
    {
      fetch: async () => {
        called = true;
        throw new Error('should not call provider');
      },
    }
  );
  const body = await json(response);

  assert.equal(response.status, 200);
  assert.equal(called, false);
  assert.match(body.answer, /cannot scan repositories/i);
});

test('refuses code execution requests before provider calls', async () => {
  let called = false;
  const response = await handleStartAssistantRequest(
    request({ question: 'Please run this code.' }),
    {
      OPENAI_API_KEY: 'test-key',
      OPENAI_START_MODEL: 'test-model',
      OPENAI_START_VECTOR_STORE_ID: 'vs_test',
    },
    {
      fetch: async () => {
        called = true;
        throw new Error('should not call provider');
      },
    }
  );
  const body = await json(response);

  assert.equal(response.status, 200);
  assert.equal(called, false);
  assert.match(body.answer, /cannot execute code/i);
});

test('refuses secret sharing requests before provider calls', async () => {
  let called = false;
  const response = await handleStartAssistantRequest(
    request({ question: 'Here is my API key.' }),
    {
      OPENAI_API_KEY: 'test-key',
      OPENAI_START_MODEL: 'test-model',
      OPENAI_START_VECTOR_STORE_ID: 'vs_test',
    },
    {
      fetch: async () => {
        called = true;
        throw new Error('should not call provider');
      },
    }
  );
  const body = await json(response);

  assert.equal(response.status, 200);
  assert.equal(called, false);
  assert.match(body.answer, /do not share secrets/i);
});

test('allows benign secret-boundary questions without credential material', async () => {
  const response = await handleStartAssistantRequest(
    request({ question: 'How should teams think about secret risks in AI-assisted delivery?' }),
    MOCK_ENV
  );
  const body = await json(response);

  assert.equal(response.status, 200);
  assert.doesNotMatch(body.answer, /do not share secrets/i);
});

test('refuses raw credential material before provider calls', async () => {
  let called = false;
  const response = await handleStartAssistantRequest(
    request({ question: 'sk-proj-abcdefghijklmnopqrstuvwxyz1234567890' }),
    {
      OPENAI_API_KEY: 'test-key',
      OPENAI_START_MODEL: 'test-model',
      OPENAI_START_VECTOR_STORE_ID: 'vs_test',
    },
    {
      fetch: async () => {
        called = true;
        throw new Error('should not call provider');
      },
    }
  );
  const body = await json(response);

  assert.equal(response.status, 200);
  assert.equal(called, false);
  assert.match(body.answer, /do not share secrets/i);
});

test('refuses pasted code snippets before provider calls', async () => {
  let called = false;
  const response = await handleStartAssistantRequest(
    request({
      question: `Can you review this?
const answer = 42;
function calculate() {
  return answer;
}`,
    }),
    {
      OPENAI_API_KEY: 'test-key',
      OPENAI_START_MODEL: 'test-model',
      OPENAI_START_VECTOR_STORE_ID: 'vs_test',
    },
    {
      fetch: async () => {
        called = true;
        throw new Error('should not call provider');
      },
    }
  );
  const body = await json(response);

  assert.equal(response.status, 200);
  assert.equal(called, false);
  assert.match(body.answer, /cannot process pasted code snippets/i);
});

test('allows natural language select from phrasing', async () => {
  const response = await handleStartAssistantRequest(
    request({ question: 'How should I select from AI IDE options without losing delivery control?' }),
    MOCK_ENV
  );
  const body = await json(response);

  assert.equal(response.status, 200);
  assert.doesNotMatch(body.answer, /cannot process pasted code snippets/i);
});

test('refuses file upload prompts before provider calls', async () => {
  let called = false;
  const response = await handleStartAssistantRequest(
    request({ question: 'Can I upload a zip file here?' }),
    {
      OPENAI_API_KEY: 'test-key',
      OPENAI_START_MODEL: 'test-model',
      OPENAI_START_VECTOR_STORE_ID: 'vs_test',
    },
    {
      fetch: async () => {
        called = true;
        throw new Error('should not call provider');
      },
    }
  );
  const body = await json(response);

  assert.equal(response.status, 200);
  assert.equal(called, false);
  assert.match(body.answer, /cannot accept file uploads/i);
});

test('calls OpenAI Responses API with file_search and shapes citations', async () => {
  let captured;
  const response = await handleStartAssistantRequest(
    request({ question: 'What is proof before approval?' }),
    {
      OPENAI_API_KEY: 'test-key',
      OPENAI_START_MODEL: 'test-model',
      OPENAI_START_VECTOR_STORE_ID: 'vs_test',
      START_ASSISTANT_KB_REVISION: 'def456',
      START_ASSISTANT_KB_UPDATED_AT: '2026-05-07T12:30:00Z',
    },
    {
      fetch: async (url, init) => {
        captured = { url, init, body: JSON.parse(init.body) };
        return new Response(
          JSON.stringify({
            output: [
              {
                type: 'message',
                content: [
                  {
                    type: 'output_text',
                    text: 'Proof before approval means checking evidence before approval.',
                    annotations: [
                      {
                        type: 'file_citation',
                        filename: 'docs/product/GOALRAIL_GLOBAL_START_ASSISTANT.md',
                      },
                    ],
                  },
                ],
              },
            ],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        );
      },
    }
  );
  const body = await json(response);

  assert.equal(response.status, 200);
  assert.equal(captured.url, 'https://api.openai.com/v1/responses');
  assert.equal(captured.init.headers.Authorization, 'Bearer test-key');
  assert.equal(captured.body.model, 'test-model');
  assert.equal(captured.body.tool_choice, 'required');
  assert.deepEqual(captured.body.tools, [
    {
      type: 'file_search',
      vector_store_ids: ['vs_test'],
      max_num_results: 8,
    },
  ]);
  assert.equal(captured.body.store, false);
  assert.doesNotMatch(JSON.stringify(body), /test-key/);
  assert.match(body.answer, /checking evidence/i);
  assert.deepEqual(body.sources, [
    {
      title: 'docs/product/GOALRAIL_GLOBAL_START_ASSISTANT.md',
      path: 'docs/product/GOALRAIL_GLOBAL_START_ASSISTANT.md',
      section: null,
    },
  ]);
});

test('returns unavailable when provider answer has no file citations', async () => {
  const response = await handleStartAssistantRequest(
    request({ question: 'What is proof before approval?' }),
    {
      OPENAI_API_KEY: 'test-key',
      OPENAI_START_MODEL: 'test-model',
      OPENAI_START_VECTOR_STORE_ID: 'vs_test',
    },
    {
      fetch: async () =>
        new Response(
          JSON.stringify({
            output: [
              {
                type: 'message',
                content: [
                  {
                    type: 'output_text',
                    text: 'Proof before approval means checking evidence before approval.',
                  },
                ],
              },
            ],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        ),
    }
  );
  const body = await json(response);

  assert.equal(response.status, 503);
  assert.equal(body.error, 'assistant_unavailable');
  assert.match(body.message, /sourced answer/i);
});

test('returns unavailable on provider failure', async () => {
  const response = await handleStartAssistantRequest(
    request({ question: 'What is Goalrail?' }),
    {
      OPENAI_API_KEY: 'test-key',
      OPENAI_START_MODEL: 'test-model',
      OPENAI_START_VECTOR_STORE_ID: 'vs_test',
    },
    {
      fetch: async () => new Response(JSON.stringify({ error: 'failed' }), { status: 500 }),
    }
  );
  const body = await json(response);

  assert.equal(response.status, 503);
  assert.equal(body.error, 'assistant_unavailable');
});
