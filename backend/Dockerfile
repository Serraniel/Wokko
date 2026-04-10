# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /wokko-api ./cmd/server

# Final stage
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /wokko-api /wokko-api
COPY --from=builder /app/migrations /migrations
EXPOSE 8080
ENTRYPOINT ["/wokko-api"]
