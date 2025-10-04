FROM 271540607717.dkr.ecr.ap-southeast-1.amazonaws.com/secure-vl-base/nodejs:20 AS build-fe

WORKDIR /home/apps
COPY --chown=apps:apps ./web /home/apps

RUN npm ci && \
    npm run build

FROM 271540607717.dkr.ecr.ap-southeast-1.amazonaws.com/secure-vl-base/golang:1.24-r1 AS build

WORKDIR /home/apps

COPY --chown=apps:apps . .

RUN go mod tidy && \
    go run ./scripts/build-openapi.go && \
    CGO_ENABLED=0 go build -o pinazu-core ./cmd && \
    chmod +x pinazu-core

FROM 271540607717.dkr.ecr.ap-southeast-1.amazonaws.com/secure-vl-base/minimal-base:1.0.0-r0

WORKDIR /home/apps
USER apps

COPY --from=build --chmod=755 /home/apps/pinazu-core /usr/bin/pinazu-core
COPY --from=build-fe --chown=apps:apps --chmod=755 /home/apps/dist /home/apps/web/dist
COPY --chown=apps:apps --chmod=755 ./sql/migrations /home/apps/sql/migrations
COPY --chown=apps:apps --chmod=644 --from=build /home/apps/api/openapi.yaml /home/apps/api/openapi.yaml

ENTRYPOINT [ "tini", "-g", "--", "/usr/bin/pinazu-core" ]
