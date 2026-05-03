import { FormEvent, useMemo, useState } from 'react';

import './App.css';

type SurfaceId = 'contracts' | 'delivery-readiness' | 'proof';
type ScreenId = 'console' | 'settings-users';
type UserStatus = 'Active' | 'Pending' | 'Disabled';
type UserRole = 'Owner' | 'Member' | 'Observer';
type RoleFilter = UserRole | 'all';
type StatusFilter = UserStatus | 'all';

interface SurfaceItem {
  id: SurfaceId;
  label: string;
}

interface ConsoleUser {
  id: string;
  name: string;
  email: string;
  role: UserRole;
  status: UserStatus;
}

const SURFACES: SurfaceItem[] = [
  { id: 'contracts', label: 'Contracts' },
  { id: 'delivery-readiness', label: 'Delivery Readiness' },
  { id: 'proof', label: 'Proof' },
];

const INITIAL_USERS: ConsoleUser[] = [
  { id: 'u1', name: 'Owner', email: 'owner@example.com', role: 'Owner', status: 'Active' },
  { id: 'u2', name: 'Product Lead', email: 'product@example.com', role: 'Member', status: 'Pending' },
  { id: 'u3', name: 'Reviewer', email: 'reviewer@example.com', role: 'Observer', status: 'Active' },
];

const EMPTY_DRAFT: Omit<ConsoleUser, 'id'> = {
  name: '',
  email: '',
  role: 'Member',
  status: 'Pending',
};

function initials(name: string) {
  return name
    .split(' ')
    .filter(Boolean)
    .map((part) => part[0])
    .slice(0, 2)
    .join('')
    .toUpperCase();
}

function statusClass(status: UserStatus) {
  if (status === 'Active') {
    return 'statusActive';
  }

  if (status === 'Pending') {
    return 'statusPending';
  }

  return 'statusDisabled';
}

function App() {
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [loginError, setLoginError] = useState('');
  const [activeSurface, setActiveSurface] = useState<SurfaceId>('contracts');
  const [screen, setScreen] = useState<ScreenId>('console');
  const [users, setUsers] = useState<ConsoleUser[]>(INITIAL_USERS);
  const [searchQuery, setSearchQuery] = useState('');
  const [roleFilter, setRoleFilter] = useState<RoleFilter>('all');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draft, setDraft] = useState<Omit<ConsoleUser, 'id'>>(EMPTY_DRAFT);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);

  const activeLabel = SURFACES.find((surface) => surface.id === activeSurface)?.label ?? 'Contracts';
  const drawerTitle = editingId ? 'Edit user' : 'Add user';

  function handleLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const email = String(form.get('email') ?? '').trim();
    const password = String(form.get('password') ?? '').trim();

    if (!email || !password) {
      setLoginError('Enter email and password to continue.');
      return;
    }

    setLoginError('');
    setIsLoggedIn(true);
  }

  function openNewUser() {
    setEditingId(null);
    setDraft(EMPTY_DRAFT);
    setIsDrawerOpen(true);
  }

  function openExistingUser(user: ConsoleUser) {
    setEditingId(user.id);
    setDraft({
      name: user.name,
      email: user.email,
      role: user.role,
      status: user.status,
    });
    setIsDrawerOpen(true);
  }

  function closeDrawer() {
    setIsDrawerOpen(false);
  }

  function saveUser() {
    const nextDraft = {
      ...draft,
      name: draft.name.trim(),
      email: draft.email.trim(),
    };

    if (!nextDraft.name || !nextDraft.email) {
      return;
    }

    if (editingId) {
      setUsers((currentUsers) =>
        currentUsers.map((user) => (user.id === editingId ? { ...user, ...nextDraft } : user))
      );
    } else {
      setUsers((currentUsers) => [
        ...currentUsers,
        {
          id: `u${currentUsers.length + 1}`,
          ...nextDraft,
        },
      ]);
    }

    setIsDrawerOpen(false);
  }

  const visibleUsers = useMemo(() => {
    const normalizedQuery = searchQuery.trim().toLowerCase();

    return users.filter((user) => {
      const matchesQuery =
        !normalizedQuery ||
        user.name.toLowerCase().includes(normalizedQuery) ||
        user.email.toLowerCase().includes(normalizedQuery);
      const matchesRole = roleFilter === 'all' || user.role === roleFilter;
      const matchesStatus = statusFilter === 'all' || user.status === statusFilter;

      return matchesQuery && matchesRole && matchesStatus;
    });
  }, [roleFilter, searchQuery, statusFilter, users]);

  const userRows = useMemo(
    () =>
      visibleUsers.map((user) => (
        <tr className="userRow" key={user.id}>
          <td>
            <div className="userName">
              <span className="avatar" aria-hidden="true">
                {initials(user.name)}
              </span>
              <span>{user.name}</span>
            </div>
          </td>
          <td className="userEmail">{user.email}</td>
          <td>
            <span className={user.role === 'Owner' ? 'pill roleOwner' : 'pill'}>{user.role}</span>
          </td>
          <td>
            <span className={`pill ${statusClass(user.status)}`}>{user.status}</span>
          </td>
          <td>
            <div className="userActions">
              <button className="iconButton" onClick={() => openExistingUser(user)} type="button">
                <span aria-hidden="true">✎</span>
                <span className="srOnly">Edit {user.name}</span>
              </button>
            </div>
          </td>
        </tr>
      )),
    [visibleUsers]
  );

  if (!isLoggedIn) {
    return (
      <main className="loginScreen" data-deployment-target="console.goalrail.dev">
        <div className="loginRails" aria-hidden="true" />
        <form className="loginCard" onSubmit={handleLogin}>
          <Brand />

          <label className="field">
            <span>Email</span>
            <input autoComplete="email" name="email" placeholder="name@example.com" type="email" />
          </label>

          <label className={loginError ? 'field fieldError' : 'field'}>
            <span>Password</span>
            <input autoComplete="current-password" name="password" type="password" />
          </label>

          {loginError ? <p className="fieldMessage">{loginError}</p> : null}

          <button className="primaryButton fullWidth" type="submit">
            Sign in
            <span aria-hidden="true">→</span>
          </button>
        </form>
      </main>
    );
  }

  return (
    <main className="consoleShell" data-deployment-target="console.goalrail.dev">
      <aside className="sidebar" aria-label="Goalrail console navigation">
        <Brand />

        <nav className="surfaceNav" aria-label="Product surfaces">
          {SURFACES.map((surface) => (
            <button
              aria-current={screen === 'console' && activeSurface === surface.id ? 'page' : undefined}
              className={screen === 'console' && activeSurface === surface.id ? 'surfaceButton active' : 'surfaceButton'}
              key={surface.id}
              onClick={() => {
                setActiveSurface(surface.id);
                setScreen('console');
              }}
              type="button"
            >
              {surface.label}
            </button>
          ))}
        </nav>

        <div className="sidebarSpacer" />

        <div className="settingsDock">
          <button
            aria-current={screen === 'settings-users' ? 'page' : undefined}
            className={screen === 'settings-users' ? 'settingsButton active' : 'settingsButton'}
            onClick={() => setScreen('settings-users')}
            type="button"
          >
            <span aria-hidden="true">⚙</span>
            <span>Settings</span>
          </button>
        </div>
      </aside>

      {screen === 'console' ? (
        <section className="emptySurface" aria-label={`${activeLabel}: empty surface`} />
      ) : (
        <section className="settingsSurface" aria-label="Settings: users">
          <header className="surfaceHeader">
            <div>
              <p className="kicker">settings · users</p>
              <h2>Settings</h2>
            </div>
            <p className="metaText">{visibleUsers.length} records</p>
          </header>

          <div className="settingsContent">
            <div className="usersHeader">
              <div>
                <h3>Users</h3>
                <p>Manage workspace access.</p>
              </div>
              <button aria-label="Add user" className="primaryButton" onClick={openNewUser} type="button">
                <span aria-hidden="true">+</span>
                <span>Add</span>
              </button>
            </div>

            <div className="usersToolbar">
              <label className="searchBox">
                <span aria-hidden="true">⌕</span>
                <input
                  aria-label="Search users"
                  onChange={(event) => setSearchQuery(event.target.value)}
                  placeholder="Search by name or email"
                  type="search"
                  value={searchQuery}
                />
              </label>
              <label className="filterBox">
                <span>Role</span>
                <select
                  aria-label="Filter by role"
                  onChange={(event) => setRoleFilter(event.target.value as RoleFilter)}
                  value={roleFilter}
                >
                  <option value="all">All roles</option>
                  <option value="Owner">Owner</option>
                  <option value="Member">Member</option>
                  <option value="Observer">Observer</option>
                </select>
              </label>
              <label className="filterBox">
                <span>Status</span>
                <select
                  aria-label="Filter by status"
                  onChange={(event) => setStatusFilter(event.target.value as StatusFilter)}
                  value={statusFilter}
                >
                  <option value="all">All statuses</option>
                  <option value="Active">Active</option>
                  <option value="Pending">Pending</option>
                  <option value="Disabled">Disabled</option>
                </select>
              </label>
            </div>

            <div className="usersTableFrame">
              <table className="usersTable" aria-label="Workspace users">
                <thead>
                  <tr className="userRow userHead">
                    <th scope="col">Name</th>
                    <th scope="col">Email</th>
                    <th scope="col">Role</th>
                    <th scope="col">Status</th>
                    <th scope="col" aria-label="Actions" />
                  </tr>
                </thead>
                <tbody>
                  {userRows}
                  {visibleUsers.length === 0 ? (
                    <tr>
                      <td className="emptyUsers" colSpan={5}>
                        No users match the selected filters.
                      </td>
                    </tr>
                  ) : null}
                </tbody>
              </table>
            </div>
          </div>
        </section>
      )}

      {isDrawerOpen ? (
        <>
          <button aria-label="Close user form" className="drawerScrim" onClick={closeDrawer} type="button" />
          <aside className="drawer" aria-label={drawerTitle}>
            <header className="drawerHeader">
              <div>
                <p className="kicker">{editingId ? 'access record' : 'workspace user'}</p>
                <h3>{drawerTitle}</h3>
              </div>
              <button className="iconButton" onClick={closeDrawer} type="button">
                <span aria-hidden="true">×</span>
                <span className="srOnly">Close</span>
              </button>
            </header>

            <div className="drawerBody">
              <label className="field">
                <span>Name</span>
                <input
                  onChange={(event) => setDraft((currentDraft) => ({ ...currentDraft, name: event.target.value }))}
                  placeholder="User name"
                  value={draft.name}
                />
              </label>

              <label className="field">
                <span>Email</span>
                <input
                  onChange={(event) => setDraft((currentDraft) => ({ ...currentDraft, email: event.target.value }))}
                  placeholder="user@example.com"
                  type="email"
                  value={draft.email}
                />
              </label>

              <label className="field">
                <span>Role</span>
                <select
                  onChange={(event) =>
                    setDraft((currentDraft) => ({ ...currentDraft, role: event.target.value as UserRole }))
                  }
                  value={draft.role}
                >
                  <option>Owner</option>
                  <option>Member</option>
                  <option>Observer</option>
                </select>
              </label>

              <label className="field">
                <span>Status</span>
                <select
                  onChange={(event) =>
                    setDraft((currentDraft) => ({ ...currentDraft, status: event.target.value as UserStatus }))
                  }
                  value={draft.status}
                >
                  <option>Active</option>
                  <option>Pending</option>
                  <option>Disabled</option>
                </select>
              </label>
            </div>

            <footer className="drawerFooter">
              <button className="ghostButton" onClick={closeDrawer} type="button">
                Cancel
              </button>
              <button className="primaryButton" onClick={saveUser} type="button">
                Save
              </button>
            </footer>
          </aside>
        </>
      ) : null}
    </main>
  );
}

function Brand() {
  return (
    <div className="brand" aria-label="Goalrail console">
      <span className="brandText">GOALRAIL</span>
    </div>
  );
}

export default App;
