FROM golang:1.26 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/swarm-orchestrator ./cmd/swarm-orchestrator

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/swarm-orchestrator /app/swarm-orchestrator
ENTRYPOINT ["/app/swarm-orchestrator"]
CMD ["--workspace", "/workspaces"]
