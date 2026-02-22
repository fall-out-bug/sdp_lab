FROM golang:1.26 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/feature-orchestrator ./cmd/feature-orchestrator

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/feature-orchestrator /app/feature-orchestrator
ENTRYPOINT ["/app/feature-orchestrator"]
CMD ["--workspace", "/workspaces", "--poll", "30s", "--max", "3", "--namespace", "sdp-workers"]
