FROM golang:1.24-alpine AS builder

WORKDIR /app
RUN apk add --no-cache ca-certificates git tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/server ./cmd/server

FROM alpine:3.20

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/server /app/server
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/.env /app/.env

EXPOSE 8080

CMD ["/app/server"]
