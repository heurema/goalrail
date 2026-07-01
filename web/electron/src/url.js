// Shared URL-normalization helpers for the desktop shell.
//
// Loaded by both the Electron main process (`require("./url")` in
// `src/main.js`) and the bundled setup page (`<script src="../src/url.js">` in
// `setup/index.html`, where it publishes `window.goalrailUrl`). One copy keeps
// the two from drifting — the setup page's plain-http warning and the main
// process's navigation must agree on what a bare URL means.
//
// Only web/Node globals (URL, fetch, AbortSignal) are used, so the same source
// runs unchanged under CommonJS (main) and in the renderer (setup page).
(function (root, factory) {
  const api = factory();
  if (typeof module === "object" && module.exports) {
    module.exports = api;
  } else {
    root.goalrailUrl = api;
  }
})(typeof globalThis !== "undefined" ? globalThis : this, function () {
  "use strict";

  /**
   * Hostnames that resolve to the local machine. A schemeless URL defaults to
   * https:// (the workspace / remote case the internal user guide documents),
   * but these default to http:// — local dev servers are virtually always plain
   * http, and the setup placeholder shows http://localhost.
   */
  const LOCAL_HOSTS = new Set(["localhost", "127.0.0.1", "[::1]", "::1"]);

  /**
   * The scheme a schemeless input should default to: http:// for loopback
   * hosts (local dev is plain http), https:// for everything else (the pasted
   * workspace-URL case). Unparseable input falls back to https:// so the
   * caller's own URL parse raises the real error.
   *
   * @param {string} trimmed A trimmed, scheme-less `host[:port][/path]`.
   * @returns {"http" | "https"}
   */
  function defaultSchemeFor(trimmed) {
    let host;
    try {
      host = new URL(`https://${trimmed}`).hostname;
    } catch {
      host = "";
    }
    return LOCAL_HOSTS.has(host) ? "http" : "https";
  }

  /**
   * Normalize a user-entered server URL into something navigable. Accepts a
   * bare `host[:port][/path]` and defaults the scheme (https://, or http:// for
   * loopback hosts), trims whitespace, and rejects anything that isn't an
   * http(s) URL — fail loud rather than navigate to garbage.
   *
   * @param {string} raw
   * @returns {string} A normalized absolute http(s) URL.
   */
  function normalizeUrl(raw) {
    const trimmed = (raw ?? "").trim();
    if (trimmed === "") throw new Error("server URL is empty");
    const withScheme = trimmed.includes("://")
      ? trimmed
      : `${defaultSchemeFor(trimmed)}://${trimmed}`;
    let url;
    try {
      url = new URL(withScheme);
    } catch (e) {
      throw new Error(`invalid URL: ${e.message}`);
    }
    if (url.protocol !== "http:" && url.protocol !== "https:") {
      throw new Error(`unsupported scheme '${url.protocol}' (use http/https)`);
    }
    return url.toString();
  }

  /**
   * True when the entered URL is unencrypted http:// to a non-local host — the
   * setup page warns before connecting. Mirrors normalizeUrl's scheme-
   * defaulting (https:// by default, http:// for loopback), so a bare remote
   * host — now https — does not trip the warning; only an explicit http:// to a
   * remote host does. Invalid URLs return false so the real error comes from
   * normalizeUrl on Connect.
   *
   * @param {string} raw
   * @returns {boolean}
   */
  function isPlainHttpRemote(raw) {
    const trimmed = (raw || "").trim();
    if (trimmed === "") return false;
    const withScheme = trimmed.includes("://")
      ? trimmed
      : `${defaultSchemeFor(trimmed)}://${trimmed}`;
    let url;
    try {
      url = new URL(withScheme);
    } catch {
      return false;
    }
    return url.protocol === "http:" && !LOCAL_HOSTS.has(url.hostname);
  }

  return {
    LOCAL_HOSTS,
    defaultSchemeFor,
    normalizeUrl,
    isPlainHttpRemote,
  };
});
