import { useState } from 'react';

import './App.css';

type SurfaceId = 'contracts' | 'delivery-readiness' | 'proof';

interface SurfaceItem {
  id: SurfaceId;
  label: string;
}

const SURFACES: SurfaceItem[] = [
  { id: 'contracts', label: 'Контракты' },
  { id: 'delivery-readiness', label: 'Готовность доставки' },
  { id: 'proof', label: 'Доказательства' },
];

function App() {
  const [activeSurface, setActiveSurface] = useState<SurfaceId>('contracts');
  const activeLabel = SURFACES.find((surface) => surface.id === activeSurface)?.label ?? 'Контракты';

  return (
    <main className="consoleShell" data-deployment-target="console.goalrail.ru">
      <aside className="sidebar" aria-label="Навигация консоли Goalrail">
        <div className="brand" aria-label="Консоль Goalrail">
          <span className="brandMark" aria-hidden="true">
            <span />
            <span />
            <span />
          </span>
          <span className="brandText">Goalrail</span>
        </div>

        <nav className="surfaceNav" aria-label="Разделы продукта">
          {SURFACES.map((surface) => (
            <button
              aria-current={activeSurface === surface.id ? 'page' : undefined}
              className={activeSurface === surface.id ? 'surfaceButton active' : 'surfaceButton'}
              key={surface.id}
              onClick={() => setActiveSurface(surface.id)}
              type="button"
            >
              {surface.label}
            </button>
          ))}
        </nav>
      </aside>

      <section className="emptySurface" aria-label={`${activeLabel}: пустой раздел`} />
    </main>
  );
}

export default App;
