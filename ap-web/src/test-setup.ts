import "@testing-library/jest-dom/vitest";
import { vi } from "vitest";

// The @lobehub icon packages have broken nested-module resolution
// under vitest; stub presentational glyphs so component modules that
// import them can still load in tests. (The Antigravity glyph additionally
// drags in @lobehub/fluent-emoji → @emoji-mart/data, whose JSON modules need
// an import attribute Node refuses under vitest — so it must be stubbed too.)
vi.mock("@/components/icons/ClaudeIcon", () => ({
  ClaudeIcon: () => null,
}));
vi.mock("@/components/icons/CodexIcon", () => ({
  CodexIcon: () => null,
}));
vi.mock("@/components/icons/OpenCodeIcon", () => ({
  OpenCodeIcon: () => null,
}));
vi.mock("@/components/icons/CursorIcon", () => ({
  CursorIcon: () => null,
}));
vi.mock("@/components/icons/GooseIcon", () => ({
  GooseIcon: () => null,
}));
vi.mock("@/components/icons/AntigravityIcon", () => ({
  AntigravityIcon: () => null,
}));

// Radix UI primitives (DropdownMenu, etc.) call these pointer-capture and
// scroll APIs that jsdom doesn't implement. Stub them so component tests
// that open a Radix menu don't throw. No-ops are sufficient — the tests
// assert on the resulting DOM, not on capture/scroll side effects.
if (!Element.prototype.hasPointerCapture) {
  Element.prototype.hasPointerCapture = () => false;
}
if (!Element.prototype.setPointerCapture) {
  Element.prototype.setPointerCapture = () => {};
}
if (!Element.prototype.releasePointerCapture) {
  Element.prototype.releasePointerCapture = () => {};
}
if (!Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = () => {};
}

Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
});

// Node 25 exposes an experimental global Web Storage object that lacks the
// browser Storage API methods unless Node is started with a localstorage file.
// In that runtime Vitest's jsdom window inherits the broken object too, so
// install a small Storage-compatible test double while keeping methods on
// Storage.prototype for tests that spy on quota/access failures.
if (typeof window.Storage !== "undefined" && typeof window.localStorage?.clear !== "function") {
  const stores = new WeakMap<Storage, Map<string, string>>();

  function dataFor(storage: Storage): Map<string, string> {
    let data = stores.get(storage);
    if (!data) {
      data = new Map<string, string>();
      stores.set(storage, data);
    }
    return data;
  }

  Object.defineProperties(window.Storage.prototype, {
    clear: {
      configurable: true,
      value(this: Storage) {
        dataFor(this).clear();
      },
    },
    getItem: {
      configurable: true,
      value(this: Storage, key: string) {
        return dataFor(this).get(String(key)) ?? null;
      },
    },
    key: {
      configurable: true,
      value(this: Storage, index: number) {
        return Array.from(dataFor(this).keys())[index] ?? null;
      },
    },
    removeItem: {
      configurable: true,
      value(this: Storage, key: string) {
        dataFor(this).delete(String(key));
      },
    },
    setItem: {
      configurable: true,
      value(this: Storage, key: string, value: string) {
        dataFor(this).set(String(key), String(value));
      },
    },
  });

  function createStorage(): Storage {
    const storage = Object.create(window.Storage.prototype) as Storage;
    stores.set(storage, new Map<string, string>());
    Object.defineProperty(storage, "length", {
      configurable: true,
      get() {
        return dataFor(storage).size;
      },
    });
    return storage;
  }

  const localStorage = createStorage();
  const sessionStorage = createStorage();
  Object.defineProperty(window, "localStorage", { configurable: true, value: localStorage });
  Object.defineProperty(window, "sessionStorage", { configurable: true, value: sessionStorage });
  Object.defineProperty(globalThis, "localStorage", { configurable: true, value: localStorage });
  Object.defineProperty(globalThis, "sessionStorage", {
    configurable: true,
    value: sessionStorage,
  });
}
