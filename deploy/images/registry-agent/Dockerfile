FROM golang:1.26 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/registry-agent ./cmd/registry-agent

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/registry-agent /app/registry-agent
ENTRYPOINT ["/app/registry-agent"]
CMD ["--addr", ":8080"]
