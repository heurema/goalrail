import { useState } from 'react';

import './App.css';

type SurfaceId = 'contracts' | 'delivery-readiness' | 'proof';

interface SurfaceItem {
  id: SurfaceId;
  label: string;
}

const SURFACES: SurfaceItem[] = [
  { id: 'contracts', label: 'Contracts' },
  { id: 'delivery-readiness', label: 'Delivery Readiness' },
  { id: 'proof', label: 'Proof' },
];

function App() {
  const [activeSurface, setActiveSurface] = useState<SurfaceId>('contracts');
  const activeLabel = SURFACES.find((surface) => surface.id === activeSurface)?.label ?? 'Contracts';

  return (
    <main className="consoleShell" data-deployment-target="console.goalrail.dev">
      <aside className="sidebar" aria-label="Goalrail console navigation">
        <Brand />

        <nav className="surfaceNav" aria-label="Product surfaces">
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

      <section className="emptySurface" aria-label={`${activeLabel} surface empty`} />
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
