import { fireEvent, screen, within } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';

import App from './App';
import { render } from '../test-utils';

const THEME_STORAGE_KEY = 'goalrail.console.theme';

function login() {
  render(<App />);

  fireEvent.change(screen.getByLabelText(/^Email$/i), { target: { value: 'owner@example.com' } });
  fireEvent.change(screen.getByLabelText(/^Пароль$/i), { target: { value: 'password' } });
  fireEvent.click(screen.getByRole('button', { name: /войти/i }));
}

describe('App', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('renders a login-only entry screen without registration', () => {
    render(<App />);

    const brand = screen.getByLabelText(/консоль goalrail/i);

    expect(brand.tagName).toBe('DIV');
    expect(brand).toHaveTextContent(/^GOALRAIL$/);
    expect(brand.querySelector('svg.brandMark')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /консоль goalrail/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: /^GoalRail Console$/ })).not.toBeInTheDocument();
    expect(screen.queryByText(/вход в рабочее пространство/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/доступ выдает администратор/i)).not.toBeInTheDocument();
    expect(screen.getByLabelText(/^Email$/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/^Пароль$/i)).toBeInTheDocument();
    expect(screen.queryByText(/регистрация|зарегистрироваться|sign up|sso/i)).not.toBeInTheDocument();
  });

  it('keeps exactly three product surfaces with honest structured empty states', () => {
    login();

    const navigation = screen.getByRole('navigation', { name: /разделы продукта/i });
    const productButtons = within(navigation).getAllByRole('button');

    expect(navigation).toBeInTheDocument();
    expect(productButtons).toHaveLength(3);
    expect(screen.getByRole('button', { name: /^Контракты$/i })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('button', { name: /^Оценка готовности$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Проверка результата$/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/контракты: structured empty state/i)).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /^Контракты$/i })).toBeInTheDocument();
    expect(screen.getByText('Goal → Contract → Task → Proof')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /^Оценка готовности$/i }));

    expect(screen.getByText(/хватает ли контекста/i)).toBeInTheDocument();
    expect(screen.getByText('НЕ ПРОВЕРЯЛОСЬ')).toBeInTheDocument();
    expect(screen.getByText(/Открытые вопросы, которые блокируют уверенный handoff/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /^Проверка результата$/i }));

    expect(screen.getByText(/проверки execution evidence через gate/i)).toBeInTheDocument();
    expect(screen.getByText('ОЖИДАЕТ VERIFIED EVIDENCE')).toBeInTheDocument();
    expect(screen.getByText(/Сохранили ли проверки и evidence доверие/i)).toBeInTheDocument();
    expect(document.body).not.toHaveTextContent(
      /trialops-demo|C-0147|readiness score|\/100|\bscan\b|proof queue|fake queue|fake pass|fake fail|pass\/fail/i
    );
  });

  it('opens appearance settings by default without making it a product surface', () => {
    login();

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));

    expect(screen.getByLabelText(/настройки: оформление/i)).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: /разделы продукта/i })).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: /разделы настроек/i })).toBeInTheDocument();
    expect(screen.queryByText(/^Раздел$/i)).not.toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /^Оформление$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Контракты$/i })).not.toHaveAttribute('aria-current', 'page');
    expect(screen.queryByText(/preview|local-only|local UI|backend|sessions|cookies|будущ/i)).not.toBeInTheDocument();
  });

  it('renders all theme presets and applies the selected theme to the shell', () => {
    login();

    expect(screen.getByRole('main')).toHaveAttribute('data-goalrail-theme', 'goalrail-default');

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));

    expect(screen.getByRole('button', { name: /Goalrail Default/i })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByRole('button', { name: /Catppuccin Mocha/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Dracula/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Nord/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Solarized Dark/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Gruvbox Dark/i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /Nord/i }));

    expect(screen.getByRole('main')).toHaveAttribute('data-goalrail-theme', 'nord');
    expect(screen.getByRole('button', { name: /Nord/i })).toHaveAttribute('aria-pressed', 'true');
    expect(window.localStorage.getItem(THEME_STORAGE_KEY)).toBe('nord');
  });

  it('initializes with a stored valid theme', () => {
    window.localStorage.setItem(THEME_STORAGE_KEY, 'solarized-dark');

    login();

    expect(screen.getByRole('main')).toHaveAttribute('data-goalrail-theme', 'solarized-dark');

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));

    expect(screen.getByRole('button', { name: /Solarized Dark/i })).toHaveAttribute('aria-pressed', 'true');
  });

  it('falls back to the default theme when stored theme is invalid', () => {
    window.localStorage.setItem(THEME_STORAGE_KEY, 'unknown-theme');

    login();

    expect(screen.getByRole('main')).toHaveAttribute('data-goalrail-theme', 'goalrail-default');

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));

    expect(screen.getByRole('button', { name: /Goalrail Default/i })).toHaveAttribute('aria-pressed', 'true');
  });

  it('opens users inside settings after theme switching', () => {
    login();

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));
    fireEvent.click(screen.getByRole('button', { name: /Nord/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Пользователи$/i }));

    expect(screen.getByLabelText(/настройки: пользователи/i)).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /^Пользователи$/i })).toBeInTheDocument();
    expect(screen.getByRole('table', { name: /пользователи рабочего пространства/i })).toBeInTheDocument();
  });

  it('adds and edits users in the settings drawer', () => {
    login();

    fireEvent.click(screen.getByRole('button', { name: /настройки/i }));
    fireEvent.click(screen.getByRole('button', { name: /^Пользователи$/i }));
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
    fireEvent.click(screen.getByRole('button', { name: /^Пользователи$/i }));
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
