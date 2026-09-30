# syntax=docker/dockerfile:1
# Build context is the repository root (see docker-compose.yml) so that
# deploy/nginx.conf is reachable.

# Build stage: type-check and bundle the SPA.
FROM node:20-alpine AS build
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# Runtime stage: nginx serves the static bundle and proxies /api to the api service.
FROM nginx:1.27-alpine
COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
