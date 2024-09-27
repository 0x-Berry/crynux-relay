FROM golang:alpine3.19 AS builder
RUN apk add --no-cache --update gcc g++

WORKDIR /crynux_relay

COPY go.* .

RUN CGO_ENABLED=1 go mod download

COPY . .

RUN CGO_ENABLED=1 go build

FROM alpine:3.19

RUN apk add --no-cache tzdata
ENV TZ=Asia/Tokyo

WORKDIR /app

COPY --from=builder /crynux_relay/crynux_relay .
COPY static ./static

CMD ["/app/crynux_relay"]
