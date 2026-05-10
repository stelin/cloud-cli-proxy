FROM node:24-alpine AS build

WORKDIR /app
COPY web/admin/package.json web/admin/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY web/admin/ .
RUN pnpm build

FROM nginx:alpine
COPY deploy/docker/admin/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
