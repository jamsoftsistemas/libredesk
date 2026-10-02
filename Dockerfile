# ---- Frontend build ----
FROM node:20-alpine AS frontend
RUN corepack enable
WORKDIR /src
COPY frontend/package.json frontend/pnpm-lock.yaml ./frontend/
RUN cd frontend && pnpm install --frozen-lockfile
COPY frontend ./frontend
COPY static ./static
COPY internal/countries ./internal/countries
RUN cd frontend && pnpm build:main && pnpm build:widget && pnpm build:widget-loader

# ---- Backend build ----
FROM golang:1.25-alpine AS backend
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/frontend/dist ./frontend/dist
COPY --from=frontend /src/static/widget.min.js ./static/widget.min.js
RUN go install github.com/knadh/stuffbin/...@latest
RUN LAST_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo unknown) && \
    VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0) && \
    BUILDSTR="${VERSION} (#${LAST_COMMIT} $(date -u +%Y-%m-%dT%H:%M:%S%z))" && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -ldflags="-X 'main.buildString=${BUILDSTR}' -X 'main.versionString=${VERSION}' -X 'github.com/abhinavxd/libredesk/internal/version.Version=${VERSION}' -s -w" \
      -o libredesk cmd/*.go && \
    "$(go env GOPATH)/bin/stuffbin" -a stuff -in libredesk -out libredesk \
      frontend/dist i18n schema.sql static/email-templates static/public static/widget.min.js:static/widget.js

# ---- Final image ----
FROM alpine:3.18
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /libredesk
COPY --from=backend /src/libredesk ./libredesk
COPY config.sample.toml config.toml
EXPOSE 9000
CMD ["./libredesk"]
