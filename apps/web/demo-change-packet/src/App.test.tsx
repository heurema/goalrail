import { fireEvent, screen, act } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import App from './App';
import { render } from '../test-utils';

describe('App', () => {
  beforeEach(() => {
    vi.useFakeTimers();
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
});
