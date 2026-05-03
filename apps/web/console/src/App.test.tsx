import { fireEvent, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import App from './App';
import { render } from '../test-utils';

function login() {
  render(<App />);

  fireEvent.change(screen.getByLabelText(/^Email$/i), { target: { value: 'owner@example.com' } });
  fireEvent.change(screen.getByLabelText(/^Password$/i), { target: { value: 'password' } });
  fireEvent.click(screen.getByRole('button', { name: /sign in/i }));
}

describe('App', () => {
  it('renders a login-only entry screen without registration', () => {
    render(<App />);

    const brand = screen.getByLabelText(/goalrail console/i);

    expect(brand.tagName).toBe('DIV');
    expect(brand).toHaveTextContent(/^GOALRAIL$/);
    expect(brand.querySelector('svg.brandMark')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /goalrail console/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: /^GoalRail Console$/ })).not.toBeInTheDocument();
    expect(screen.queryByText(/sign in to your workspace/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/administrator grants access/i)).not.toBeInTheDocument();
    expect(screen.getByLabelText(/^Email$/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/^Password$/i)).toBeInTheDocument();
    expect(screen.queryByText(/registration|register|sign up|sso/i)).not.toBeInTheDocument();
  });

  it('keeps the three product surfaces empty after login', () => {
    login();

    const navigation = screen.getByRole('navigation', { name: /product surfaces/i });

    expect(navigation).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Contracts$/i })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('button', { name: /^Delivery Readiness$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Proof$/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/contracts: empty surface/i)).toBeInTheDocument();
    expect(screen.queryByText(/trialops-demo|C-0147|readiness score|proof queue/i)).not.toBeInTheDocument();
  });

  it('opens users settings from the bottom utility without making it a product surface', () => {
    login();

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));

    expect(screen.getByLabelText(/settings: users/i)).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: /product surfaces/i })).toBeInTheDocument();
    expect(screen.queryByRole('navigation', { name: /settings sections/i })).not.toBeInTheDocument();
    expect(screen.queryByText(/^Section$/i)).not.toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /^Users$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Contracts$/i })).not.toHaveAttribute('aria-current', 'page');
    expect(screen.queryByText(/preview|local-only|local UI|backend|sessions|cookies|future/i)).not.toBeInTheDocument();
  });

  it('adds and edits users in the settings drawer', () => {
    login();

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    fireEvent.click(screen.getByRole('button', { name: /add user/i }));

    expect(screen.getByRole('complementary', { name: /add user/i })).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText(/^Name$/i), { target: { value: 'QA Lead' } });
    fireEvent.change(screen.getByLabelText(/^Email$/i), { target: { value: 'qa@example.com' } });
    fireEvent.click(screen.getByRole('button', { name: /^Save$/i }));

    expect(screen.getByText('QA Lead')).toBeInTheDocument();
    expect(screen.getByText('qa@example.com')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /edit QA Lead/i }));
    fireEvent.change(screen.getByLabelText(/^Name$/i), { target: { value: 'QA Owner' } });
    fireEvent.click(screen.getByRole('button', { name: /^Save$/i }));

    expect(screen.getByText('QA Owner')).toBeInTheDocument();
    expect(screen.queryByText('QA Lead')).not.toBeInTheDocument();
  });

  it('filters users by search, role, and status', () => {
    login();

    fireEvent.click(screen.getByRole('button', { name: /settings/i }));
    const table = screen.getByRole('table', { name: /workspace users/i });
    expect(table).toBeInTheDocument();
    expect(screen.getByRole('searchbox', { name: /search users/i })).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText(/search by name or email/i), {
      target: { value: 'reviewer' },
    });

    expect(within(table).getByText('Reviewer')).toBeInTheDocument();
    expect(within(table).queryByText('Owner')).not.toBeInTheDocument();
    expect(within(table).queryByText('Product Lead')).not.toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText(/search by name or email/i), {
      target: { value: '' },
    });
    fireEvent.change(screen.getByLabelText(/filter by role/i), {
      target: { value: 'Member' },
    });

    expect(within(table).getByText('Product Lead')).toBeInTheDocument();
    expect(within(table).queryByText('Owner')).not.toBeInTheDocument();
    expect(within(table).queryByText('Reviewer')).not.toBeInTheDocument();

    fireEvent.change(screen.getByLabelText(/filter by status/i), {
      target: { value: 'Active' },
    });

    expect(screen.getByText(/no users/i)).toBeInTheDocument();
  });
});
