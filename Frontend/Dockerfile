# BuildKit cache mount speeds up repeat `npm ci` (persists tarballs across builds).
FROM node:20-alpine AS builder
WORKDIR /app
ARG NPM_REGISTRY=https://registry.npmjs.org/
ENV NPM_CONFIG_REGISTRY=${NPM_REGISTRY}
COPY package.json package-lock.json ./
RUN npm ci
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

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
