# AVIBE web: Nuxt 4 SSR on a Node server (local/staging verify).
# Edge production uses the cloudflare-module build instead (WI-P1 dual-build).
FROM node:22-alpine AS build
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG NUXT_API_PROXY=http://api:8080
ENV NUXT_API_PROXY=$NUXT_API_PROXY
RUN npm run build

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production NITRO_PORT=3000 NITRO_HOST=0.0.0.0
COPY --from=build /src/.output ./
EXPOSE 3000
CMD ["node", "server/index.mjs"]
