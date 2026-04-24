import { fireEvent, screen, act } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import App from './App';
import { render } from '../test-utils';

function mockMatchMedia(matches: boolean) {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
}

describe('App', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mockMatchMedia(false);
  });

  afterEach(() => {
    vi.runOnlyPendingTimers();
    vi.useRealTimers();
  });

  it('renders the change packet intake shell', () => {
    render(<App />);

    expect(screen.getByText(/Raw request · inbound/i)).toBeInTheDocument();
    expect(screen.getByText(/Ambiguity inspector/i)).toBeInTheDocument();
    expect(screen.getByText(/Change spine · cp-0147/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Begin/i })).toBeInTheDocument();
  });

  it('advances through clarification and contract states', () => {
    render(<App />);

    fireEvent.click(screen.getByRole('button', { name: /Begin/i }));

    expect(screen.getByText(/Clarification cards · 5 of 5/i)).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(3000);
    });

    expect(screen.getAllByText(/Answer pinned to contract/i)).toHaveLength(5);

    fireEvent.click(screen.getByRole('button', { name: /Begin/i }));

    expect(screen.getByText(/Working contract · draft v3/i)).toBeInTheDocument();
    expect(screen.getByText(/Introduce a bounded/i)).toBeInTheDocument();
  });

  it('switches between workspace-level readiness and proof surfaces', () => {
    render(<App />);

    expect(screen.getByText(/Active contracts/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/Search contracts/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /^trialops-demo$/i }));
    expect(screen.getByRole('listbox', { name: /Repo selector/i })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: /All repos/i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /^Delivery Readiness$/i }));

    expect(screen.queryByText(/Active contracts/i)).not.toBeInTheDocument();
    expect(screen.queryByPlaceholderText(/Search contracts/i)).not.toBeInTheDocument();
    expect(screen.getAllByText(/Repositories/i).length).toBeGreaterThan(0);
    expect(screen.getByPlaceholderText(/Search repos/i)).toBeInTheDocument();
    expect(screen.getByRole('group', { name: /Repo readiness shelf/i })).toBeInTheDocument();
    expect(screen.getByText(/Repo-level setup and operating mode/i)).toBeInTheDocument();
    expect(screen.getAllByText(/frontend-console/i).length).toBeGreaterThan(0);
    expect(screen.getByRole('button', { name: /^Add repository$/i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /^Proof Feed$/i }));

    expect(screen.queryByText(/Active contracts/i)).not.toBeInTheDocument();
    expect(screen.getAllByText(/Proof queue/i).length).toBeGreaterThan(0);
    expect(screen.getByPlaceholderText(/Search proofs/i)).toBeInTheDocument();
    expect(screen.getByRole('group', { name: /Proof queue shelf/i })).toBeInTheDocument();
    expect(screen.getByText(/Cross-contract evidence and decisions/i)).toBeInTheDocument();
    expect(screen.getAllByText(/C-0082/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/Proof archive \/ hash/i)).toBeInTheDocument();
  });

  it('shows the mobile companion instead of the desktop console below the breakpoint', () => {
    mockMatchMedia(true);

    render(<App />);

    expect(screen.getByText(/Goalrail Mobile Companion/i)).toBeInTheDocument();
    expect(screen.getByText(/Focused review mode for contracts, readiness, and proof decisions/i)).toBeInTheDocument();
    expect(screen.queryByText(/Ambiguity inspector/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Change spine · cp-0147/i)).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Proof$/i })).toHaveClass('active');
    expect(screen.getByText(/C-0147 · Awaiting approval · 5\/5/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Review decision/i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /^Contracts$/i }));
    expect(screen.getByText(/Goal intake · 1\/7/i)).toBeInTheDocument();
    expect(screen.getByText(/C-0148 · CSV export filters · Executing/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /^Readiness$/i }));
    expect(screen.getByText(/3 repos/i)).toBeInTheDocument();
    expect(screen.getByText(/57\/100 avg/i)).toBeInTheDocument();
    expect(screen.getByText(/trialops-demo · 72\/100 · Ready/i)).toBeInTheDocument();
    expect(screen.getByText(/Desktop setup recommended/i)).toBeInTheDocument();
  });
});
