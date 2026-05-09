import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

const startAssistantProxyTarget = process.env.START_ASSISTANT_PROXY_TARGET || 'https://goalrail.dev';

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      input: {
        main: fileURLToPath(new URL('./index.html', import.meta.url)),
        start: fileURLToPath(new URL('./start/index.html', import.meta.url)),
      },
    },
  },
  server: {
    proxy: {
      '/api/start-chat': {
        target: startAssistantProxyTarget,
        changeOrigin: true,
      },
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './vitest.setup.ts',
    css: true,
  },
});
