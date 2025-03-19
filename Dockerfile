FROM golang:alpine AS builder

WORKDIR /build
ADD go.mod .
COPY . .
RUN go build -o bot_binary cmd/main.go

FROM alpine

WORKDIR /app
COPY --from=builder /build/bot_binary .
CMD [ "./bot_binary", "-config", "config.yaml" ]
