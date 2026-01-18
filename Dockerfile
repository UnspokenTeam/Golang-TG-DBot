FROM golang:1.24.5-alpine3.22 AS build

WORKDIR /app
RUN apk add --no-cache curl
RUN curl -fsSL https://raw.githubusercontent.com/pressly/goose/master/install.sh | sh
COPY . .

RUN go mod download
RUN go build -o Bot github.com/unspokenteam/golang-tg-dbot

FROM alpine:3.22
WORKDIR /app

COPY --from=build /internal/app/Bot .
COPY --from=build /usr/local/bin/goose /usr/local/bin/goose
COPY ./sql ./sql

EXPOSE 8000
ENTRYPOINT goose -dir ./sql postgres "host=$DB_HOST port=$DB_PORT user=$DB_USER dbname=$DB_NAME password=$DB_PASSWORD sslmode=disable" up && ./Bot