// Tests for the shared desktop URL helpers (src/url.js), run with
// `node --test` (no extra deps). Covers the scheme-defaulting that lets a
// pasted schemeless remote URL connect and the plain-http warning.

const { describe, it } = require("node:test");
const assert = require("node:assert/strict");

const { defaultSchemeFor, normalizeUrl, isPlainHttpRemote } = require("../src/url");

describe("defaultSchemeFor", () => {
  it("defaults remote hosts to https", () => {
    assert.equal(defaultSchemeFor("goalrail.example.com/goalrail"), "https");
    assert.equal(defaultSchemeFor("example.com"), "https");
  });

  it("defaults loopback hosts to http", () => {
    assert.equal(defaultSchemeFor("localhost:6767"), "http");
    assert.equal(defaultSchemeFor("127.0.0.1:6767"), "http");
    assert.equal(defaultSchemeFor("[::1]:6767"), "http");
  });

  it("defaults unparseable input to https", () => {
    assert.equal(defaultSchemeFor("exa mple"), "https");
  });
});

describe("normalizeUrl", () => {
  it("defaults a schemeless remote /goalrail URL to https", () => {
    assert.equal(
      normalizeUrl("goalrail.example.com/goalrail"),
      "https://goalrail.example.com/goalrail",
    );
  });

  it("defaults a bare remote host to https", () => {
    assert.equal(normalizeUrl("goalrail.example.com"), "https://goalrail.example.com/");
  });

  it("defaults loopback hosts to http", () => {
    assert.equal(normalizeUrl("localhost:6767"), "http://localhost:6767/");
    assert.equal(normalizeUrl("127.0.0.1:6767"), "http://127.0.0.1:6767/");
    assert.equal(normalizeUrl("[::1]:6767"), "http://[::1]:6767/");
  });

  it("preserves an explicit scheme (even http to a remote host)", () => {
    assert.equal(normalizeUrl("http://localhost:6767"), "http://localhost:6767/");
    assert.equal(normalizeUrl("https://example.com"), "https://example.com/");
    assert.equal(normalizeUrl("http://goalrail.example.com"), "http://goalrail.example.com/");
  });

  it("trims surrounding whitespace", () => {
    assert.equal(normalizeUrl("  example.com/goalrail  "), "https://example.com/goalrail");
  });

  it("rejects empty input", () => {
    assert.throws(() => normalizeUrl(""), /server URL is empty/);
    assert.throws(() => normalizeUrl("   "), /server URL is empty/);
  });

  it("rejects a non-http(s) scheme", () => {
    assert.throws(() => normalizeUrl("ftp://example.com"), /unsupported scheme/);
  });
});

describe("isPlainHttpRemote", () => {
  it("does not warn for a bare remote host (now https)", () => {
    assert.equal(isPlainHttpRemote("goalrail.example.com"), false);
    assert.equal(isPlainHttpRemote("goalrail.example.com/goalrail"), false);
  });

  it("warns for an explicit http:// to a remote host", () => {
    assert.equal(isPlainHttpRemote("http://goalrail.example.com"), true);
  });

  it("does not warn for loopback hosts", () => {
    assert.equal(isPlainHttpRemote("localhost:6767"), false);
    assert.equal(isPlainHttpRemote("http://localhost:6767"), false);
    assert.equal(isPlainHttpRemote("http://127.0.0.1:6767"), false);
  });

  it("does not warn for https or empty/invalid input", () => {
    assert.equal(isPlainHttpRemote("https://goalrail.example.com"), false);
    assert.equal(isPlainHttpRemote(""), false);
    assert.equal(isPlainHttpRemote("ht tp://nope"), false);
  });
});
