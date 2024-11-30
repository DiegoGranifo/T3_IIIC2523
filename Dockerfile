# Dockerfile

# Etapa de construcción
FROM golang:1.20-alpine AS builder

# Instalar dependencias necesarias
RUN apk update && apk add --no-cache git

# Establecer el directorio de trabajo dentro del contenedor
WORKDIR /app

# Copiar solo go.mod primero para aprovechar la caché de Docker
COPY go.mod ./

# Descargar las dependencias del proyecto (si las hay)
RUN go mod download

# Copiar el resto del código fuente
COPY . .

# Compilar la aplicación Go
RUN go build -o king-algorithm main.go

# Etapa de ejecución
FROM alpine:latest

# Establecer el directorio de trabajo dentro del contenedor
WORKDIR /app

# Copiar el binario compilado desde la etapa de construcción
COPY --from=builder /app/king-algorithm .

# Exponer los puertos necesarios (8001 a 8005)
EXPOSE 8001 8002 8003 8004 8005

# Comando por defecto para ejecutar la aplicación
CMD ["./king-algorithm"]
