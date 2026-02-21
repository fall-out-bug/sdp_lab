FROM golang:1.26 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/adapter-controller ./cmd/adapter-controller

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/adapter-controller /app/adapter-controller
ENTRYPOINT ["/app/adapter-controller"]
