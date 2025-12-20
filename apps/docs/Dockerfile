# Base stage
FROM node:20-alpine AS base

WORKDIR /app

# Enable pnpm
RUN corepack enable

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/docs/package.json ./apps/docs/

# Install dependencies
RUN pnpm install --frozen-lockfile

# Development stage
FROM base AS dev
WORKDIR /app/apps/docs
EXPOSE 3000
CMD ["pnpm", "start", "--port", "3000", "--host", "0.0.0.0", "--poll", "1000"]

# Builder stage
FROM base AS builder
COPY apps/docs ./apps/docs
WORKDIR /app/apps/docs
RUN pnpm build

# Serve stage
FROM node:20-alpine AS runner
WORKDIR /app
RUN corepack enable
COPY --from=builder /app/apps/docs/build ./build
RUN npm install -g serve
EXPOSE 3000
CMD ["serve", "-s", "build", "-p", "3000"]
