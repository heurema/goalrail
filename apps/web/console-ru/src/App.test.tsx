import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import App from './App';
import { render } from '../test-utils';

function login() {
  render(<App />);

  fireEvent.change(screen.getByLabelText(/^Email$/i), { target: { value: 'owner@example.com' } });
  fireEvent.change(screen.getByLabelText(/^Пароль$/i), { target: { value: 'password' } });
  fireEvent.click(screen.getByRole('button', { name: /войти/i }));
}

describe('App', () => {
  it('renders a login-only entry screen without registration', () => {
    render(<App />);

    expect(screen.getByRole('heading', { name: /^GoalRail Console$/ })).toBeInTheDocument();
    expect(screen.getByLabelText(/^Email$/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/^Пароль$/i)).toBeInTheDocument();
    expect(screen.getByText(/доступ выдает администратор/i)).toBeInTheDocument();
    expect(screen.queryByText(/регистрация|зарегистрироваться|sign up|sso/i)).not.toBeInTheDocument();
  });

  it('keeps the three product surfaces empty after login', () => {
    login();

    const navigation = screen.getByRole('navigation', { name: /разделы продукта/i });

    expect(navigation).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Контракты$/i })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('button', { name: /^Оценка готовности$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Проверка результата$/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/контракты: пустой раздел/i)).toBeInTheDocument();
    expect(screen.queryByText(/trialops-demo|C-0147|readiness score|proof queue/i)).not.toBeInTheDocument();
  });

  it('opens users settings from the bottom utility without making it a product surface', () => {
    login();

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));

    expect(screen.getByLabelText(/настройки: пользователи/i)).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: /разделы продукта/i })).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: /разделы настроек/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /пользователи\s*access/i })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('button', { name: /^Контракты$/i })).not.toHaveAttribute('aria-current', 'page');
    expect(screen.queryByText(/preview|local-only|local UI|backend|sessions|cookies|будущ/i)).not.toBeInTheDocument();
  });

  it('adds and edits users in the settings drawer', () => {
    login();

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));
    fireEvent.click(screen.getByRole('button', { name: /добавить пользователя/i }));

    expect(screen.getByRole('complementary', { name: /добавить пользователя/i })).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText(/^Имя$/i), { target: { value: 'QA Lead' } });
    fireEvent.change(screen.getByLabelText(/^Email$/i), { target: { value: 'qa@example.com' } });
    fireEvent.click(screen.getByRole('button', { name: /^Сохранить$/i }));

    expect(screen.getByText('QA Lead')).toBeInTheDocument();
    expect(screen.getByText('qa@example.com')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /редактировать QA Lead/i }));
    fireEvent.change(screen.getByLabelText(/^Имя$/i), { target: { value: 'QA Owner' } });
    fireEvent.click(screen.getByRole('button', { name: /^Сохранить$/i }));

    expect(screen.getByText('QA Owner')).toBeInTheDocument();
    expect(screen.queryByText('QA Lead')).not.toBeInTheDocument();
  });

  it('filters users by search, role, and status', () => {
    login();

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));
    expect(screen.getByRole('table', { name: /пользователи рабочего пространства/i })).toBeInTheDocument();
    expect(screen.getByRole('searchbox', { name: /поиск пользователей/i })).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText(/поиск по имени или email/i), {
      target: { value: 'reviewer' },
    });

    expect(screen.getByText('Reviewer')).toBeInTheDocument();
    expect(screen.queryByText('Owner')).not.toBeInTheDocument();
    expect(screen.queryByText('Product Lead')).not.toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText(/поиск по имени или email/i), {
      target: { value: '' },
    });
    fireEvent.change(screen.getByLabelText(/фильтр по роли/i), {
      target: { value: 'Участник' },
    });

    expect(screen.getByText('Product Lead')).toBeInTheDocument();
    expect(screen.queryByText('Owner')).not.toBeInTheDocument();
    expect(screen.queryByText('Reviewer')).not.toBeInTheDocument();

    fireEvent.change(screen.getByLabelText(/фильтр по статусу/i), {
      target: { value: 'Активен' },
    });

    expect(screen.getByText(/нет пользователей/i)).toBeInTheDocument();
  });
});
