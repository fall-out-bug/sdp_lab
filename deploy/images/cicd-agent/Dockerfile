FROM golang:1.26 AS builder
WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/cicd-agent ./cmd/cicd-agent

# Build bd from beads repo (required for deploy chore issues)
ARG BD_VERSION=v0.49.6
RUN git clone --depth 1 --branch "${BD_VERSION}" https://github.com/steveyegge/beads /tmp/beads \
  && cd /tmp/beads && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/bd ./cmd/bd

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /out/cicd-agent /app/cicd-agent
COPY --from=builder /out/bd /usr/local/bin/bd
ENTRYPOINT ["/app/cicd-agent"]
CMD ["-dir", "/workspace"]
