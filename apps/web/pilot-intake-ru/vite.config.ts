import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

const startAssistantProxyTarget = process.env.START_ASSISTANT_PROXY_TARGET || 'https://goalrail.dev';

export default defineConfig({
  plugins: [react()],
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
