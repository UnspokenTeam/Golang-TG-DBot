FROM golang:1.25.6-alpine AS builder

WORKDIR /build

COPY go.mod go.sum go.work go.work.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -a \
    -installsuffix cgo \
    -o bot \
    ./cmd/bot/main.go

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/bot /bot

EXPOSE 8080
ENTRYPOINT ["/bot"]