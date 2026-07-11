## Build stage
FROM node:20-alpine AS builder

WORKDIR /build

RUN npm install -g pnpm

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY . .

RUN pnpm build

## Runtime stage
FROM node:20-alpine

WORKDIR /app

RUN npm install -g pnpm

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --prod

COPY --from=builder /build/build ./build

EXPOSE 3000

ENV NODE_ENV=production

CMD ["node", "build"]
