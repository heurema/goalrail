/* global React, ReactDOM */
const { useState, useMemo } = React;

// =============== ICONS ===============
const Ic = {
  contract: (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round">
      <path d="M3 2h7l3 3v9H3z" /><path d="M10 2v3h3" /><path d="M5.5 8h5M5.5 10.5h3.5" />
    </svg>
  ),
  check: (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="8" cy="8" r="5.5" /><path d="M5.7 8.2l1.6 1.6 3-3.4" />
    </svg>
  ),
  shield: (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round">
      <path d="M8 2l5 2v4.5c0 3-2.2 4.7-5 5.5-2.8-.8-5-2.5-5-5.5V4z" /><path d="M5.7 8.3l1.6 1.6 3-3.4" />
    </svg>
  ),
  gear: (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="8" cy="8" r="2.2" />
      <path d="M8 1.6v1.6M8 12.8v1.6M2.2 8h1.6M12.2 8h1.6M3.7 3.7l1.1 1.1M11.2 11.2l1.1 1.1M3.7 12.3l1.1-1.1M11.2 4.8l1.1-1.1" />
    </svg>
  ),
  plus: (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
      <path d="M8 3.5v9M3.5 8h9" />
    </svg>
  ),
  edit: (
    <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round">
      <path d="M3 13l.6-2.4 7-7 1.8 1.8-7 7L3 13z" /><path d="M9.6 4l1.8 1.8" />
    </svg>
  ),
  search: (
    <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="7" cy="7" r="4" /><path d="M10 10l3 3" />
    </svg>
  ),
  close: (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round">
      <path d="M4 4l8 8M12 4l-8 8" />
    </svg>
  ),
  menu: (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round">
      <path d="M3 4.5h10M3 8h10M3 11.5h10" />
    </svg>
  ),
  arrow: (
    <svg width="11" height="11" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round">
      <path d="M3 8h10M9 4l4 4-4 4" />
    </svg>
  ),
};

// =============== BRAND ===============
function Brand({ small }) {
  return (
    <div className="brand">
      <div className="brand-mark"><span /></div>
      <div className="name" style={small ? { fontSize: 10.5 } : {}}>Goalrail</div>
    </div>
  );
}

// =============== LOGIN ===============
function LoginFrame({ withError = false }) {
  return (
    <div className="frame" data-screen-label="01 Вход">
      <div className="login">
        <div className="login-rails" />
        <div className="login-card">
          <Brand />
          <h1>Goalrail Console</h1>
          <p className="sub">Контрактный контур управления delivery.</p>
          <p className="helper">Доступ выдает администратор рабочего пространства.</p>

          <div className="field">
            <label>Email</label>
            <input type="email" defaultValue="vitaly@example.com" />
          </div>
          <div className={"field" + (withError ? " error" : "")}>
            <label>Пароль</label>
            <input type="password" defaultValue={withError ? "" : "••••••••••••"} />
            {withError && <div className="field-msg">Введите пароль для продолжения.</div>}
          </div>

          <button className="btn primary full" style={{ marginTop: 8 }}>Войти {Ic.arrow}</button>

          <div className="login-foot">
            <span>UI PREVIEW · LOCAL-ONLY</span>
            <span>Доступ выдает администратор</span>
          </div>
        </div>
      </div>
    </div>
  );
}

// =============== SIDEBAR ===============
function Sidebar({ section, surface, onSection, onSurface }) {
  return (
    <aside className="sidebar">
      <div className="sidebar-brand"><Brand /></div>

      <div className="sidebar-section">
        <div className="kicker">Workspace</div>
        <button className="nav-item" disabled style={{ opacity: 1 }}>
          <span className="nav-ico">
            <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4">
              <rect x="2.5" y="3.5" width="11" height="9" rx="1.2" /><path d="M2.5 6.5h11" />
            </svg>
          </span>
          <span>trialops · prod</span>
          <span className="nav-meta">ws-01</span>
        </button>
      </div>

      <div className="sidebar-section">
        <div className="kicker">Surfaces</div>
        <button className={"nav-item" + (section === "console" && surface === "contracts" ? " active" : "")}
                onClick={() => { onSection("console"); onSurface("contracts"); }}>
          <span className="nav-ico">{Ic.contract}</span>
          <span>Контракты</span>
        </button>
        <button className={"nav-item" + (section === "console" && surface === "readiness" ? " active" : "")}
                onClick={() => { onSection("console"); onSurface("readiness"); }}>
          <span className="nav-ico">{Ic.check}</span>
          <span>Оценка готовности</span>
        </button>
        <button className={"nav-item" + (section === "console" && surface === "proof" ? " active" : "")}
                onClick={() => { onSection("console"); onSurface("proof"); }}>
          <span className="nav-ico">{Ic.shield}</span>
          <span>Проверка результата</span>
        </button>
      </div>

      <div className="sidebar-spacer" />

      <div className="sidebar-utility">
        <button className={"utility-btn" + (section === "settings" ? " active" : "")}
                onClick={() => onSection("settings")}>
          <span className="nav-ico">{Ic.gear}</span>
          <span>Настройки</span>
        </button>
        <div className="utility-foot">
          <span>console preview</span>
          <span>local-only</span>
        </div>
      </div>
    </aside>
  );
}

// =============== TOPBAR ===============
function Topbar({ crumb, title, right }) {
  return (
    <div className="topbar">
      <div>
        <div className="crumb">{crumb}</div>
        <h1>{title}</h1>
      </div>
      <div className="right">
        {right}
        <div className="user-chip">
          <div className="avatar">VK</div>
          <span>Vitaly · Владелец</span>
        </div>
      </div>
    </div>
  );
}

// =============== SURFACE: CONTRACTS ===============
function ContractsSurface() {
  return (
    <div className="surface">
      <div className="surface-head">
        <div>
          <div className="kicker" style={{ marginBottom: 6 }}>workspace · trialops · prod</div>
          <h2>Контракты</h2>
          <p className="lede">Рабочая область контрактов появится здесь после подключения server-backed flow. Каждый контракт держит границы между бизнес-целью и реальной поставкой.</p>
        </div>
        <div className="meta">
          <span>0 контрактов</span>
          <span>контур · не подключен</span>
        </div>
      </div>

      <div className="empty-rail">
        <div className="empty-card">
          <p className="lede">Когда контур будет подключен, здесь появится список контрактов с явной стадией и явным человеческим решением. Сейчас Goalrail Console работает в режиме shell — структура зафиксирована, исполнение ожидается.</p>
          <p className="helper"># shell ready · waiting for server-backed flow</p>
        </div>

        <div className="skeleton-row" aria-hidden="true">
          <div className="id">C-0000</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <div className="skel-bar w1" />
            <div className="skel-bar w2" />
          </div>
          <span className="pill">placeholder</span>
          <span className="pill no-dot mono" style={{ color: 'var(--mute-2)' }}>repo · —</span>
        </div>

        <div className="flow-strip">
          <span className="step"><b>Goal</b></span>
          <span className="arrow">{Ic.arrow}</span>
          <span className="step"><b>Contract</b></span>
          <span className="arrow">{Ic.arrow}</span>
          <span className="step"><b>Task</b></span>
          <span className="arrow">{Ic.arrow}</span>
          <span className="step"><b>Proof</b></span>
          <span style={{ marginLeft: 'auto' }} className="kicker">контур поставки</span>
        </div>
      </div>
    </div>
  );
}

// =============== SURFACE: READINESS ===============
function ReadinessSurface() {
  return (
    <div className="surface">
      <div className="surface-head">
        <div>
          <div className="kicker" style={{ marginBottom: 6 }}>readiness · pre-contract</div>
          <h2>Оценка готовности</h2>
          <p className="lede">Здесь будет видна готовность задачи к контракту: недостающий контекст, ограничения и критерии приемки. Поверхность работает на структурном уровне — без живых сканов и числовых оценок.</p>
        </div>
        <div className="meta">
          <span>context · waiting</span>
          <span>готовность · не проверялась</span>
        </div>
      </div>

      <div className="empty-card">
        <p className="lede">Назначение готовности — поймать недостающие условия до того, как они превратятся в проблему на этапе поставки. Каждая колонка ниже соберет конкретные сигналы, когда контракт будет получен.</p>
        <p className="helper"># readiness shell · awaiting first contract</p>
      </div>

      <div className="lane-grid">
        {[
          { n: 'Контекст', h: 'Что система должна знать о задаче, чтобы принять решение.' },
          { n: 'Ограничения', h: 'Зоны, в которые исполнение заходить не должно.' },
          { n: 'Критерии приемки', h: 'Условия, при которых результат считается принятым.' },
          { n: 'Риски', h: 'Известные неопределенности до начала работы.' },
        ].map((l) => (
          <div className="lane" key={l.n}>
            <div className="lname">{l.n}</div>
            <div className="lhint">{l.h}</div>
            <div className="ldot"><i /><i /><i /></div>
          </div>
        ))}
      </div>
    </div>
  );
}

// =============== SURFACE: PROOF ===============
function ProofSurface() {
  return (
    <div className="surface">
      <div className="surface-head">
        <div>
          <div className="kicker" style={{ marginBottom: 6 }}>proof · post-execution</div>
          <h2>Проверка результата</h2>
          <p className="lede">Здесь появится proof после проверки результата. Поверхность держит четыре дорожки и не претендует на наличие живой проверки до подключения контура.</p>
        </div>
        <div className="meta">
          <span>очередь · пусто</span>
          <span>proof · ожидается</span>
        </div>
      </div>

      <div className="empty-card">
        <p className="lede">Каждый proof говорит, что изменилось, что не изменилось и почему этому можно доверять. Без proof результат не считается принятым — это намеренная граница.</p>
        <p className="helper"># proof shell · no completed proofs yet</p>
      </div>

      <div className="lane-grid">
        {[
          { n: 'Scope', h: 'Что вошло в изменение и что осталось нетронутым.' },
          { n: 'Integrity', h: 'Связность доказательств между критериями и артефактами.' },
          { n: 'Policy', h: 'Соответствие правилам рабочего пространства.' },
          { n: 'Target', h: 'Совпадение результата с зафиксированной целью.' },
        ].map((l) => (
          <div className="lane" key={l.n}>
            <div className="lname">{l.n}</div>
            <div className="lhint">{l.h}</div>
            <div className="ldot"><i /><i /><i /><i /></div>
          </div>
        ))}
      </div>
    </div>
  );
}

// =============== SETTINGS / APPEARANCE ===============
const THEMES = [
  { id: 'goalrail', name: 'Goalrail Default', tag: 'default · warm', bg: 'oklch(0.205 0.008 55)', panel: 'oklch(0.255 0.007 55)', accent: 'oklch(0.72 0.045 330)', ink: 'oklch(0.78 0.01 72)' },
  { id: 'mocha',    name: 'Catppuccin Mocha', tag: 'pastel · soft',   bg: 'oklch(0.235 0.020 285)', panel: 'oklch(0.285 0.020 285)', accent: 'oklch(0.78 0.075 320)', ink: 'oklch(0.80 0.020 70)' },
  { id: 'dracula',  name: 'Dracula',          tag: 'violet · sharp',  bg: 'oklch(0.235 0.025 295)', panel: 'oklch(0.290 0.025 295)', accent: 'oklch(0.76 0.105 340)', ink: 'oklch(0.82 0.015 90)' },
  { id: 'nord',     name: 'Nord',             tag: 'arctic · calm',   bg: 'oklch(0.265 0.020 250)', panel: 'oklch(0.325 0.020 250)', accent: 'oklch(0.74 0.080 230)', ink: 'oklch(0.82 0.018 240)' },
  { id: 'solarized',name: 'Solarized Dark',   tag: 'classic · cyan',  bg: 'oklch(0.255 0.030 220)', panel: 'oklch(0.310 0.030 220)', accent: 'oklch(0.74 0.090 195)', ink: 'oklch(0.78 0.025 85)' },
  { id: 'gruvbox',  name: 'Gruvbox Dark',     tag: 'retro · warm',    bg: 'oklch(0.250 0.018 65)',  panel: 'oklch(0.305 0.018 65)',  accent: 'oklch(0.76 0.110 60)',  ink: 'oklch(0.78 0.030 80)' },
];

function ThemeCard({ t, selected, onSelect }) {
  return (
    <div className={"theme-card" + (selected ? " selected" : "")} data-theme={t.id} onClick={onSelect}>
      <div className="tname">{t.name}<span className="check" /></div>
      <div className="theme-preview" style={{ background: t.bg, borderColor: 'color-mix(in oklab, white 8%, transparent)' }}>
        <div className="pv-rail" style={{ background: `color-mix(in oklab, ${t.panel} 80%, ${t.bg})` }} />
        <div className="pv-body">
          <div className="pv-top" style={{ background: `color-mix(in oklab, ${t.panel} 60%, ${t.bg})` }} />
          <div className="pv-content" style={{ background: t.bg }}>
            <div className="pv-line" style={{ background: t.ink, width: '60%', opacity: 0.55 }} />
            <div className="pv-line" style={{ background: t.accent, width: '38%' }} />
            <div className="pv-line" style={{ background: t.ink, width: '48%', opacity: 0.35 }} />
          </div>
        </div>
      </div>
      <div className="swatches">
        <div className="swatch" style={{ background: t.bg }} />
        <div className="swatch" style={{ background: t.panel }} />
        <div className="swatch" style={{ background: t.accent }} />
        <div className="swatch" style={{ background: t.ink }} />
      </div>
      <div className="meta"><span>{t.id}</span><span>{t.tag}</span></div>
    </div>
  );
}

function SettingsAppearance({ themeId, onTheme }) {
  return (
    <>
      <div className="surface-head">
        <div>
          <div className="kicker" style={{ marginBottom: 6 }}>settings · appearance</div>
          <h2>Настройки</h2>
        </div>
        <div className="meta"><span>preset · {themeId}</span></div>
      </div>

      <div className="settings-layout">
        <nav className="subnav">
          <div className="kicker skicker">Раздел</div>
          <button className="active">Оформление<span className="sm">UI</span></button>
          <button>Пользователи<span className="sm">access</span></button>
        </nav>
        <div className="settings-content">
          <h3>Оформление</h3>
          <p className="desc">Выберите визуальный пресет консоли. Это влияет только на интерфейс, не на delivery logic.</p>
          <div className="theme-grid">
            {THEMES.map(t => <ThemeCard key={t.id} t={t} selected={themeId === t.id} onSelect={() => onTheme(t.id)} />)}
          </div>
          <p className="helper" style={{ marginTop: 18, color: 'var(--mute)', fontFamily: 'var(--mono)', fontSize: 11, letterSpacing: '0.02em' }}>
            # terminal-inspired visual presets · не связаны с авторами оригинальных схем
          </p>
        </div>
      </div>
    </>
  );
}

// =============== SETTINGS / USERS ===============
const USERS_INIT = [
  { id: 'u1', name: 'Vitaly',       email: 'vitaly@example.com',   role: 'Владелец',    status: 'Активен' },
  { id: 'u2', name: 'Product Lead', email: 'product@example.com',  role: 'Участник',    status: 'Ожидает' },
  { id: 'u3', name: 'Reviewer',     email: 'reviewer@example.com', role: 'Наблюдатель', status: 'Активен' },
];

function statusPill(s) {
  if (s === 'Активен') return 'pass';
  if (s === 'Ожидает') return 'warn';
  return 'danger';
}
function rolePill(r) {
  if (r === 'Владелец') return 'accent';
  return '';
}
function initials(name) {
  return name.split(' ').map(s => s[0]).slice(0, 2).join('').toUpperCase();
}

function SettingsUsers({ onOpen, openDrawer }) {
  const [users, setUsers] = useState(USERS_INIT);
  const [draft, setDraft] = useState({ name: '', email: '', role: 'Участник', status: 'Ожидает' });
  const [editingId, setEditingId] = useState(null);

  function open(user) {
    if (user) {
      setEditingId(user.id);
      setDraft({ name: user.name, email: user.email, role: user.role, status: user.status });
    } else {
      setEditingId(null);
      setDraft({ name: '', email: '', role: 'Участник', status: 'Ожидает' });
    }
    onOpen(true);
  }
  function save() {
    if (editingId) {
      setUsers(us => us.map(u => u.id === editingId ? { ...u, ...draft } : u));
    } else {
      setUsers(us => [...us, { id: 'u' + (us.length + 1), ...draft }]);
    }
    onOpen(false);
  }

  return (
    <>
      <div className="surface-head">
        <div>
          <div className="kicker" style={{ marginBottom: 6 }}>settings · users</div>
          <h2>Настройки</h2>
        </div>
        <div className="meta"><span>{users.length} записи</span></div>
      </div>

      <div className="settings-layout">
        <nav className="subnav">
          <div className="kicker skicker">Раздел</div>
          <button>Оформление<span className="sm">UI</span></button>
          <button className="active">Пользователи<span className="sm">access</span></button>
        </nav>
        <div className="settings-content">
          <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', gap: 16, marginBottom: 6 }}>
            <div>
              <h3>Пользователи</h3>
              <p className="desc" style={{ marginBottom: 0 }}>Управление доступом к рабочему пространству.</p>
            </div>
            <button className="btn primary" onClick={() => open(null)}>{Ic.plus}<span>Добавить пользователя</span></button>
          </div>

          <div className="users-toolbar" style={{ marginTop: 18 }}>
            <div className="search">{Ic.search}<input placeholder="Поиск по имени или email" /></div>
            <button className="btn ghost">Все роли</button>
            <button className="btn ghost">Все статусы</button>
          </div>

          <div className="utable">
            <div className="urow head">
              <div>Имя</div><div>Email</div><div>Роль</div><div>Статус</div><div style={{ textAlign: 'right' }}>—</div>
            </div>
            {users.map(u => (
              <div className="urow" key={u.id}>
                <div className="uname"><div className="avatar">{initials(u.name)}</div>{u.name}</div>
                <div className="uemail">{u.email}</div>
                <div className="urole"><span className={"pill " + rolePill(u.role)}>{u.role}</span></div>
                <div><span className={"pill " + statusPill(u.status)}>{u.status}</span></div>
                <div className="uactions"><button className="icon-btn" onClick={() => open(u)} title="Редактировать">{Ic.edit}</button></div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {openDrawer && (
        <>
          <div className="scrim" onClick={() => onOpen(false)} />
          <div className="drawer">
            <div className="drawer-head">
              <div>
                <div className="kicker">{editingId ? 'access record' : 'workspace user'}</div>
                <h4>{editingId ? 'Редактировать пользователя' : 'Добавить пользователя'}</h4>
              </div>
              <button className="icon-btn" onClick={() => onOpen(false)}>{Ic.close}</button>
            </div>
            <div className="drawer-body">
              <div className="field">
                <label>Имя</label>
                <input value={draft.name} onChange={e => setDraft(d => ({ ...d, name: e.target.value }))} placeholder="Имя пользователя" />
              </div>
              <div className="field">
                <label>Email</label>
                <input value={draft.email} onChange={e => setDraft(d => ({ ...d, email: e.target.value }))} placeholder="user@example.com" />
              </div>
              <div className="field">
                <label>Роль</label>
                <select value={draft.role} onChange={e => setDraft(d => ({ ...d, role: e.target.value }))}>
                  <option>Владелец</option>
                  <option>Участник</option>
                  <option>Наблюдатель</option>
                </select>
              </div>
              <div className="field">
                <label>Статус</label>
                <select value={draft.status} onChange={e => setDraft(d => ({ ...d, status: e.target.value }))}>
                  <option>Активен</option>
                  <option>Ожидает</option>
                  <option>Отключен</option>
                </select>
              </div>
              <p style={{ marginTop: 14, color: 'var(--mute)', fontSize: 11.5, fontFamily: 'var(--mono)', letterSpacing: '0.02em' }}>
                # доступ применится после подключения server-backed flow
              </p>
            </div>
            <div className="drawer-foot">
              <button className="btn ghost" onClick={() => onOpen(false)}>Отмена</button>
              <button className="btn primary" onClick={save}>Сохранить</button>
            </div>
          </div>
        </>
      )}
    </>
  );
}

// =============== CONSOLE FRAME (composes everything) ===============
function ConsoleFrame({ initialSection = 'console', initialSurface = 'contracts', initialSettings = 'appearance', drawer = false, label }) {
  const [section, setSection] = useState(initialSection);
  const [surface, setSurface] = useState(initialSurface);
  const [settingsTab, setSettingsTab] = useState(initialSettings);
  const [themeId, setThemeId] = useState('goalrail');
  const [openDrawer, setOpenDrawer] = useState(drawer);

  const titleMap = {
    contracts: { crumb: 'console · contracts', title: 'Контракты' },
    readiness: { crumb: 'console · readiness', title: 'Оценка готовности' },
    proof:     { crumb: 'console · proof',     title: 'Проверка результата' },
  };
  const settingsTitle = { crumb: 'console · settings · ' + (settingsTab === 'appearance' ? 'appearance' : 'users'), title: 'Настройки' };
  const top = section === 'console' ? titleMap[surface] : settingsTitle;

  return (
    <div className="frame" data-screen-label={label} data-theme={themeId} style={{ position: 'relative' }}>
      <div className="shell">
        <Sidebar section={section} surface={surface} onSection={setSection} onSurface={setSurface} />
        <main className="main">
          <Topbar crumb={top.crumb} title={top.title} />
          {section === 'console' && surface === 'contracts' && <ContractsSurface />}
          {section === 'console' && surface === 'readiness' && <ReadinessSurface />}
          {section === 'console' && surface === 'proof' && <ProofSurface />}
          {section === 'settings' && (
            <div className="surface">
              {/* in settings, swap left subnav handles which view */}
              {settingsTab === 'appearance'
                ? <SettingsAppearanceWrap themeId={themeId} setThemeId={setThemeId} setSettingsTab={setSettingsTab} />
                : <SettingsUsersWrap setSettingsTab={setSettingsTab} openDrawer={openDrawer} setOpenDrawer={setOpenDrawer} />
              }
            </div>
          )}
        </main>
      </div>
    </div>
  );
}

// Wrappers that wire the subnav buttons to switch tabs
function SettingsAppearanceWrap({ themeId, setThemeId, setSettingsTab }) {
  return (
    <>
      <div className="surface-head">
        <div>
          <div className="kicker" style={{ marginBottom: 6 }}>settings · appearance</div>
          <h2>Настройки</h2>
        </div>
        <div className="meta"><span>preset · {themeId}</span></div>
      </div>
      <div className="settings-layout">
        <nav className="subnav">
          <div className="kicker skicker">Раздел</div>
          <button className="active" onClick={() => setSettingsTab('appearance')}>Оформление<span className="sm">UI</span></button>
          <button onClick={() => setSettingsTab('users')}>Пользователи<span className="sm">access</span></button>
        </nav>
        <div className="settings-content">
          <h3>Оформление</h3>
          <p className="desc">Выберите визуальный пресет консоли. Это влияет только на интерфейс, не на delivery logic.</p>
          <div className="theme-grid">
            {THEMES.map(t => <ThemeCard key={t.id} t={t} selected={themeId === t.id} onSelect={() => setThemeId(t.id)} />)}
          </div>
          <p style={{ marginTop: 18, color: 'var(--mute)', fontFamily: 'var(--mono)', fontSize: 11, letterSpacing: '0.02em' }}>
            # terminal-inspired visual presets · не связаны с авторами оригинальных схем
          </p>
        </div>
      </div>
    </>
  );
}

function SettingsUsersWrap({ setSettingsTab, openDrawer, setOpenDrawer }) {
  const [users, setUsers] = useState(USERS_INIT);
  const [editingId, setEditingId] = useState(null);
  const [draft, setDraft] = useState({ name: '', email: '', role: 'Участник', status: 'Ожидает' });

  function open(user) {
    if (user) {
      setEditingId(user.id);
      setDraft({ name: user.name, email: user.email, role: user.role, status: user.status });
    } else {
      setEditingId(null);
      setDraft({ name: '', email: '', role: 'Участник', status: 'Ожидает' });
    }
    setOpenDrawer(true);
  }
  function save() {
    if (!draft.name || !draft.email) { setOpenDrawer(false); return; }
    if (editingId) setUsers(us => us.map(u => u.id === editingId ? { ...u, ...draft } : u));
    else setUsers(us => [...us, { id: 'u' + (us.length + 1), ...draft }]);
    setOpenDrawer(false);
  }

  return (
    <>
      <div className="surface-head">
        <div>
          <div className="kicker" style={{ marginBottom: 6 }}>settings · users</div>
          <h2>Настройки</h2>
        </div>
        <div className="meta"><span>{users.length} записи</span></div>
      </div>

      <div className="settings-layout">
        <nav className="subnav">
          <div className="kicker skicker">Раздел</div>
          <button onClick={() => setSettingsTab('appearance')}>Оформление<span className="sm">UI</span></button>
          <button className="active" onClick={() => setSettingsTab('users')}>Пользователи<span className="sm">access</span></button>
        </nav>
        <div className="settings-content">
          <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', gap: 16, marginBottom: 6 }}>
            <div>
              <h3>Пользователи</h3>
              <p className="desc" style={{ marginBottom: 0 }}>Управление доступом к рабочему пространству.</p>
            </div>
            <button className="btn primary" onClick={() => open(null)}>{Ic.plus}<span>Добавить пользователя</span></button>
          </div>

          <div className="users-toolbar" style={{ marginTop: 18 }}>
            <div className="search">{Ic.search}<input placeholder="Поиск по имени или email" /></div>
            <button className="btn ghost">Все роли</button>
            <button className="btn ghost">Все статусы</button>
          </div>

          <div className="utable">
            <div className="urow head">
              <div>Имя</div><div>Email</div><div>Роль</div><div>Статус</div><div style={{ textAlign: 'right' }}>—</div>
            </div>
            {users.map(u => (
              <div className="urow" key={u.id}>
                <div className="uname"><div className="avatar">{initials(u.name)}</div>{u.name}</div>
                <div className="uemail">{u.email}</div>
                <div className="urole"><span className={"pill " + rolePill(u.role)}>{u.role}</span></div>
                <div><span className={"pill " + statusPill(u.status)}>{u.status}</span></div>
                <div className="uactions"><button className="icon-btn" onClick={() => open(u)}>{Ic.edit}</button></div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {openDrawer && (
        <>
          <div className="scrim" onClick={() => setOpenDrawer(false)} />
          <div className="drawer">
            <div className="drawer-head">
              <div>
                <div className="kicker">{editingId ? 'access record' : 'workspace user'}</div>
                <h4>{editingId ? 'Редактировать пользователя' : 'Добавить пользователя'}</h4>
              </div>
              <button className="icon-btn" onClick={() => setOpenDrawer(false)}>{Ic.close}</button>
            </div>
            <div className="drawer-body">
              <div className="field">
                <label>Имя</label>
                <input value={draft.name} onChange={e => setDraft(d => ({ ...d, name: e.target.value }))} placeholder="Имя пользователя" />
              </div>
              <div className="field">
                <label>Email</label>
                <input value={draft.email} onChange={e => setDraft(d => ({ ...d, email: e.target.value }))} placeholder="user@example.com" />
              </div>
              <div className="field">
                <label>Роль</label>
                <select value={draft.role} onChange={e => setDraft(d => ({ ...d, role: e.target.value }))}>
                  <option>Владелец</option>
                  <option>Участник</option>
                  <option>Наблюдатель</option>
                </select>
              </div>
              <div className="field">
                <label>Статус</label>
                <select value={draft.status} onChange={e => setDraft(d => ({ ...d, status: e.target.value }))}>
                  <option>Активен</option>
                  <option>Ожидает</option>
                  <option>Отключен</option>
                </select>
              </div>
              <p style={{ marginTop: 14, color: 'var(--mute)', fontSize: 11.5, fontFamily: 'var(--mono)', letterSpacing: '0.02em' }}>
                # доступ применится после подключения server-backed flow
              </p>
            </div>
            <div className="drawer-foot">
              <button className="btn ghost" onClick={() => setOpenDrawer(false)}>Отмена</button>
              <button className="btn primary" onClick={save}>Сохранить</button>
            </div>
          </div>
        </>
      )}
    </>
  );
}

// =============== MOBILE ===============
function MobileFrame() {
  const [tab, setTab] = useState('contracts');
  return (
    <div className="frame" data-screen-label="08 Mobile">
      <div className="mob">
        <div className="mob-top">
          <Brand />
          <div className="right">
            <button className="icon-btn" title="Настройки">{Ic.gear}</button>
            <button className="icon-btn" title="Меню">{Ic.menu}</button>
          </div>
        </div>
        <div className="mob-tabs">
          <div className={"mob-tab" + (tab === 'contracts' ? " active" : "")} onClick={() => setTab('contracts')}>Контракты</div>
          <div className={"mob-tab" + (tab === 'readiness' ? " active" : "")} onClick={() => setTab('readiness')}>Готовность</div>
          <div className={"mob-tab" + (tab === 'proof' ? " active" : "")} onClick={() => setTab('proof')}>Проверка</div>
        </div>
        <div className="mob-body">
          {tab === 'contracts' && <>
            <div className="kicker" style={{ marginBottom: 6 }}>console · trialops · prod</div>
            <h3 className="mob-h">Контракты</h3>
            <p className="mob-lede">Контур поставки появится здесь после подключения server-backed flow.</p>
            <div className="mob-card">
              <div className="row1"><div className="id">C-0000</div><span className="pill">placeholder</span></div>
              <h5>Карточка контракта</h5>
              <p>Goal → Contract → Task → Proof. Структура зафиксирована, исполнение ожидается.</p>
            </div>
            <div className="mob-card" style={{ borderStyle: 'dashed', opacity: 0.65 }}>
              <div className="row1"><div className="id">— — — —</div><span className="pill no-dot mono" style={{ color: 'var(--mute-2)' }}>repo · —</span></div>
              <div style={{ height: 8, background: 'color-mix(in oklab, var(--ink) 6%, var(--panel-2))', borderRadius: 2, width: '60%', marginBottom: 6 }} />
              <div style={{ height: 8, background: 'color-mix(in oklab, var(--ink) 6%, var(--panel-2))', borderRadius: 2, width: '40%' }} />
            </div>
          </>}
          {tab === 'readiness' && <>
            <h3 className="mob-h">Оценка готовности</h3>
            <p className="mob-lede">Структурный обзор недостающего контекста, ограничений и критериев приемки.</p>
            {['Контекст','Ограничения','Критерии приемки','Риски'].map(n => (
              <div key={n} className="mob-card">
                <div className="row1"><div className="id mono">{n}</div><span className="pill">queued</span></div>
                <p>Сигналы появятся после подключения контура.</p>
              </div>
            ))}
          </>}
          {tab === 'proof' && <>
            <h3 className="mob-h">Проверка результата</h3>
            <p className="mob-lede">Четыре дорожки proof: scope, integrity, policy, target.</p>
            {['Scope','Integrity','Policy','Target'].map(n => (
              <div key={n} className="mob-card">
                <div className="row1"><div className="id mono">{n}</div><span className="pill">empty</span></div>
                <p>Без proof результат не считается принятым.</p>
              </div>
            ))}
          </>}
        </div>
        <div className="mob-utility">
          <button className="utility-btn" style={{ flex: 1, minHeight: 44 }}>{Ic.gear}<span>Настройки</span></button>
          <div style={{ fontFamily: 'var(--mono)', fontSize: 9.5, color: 'var(--mute-2)', letterSpacing: '0.10em', textTransform: 'uppercase' }}>console preview</div>
        </div>
      </div>
    </div>
  );
}

// =============== DESIGN SYSTEM FRAME ===============
function DesignSystemFrame() {
  const tokens = [
    ['Background','--bg'], ['Surface','--bg-2'], ['Surface elevated','--panel-2'],
    ['Border','--line'], ['Soft border','--line-soft'], ['Primary text','--ink'],
    ['Secondary text','--ink-2'], ['Muted text','--mute'], ['Accent','--accent'],
    ['Accent soft','--accent-soft'], ['Success','--pass'], ['Warning','--warn'], ['Danger','--danger'],
  ];

  return (
    <div className="frame" data-screen-label="09 Design system" style={{ overflow: 'auto' }}>
      <div style={{ padding: '32px 36px 40px', display: 'flex', flexDirection: 'column', gap: 26 }}>
        <header>
          <div className="kicker" style={{ marginBottom: 6 }}>design system</div>
          <h2 style={{ margin: 0, fontSize: 22, fontWeight: 600 }}>Goalrail Console — visual primitives</h2>
          <p style={{ color: 'var(--ink-2)', maxWidth: 720, marginTop: 6 }}>
            Контейнер дизайн-системы для команды разработчиков. Premium, restrained, terminal-adjacent.
          </p>
        </header>

        <section>
          <div className="kicker" style={{ marginBottom: 10 }}>1 · Typography</div>
          <div className="type-stack">
            <div className="type-row"><div className="t-meta">UI · Inter 22 / 600</div><div style={{ fontSize: 22, fontWeight: 600, letterSpacing: '-0.01em' }}>Контракт-первый контур</div></div>
            <div className="type-row"><div className="t-meta">UI · Inter 15 / 600</div><div style={{ fontSize: 15, fontWeight: 600 }}>Surface header</div></div>
            <div className="type-row"><div className="t-meta">UI · Inter 13 / 400</div><div style={{ fontSize: 13 }}>Body — calm copy для описаний и hints.</div></div>
            <div className="type-row"><div className="t-meta">Mono · JetBrains 10 / 0.14em</div><div style={{ fontFamily: 'var(--mono)', fontSize: 10, letterSpacing: '0.14em', textTransform: 'uppercase', color: 'var(--mute)' }}>console · contracts · ws-01</div></div>
            <div className="type-row"><div className="t-meta">Mono · IDs</div><div style={{ fontFamily: 'var(--mono)', fontSize: 11.5, letterSpacing: '0.04em', color: 'var(--ink-2)' }}>C-0147 · PF-0091 · ws-01</div></div>
          </div>
        </section>

        <section>
          <div className="kicker" style={{ marginBottom: 10 }}>2 · Color tokens</div>
          <div className="ds-tokens">
            {tokens.map(([n,v]) => (
              <div className="token" key={v}>
                <div className="sw" style={{ background: `var(${v})` }} />
                <div className="info"><div className="n">{n}</div><div className="v">{v}</div></div>
              </div>
            ))}
          </div>
        </section>

        <section>
          <div className="kicker" style={{ marginBottom: 10 }}>3 · Spacing & layout</div>
          <div className="ds-grid">
            <div className="component-cell">
              <div className="label">Sidebar width</div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <div style={{ width: 232, height: 60, border: '1px solid var(--line)', borderRadius: 4, background: 'var(--bg-2)', display: 'grid', placeItems: 'center', fontFamily: 'var(--mono)', fontSize: 10, color: 'var(--mute)', letterSpacing: '0.1em' }}>232 PX</div>
                <span className="kicker">desktop</span>
              </div>
            </div>
            <div className="component-cell">
              <div className="label">Whitespace</div>
              <p style={{ color: 'var(--ink-2)', margin: 0 }}>Surface: 28/32 padding · Cards: 14–18 · Lists: 10–14 row gap. Compact но не плотный.</p>
            </div>
          </div>
        </section>

        <section>
          <div className="kicker" style={{ marginBottom: 10 }}>4 · Components</div>
          <div className="component-grid">
            <div className="component-cell">
              <div className="label">Sidebar nav item</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <div className="nav-item active" style={{ width: 220 }}><span className="nav-ico">{Ic.contract}</span><span>Контракты</span></div>
                <div className="nav-item" style={{ width: 220 }}><span className="nav-ico">{Ic.check}</span><span>Оценка готовности</span></div>
              </div>
            </div>
            <div className="component-cell">
              <div className="label">Utility settings button</div>
              <div className="utility-btn" style={{ width: 220 }}><span className="nav-ico">{Ic.gear}</span><span>Настройки</span></div>
            </div>
            <div className="component-cell">
              <div className="label">Inputs</div>
              <div className="field" style={{ marginBottom: 0 }}>
                <label>Email</label>
                <input defaultValue="user@example.com" />
              </div>
              <div className="field error" style={{ marginBottom: 0 }}>
                <label>Пароль</label>
                <input defaultValue="" placeholder="•••••••" />
                <div className="field-msg">Введите пароль для продолжения.</div>
              </div>
            </div>
            <div className="component-cell">
              <div className="label">Buttons</div>
              <div className="btn-row">
                <button className="btn primary">Войти</button>
                <button className="btn">Отмена</button>
                <button className="btn ghost">Ghost</button>
                <button className="btn danger">Block</button>
              </div>
            </div>
            <div className="component-cell">
              <div className="label">Status pills</div>
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                <span className="pill accent">Владелец</span>
                <span className="pill pass">Активен</span>
                <span className="pill warn">Ожидает</span>
                <span className="pill danger">Отключен</span>
                <span className="pill">placeholder</span>
              </div>
            </div>
            <div className="component-cell">
              <div className="label">Theme card (compact)</div>
              <div style={{ width: 220 }}>
                <ThemeCard t={THEMES[0]} selected onSelect={() => {}} />
              </div>
            </div>
            <div className="component-cell" style={{ gridColumn: '1 / -1' }}>
              <div className="label">User table row</div>
              <div className="utable">
                <div className="urow">
                  <div className="uname"><div className="avatar">VK</div>Vitaly</div>
                  <div className="uemail">vitaly@example.com</div>
                  <div><span className="pill accent">Владелец</span></div>
                  <div><span className="pill pass">Активен</span></div>
                  <div className="uactions"><button className="icon-btn">{Ic.edit}</button></div>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section>
          <div className="kicker" style={{ marginBottom: 10 }}>5 · Rationale</div>
          <div className="rationale">
            <div className="item"><h6><span className="num">01</span>Только три поверхности</h6><p>Контракты — Готовность — Проверка результата. Это и есть продуктовая дуга Goalrail: от запроса к контракту, от контракта к исполнению, от исполнения к proof. Любая четвёртая вкладка размывает посыл «контракт-первый».</p></div>
            <div className="item"><h6><span className="num">02</span>Настройки внизу слева</h6><p>Settings — это утилита workspace, а не продуктовая поверхность. Нижне-левое расположение — устоявшаяся метафора утилитарных контролов терминал-консолей: всегда под рукой, никогда не конкурирует за внимание с рабочей областью.</p></div>
            <div className="item"><h6><span className="num">03</span>Логин без регистрации</h6><p>Доступ выдаёт администратор рабочего пространства. Регистрация и SSO здесь — лишняя поверхность, которая обещает самообслуживание там, где его нет. Минимальный экран — два поля и одна кнопка — задаёт тон всей консоли.</p></div>
            <div className="item"><h6><span className="num">04</span>Именованные terminal-пресеты</h6><p>Именованные пресеты — Goalrail Default, Mocha, Dracula, Nord, Solarized, Gruvbox — обращаются к привычкам разработчиков и operator-аудитории. Бинарный light/dark не передаёт настроения долгого рабочего сеанса.</p></div>
            <div className="item"><h6><span className="num">05</span>Пользователи внутри Settings</h6><p>Управление доступом — административная задача, не часть delivery loop. Размещение Users рядом с Appearance честно говорит: это конфигурация рабочего пространства, а не продукт.</p></div>
            <div className="item"><h6><span className="num">06</span>Что мы не показываем</h6><p>Никаких фейковых метрик, скан-результатов, чат-боксов, моделей и аватаров ассистентов. Консоль показывает структуру и контроль, а не «AI-магию». Пустые состояния честны: shell готов, contour ожидается.</p></div>
          </div>
        </section>
      </div>
    </div>
  );
}

// =============== CANVAS LAYOUT ===============
function App() {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerOpen2, setDrawerOpen2] = useState(true); // for the dedicated drawer frame

  return (
    <DesignCanvas title="Goalrail Console — v0.4">
      <DCSection id="auth" title="01 · Login">
        <DCArtboard id="login" label="Login screen" width={1280} height={820}>
          <LoginFrame />
        </DCArtboard>
        <DCArtboard id="login-error" label="Login — validation error" width={1280} height={820}>
          <LoginFrame withError />
        </DCArtboard>
      </DCSection>

      <DCSection id="console" title="02 · Console shell — three product surfaces">
        <DCArtboard id="contracts" label="Контракты (selected)" width={1320} height={820}>
          <ConsoleFrame initialSection="console" initialSurface="contracts" label="02 Контракты" />
        </DCArtboard>
        <DCArtboard id="readiness" label="Оценка готовности" width={1320} height={820}>
          <ConsoleFrame initialSection="console" initialSurface="readiness" label="03 Готовность" />
        </DCArtboard>
        <DCArtboard id="proof" label="Проверка результата" width={1320} height={820}>
          <ConsoleFrame initialSection="console" initialSurface="proof" label="04 Проверка" />
        </DCArtboard>
      </DCSection>

      <DCSection id="settings" title="03 · Settings — utility, not a fourth surface">
        <DCArtboard id="appearance" label="Settings · Оформление" width={1320} height={820}>
          <ConsoleFrame initialSection="settings" initialSettings="appearance" label="05 Оформление" />
        </DCArtboard>
        <DCArtboard id="users" label="Settings · Пользователи" width={1320} height={820}>
          <ConsoleFrame initialSection="settings" initialSettings="users" label="06 Пользователи" />
        </DCArtboard>
        <DCArtboard id="user-drawer" label="Add / Edit user — drawer" width={1320} height={820}>
          <ConsoleFrame initialSection="settings" initialSettings="users" drawer={true} label="07 Drawer" />
        </DCArtboard>
      </DCSection>

      <DCSection id="mobile" title="04 · Mobile direction">
        <DCArtboard id="mobile-console" label="Console · mobile (390)" width={390} height={820}>
          <MobileFrame />
        </DCArtboard>
        <DCArtboard id="mobile-login" label="Login · mobile (390)" width={390} height={820}>
          <LoginFrame />
        </DCArtboard>
      </DCSection>

      <DCSection id="ds" title="05 · Design system & rationale">
        <DCArtboard id="ds-board" label="Design system" width={1320} height={1500}>
          <DesignSystemFrame />
        </DCArtboard>
      </DCSection>
    </DesignCanvas>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(<App />);
