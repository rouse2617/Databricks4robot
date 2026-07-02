# BuildKit cache mount speeds up repeat `npm ci` (persists tarballs across builds).
FROM mirror.gcr.io/library/node:20-alpine AS builder
WORKDIR /app
ENV HTTP_PROXY=
ENV HTTPS_PROXY=
ENV ALL_PROXY=
ENV http_proxy=
ENV https_proxy=
ENV all_proxy=
ARG NPM_REGISTRY=https://registry.npmjs.org/
ENV NPM_CONFIG_REGISTRY=${NPM_REGISTRY}
COPY package.json package-lock.json ./
RUN env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u http_proxy -u https_proxy -u all_proxy \
    -u NPM_CONFIG_PROXY -u NPM_CONFIG_HTTPS_PROXY -u npm_config_proxy -u npm_config_https_proxy \
    npm ci --include=dev --no-audit --no-fund --cache /tmp/npm-cache \
    && test -x node_modules/.bin/tsc
COPY . .
ARG VITE_API_BASE_URL=http://localhost:8080
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ARG VITE_DEV_ACCESS_TOKEN=
ENV VITE_DEV_ACCESS_TOKEN=$VITE_DEV_ACCESS_TOKEN
ARG VITE_APP_VERSION=0.1.0
ENV VITE_APP_VERSION=$VITE_APP_VERSION
ARG VITE_BUILD_REF=local
ENV VITE_BUILD_REF=$VITE_BUILD_REF
RUN npm run build

FROM mirror.gcr.io/library/nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
# nginx official entrypoint will envsubst templates under /etc/nginx/templates/*.template
ENV BACKEND_UPSTREAM_HOST=cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app
ENV MCAP_PREVIEW_UPSTREAM_HOST=mcap-preview-dev-wtttm6suaq-uc.a.run.app
COPY nginx.conf /etc/nginx/templates/default.conf.template
EXPOSE 80
