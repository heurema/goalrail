import path from "node:path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import type { ProxyOptions } from "vite";
import { defineConfig } from "vitest/config";

const GOALRAIL_URL = process.env.GOALRAIL_URL ?? "http://localhost:6767";
const authToken = process.env.GOALRAIL_AUTH_TOKEN ?? null;

function configureProxy(target: string, useAuth: boolean): NonNullable<ProxyOptions["configure"]> {
  const parsed = new URL(target);
  // The URL pathname becomes a prefix prepended to every proxied request.
  // e.g. GOALRAIL_URL=https://host.com/api/2.0/goalrail means the browser's
  // /v1/sessions is rewritten to /api/2.0/goalrail/v1/sessions before forwarding.
  const basePath = parsed.pathname.replace(/\/$/, "");

  return (proxy) => {
    proxy.on("proxyReq", (proxyReq) => {
      if (basePath) proxyReq.path = `${basePath}${proxyReq.path}`;
      if (useAuth) {
        if (authToken) proxyReq.setHeader("Authorization", `Bearer ${authToken}`);
      }
    });

    proxy.on("proxyReqWs", (proxyReq) => {
      if (basePath) proxyReq.path = `${basePath}${proxyReq.path}`;
      if (useAuth) {
        if (authToken) proxyReq.setHeader("Authorization", `Bearer ${authToken}`);
      }
    });

    proxy.on("proxyRes", (proxyRes, _req, res) => {
      const contentType = proxyRes.headers["content-type"] ?? "";
      if (typeof contentType === "string" && contentType.includes("text/event-stream")) {
        // http-proxy applies upstream headers after its own proxyRes listener
        // runs. Defer flushing until after those headers have been copied.
        setImmediate(() => res.flushHeaders());
      }
    });
  };
}

function createProxyConfig(target: string, useAuth: boolean): Record<string, ProxyOptions> {
  const origin = new URL(target).origin;
  const configure = configureProxy(target, useAuth);

  return {
    "/v1": {
      target: origin,
      changeOrigin: true,
      ws: true,
      configure,
    },
    "/api": {
      target: origin,
      changeOrigin: true,
      configure,
    },
    "/auth": {
      target: origin,
      changeOrigin: true,
      configure,
    },
    "/health": {
      target: origin,
      changeOrigin: true,
      configure,
    },
  };
}

const useAuth = authToken != null;

if (useAuth) {
  console.log(`[dev-proxy] target=${GOALRAIL_URL} (authenticated)`);
} else {
  console.log(`[dev-proxy] target=${GOALRAIL_URL}`);
}

const proxyConfig = createProxyConfig(GOALRAIL_URL, useAuth);

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["./src/test-setup.ts"],
    // Scope discovery to src/ — the ap-web suite lives there. Without this,
    // vitest's default glob descends into the nested electron package and
    // tries to run its node:test files (which aren't vitest suites).
    include: ["src/**/*.{test,spec}.?(c|m)[jt]s?(x)"],
    coverage: {
      provider: "v8",
      // With `include` set, vitest counts every matching source file (untested
      // ones as 0%), so the total reflects the whole frontend — parity with the
      // backend's --cov=goalrail, not just files a test happened to import.
      include: ["src/**/*.{ts,tsx}"],
      exclude: [
        "src/**/*.test.{ts,tsx}",
        "src/**/*.d.ts",
        "src/test-setup.ts",
        // Vendored UI kit, not product code (see tests/e2e_ui/COVERAGE_GAPS.md).
        "src/components/ai-elements/**",
      ],
      reportsDirectory: "./coverage",
      // text-summary: human-readable console line; json-summary: machine-
      // readable coverage/coverage-summary.json that CI distills to total.txt.
      reporter: ["text-summary", "json-summary"],
    },
  },
  server: {
    proxy: proxyConfig,
  },
  build: {
    outDir: path.resolve(__dirname, "../goalrail/server/static/web-ui"),
    emptyOutDir: true,
  },
});
