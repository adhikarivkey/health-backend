## Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install git (required for some dependencies)
RUN apk add --no-cache git

# Copy go mod
COPY go.mod go.sum ./
RUN go mod download

# Copy full project
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/app ./server/main.go

## Final stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/app /app/app

EXPOSE 8000

CMD ["/app/app"]
