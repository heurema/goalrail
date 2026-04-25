import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import App from './App';
import { render } from '../test-utils';

describe('App', () => {
  it('renders only the three empty Russian console surfaces in the left navigation', () => {
    render(<App />);

    const navigation = screen.getByRole('navigation', { name: /разделы продукта/i });
    const buttons = screen.getAllByRole('button');

    expect(navigation).toBeInTheDocument();
    expect(buttons).toHaveLength(3);
    expect(screen.getByRole('button', { name: /^Контракты$/i })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('button', { name: /^Готовность доставки$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Доказательства$/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/контракты: пустой раздел/i)).toBeInTheDocument();
    expect(screen.queryByText(/trialops-demo|C-0147|readiness score|proof queue/i)).not.toBeInTheDocument();
  });

  it('switches the active empty Russian surface without rendering data', () => {
    render(<App />);

    fireEvent.click(screen.getByRole('button', { name: /^Готовность доставки$/i }));
    expect(screen.getByRole('button', { name: /^Готовность доставки$/i })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByLabelText(/готовность доставки: пустой раздел/i)).toBeInTheDocument();
    expect(screen.queryByText(/add repository|connected|score|status/i)).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /^Доказательства$/i }));
    expect(screen.getByRole('button', { name: /^Доказательства$/i })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByLabelText(/доказательства: пустой раздел/i)).toBeInTheDocument();
    expect(screen.queryByText(/archive|decision|evidence/i)).not.toBeInTheDocument();
  });
});
