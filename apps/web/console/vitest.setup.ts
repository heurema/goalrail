import '@testing-library/jest-dom/vitest';

import { vi } from 'vitest';

const { getComputedStyle } = window;
window.getComputedStyle = (element) => getComputedStyle(element);
window.HTMLElement.prototype.scrollIntoView = () => {};

function createStorageMock() {
  const storageState = new Map<string, string>();

  return {
    get length() {
      return storageState.size;
    },
    clear: vi.fn(() => storageState.clear()),
    getItem: vi.fn((key: string) => storageState.get(key) ?? null),
    key: vi.fn((index: number) => Array.from(storageState.keys())[index] ?? null),
    removeItem: vi.fn((key: string) => {
      storageState.delete(key);
    }),
    setItem: vi.fn((key: string, value: string) => {
      storageState.set(key, value);
    }),
  };
}

Object.defineProperty(window, 'localStorage', {
  configurable: true,
  value: createStorageMock(),
});

Object.defineProperty(window, 'sessionStorage', {
  configurable: true,
  value: createStorageMock(),
});

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});

Object.defineProperty(document, 'fonts', {
  value: { addEventListener: vi.fn(), removeEventListener: vi.fn() },
});

class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}

window.ResizeObserver = ResizeObserver;
