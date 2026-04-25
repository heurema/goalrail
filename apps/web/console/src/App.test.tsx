import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import App from './App';
import { render } from '../test-utils';

describe('App', () => {
  it('renders only the three empty console surfaces in the left navigation', () => {
    render(<App />);

    const navigation = screen.getByRole('navigation', { name: /product surfaces/i });
    const buttons = screen.getAllByRole('button');

    expect(navigation).toBeInTheDocument();
    expect(buttons).toHaveLength(3);
    expect(screen.getByRole('button', { name: /^Contracts$/i })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('button', { name: /^Delivery Readiness$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Proof$/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/contracts surface empty/i)).toBeInTheDocument();
    expect(screen.queryByText(/trialops-demo|C-0147|readiness score|proof queue/i)).not.toBeInTheDocument();
  });

  it('switches the active empty surface without rendering data', () => {
    render(<App />);

    fireEvent.click(screen.getByRole('button', { name: /^Delivery Readiness$/i }));
    expect(screen.getByRole('button', { name: /^Delivery Readiness$/i })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByLabelText(/delivery readiness surface empty/i)).toBeInTheDocument();
    expect(screen.queryByText(/add repository|connected|score|status/i)).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /^Proof$/i }));
    expect(screen.getByRole('button', { name: /^Proof$/i })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByLabelText(/proof surface empty/i)).toBeInTheDocument();
    expect(screen.queryByText(/archive|decision|evidence/i)).not.toBeInTheDocument();
  });
});
