# syntax=docker/dockerfile:1.7
FROM node:20-bookworm-slim AS builder
WORKDIR /app/Frontend

COPY Frontend/package.json Frontend/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm_ci_start="$(date -u +%s)" \
    && npm ci \
    && npm_ci_end="$(date -u +%s)" \
    && echo "TIMING frontend_npm_ci_seconds=$((npm_ci_end - npm_ci_start))"

COPY Frontend/ ./
ARG VITE_APP_VERSION=preview
ARG VITE_BUILD_REF=preview
ARG VITE_APP_ENV=preview
ARG VITE_BASE_PATH=/
ARG VITE_API_BASE_URL=
ENV VITE_APP_VERSION=${VITE_APP_VERSION}
ENV VITE_BUILD_REF=${VITE_BUILD_REF}
ENV VITE_APP_ENV=${VITE_APP_ENV}
ENV VITE_BASE_PATH=${VITE_BASE_PATH}
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
RUN --mount=type=cache,target=/app/Frontend/.preview-cache/tsc,sharing=private \
    --mount=type=cache,target=/app/Frontend/node_modules/.vite,sharing=private \
    vite_start="$(date -u +%s)" \
    && ./node_modules/.bin/tsc -b tsconfig.preview.json \
    && ./node_modules/.bin/vite build \
    && vite_end="$(date -u +%s)" \
    && echo "TIMING frontend_vite_build_seconds=$((vite_end - vite_start))"

FROM nginx:alpine

COPY --from=builder /app/Frontend/dist /usr/share/nginx/html
COPY Frontend/nginx.conf /etc/nginx/templates/default.conf.template

ENV BACKEND_UPSTREAM_HOST=cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app
ENV MCAP_PREVIEW_UPSTREAM_HOST=mcap-preview-dev-wtttm6suaq-uc.a.run.app

EXPOSE 80
