FROM golang:1.25-alpine AS builder

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install github.com/a-h/templ/cmd/templ@v0.3.1001 && templ generate

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o holo ./cmd/holo

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

RUN mkdir -p /app/data

COPY --from=builder /app/holo .

EXPOSE 8080

ENTRYPOINT ["./holo"]
