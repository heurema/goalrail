import '@mantine/core/styles.css';

import React from 'react';
import ReactDOM from 'react-dom/client';
import { MantineProvider } from '@mantine/core';

import App from './App';
import './i18n';
import { theme } from './theme';

function isStartRoute() {
  return window.location.pathname.replace(/\/+$/, '') === '/start';
}

const app = <App />;

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    {isStartRoute() ? app : <MantineProvider theme={theme}>{app}</MantineProvider>}
  </React.StrictMode>
);
