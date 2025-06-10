# Start OpenSearch services
Write-Host "Starting OpenSearch services..." -ForegroundColor Green
docker compose up -d --build

# Wait for OpenSearch to be ready
Write-Host "Waiting for OpenSearch to start..." -ForegroundColor Yellow
# Improved health check that waits for OpenSearch to report status as green or yellow
$maxAttempts = 30
$attempt = 0

while ($attempt -lt $maxAttempts) {
    Write-Host "Attempt $attempt of $maxAttempts..." -ForegroundColor Cyan
    
    # First check if the container is running
    $opensearchRunning = docker ps | Select-String "opensearch"
    if (-not $opensearchRunning) {
        Write-Host "OpenSearch container is not running. Checking logs:" -ForegroundColor Red
        docker compose logs opensearch --tail 25
        exit 1
    }
    
    # Try to check cluster health
    try {
        $healthResponse = Invoke-WebRequest -Uri "http://localhost:9200/_cat/health" -UseBasicParsing -TimeoutSec 5 -ErrorAction SilentlyContinue
        if ($healthResponse.StatusCode -eq 200) {
            $healthStatus = ($healthResponse.Content -split '\s+')[3]
            if ($healthStatus -eq "green" -or $healthStatus -eq "yellow") {
                Write-Host "OpenSearch is ready with status: $healthStatus" -ForegroundColor Green
                
                # Additional check for KNN plugin
                try {
                    $knnResponse = Invoke-WebRequest -Uri "http://localhost:9200/_plugins/_knn/stats" -UseBasicParsing -TimeoutSec 5 -ErrorAction SilentlyContinue
                    if ($knnResponse.StatusCode -ne 200) {
                        Write-Host "Warning: KNN plugin may not be properly installed or configured." -ForegroundColor Yellow
                        Write-Host "Checking OpenSearch plugins:" -ForegroundColor Yellow
                        docker exec opensearch /usr/share/opensearch/bin/opensearch-plugin list
                    } else {
                        Write-Host "KNN plugin is active and responding." -ForegroundColor Green
                    }
                } catch {
                    Write-Host "Warning: Could not check KNN plugin status." -ForegroundColor Yellow
                }
                
                break
            }
        }
    } catch {
        # Health check failed, continue waiting
    }
    
    $attempt++
    Start-Sleep -Seconds 3
}

if ($attempt -eq $maxAttempts) {
    Write-Host "OpenSearch did not become ready in time. Showing the most recent logs:" -ForegroundColor Red
    docker compose logs opensearch --tail 50
    exit 1
}

Write-Host "OpenSearch is ready!" -ForegroundColor Green

# Install required analysis plugins for multilingual support
Write-Host "Installing language analysis plugins..." -ForegroundColor Yellow
docker exec opensearch /usr/share/opensearch/bin/opensearch-plugin install analysis-nori
docker exec opensearch /usr/share/opensearch/bin/opensearch-plugin install analysis-kuromoji  
docker exec opensearch /usr/share/opensearch/bin/opensearch-plugin install analysis-smartcn

# Restart OpenSearch to load the plugins
Write-Host "Restarting OpenSearch to load plugins..." -ForegroundColor Yellow
docker compose restart opensearch

# Wait for OpenSearch to be ready again after restart
Write-Host "Waiting for OpenSearch to restart..." -ForegroundColor Yellow
$attempt = 0

while ($attempt -lt $maxAttempts) {
    Write-Host "Restart attempt $attempt of $maxAttempts..." -ForegroundColor Cyan
    
    try {
        $healthResponse = Invoke-WebRequest -Uri "http://localhost:9200/_cat/health" -UseBasicParsing -TimeoutSec 5 -ErrorAction SilentlyContinue
        if ($healthResponse.StatusCode -eq 200) {
            $healthStatus = ($healthResponse.Content -split '\s+')[3]
            if ($healthStatus -eq "green" -or $healthStatus -eq "yellow") {
                Write-Host "OpenSearch is ready again with status: $healthStatus" -ForegroundColor Green
                break
            }
        }
    } catch {
        # Health check failed, continue waiting
    }
    
    $attempt++
    Start-Sleep -Seconds 3
}

if ($attempt -eq $maxAttempts) {
    Write-Host "OpenSearch did not restart properly. Showing logs:" -ForegroundColor Red
    docker compose logs opensearch --tail 50
    exit 1
}

# Check if Ollama is running and pull multilingual embedding model
Write-Host "Checking Ollama setup..." -ForegroundColor Yellow
$ollamaExists = Get-Command ollama -ErrorAction SilentlyContinue
if ($ollamaExists) {
    Write-Host "Ollama found. Pulling multilingual embedding model..." -ForegroundColor Green
    ollama pull paraphrase-multilingual
    Write-Host "Multilingual embedding model ready!" -ForegroundColor Green
} else {
    Write-Host "Warning: Ollama not found. Please install Ollama and run 'ollama pull paraphrase-multilingual'" -ForegroundColor Yellow
}

# Create the index with multilingual mappings for keyword search and embeddings
Write-Host "Creating the open-librarian-articles index..." -ForegroundColor Yellow

$indexMapping = @{
    settings = @{
        analysis = @{
            analyzer = @{
                multilingual = @{
                    type = "standard"
                }
            }
        }
    }
    mappings = @{
        properties = @{
            lang = @{ type = "keyword" }
            title = @{
                type = "text"
                analyzer = "standard"
                fields = @{
                    ko = @{ type = "text"; analyzer = "nori" }
                    en = @{ type = "text"; analyzer = "english" }
                    ja = @{ type = "text"; analyzer = "kuromoji" }
                    zh = @{ type = "text"; analyzer = "smartcn" }
                    es = @{ type = "text"; analyzer = "spanish" }
                    fr = @{ type = "text"; analyzer = "french" }
                    de = @{ type = "text"; analyzer = "german" }
                    ru = @{ type = "text"; analyzer = "russian" }
                }
            }
            summary = @{
                type = "text"
                analyzer = "standard"
                fields = @{
                    ko = @{ type = "text"; analyzer = "nori" }
                    en = @{ type = "text"; analyzer = "english" }
                    ja = @{ type = "text"; analyzer = "kuromoji" }
                    zh = @{ type = "text"; analyzer = "smartcn" }
                    es = @{ type = "text"; analyzer = "spanish" }
                    fr = @{ type = "text"; analyzer = "french" }
                    de = @{ type = "text"; analyzer = "german" }
                    ru = @{ type = "text"; analyzer = "russian" }
                }
            }
            content = @{
                type = "text"
                analyzer = "standard"
                fields = @{
                    ko = @{ type = "text"; analyzer = "nori" }
                    en = @{ type = "text"; analyzer = "english" }
                    ja = @{ type = "text"; analyzer = "kuromoji" }
                    zh = @{ type = "text"; analyzer = "smartcn" }
                    es = @{ type = "text"; analyzer = "spanish" }
                    fr = @{ type = "text"; analyzer = "french" }
                    de = @{ type = "text"; analyzer = "german" }
                    ru = @{ type = "text"; analyzer = "russian" }
                }
            }
            title_embedding = @{
                type = "knn_vector"
                dimension = 768
                method = @{
                    name = "hnsw"
                    space_type = "cosinesimil"
                    engine = "nmslib"
                }
            }
            summary_embedding = @{
                type = "knn_vector"
                dimension = 768
                method = @{
                    name = "hnsw"
                    space_type = "cosinesimil"
                    engine = "nmslib"
                }
            }
            tags = @{ type = "keyword" }
            original_url = @{ type = "keyword" }
            author = @{ type = "keyword" }
            created_date = @{ type = "date" }
            registrar = @{ type = "keyword" }
        }
    }
} | ConvertTo-Json -Depth 10

try {
    $indexResponse = Invoke-RestMethod -Uri "http://localhost:9200/open-librarian-articles" -Method PUT -Body $indexMapping -ContentType "application/json"
    Write-Host "Index creation response:" -ForegroundColor Green
    Write-Host ($indexResponse | ConvertTo-Json -Depth 3) -ForegroundColor White
} catch {
    Write-Host "Index creation failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Response: $($_.Exception.Response)" -ForegroundColor Red
}

Write-Host "Setup complete!" -ForegroundColor Green
Write-Host "OpenSearch is running at: http://localhost:9200" -ForegroundColor Cyan
Write-Host "OpenSearch Dashboards is running at: http://localhost:5601" -ForegroundColor Cyan
