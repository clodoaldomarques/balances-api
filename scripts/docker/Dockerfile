## build
FROM golang:1.27.1-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /balances-api ./cmd

## deploy
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /balances-api /balances-api

EXPOSE 5000

ENTRYPOINT [ "/balances-api" ]