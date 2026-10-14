FROM node:26.8.1-bookworm-slim AS build

WORKDIR /app
RUN npm install --global pnpm@11.24.0

COPY package.json pnpm-lock.yaml .npmrc ./
ENV PNPM_CONFIG_DANGEROUSLY_ALLOW_ALL_BUILDS=true
RUN pnpm install --frozen-lockfile

COPY . .
RUN pnpm build
RUN pnpm browser-bundle:check

FROM node:26.8.1-bookworm-slim AS runtime

ENV NODE_ENV=production
ENV NITRO_HOST=0.0.0.0
ENV NITRO_PORT=3000
WORKDIR /app

RUN groupadd --system --gid 10001 console \
  && useradd --system --uid 10001 --gid console --home-dir /app console
COPY --from=build --chown=console:console /app/.output ./.output

USER console
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD node -e 'fetch("http://127.0.0.1:3000/api/health").then(r => { if (!r.ok) process.exit(1) }).catch(() => process.exit(1))'
CMD ["node", ".output/server/index.mjs"]
