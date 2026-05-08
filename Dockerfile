FROM golang:1.22-bullseye AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/main .

RUN chmod +x /app/main

ENV MONGO_URI="${MONGO_URI:-mongodb://localhost:27017}"
ENV MONGO_DB="${MONGO_DB:-vetflow}"
ENV PORT="${PORT:-8080}"

EXPOSE 8080

CMD ["/app/main"]
