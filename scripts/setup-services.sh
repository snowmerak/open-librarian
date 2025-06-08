#!/bin/bash

# Start OpenSearch services
echo "Starting OpenSearch services..."
docker compose up -d --build

# Wait for OpenSearch to be ready
echo "Waiting for OpenSearch to start..."
# Improved health check that waits for OpenSearch to report status as green or yellow
max_attempts=30
attempt=0
while [ $attempt -lt $max_attempts ]
do
    echo "Attempt $attempt of $max_attempts..."
    
    # First check if the container is running
    if ! docker ps | grep -q opensearch; then
        echo "OpenSearch container is not running. Checking logs:"
        docker compose logs opensearch --tail 25
        exit 1
    fi
    
    # Try to check cluster health
    health_code=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:9200/_cat/health)
    if [ "$health_code" = "200" ]; then
        health_status=$(curl -s http://localhost:9200/_cat/health | awk '{print $4}')
        if [ "$health_status" = "green" ] || [ "$health_status" = "yellow" ]; then
            echo "OpenSearch is ready with status: $health_status"
            
            # Additional check for KNN plugin
            knn_status=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:9200/_plugins/_knn/stats)
            if [ "$knn_status" != "200" ]; then
                echo "Warning: KNN plugin may not be properly installed or configured."
                echo "Checking OpenSearch plugins:"
                docker exec opensearch /usr/share/opensearch/bin/opensearch-plugin list
            else
                echo "KNN plugin is active and responding."
            fi
            
            break
        fi
    fi
    
    attempt=$((attempt+1))
    sleep 3
done

if [ $attempt -eq $max_attempts ]; then
    echo "OpenSearch did not become ready in time. Showing the most recent logs:"
    docker compose logs opensearch --tail 50
    exit 1
fi

echo "OpenSearch is ready!"

# Install required analysis plugins for multilingual support
echo "Installing language analysis plugins..."
docker exec opensearch /usr/share/opensearch/bin/opensearch-plugin install analysis-nori
docker exec opensearch /usr/share/opensearch/bin/opensearch-plugin install analysis-kuromoji  
docker exec opensearch /usr/share/opensearch/bin/opensearch-plugin install analysis-smartcn

# Restart OpenSearch to load the plugins
echo "Restarting OpenSearch to load plugins..."
docker compose restart opensearch

# Wait for OpenSearch to be ready again after restart
echo "Waiting for OpenSearch to restart..."
attempt=0
while [ $attempt -lt $max_attempts ]
do
    echo "Restart attempt $attempt of $max_attempts..."
    
    health_code=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:9200/_cat/health)
    if [ "$health_code" = "200" ]; then
        health_status=$(curl -s http://localhost:9200/_cat/health | awk '{print $4}')
        if [ "$health_status" = "green" ] || [ "$health_status" = "yellow" ]; then
            echo "OpenSearch is ready again with status: $health_status"
            break
        fi
    fi
    
    attempt=$((attempt+1))
    sleep 3
done

if [ $attempt -eq $max_attempts ]; then
    echo "OpenSearch did not restart properly. Showing logs:"
    docker compose logs opensearch --tail 50
    exit 1
fi

# Check if Ollama is running and pull multilingual embedding model
echo "Checking Ollama setup..."
if command -v ollama &> /dev/null; then
    echo "Ollama found. Pulling multilingual embedding model..."
    ollama pull paraphrase-multilingual
    echo "Multilingual embedding model ready!"
else
    echo "Warning: Ollama not found. Please install Ollama and run 'ollama pull paraphrase-multilingual'"
fi

# Create the index with multilingual mappings for keyword search and embeddings
echo "Creating the open-librarian-articles index..."
curl_result=$(curl -X PUT "http://localhost:9200/open-librarian-articles" -H "Content-Type: application/json" -d '{
  "settings": {
    "analysis": {
      "analyzer": {
        "multilingual": {
          "type": "standard"
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "lang": { "type": "keyword" },
      "title": {
        "type": "text",
        "analyzer": "standard",
        "fields": {
          "ko": { "type": "text", "analyzer": "nori" },
          "en": { "type": "text", "analyzer": "english" },
          "ja": { "type": "text", "analyzer": "kuromoji" },
          "zh": { "type": "text", "analyzer": "smartcn" },
          "es": { "type": "text", "analyzer": "spanish" },
          "fr": { "type": "text", "analyzer": "french" },
          "de": { "type": "text", "analyzer": "german" },
          "ru": { "type": "text", "analyzer": "russian" }
        }
      },
      "summary": {
        "type": "text",
        "analyzer": "standard",
        "fields": {
          "ko": { "type": "text", "analyzer": "nori" },
          "en": { "type": "text", "analyzer": "english" },
          "ja": { "type": "text", "analyzer": "kuromoji" },
          "zh": { "type": "text", "analyzer": "smartcn" },
          "es": { "type": "text", "analyzer": "spanish" },
          "fr": { "type": "text", "analyzer": "french" },
          "de": { "type": "text", "analyzer": "german" },
          "ru": { "type": "text", "analyzer": "russian" }
        }
      },
      "content": {
        "type": "text",
        "analyzer": "standard",
        "fields": {
          "ko": { "type": "text", "analyzer": "nori" },
          "en": { "type": "text", "analyzer": "english" },
          "ja": { "type": "text", "analyzer": "kuromoji" },
          "zh": { "type": "text", "analyzer": "smartcn" },
          "es": { "type": "text", "analyzer": "spanish" },
          "fr": { "type": "text", "analyzer": "french" },
          "de": { "type": "text", "analyzer": "german" },
          "ru": { "type": "text", "analyzer": "russian" }
        }
      },
      "title_embedding": {
        "type": "knn_vector",
        "dimension": 768,
        "method": {
          "name": "hnsw",
          "space_type": "cosinesimil",
          "engine": "nmslib"
        }
      },
      "summary_embedding": {
        "type": "knn_vector",
        "dimension": 768,
        "method": {
          "name": "hnsw",
          "space_type": "cosinesimil",
          "engine": "nmslib"
        }
      },
      "tags": { "type": "keyword" },
      "original_url": { "type": "keyword" },
      "author": { "type": "keyword" },
      "created_date": { "type": "date" }
    }
  }
}')

echo "Index creation response:"
echo "$curl_result"

echo "Setup complete!"
echo "OpenSearch is running at: http://localhost:9200"
echo "OpenSearch Dashboards is running at: http://localhost:5601"
