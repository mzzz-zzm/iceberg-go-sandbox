#!/bin/sh

set -e

echo "1. Downloading Go dependencies..."
go mod tidy

echo "2. Waiting for MinIO and Catalog to start..."
while ! wget -q -O - http://minio:9000/minio/health/live > /dev/null; do
    echo "Waiting for MinIO..."
    sleep 2
done

sleep 3

echo "3. Starting Build Up and Execution stage..."
go run setup.go main.go

echo "4. Sandbox execution finished. Keeping container alive for debugging..."
sleep infinity