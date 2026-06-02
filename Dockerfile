# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copiar go.mod y go.sum primero para cachear dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiar el resto del código
COPY . .

# Compilar binario estático
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/arkivy-api ./cmd/main.go

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copiar binario
COPY --from=builder /app/arkivy-api .

# Copiar key.json si existe (para Zitadel service account)
COPY key.json* ./

EXPOSE 9090

CMD ["./arkivy-api"]