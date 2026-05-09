import type { ReactNode } from 'react';

import { MantineProvider } from '@mantine/core';
import { render as testingLibraryRender } from '@testing-library/react';

import { theme } from '../src/theme';

interface RenderOptions {
  children?: ReactNode;
}

function normalizedPathname() {
  return window.location.pathname.replace(/\/+$/, '') || '/';
}

function isPublicStartSurfaceRoute() {
  const pathname = normalizedPathname();
  return pathname === '/' || pathname === '/start';
}

export function render(ui: ReactNode) {
  if (isPublicStartSurfaceRoute()) {
    return testingLibraryRender(<>{ui}</>);
  }

  return testingLibraryRender(<>{ui}</>, {
    wrapper: ({ children }: RenderOptions) => (
      <MantineProvider env="test" theme={theme}>
        {children}
      </MantineProvider>
    ),
  });
}
