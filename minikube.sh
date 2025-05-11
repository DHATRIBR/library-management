#!/usr/bin/env bash

set -e  # Exit on any error
WORK_DIR=$(pwd)

# Define colors
GREEN="\033[0;32m"
BLUE="\033[0;34m"
RED="\033[0;31m"
NC="\033[0m" # No Color

announce() {
    echo -e "${BLUE}==> $1${NC}"
}

success() {
    echo -e "${GREEN}✔ $1${NC}"
}

error_exit() {
    echo -e "${RED}✖ Error: $1${NC}"
    exit 1
}

apply_k8s_config() {
    local desc=$1
    local path=$2

    announce "Applying $desc configuration..."
    if kubectl apply -f "$path"; then
        success "$desc applied successfully."
    else
        error_exit "Failed to apply $desc from $path"
    fi
}

# Uncomment if needed
announce "Cleaning up Docker system..."
docker system prune -f

# Uncomment if needed
announce "Setting up Minikube Docker environment..."
eval $(minikube docker-env)

announce "Setting up Ingress..."
minikube addons enable ingress

announce "Building Docker images..."
for service_dir in book-service borrow-service NotificationService; do
    case $service_dir in
        book-service)
            image_name="library/book-service:latest"
            ;;
        borrow-service)
            image_name="library/borrow-service:latest"
            ;;
        NotificationService)
            image_name="library/notification:latest"
            ;;
        *)
            error_exit "Unknown service: $service_dir"
            ;;
    esac

    announce "Building image for $service_dir → $image_name"
    pushd "services/$service_dir" >/dev/null
    docker build -t "$image_name" .
    popd >/dev/null
    success "Built $image_name"
done

announce "Starting Kubernetes deployment..."

pushd "$WORK_DIR/services" >/dev/null

# Notification Service
apply_k8s_config "NotificationService MySQL" "NotificationService/mysql.yaml"
apply_k8s_config "NotificationService Deployment" "NotificationService/notification-service.yaml"

# Book Service
apply_k8s_config "BookService MongoDB" "book-service/mongodb.yaml"
apply_k8s_config "BookService Deployment" "book-service/book-service.yaml"

# Borrow Service
apply_k8s_config "BorrowService MySQL" "borrow-service/mysql.yaml"
apply_k8s_config "BorrowService Deployment" "borrow-service/borrow-service.yaml"

# Borrow Service
apply_k8s_config "User Management MySQL" "user-mgmt-service/mysql.yaml"
apply_k8s_config "User Management Deployment" "user-mgmt-service/borrow-service.yaml"

popd >/dev/null

# Ingress
apply_k8s_config "Ingress Controller" "$WORK_DIR/services/ingress.yaml"

# Kafka
apply_k8s_config "Kafka" "$WORK_DIR/services/kafka.yaml"
apply_k8s_config "Kafka Init Script" "$WORK_DIR/services/kafka-init.yaml"

announce "All services deployed successfully."