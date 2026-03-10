FROM golang:1.23 AS builder
WORKDIR /src
COPY . .

# Build adapter-controller and beads-fsm from this repo
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/adapter-controller ./cmd/adapter-controller
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/beads-fsm ./cmd/beads-fsm

# Build bd from beads repo (required for beads operations when workspace available)
ARG BD_VERSION=v0.59.0
RUN git clone --depth 1 --branch "${BD_VERSION}" https://github.com/steveyegge/beads /tmp/beads \
  && cd /tmp/beads && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/bd ./cmd/bd

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /out/adapter-controller /app/adapter-controller
COPY --from=builder /out/beads-fsm /usr/local/bin/beads-fsm
COPY --from=builder /out/bd /usr/local/bin/bd

ENTRYPOINT ["/app/adapter-controller"]
