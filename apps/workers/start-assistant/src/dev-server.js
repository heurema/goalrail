import http from 'node:http';

import { handleStartAssistantRequest } from './worker.js';

const host = process.env.HOST || '127.0.0.1';
const port = Number(process.env.PORT || process.env.START_ASSISTANT_PORT || 8787);

const server = http.createServer(async (nodeRequest, nodeResponse) => {
  const chunks = [];

  for await (const chunk of nodeRequest) {
    chunks.push(chunk);
  }

  const body = chunks.length > 0 ? Buffer.concat(chunks) : undefined;
  const headers = new Headers();

  for (const [name, value] of Object.entries(nodeRequest.headers)) {
    if (Array.isArray(value)) {
      headers.set(name, value.join(', '));
    } else if (value !== undefined) {
      headers.set(name, value);
    }
  }

  const request = new Request(`http://${host}:${port}${nodeRequest.url || '/'}`, {
    method: nodeRequest.method,
    headers,
    body: body && body.length > 0 ? body : undefined,
  });

  const env = {
    ...process.env,
    START_ASSISTANT_PROVIDER_MODE:
      process.env.START_ASSISTANT_PROVIDER_MODE ||
      (process.env.OPENAI_API_KEY && process.env.OPENAI_START_MODEL && process.env.OPENAI_START_VECTOR_STORE_ID ? '' : 'mock'),
  };

  const response = await handleStartAssistantRequest(request, env);
  const responseBody = Buffer.from(await response.arrayBuffer());

  nodeResponse.writeHead(response.status, Object.fromEntries(response.headers.entries()));
  nodeResponse.end(responseBody);
});

server.listen(port, host, () => {
  console.log(`start-assistant worker dev server listening on http://${host}:${port}`);
});
