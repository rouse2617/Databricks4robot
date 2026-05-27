# Override with: docker build --build-arg BASE_IMAGE=... (see deploy/cloudrun/frontend-dev.sh).
ARG BASE_IMAGE=us-central1-docker.pkg.dev/green-valley-442103/video-proc-images/cyber-databrew-frontend:dev-latest
FROM ${BASE_IMAGE}

COPY deploy/cloudrun/frontend-nginx.conf /etc/nginx/conf.d/default.conf
COPY databrew-pipeline/argo-ui/dist /usr/share/nginx/html/argo
