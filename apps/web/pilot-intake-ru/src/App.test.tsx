import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import App from './App';
import { render } from '../test-utils';

describe('App', () => {
  it('renders the Russian pilot intake landing', () => {
    render(<App />);

    expect(screen.getByRole('heading', { name: /Разработка с ИИ.*под контролем/i })).toBeInTheDocument();
    expect(screen.getByText(/Пилот это управляемый эксперимент/i)).toBeInTheDocument();
    expect(screen.getByText(/ПИЛОТ ОТКРЫТ/i)).toBeInTheDocument();
    expect(screen.getByText(/Публичная страница пилота/i)).toBeInTheDocument();
  });

  it('keeps the pilot-first proof and non-autopilot messaging visible', () => {
    render(<App />);

    expect(screen.getAllByText(/рабочий контракт/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/проверяем результат/i)).toBeInTheDocument();
    expect(screen.getByText(/не продаём «полный автопилот»/i)).toBeInTheDocument();
    expect(screen.getByText(/не заменяем трекер задач или среду разработки/i)).toBeInTheDocument();
  });

  it('exposes the email CTA with the direct contact address', () => {
    render(<App />);

    const form = screen.getByRole('form', { name: /Форма заявки на пилот/i });
    const input = screen.getByRole('textbox', { name: /Рабочая почта/i });
    const button = screen.getByRole('button', { name: /Обсудить пилот/i });
    const email = screen.getByRole('link', { name: /hello@goalrail\.dev/i });

    expect(document.querySelectorAll('form')).toHaveLength(1);
    expect(form).toHaveAttribute('action', 'mailto:hello@goalrail.dev');
    expect(input).toHaveAttribute('type', 'email');
    expect(input).toHaveAttribute('placeholder', 'ваша@компания.ru');
    expect(input).not.toHaveAttribute('placeholder', 'hello@goalrail.dev');
    expect(button).toHaveAttribute('type', 'submit');
    expect(email).toHaveAttribute('href', 'mailto:hello@goalrail.dev');
  });
});
