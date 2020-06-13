FROM golang:1.13-alpine as builder

RUN mkdir /build
ADD . /build/

WORKDIR /build

RUN go build -o fibbage-game-server cmd/server/main.go

FROM alpine:latest

COPY --from=builder /build/fibbage-game-server /app/
COPY --from=builder /build/config.yml /app/
WORKDIR /app

CMD ["./fibbage-game-server"]
