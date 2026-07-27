FROM golang:1.25-alpine AS builder

WORKDIR /src

# Copy the shared gen module first (replace directive points to ../gen).
COPY gen/ /gen/

# Copy cloud module and download dependencies.
COPY cloud/go.mod cloud/go.sum ./
RUN go mod download
COPY cloud/ .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /server /server
COPY --from=builder /src/internal/repository/postgres/migrations /migrations
EXPOSE 8080 50051
ENTRYPOINT ["/server"]
