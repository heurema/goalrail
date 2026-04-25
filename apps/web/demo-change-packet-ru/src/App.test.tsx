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

    expect(screen.getAllByText(/Входящий запрос/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/Главные вводные · 5 всего/i)).toBeInTheDocument();
    expect(screen.getByText(/Цепочка изменения · cp-0147/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Начать/i })).toBeInTheDocument();
  });

  it('advances through clarification and contract states', () => {
    render(<App />);

    fireEvent.click(screen.getByRole('button', { name: /Начать/i }));

    expect(screen.getByText(/Уточнения · 5 из 5/i)).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(3000);
    });

    expect(screen.getAllByText(/Ответ закреплен в контракте/i)).toHaveLength(5);

    fireEvent.click(screen.getByRole('button', { name: /К контракту/i }));

    expect(screen.getByText(/Рабочий контракт · черновик v3/i)).toBeInTheDocument();
    expect(screen.getByText(/Показать, как один запрос по репозиторию/i)).toBeInTheDocument();
  });

  it('switches between workspace-level readiness and proof surfaces', () => {
    render(<App />);

    expect(screen.getByText(/Активные контракты/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/Искать контракты/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /^trialops-demo$/i }));
    expect(screen.getByRole('listbox', { name: /Переключатель репозитория/i })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: /Все репозитории/i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /^Готовность$/i }));

    expect(screen.queryByText(/Активные контракты/i)).not.toBeInTheDocument();
    expect(screen.queryByPlaceholderText(/Искать контракты/i)).not.toBeInTheDocument();
    expect(screen.getAllByText(/Репозитории/i).length).toBeGreaterThan(0);
    expect(screen.getByPlaceholderText(/Искать репозитории/i)).toBeInTheDocument();
    expect(screen.getByRole('group', { name: /Список готовности репозиториев/i })).toBeInTheDocument();
    expect(screen.getByText(/Настройка репозитория и режим работы/i)).toBeInTheDocument();
    expect(screen.getAllByText(/frontend-console/i).length).toBeGreaterThan(0);
    expect(screen.getByRole('button', { name: /^Добавить репозиторий$/i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /^Подтверждения$/i }));

    expect(screen.queryByText(/Активные контракты/i)).not.toBeInTheDocument();
    expect(screen.getAllByText(/Очередь подтверждений|Очередь пакетов подтверждения/i).length).toBeGreaterThan(0);
    expect(screen.getByPlaceholderText(/Искать подтверждения/i)).toBeInTheDocument();
    expect(screen.getByRole('group', { name: /Очередь подтверждений/i })).toBeInTheDocument();
    expect(screen.getByText(/Артефакты и решения по контрактам/i)).toBeInTheDocument();
    expect(screen.getAllByText(/C-0082/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/Архив подтверждения \/ хеш/i)).toBeInTheDocument();
  });

  it('shows the mobile companion instead of the desktop console below the breakpoint', () => {
    mockMatchMedia(true);

    render(<App />);

    expect(screen.getByRole('heading', { name: 'Goalrail' })).toBeInTheDocument();
    expect(screen.getByText(/Короткий режим проверки: контракты, готовность и подтверждения/i)).toBeInTheDocument();
    expect(screen.queryByText(/Главные вводные/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Цепочка изменения · cp-0147/i)).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Итог$/i })).toHaveClass('active');
    expect(screen.getByText(/C-0147 · Ждет решения · 5\/5/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Проверить решение/i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /^Контракты$/i }));
    expect(screen.getByText(/Стадия: Входящий запрос · 1\/8/i)).toBeInTheDocument();
    expect(screen.getByText(/C-0148 · Фильтры CSV-экспорта · В работе/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /^Готовность$/i }));
    expect(screen.getByText(/3 репозитория/i)).toBeInTheDocument();
    expect(screen.getByText(/57\/100 среднее/i)).toBeInTheDocument();
    expect(screen.getByText(/trialops-demo · 72\/100 · Готово/i)).toBeInTheDocument();
    expect(screen.getByText(/Настройку лучше делать/i)).toBeInTheDocument();
  });
});
