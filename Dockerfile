FROM golang:1.26.1-alpine3.22 AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o firstclick ./cmd/firstclick

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /app/firstclick .
COPY --from=builder /app/static ./static

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

CMD ["./firstclick"]