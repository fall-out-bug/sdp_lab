FROM golang:1.26 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/intake-gateway ./cmd/intake-gateway

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/intake-gateway /app/intake-gateway
ENTRYPOINT ["/app/intake-gateway"]
CMD ["--addr", ":8081"]
