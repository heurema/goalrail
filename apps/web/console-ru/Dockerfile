# syntax=docker/dockerfile:1

FROM node:24 AS build

ARG VITE_GOALRAIL_API_BASE_URL
ENV VITE_GOALRAIL_API_BASE_URL=${VITE_GOALRAIL_API_BASE_URL}

WORKDIR /src
COPY apps/web/ ./apps/web/

WORKDIR /src/apps/web
RUN npm ci --no-audit --no-fund
RUN LIBC="$(node -p "(() => { const fs = require('node:fs'); const report = process.report?.getReport?.(); if (report?.header?.glibcVersionRuntime) return 'gnu'; try { return fs.readFileSync('/usr/bin/ldd', 'utf8').includes('musl') ? 'musl' : 'gnu'; } catch { return 'gnu'; } })()")" \
  && ARCH="$(node -p 'process.arch')" \
  && ROLLDOWN_VERSION="$(node -p "require('./node_modules/rolldown/package.json').version")" \
  && npm install --no-save --package-lock=false "@rolldown/binding-linux-${ARCH}-${LIBC}@${ROLLDOWN_VERSION}"
RUN npm run console-ru:build

FROM nginxinc/nginx-unprivileged:1.29-alpine

COPY --from=build /src/apps/web/console-ru/dist/ /usr/share/nginx/html/

EXPOSE 8080
