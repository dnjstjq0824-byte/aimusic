# AVIBE web: Nuxt 4 SSR on a Node server.
# This image IS production (avibe-prod-web on the VPS, shipped by
# deploy/rollout-redesign.sh step 4; see docker-compose.prod.yml `web`).
# The Cloudflare Worker serves version-matched static assets + proxies SSR.
FROM node:22-alpine AS build
WORKDIR /src
COPY package.json package-lock.json .npmrc ./
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
