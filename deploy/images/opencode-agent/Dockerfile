FROM golang:1.26 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/opencode-agent ./cmd/opencode-agent

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/opencode-agent /app/opencode-agent
ENTRYPOINT ["/app/opencode-agent"]
CMD ["--loop", "--interval", "30s"]
