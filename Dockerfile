# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# templ-generated files are committed, so no codegen needed in CI.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/shouyu ./cmd/shouyu

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /out/shouyu /shouyu
COPY web/static /web/static
# NOTE (W3.1 finding): no COPY of migrations — the schema is //go:embed-ed
# into the binary at build time via internal/notes/repo.go, so the migrations
# directory is never read from disk at runtime.
USER nonroot:nonroot
ENV STATIC_DIR=/web/static
EXPOSE 8080
ENTRYPOINT ["/shouyu"]
