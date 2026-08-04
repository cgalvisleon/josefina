#!/bin/bash

set -e                                                                    # Salir si hay errores

# Variables de configuración
CURRENT_DATETIME=$(date '+%Y%m%d%H%M%S')                                  # Fecha y hora actual
TEMPLATE_FILE="./deployments/template-deploy.yml"                         # Archivo de plantilla
OUTPUT_FILE="./deployments/deploy.yml"                                    # Archivo de salida
PORT=1377                                                                 # Puerto de la aplicación
HOST="http://josephine"                                             
PATH_URL="/josephine"                                             
APP="josephine"                                                      # Nombre de la aplicación
CMD="josephine"                                                              # Comando de la aplicación
ROLE="josephine"                                                     # Valor para reemplazar $ROLE
IMAGE="cgalvisleon/josephine"                                        # Valor $IMAGE
VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")    # Valor para reemplazar $VERSION
RELEASE="$VERSION-$CURRENT_DATETIME"                                      # Valor para reemplazar $RELEASE
BRANCH=$(git branch --show-current)                                       # Valor para validar el namespace
LIB_NAME="jdblib"                                                         # Nombre de la librería
PRD=false

if [ "$BRANCH" == "main" ]; then
    NAMESPACE="prod"                                                      # Valor para reemplazar $NS
    IMAGE_VERSION="$IMAGE:$VERSION"                                       # Valor para reemplazar $IMAGE
    HISTORY_LIMIT=15                                                      # Límite de historial de versiones
    REPLICAS=3                                                            # Número de réplicas
    PRODUCTION=true                                                       # Bandera para indicar que es producción
    CPU_REQUEST="250m"                                                    # CPU request
    CPU_LIMIT="500m"                                                      # CPU limit
    MEMORY_REQUEST="256Mi"                                                # Memory request
    MEMORY_LIMIT="512Mi"                                                  # Memory limit
    MAX_PODS_AVAILABLE=1                                                  # Número máximo de pods disponibles
    MAX_PODS_SURGE=1                                                      # Número máximo de pods en exceso
    PRD=true
else
    NAMESPACE="stage"                                                     # Valor para reemplazar $NS
    IMAGE_VERSION="$IMAGE:$VERSION-beta"                                  # Valor para reemplazar $IMAGE
    HISTORY_LIMIT=1                                                       # Límite de historial de versiones
    REPLICAS=1                                                            # Número de réplicas
    PRODUCTION=false                                                      # Bandera para indicar que es producción
    CPU_REQUEST="50m"                                                     # CPU request
    CPU_LIMIT="100m"                                                      # CPU limit
    MEMORY_REQUEST="64Mi"                                                 # Memory request
    MEMORY_LIMIT="128Mi"                                                  # Memory limit
    MAX_PODS_AVAILABLE=1                                                  # Número máximo de pods disponibles
    MAX_PODS_SURGE=1                                                      # Número máximo de pods en exceso
    KIND="StatefulSet"                                                    # Tipo de objeto de Kubernetes
    TEMPLATE_FILE="./deployments/template-statefulset.yml"                # Archivo de plantilla
fi

HELP=false
BUILD=false
BUILD_MOBILE=false
DEPLOY=false
UNDO=false
DELETE=false
DEBUG=false
LIST=false

# Parsear opciones
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --h | --help) HELP=true ;;                    # Activar la bandera si se proporciona --help
        --d | --deploy) DEPLOY=true ;;                # Activar la bandera si se proporciona --deploy
        --u | --undo) UNDO=true ;;                    # Activar la bandera si se proporciona --undo
        --delete) DELETE=true ;;                      # Activar la bandera si se proporciona --delete
        --b | --build) BUILD=true ;;                  # Activar la bandera si se proporciona --build
        --b-m | --build-mobile) BUILD_MOBILE=true ;; # Activar la bandera si se proporciona --build-mobile
        --r | --replicas) REPLICAS="$2"; shift ;;     # Cambiar el número de réplicas
        --debug) DEBUG=true ;;                        # Activar la bandera si se proporciona --debug
        --p | --production) PRODUCTION="$2"; shift ;; # Activar la bandera si se proporciona --production
        --l | --list) LIST=true ;;                    # Activar la bandera si se proporciona --list
        *) echo "Opción desconocida: $1"; exit 1 ;;
    esac
    shift
done

# Mostrar las opciones elegidas
echo "Opciones elegidas:"
if [ "$BUILD_MOBILE" == true ]; then
    echo " - Build Mobile: Activado"
else
    [[ "$DEPLOY" == true ]] && echo " - Deploy: Activado"
    [[ "$UNDO" == true ]] && echo " - Undo: Activado"
    [[ "$DELETE" == true ]] && echo " - Delete: Activado"
    [[ "$BUILD" == true ]] && echo " - Build: Activado"
    [[ "$REPLICAS" -gt 0 ]] && echo " - Réplicas: $REPLICAS"
    [[ "$PRODUCTION" == true ]] && echo " - Producción: Activado"
    [[ "$PRODUCTION" == false ]] && echo " - Producción: Desactivado"
    [[ "$DEBUG" == true ]] && echo " - Debug: Activado"
    [[ "$LIST" == true ]] && echo " - List: Activado"
    echo " - Build Mobile: Activado"
fi

# Función para mostrar la ayuda
help() {
    echo "Uso: deploy.sh [opciones]"
    echo "Opciones:"
    echo "  --h, --help: Muestra este mensaje de ayuda."
    echo "  --d, --deploy: Despliega la aplicación en el clúster de Kubernetes."
    echo "  --u, --undo: Deshace el despliegue de la aplicación en el clúster de Kubernetes."
    echo "  --delete: Elimina el despliegue de la aplicación en el clúster de Kubernetes."
    echo "  --b, --build: Construye la imagen de Docker de la aplicación."
    echo "  --b-m, --build-mobile: Construye la imagen de Docker de la aplicación para Android y IOS."
    echo "  --r, --replicas: Cambia el número de réplicas de la aplicación."
    echo "  --p, --production: Activa o desactiva el modo de producción."
    echo "  --debug: Activa el modo de depuración."
    echo "  --l, --list: Muestra la lista de imágenes de Docker."
    exit 0
}

# Función para construir la imagen de Docker
build_image() {
    local platform=$1
    local image=$2
    local dockerfile=$3
    local tag_latest=$4

    # Si tag_latest es true, taggear como latest
    if [ "$tag_latest" = true ]; then      
        docker buildx build --no-cache --platform "$platform" \
            -t "$IMAGE:latest" \
            -t "$image" \
            -f "$dockerfile" --push .
        echo "Imagen $image y $IMAGE:latest creadas con éxito."
    else
        docker buildx build --no-cache --platform "$platform" \
            -t "$image" \
            -f "$dockerfile" --push .
        echo "Imagen $image creada con éxito."
    fi
}

# Función para aplicar el archivo de configuración de Kubernetes
apply_k8s() {
    local namespace=$1
    local file=$2

    kubectl apply -f "$file"
    kubectl -n "$namespace" get pods

    echo "Deploy $file en el namespace $namespace."
}

# Reemplazar valores en el archivo de plantilla y guardar en el archivo de salida
build_manifest() {
    sed -e "s#\$PORT#$PORT#g" \
        -e "s#\$HOST#$HOST#g" \
        -e "s#\$REPLICAS#$REPLICAS#g" \
        -e "s#\$PRODUCTION#$PRODUCTION#g" \
        -e "s#\$DEBUG#$DEBUG#g" \
        -e "s#\$APP#$APP#g" \
        -e "s#\$ROLE#$ROLE#g" \
        -e "s#\$PATH_URL#$PATH_URL#g" \
        -e "s#\$NS#$NAMESPACE#g" \
        -e "s#\$IMAGE#$IMAGE_VERSION#g" \
        -e "s#\$CPU_REQUEST#$CPU_REQUEST#g" \
        -e "s#\$CPU_LIMIT#$CPU_LIMIT#g" \
        -e "s#\$MAX_PODS_AVAILABLE#$MAX_PODS_AVAILABLE#g" \
        -e "s#\$MAX_PODS_SURGE#$MAX_PODS_SURGE#g" \
        -e "s#\$MEMORY_REQUEST#$MEMORY_REQUEST#g" \
        -e "s#\$MEMORY_LIMIT#$MEMORY_LIMIT#g" \
        -e "s#\$HISTORY_LIMIT#$HISTORY_LIMIT#g" \
        -e "s#\$RELEASE#$RELEASE#g" "$TEMPLATE_FILE" > "$OUTPUT_FILE"

    echo "Archivo $OUTPUT_FILE generado con éxito."
}

# Función para listar los eventos de Kubernetes
logs() {
    local namespace=$1

    kubectl -n "$namespace" get events --sort-by='.metadata.creationTimestamp'
}

# Función para listar los pods
list_pods() {
    local namespace=$1

    kubectl -n "$namespace" get pods -o custom-columns="POD:metadata.name,CPU Requests:spec.containers[*].resources.requests.cpu,CPU Limits:spec.containers[*].resources.limits.cpu,Memory Requests:spec.containers[*].resources.requests.memory,Memory Limits:spec.containers[*].resources.limits.memory"
    echo "Listado de pods."
}

build_mobile() {
    gomobile bind -target=android -androidapi 21 -o "./flutter/android/libs/$LIB_NAME.aar" "./mobile/$LIB_NAME"
    gomobile bind -target=ios -o "./flutter/ios/$LIB_NAME.xcframework" "./mobile/$LIB_NAME"

    echo "Librería $LIB_NAME creada con éxito."
}

if [ "$HELP" = true ]; then
    help
elif [ "$LIST" = true ]; then
    list_pods "$NAMESPACE"
elif [ "$UNDO" = true ]; then
    kubectl rollout undo deployment "$ROLE" -n "$NAMESPACE"
    kubectl -n "$NAMESPACE" get pods
    echo "Desplegado deshecho."
elif [ "$BUILD" = true ]; then
    build_manifest
    build_image "linux/amd64,linux/arm64" "$IMAGE_VERSION" "./cmd/$CMD/Dockerfile" $PRD
elif [ "$BUILD_MOBILE" = true ]; then
    build_mobile
elif [ "$DELETE" = true ]; then
    kubectl delete -f "$OUTPUT_FILE"
    kubectl -n "$NAMESPACE" get all
    echo "Despliegue eliminado."
elif [ "$DEPLOY" = true ]; then
    build_manifest
    apply_k8s "$NAMESPACE" "$OUTPUT_FILE"
elif [ "$BRANCH" == "main" ]; then
    build_manifest
    build_image "linux/amd64,linux/arm64" "$IMAGE_VERSION" "./cmd/$CMD/Dockerfile" $PRD
    apply_k8s "$NAMESPACE" "$OUTPUT_FILE"
else
    help
fi

# Línea en blanco al final