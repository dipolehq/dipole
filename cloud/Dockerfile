FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server ./cmd/server

FROM scratch
COPY --from=builder /server /server
COPY --from=builder /src/internal/repository/postgres/migrations /migrations
EXPOSE 8080 50051
ENTRYPOINT ["/server"]
