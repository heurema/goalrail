import type { ReactNode } from 'react';

import { MantineProvider } from '@mantine/core';
import { render as testingLibraryRender } from '@testing-library/react';

import { theme } from '../src/theme';

interface RenderOptions {
  children?: ReactNode;
}

export function render(ui: ReactNode) {
  return testingLibraryRender(<>{ui}</>, {
    wrapper: ({ children }: RenderOptions) => (
      <MantineProvider env="test" theme={theme}>
        {children}
      </MantineProvider>
    ),
  });
}
