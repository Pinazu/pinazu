# Build frontend with Node.js
FROM node:20-alpine AS build-fe

WORKDIR /home/apps

# Copy only package files first for better caching
COPY ./web/package*.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci

# Then copy source code and build
COPY ./web .
RUN npm run build

# Build Go application
FROM golang:1.25-alpine AS build

# Install git which is required by go mod
RUN apk add --no-cache git

WORKDIR /home/apps

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Then copy source code
COPY ./api ./api
COPY ./cli ./cli
COPY ./cmd ./cmd
COPY ./configs ./configs
COPY ./internal ./internal
COPY ./sql ./sql
COPY ./scripts ./scripts
COPY ./sqlc.yaml .
COPY ./oapi-codegen.yaml .

# Build with cache
RUN --mount=type=cache,target=/go/pkg/mod \
    go run ./scripts/build-openapi.go && \
    CGO_ENABLED=0 go build -o pinazu-core ./cmd && \
    chmod +x pinazu-core

# Final minimal runtime image
FROM alpine:3.19

# Install tini for proper signal handling
RUN apk add --no-cache tini && \
    addgroup -g 1000 apps && \
    adduser -D -u 1000 -G apps apps

WORKDIR /home/apps
USER apps

COPY --from=build --chmod=755 /home/apps/pinazu-core /usr/bin/pinazu-core
COPY --from=build-fe --chown=apps:apps --chmod=755 /home/apps/dist /home/apps/web/dist
COPY --chown=apps:apps --chmod=755 ./sql/migrations /home/apps/sql/migrations
COPY --chown=apps:apps --chmod=644 --from=build /home/apps/api/openapi.yaml /home/apps/api/openapi.yaml

ENTRYPOINT [ "tini", "-g", "--", "/usr/bin/pinazu-core" ]
