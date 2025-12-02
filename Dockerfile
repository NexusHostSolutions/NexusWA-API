# Build Stage
FROM golang:1.22-alpine AS builder

# Instalar GCC e ferramentas de build (Necessário para SQLite/WhatsMeow)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Habilitar CGO para o SQLite
ENV CGO_ENABLED=1
RUN go build -o nexuswa cmd/server/main.go

# Run Stage
FROM alpine:latest

# Instalar bibliotecas de sistema necessárias
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/nexuswa .
COPY --from=builder /app/.env . 
COPY --from=builder /app/public ./public 
# Importante copiar a pasta public para o site funcionar

# Criar pasta para banco de dados
RUN mkdir sessions

EXPOSE 8082

CMD ["./nexuswa"]