# Build Stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod .
COPY main.go .
# Ensure dependencies are tidy and downloaded
RUN go mod tidy
RUN go mod download
# Disable CGO for static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o store-api main.go

# Run Stage
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /app/store-api .
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/store-api"]
