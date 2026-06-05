FROM golang:alpine AS build

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o transmission-telegram .

FROM alpine:latest AS certs
RUN apk --update add ca-certificates

FROM bash:latest
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /app/transmission-telegram /transmission-telegram
RUN chmod 755 /transmission-telegram

ENTRYPOINT ["/transmission-telegram"]
