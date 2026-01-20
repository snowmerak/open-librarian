# Open Librarian 🔍

An intelligent search service that provides AI-powered answers based on stored articles with real-time communication and user authentication.

![open-librarian-main](/img/main_page.png)

## 📖 Overview

Open Librarian is a modern, AI-powered search platform that allows users to upload, store, and intelligently search through their articles. Built with Go and designed for performance, it combines traditional keyword search with vector-based semantic search to provide comprehensive answers to user queries.

## ✨ Key Features

### 🔍 Intelligent Search
- **Hybrid Search**: Combines keyword and vector-based semantic search for comprehensive results
- **AI-Powered Answers**: Get contextual answers generated from your article collection
- **Real-time Results**: Streaming search responses with WebSocket support
- **Multi-language Support**: Automatic language detection for 8 languages

### 🔐 User Management
- **Secure Authentication**: JWT-based auth with Argon2 password hashing
- **User Profiles**: Complete user management system
- **Article Ownership**: Users can manage their own uploaded content
- **Session Management**: Automatic token refresh and secure logout

### 📚 Article Management
- **Multi-format Support**: Support for PDF, DOCX, XLSX, TXT, and JSONL file processing
- **Metadata Support**: Author information, creation dates, and source URLs
- **Content Validation**: Real-time feedback during upload process
- **Unified Upload**: Single interface for all file types

### 💬 Interactive Chat
- **Conversational Interface**: Chat-like experience for queries and answers
- **Session History**: Access, load more, and manage past chat sessions
- **Source Citations**: Detailed view of sources used for answers with modal support
- **LLM Integration**: Enhanced query handling with OpenRouter and local tools

### 🌐 Multi-language Support
- **Internationalization**: Complete i18n framework
- **Supported Languages**: English, Korean, Chinese, Japanese, Spanish
- **Dynamic Switching**: Change languages without page reload
- **Persistent Settings**: Language preferences saved locally

### 🔗 API Access
- **External Integrations**: Read-only APIs for external systems and agents
- **Rate Limiting**: Built-in protection against abuse
- **Public Endpoints**: No authentication required for read operations
- **Developer Friendly**: Interactive API documentation powered by Scalar UI (`/swagger/`)

## 🏗 Architecture

Open Librarian follows a modern three-tier architecture designed for scalability and maintainability:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   Backend       │    │   Databases     │
│                 │    │                 │    │                 │
│ • Vanilla JS    │◄──►│ • Go (Chi)      │◄──►│ • MongoDB       │
│ • Tailwind CSS  │    │ • WebSocket     │    │ • OpenSearch    │
│ • IndexedDB     │    │ • JWT Auth      │    │ • Qdrant        │
│ • i18n Support  │    │ • Rate Limiting │    │ • Vector Store  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Technology Stack

| **Component** | **Technology** | **Purpose** |
|---------------|----------------|-------------|
| **Backend** | Go 1.21+ with Chi Router | High-performance HTTP server and routing |
| **Frontend** | Vanilla JavaScript + Tailwind CSS | Modern, responsive web interface |
| **Authentication** | JWT + Argon2 | Secure user authentication and authorization |
| **User Data** | MongoDB | User accounts, article metadata, and relationships |
| **Search Engine** | OpenSearch | Full-text search and keyword matching |
| **Vector Database** | Qdrant | Semantic similarity search and embeddings |
| **AI Integration** | Ollama | Local LLM inference for answer generation |
| **Documentation** | Scalar + OpenAPI 3.0 | Interactive API Reference and examples |
| **Real-time** | WebSocket | Live updates and streaming responses |
| **Containerization** | Docker + Docker Compose | Easy deployment and service orchestration |

### Data Flow

1. **Article Upload**: Users upload articles → Processed and stored in MongoDB → Indexed in OpenSearch → Embeddings stored in Qdrant
2. **Search Query**: User submits query → Hybrid search (keyword + vector) → AI generates contextual answer → Real-time response streaming
3. **Authentication**: Login request → JWT token generation → Token validation on protected endpoints

## 📋 System Requirements

### Prerequisites
- **Go**: Version 1.21 or higher
- **Docker**: Latest version with Docker Compose
- **Memory**: Minimum 4GB RAM (8GB recommended)
- **Storage**: At least 2GB free space
- **Network**: Internet connection for AI model downloads
## 🚀 Getting Started

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/snowmerak/open-librarian.git
cd open-librarian
```

2. **Set up environment variables**
```bash
cp .env.example .env
# Edit .env with your configuration
```

3. **Start services with Docker**
```bash
./scripts/setup-services.sh
```

4. **Run the application**
```bash
go run cmd/server/main.go
```

5. **Access the application**
- **Web Interface**: http://localhost:8080
- **API Documentation**: http://localhost:8080/swagger/
- **Health Check**: http://localhost:8080/health

### First Steps

1. **Create an account** through the web interface
2. **Upload your first articles** using the bulk upload feature
3. **Start searching** and get AI-powered answers
4. **Explore the API** using the Swagger documentation

## ⚙️ Configuration

### Environment Variables

Configure the following environment variables in your `.env` file:

```bash
# Server Configuration
PORT=8080                              # Application port
JWT_SECRET=your-super-secret-jwt-key   # JWT signing secret (change this!)

# Database Connections
MONGODB_URI=mongodb://localhost:27017/open_librarian  # MongoDB connection string
OPENSEARCH_URL=http://localhost:9200                  # OpenSearch cluster URL
QDRANT_HOST=localhost                                 # Qdrant vector database host
QDRANT_PORT=6334                                     # Qdrant port

# AI Service
OLLAMA_URL=http://localhost:11434      # Ollama API endpoint for LLM inference
```

### Database Setup

#### MongoDB Configuration
```bash
# Connect to MongoDB
mongosh

# Create database and user
use open_librarian
db.createUser({
  user: "librarian",
  pwd: "your-secure-password",
  roles: ["readWrite", "dbAdmin"]
})

# Create indexes for better performance
db.users.createIndex({ "email": 1 }, { unique: true })
db.articles.createIndex({ "author_id": 1 })
db.articles.createIndex({ "created_at": -1 })
```

#### OpenSearch Setup
OpenSearch will automatically create necessary indexes when first used. No manual configuration required.

#### Qdrant Setup
Qdrant collections are automatically created when the application starts. Default configuration uses cosine similarity for vector search.

## 🔌 Usage Guide

### Web Interface

The web interface provides an intuitive way to interact with Open Librarian:

- **Dashboard**: Overview of your articles and recent activity
- **Search**: Intelligent search with real-time AI answers
- **Upload**: Bulk article upload with progress tracking
- **Profile**: User account management and settings

### Article Upload Formats

Open Librarian supports JSONL (JSON Lines) format for bulk uploads:

```json
{"title": "Article Title", "content": "Article content...", "author": "Author Name", "url": "https://example.com"}
{"title": "Another Article", "content": "More content...", "author": "Different Author"}
```

### Search Capabilities

- **Keyword Search**: Traditional full-text search across article content
- **Semantic Search**: Vector-based similarity search for conceptual matches
- **Hybrid Results**: Combines both approaches for comprehensive results
- **AI Answers**: Contextual answers generated from relevant articles

### Real-time Features

- **Live Search**: Get results as you type
- **Upload Progress**: Real-time feedback during article processing
- **Streaming Answers**: AI responses delivered in real-time
- **WebSocket Fallback**: Automatic fallback to HTTP when WebSocket unavailable

## 📖 API Reference

Open Librarian provides a comprehensive REST API for all operations. Full documentation is available at `/swagger/` when the server is running.

### Key API Endpoints

#### Authentication
| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/users/` | User registration |
| `POST` | `/api/v1/users/auth` | User login |
| `POST` | `/api/v1/users/refresh` | Token refresh |
| `GET` | `/api/v1/users/me` | Get current user profile |

#### Article Management
| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| `POST` | `/api/v1/articles` | Add single article | ✅ |
| `POST` | `/api/v1/articles/bulk` | Bulk article upload | ✅ |
| `DELETE` | `/api/v1/articles/{id}` | Delete article | ✅ (Owner only) |
| `GET` | `/api/v1/articles/{id}` | Get article details | ❌ |

#### Search & AI
| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/search` | Hybrid search with AI answers |
| `GET` | `/api/v1/search/keyword` | Keyword-only search |
| `WS` | `/api/v1/search/ws` | WebSocket real-time search |

#### External APIs (Public Access)
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/external/articles` | List articles for external agents |
| `GET` | `/api/v1/external/articles/{id}` | Get article by ID |
| `GET` | `/api/v1/external/search` | Public search endpoint |

## 💡 Example Usage

### Basic Search Example
```bash
# Simple search request
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "machine learning algorithms", "size": 5}'
```

### Article Upload Example
```bash
# Upload a single article (requires authentication)
curl -X POST http://localhost:8080/api/v1/articles \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "title": "Introduction to AI",
    "content": "Artificial Intelligence is...",
    "author": "John Doe",
    "url": "https://example.com/ai-intro"
  }'
```

### WebSocket Search Example
```javascript
// Real-time search with WebSocket
const ws = new WebSocket('ws://localhost:8080/api/v1/search/ws');

ws.onopen = () => {
    ws.send(JSON.stringify({
        query: "Your search query",
        size: 10
    }));
};

ws.onmessage = (event) => {
    const message = JSON.parse(event.data);
    switch(message.type) {
        case 'status':
            console.log('Status:', message.data);
            break;
        case 'sources':
            console.log('Sources:', message.data);
            break;
        case 'answer':
            console.log('AI Answer chunk:', message.data);
            break;
        case 'done':
            console.log('Search completed');
            break;
    }
};
```

## 🔧 Development

### Project Structure
```
open-librarian/
├── cmd/server/          # Application entry point
├── lib/                 # Core libraries
│   ├── aggregator/      # Main application logic
│   ├── client/          # Database and service clients
│   └── util/            # Utility functions
├── docs/                # Documentation and examples
├── scripts/             # Build and setup scripts
└── docker-compose.yaml  # Container orchestration
```

### Building from Source
```bash
# Build the application
go build -o bin/open-librarian cmd/server/main.go

# Run with development settings
go run cmd/server/main.go
```

### Docker Development
```bash
# Run with docker-compose
./scripts/setup-services.sh

# Rebuild Server
./scripts/rebuild-server.sh
```

## 🛡️ Security & Performance

### Security Features
- **Password Security**: Argon2 hashing with salt for maximum protection
- **JWT Authentication**: Stateless, secure token-based authentication
- **CORS Protection**: Configurable cross-origin resource sharing policies
- **Rate Limiting**: Built-in API abuse prevention with configurable thresholds
- **Input Validation**: Comprehensive request validation and sanitization
- **Secure Headers**: Security headers for web interface protection

### Performance Optimizations
- **Vector Caching**: Intelligent caching of embeddings and search results
- **Database Indexing**: Optimized MongoDB indexes for fast queries
- **Connection Pooling**: Efficient database connection management
- **Streaming Responses**: Reduced latency with real-time data streaming
- **Compression**: Response compression for bandwidth optimization

## 🌐 Multilingual Support

Open Librarian provides comprehensive internationalization:

### Supported Languages
- **English** (en) - Default language
- **Korean** (ko) - 한국어 지원
- **Chinese** (zh) - 中文支持
- **Japanese** (ja) - 日本語サポート
- **Spanish** (es) - Soporte en español

### Features
- **Automatic Detection**: Language detection for uploaded content
- **Dynamic Switching**: Change interface language without reload
- **Persistent Settings**: User language preferences saved locally
- **Content-Aware Search**: Search results optimized for content language

## 📊 Monitoring & Health Checks

### Health Monitoring
- **Service Health**: Real-time status of all connected services
- **Database Connectivity**: MongoDB, OpenSearch, and Qdrant status
- **AI Service Status**: Ollama availability and model status
- **Performance Metrics**: Response times and system resource usage

### Logging
- **Structured Logging**: JSON-formatted logs for easy parsing
- **Request Tracing**: Complete request/response cycle tracking
- **Error Tracking**: Comprehensive error reporting and stack traces
- **Audit Logging**: User actions and system events tracking

## 🚀 Deployment

### Production Deployment

For production environments, consider the following:

```bash
# Production environment variables
export NODE_ENV=production
export JWT_SECRET=$(openssl rand -base64 32)
export MONGODB_URI="mongodb://username:password@your-mongo-host:27017/open_librarian"

# Use production-ready containers
docker-compose -f docker-compose.prod.yml up -d
```

### Docker Deployment
```bash
# Build and deploy with Docker
docker build -t open-librarian:latest .
docker run -d -p 8080:8080 --env-file .env open-librarian:latest
```

### Scaling Considerations
- **Load Balancing**: Use nginx or similar for multiple instances
- **Database Scaling**: Consider MongoDB replica sets for high availability
- **Vector Database**: Qdrant clustering for large-scale deployments
- **AI Processing**: Multiple Ollama instances for concurrent AI requests

## 📅 Recent Updates

### 2026-01-21
- **API**: Migrated documentation to Scalar UI for better interactivity
- **Upload**: Added unified file upload support for PDF, DOCX, XLSX, and TXT formats
- **UI**: Improved accessibility (Aria labels) and visual enhancements (Glass effect)
- **Fixes**: Resolved issues with article form initialization and search scoring

### 2026-01-20
- **Chat**: Implemented full chat interface with history and session management
- **Search**: Added WebSocket streaming for real-time search results
- **LLM**: Integrated OpenRouter and local tool support
- **Infrastructure**: Added Docker tasks for server maintenance

## 🤝 Contributing

We welcome contributions to Open Librarian! Here's how you can help:

### Development Setup
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Set up the development environment
4. Make your changes and add tests
5. Commit your changes (`git commit -m 'Add some amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Code Guidelines
- Follow Go best practices and formatting (`go fmt`, `go vet`)
- Write comprehensive tests for new features
- Update documentation for API changes
- Ensure all tests pass before submitting

### Issue Reporting
- Use GitHub Issues for bug reports and feature requests
- Provide detailed reproduction steps for bugs
- Include system information and logs when relevant

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

Open Librarian is built on the shoulders of amazing open-source projects:

- **[OpenSearch](https://opensearch.org/)** - Powerful full-text search and analytics engine
- **[Qdrant](https://qdrant.tech/)** - High-performance vector similarity search engine
- **[Ollama](https://ollama.ai/)** - Local large language model inference platform
- **[MongoDB](https://www.mongodb.com/)** - Flexible document database for user data
- **[Go Chi](https://go-chi.io/)** - Lightweight HTTP router and middleware framework
- **[Lingua-Go](https://github.com/pemistahl/lingua-go)** - Natural language detection library

---

**Ready to get started?** Follow the [Getting Started](#-getting-started) guide to set up your own Open Librarian instance!
