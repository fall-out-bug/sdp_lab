FROM golang:1.26 AS builder
WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/telemetry-analyzer ./cmd/telemetry-analyzer

# Build bd from beads repo (required for bd create/list)
ARG BD_VERSION=v0.59.0
RUN git clone --depth 1 --branch "${BD_VERSION}" https://github.com/steveyegge/beads /tmp/beads \
  && cd /tmp/beads && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/bd ./cmd/bd

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /out/telemetry-analyzer /app/telemetry-analyzer
COPY --from=builder /out/bd /usr/local/bin/bd

ENTRYPOINT ["/app/telemetry-analyzer"]
CMD ["-dir", "/workspaces"]
