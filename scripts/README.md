# Scripts

This directory contains utility scripts for managing the Open Librarian application.

## rebuild-server.sh / rebuild-server.bat

Rebuilds and restarts only the `open-librarian-server` service without affecting other services (OpenSearch, OpenSearch Dashboards).

### Usage

**Linux/macOS:**
```bash
# Make executable (first time only)
chmod +x scripts/rebuild-server.sh

# Run the script
./scripts/rebuild-server.sh
```

**Windows:**
```cmd
scripts\rebuild-server.bat
```

### What it does

1. Stops the current `open-librarian-server` container
2. Removes the container (keeping the image layers cached)
3. Rebuilds the server image with latest code changes
4. Starts the new container
5. Waits for health check to pass
6. Provides status information

### Benefits

- Faster than rebuilding the entire stack
- Preserves OpenSearch data and state
- Automatically waits for health check
- Provides clear status feedback

### Requirements

- Docker and Docker Compose installed
- Must be run from the project root directory
- OpenSearch service should be running
