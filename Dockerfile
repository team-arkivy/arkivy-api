# ---- Step 1: compile ----
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy dependencies and start
COPY go.mod go.sum ./
RUN go mod download

# Copiar the rest of the code and compile
COPY . .
RUN go build -o main ./cmd/main.go

# ---- Step 2: final light image ----
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 9090

CMD ["./main"]