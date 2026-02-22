FROM golang:1.26 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/cicd-agent ./cmd/cicd-agent

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/cicd-agent /app/cicd-agent
ENTRYPOINT ["/app/cicd-agent"]
CMD ["-dir", "/workspace"]
