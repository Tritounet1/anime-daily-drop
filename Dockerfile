FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod .
COPY . .
RUN go build -o scrape .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /build/scrape .
CMD ["/app/scrape", "schedule"]
