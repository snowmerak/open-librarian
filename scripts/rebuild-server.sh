#!/bin/bash

# echo "📦 Stopping open-librarian-server container..."
docker-compose stop open-librarian-server

echo "🗑️  Removing open-librarian-server container..."
docker-compose rm -f open-librarian-server

echo "🔨 Building open-librarian-server image..."
docker-compose build open-librarian-server

echo "🚀 Starting open-librarian-server..."
docker-compose up -d open-librarian-serverebuild and restart only the open-librarian-server service
# Usage: ./scripts/rebuild-server.sh

set -e  # Exit on any error

echo "🔧 Rebuilding Open Librarian Server..."

# Navigate to project root (assuming script is run from project root)
if [ ! -f "docker-compose.yaml" ]; then
    echo "❌ Error: docker-compose.yaml not found. Please run this script from the project root."
    exit 1
fi

echo "📦 Stopping open-librarian-server container..."
docker-compose stop open-librarian-server

echo "🗑️  Removing open-librarian-server container..."
docker-compose rm -f open-librarian-server

echo "🏗️  Building new server image..."
docker-compose build open-librarian-server

echo "🚀 Starting open-librarian-server..."
docker-compose up -d open-librarian-server

echo "⏳ Waiting for server to be healthy..."
timeout=60
counter=0
while [ $counter -lt $timeout ]; do
    if docker-compose ps open-librarian-server | grep -q "healthy"; then
        echo "✅ Server is healthy and ready!"
        echo "🌐 Server available at: http://localhost:8080"
        echo "📖 Swagger UI available at: http://localhost:8080/swagger/"
        exit 0
    fi
    
    if docker-compose ps open-librarian-server | grep -q "unhealthy"; then
        echo "❌ Server health check failed. Check logs:"
        docker-compose logs --tail=20 open-librarian-server
        exit 1
    fi
    
    echo "⏳ Still waiting for server to be ready... ($counter/$timeout)"
    sleep 2
    counter=$((counter + 2))
done

echo "⚠️  Timeout waiting for server to be healthy. Check logs:"
docker-compose logs --tail=20 open-librarian-server
exit 1
