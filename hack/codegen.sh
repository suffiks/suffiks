#!/bin/bash -e

# Copy from traefik: https://github.com/traefik/traefik/blob/master/script/codegen.Dockerfile

set -e -o pipefail

PROJECT_MODULE="github.com/suffiks/suffiks"
IMAGE_NAME="kubernetes-codegen:latest"

echo "Building codegen Docker image..."
docker build --build-arg KUBE_VERSION=v0.25.0 --build-arg USER=$USER --build-arg UID=$(id -u) --build-arg GID=$(id -g) -f "./hack/Dockerfile.codegen" \
             -t "${IMAGE_NAME}" \
             "."

echo "Generating Traefik clientSet code ..."
cmd="/go/src/k8s.io/code-generator/generate-groups.sh client,lister,informer ${PROJECT_MODULE}/pkg/client/generated ${PROJECT_MODULE}/apis suffiks:v1 --go-header-file=/go/src/${PROJECT_MODULE}/hack/boilerplate.go.txt"
docker run --rm \
           -v "$(pwd):/go/src/${PROJECT_MODULE}" \
           -w "/go/src/${PROJECT_MODULE}" \
           "${IMAGE_NAME}" $cmd
