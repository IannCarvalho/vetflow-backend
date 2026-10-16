FROM golang:1.22-bullseye AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# -trimpath removes local filesystem paths from the binary; -ldflags "-s -w"
# strips the symbol table and DWARF debug info for a smaller static binary.
# CGO is disabled so the result is a fully static Linux binary.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o main ./cmd/main.go

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/main .

RUN chmod +x /app/main

# Run as an unprivileged user. The application never writes to the filesystem
# (attachments and profile images are stored base64-encoded in MongoDB; remote
# images are downloaded into memory), so a read-only, non-root runtime is safe.
RUN useradd --system --no-create-home --uid 10001 appuser
USER appuser

EXPOSE 8080

CMD ["/app/main"]
