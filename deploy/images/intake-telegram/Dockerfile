FROM golang:1.26 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/intake-telegram ./cmd/intake-telegram

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/intake-telegram /app/intake-telegram
ENTRYPOINT ["/app/intake-telegram"]
CMD ["--addr", ":8082"]
