import '@testing-library/jest-dom/vitest';

import { vi } from 'vitest';

const { getComputedStyle } = window;
window.getComputedStyle = (element) => getComputedStyle(element);
window.HTMLElement.prototype.scrollIntoView = () => {};

const localStorageState = new Map<string, string>();

Object.defineProperty(window, 'localStorage', {
  configurable: true,
  value: {
    get length() {
      return localStorageState.size;
    },
    clear: vi.fn(() => localStorageState.clear()),
    getItem: vi.fn((key: string) => localStorageState.get(key) ?? null),
    key: vi.fn((index: number) => Array.from(localStorageState.keys())[index] ?? null),
    removeItem: vi.fn((key: string) => {
      localStorageState.delete(key);
    }),
    setItem: vi.fn((key: string, value: string) => {
      localStorageState.set(key, value);
    }),
  },
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
