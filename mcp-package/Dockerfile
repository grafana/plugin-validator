FROM golang:1.25-alpine3.22@sha256:fa3380ab0d73b706e6b07d2a306a4dc68f20bfc1437a6a6c47c8f88fe4af6f75 AS builder

ARG MCP_VERSION=dev

WORKDIR /build
COPY pkg/cmd/mcpserver ./pkg/cmd/mcpserver
COPY go.mod go.sum ./

RUN apk add --no-cache git ca-certificates && \
    update-ca-certificates && \
    CGO_ENABLED=0 GO111MODULE=on go build -ldflags "-s -w -X main.version=${MCP_VERSION}" -o plugin-validator-mcp ./pkg/cmd/mcpserver

FROM alpine:3.22@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412

RUN apk add --no-cache ca-certificates docker-cli nodejs npm && \
    update-ca-certificates

WORKDIR /app
COPY --from=builder /build/plugin-validator-mcp /app/plugin-validator-mcp

# MCP servers communicate over stdin/stdout
ENTRYPOINT ["/app/plugin-validator-mcp"]
