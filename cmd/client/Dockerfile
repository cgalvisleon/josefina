# Versión de Go como argumento3
ARG GO_VERSION=1.23

# Stage 1: Compilación (builder)
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

# Argumentos para el sistema operativo y la arquitectura
ARG TARGETOS
ARG TARGETARCH

# Instalación de dependencias necesarias
RUN apk update && apk add --no-cache ca-certificates openssl git \
    && update-ca-certificates

# Configuración de las variables de entorno para la build
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}

# Directorio de trabajo
WORKDIR /src

# Descargar dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiar el código fuente
COPY . .

# Formatear el código Go
RUN gofmt -w .

# Compilar el binario
RUN go build -a -v -o /client ./cmd/client

# Cambiar permisos del binario
RUN chmod +x /client

# Stage 2: Imagen final mínima
FROM scratch

# Copiar certificados y binario
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /projects /projects

# Establecer el binario como punto de entrada
ENTRYPOINT ["/projects"]
