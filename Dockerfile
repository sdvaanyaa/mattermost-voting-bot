FROM golang:1.23.4 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o bot cmd/main.go

FROM alpine:3.18

WORKDIR /app

COPY --from=builder /app/bot /app/bot

RUN apk --no-cache add ca-certificates

CMD ["/app/bot"]